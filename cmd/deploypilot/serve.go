package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/metrics"
	"github.com/Yogdunana/deploypilot/internal/sandbox"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// localExecutor runs commands on the local machine.
type localExecutor struct{}

func (e *localExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	out, err := c.CombinedOutput()
	return string(out), err
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the DeployPilot MCP server",
	Long:  "Start the MCP server that exposes deployment tools to AI assistants.",
	RunE: func(cmd *cobra.Command, args []string) error {
		transport, _ := cmd.Flags().GetString("transport")

		if transport == "sse" {
			return fmt.Errorf("SSE transport is not yet implemented. Use --transport stdio")
		}

		// Load config
		cfg, err := config.Load("")
		if err != nil {
			slog.Warn("config load failed, using defaults", "error", err)
			cfg = config.DefaultConfig()
		}

		// Ensure data directory exists
		dataDir := filepath.Dir(cfg.Database.DSN)
		if dataDir != "" && dataDir != "." {
			if err := os.MkdirAll(dataDir, 0750); err != nil {
				return fmt.Errorf("create data directory: %w", err)
			}
		}

		// Open database
		db, err := database.Connect(cfg.Database.Type, cfg.Database.DSN)
		if err != nil {
			return fmt.Errorf("database connect: %w", err)
		}

		// Migrate
		if err := database.Migrate(db); err != nil {
			return fmt.Errorf("database migrate: %w", err)
		}

		// Seed
		if err := database.Seed(db); err != nil {
			if !errors.Is(err, gorm.ErrDuplicatedKey) {
				slog.Warn("database seed error", "error", err)
			}
		}

		// Create executor + sandbox + bridge
		var executor deployer.CommandExecutor = &localExecutor{}
		sb := sandbox.New(sandbox.DefaultConfig())
		sandboxedExecutor := deployer.NewSandboxExecutor(executor, sb)

		// Load or generate encryption key
		encKey, err := crypto.LoadEncryptionKeyFromEnv()
		if err != nil {
			return fmt.Errorf("encryption key: %w", err)
		}
		if encKey == nil {
			encKey = crypto.NewEncryptionKey()
			slog.Warn("DEPLOYPILOT_ENCRYPTION_KEY not set, generated a temporary key (credentials will be lost on restart)")
		}
		bridge := service.NewBridge(db, sandboxedExecutor, encKey, nil)

		// Create MCP server
		mcpServer := mcp.NewServer(bridge)

		// Initialize and start Prometheus metrics server
		metrics.Init()
		go func() {
			metricsPort := strconv.Itoa(cfg.Monitor.MetricsPort)
			slog.Info("starting metrics server", "port", metricsPort)
			metricsServer := &http.Server{
				Addr:         ":" + metricsPort,
				Handler:      metrics.Handler(),
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
			}
			if err := metricsServer.ListenAndServe(); err != nil {
				slog.Error("metrics server failed", "error", err)
			}
		}()

		slog.Info("DeployPilot MCP server starting", "version", version, "transport", "stdio")
		return server.ServeStdio(mcpServer)
	},
}

func init() {
	serveCmd.Flags().StringP("transport", "t", "stdio", "transport type: stdio, sse")
	serveCmd.Flags().IntP("port", "p", 8080, "port for SSE transport")
	rootCmd.AddCommand(serveCmd)
}
