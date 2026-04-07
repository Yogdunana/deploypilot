package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/mcp"
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
			log.Printf("warning: config load failed, using defaults: %v", err)
			cfg = &config.Config{}
		}

		// Ensure data directory exists
		dataDir := filepath.Dir(cfg.Database.DSN)
		if dataDir != "" && dataDir != "." {
			os.MkdirAll(dataDir, 0755)
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
				log.Printf("warning: database seed: %v", err)
			}
		}

		// Create executor + bridge
		var executor deployer.CommandExecutor = &localExecutor{}
		bridge := service.NewBridge(db, executor)

		// Create MCP server
		mcpServer := mcp.NewServer(bridge)

		log.Printf("DeployPilot MCP server v%s starting (stdio)", version)
		return server.ServeStdio(mcpServer)
	},
}

func init() {
	serveCmd.Flags().StringP("transport", "t", "stdio", "transport type: stdio, sse")
	serveCmd.Flags().IntP("port", "p", 8080, "port for SSE transport")
	rootCmd.AddCommand(serveCmd)
}
