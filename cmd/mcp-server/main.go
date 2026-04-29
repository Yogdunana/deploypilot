package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/service"
	appversion "github.com/Yogdunana/deploypilot/internal/version"
	"github.com/mark3labs/mcp-go/server"
	"gorm.io/gorm"
)

// version is set via -ldflags at build time (kept for --version flag compatibility).
var version = "dev"

// localExecutor runs commands on the local machine.
type localExecutor struct{}

func (e *localExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	out, err := c.CombinedOutput()
	return string(out), err
}

func main() {
	// Flags
	showVersion := flag.Bool("version", false, "print version and exit")
	showHelp := flag.Bool("help", false, "print help and exit")
	configPath := flag.String("config", "", "path to config.yaml file")
	dbDriver := flag.String("db-driver", "", "database driver: sqlite, postgres (default: sqlite)")
	dbDSN := flag.String("db-dsn", "", "database DSN (default: ./data/deploypilot.db)")
	flag.BoolVar(showHelp, "h", false, "print help and exit")
	flag.Parse()

	if *showHelp {
		fmt.Fprintf(os.Stderr, "DeployPilot MCP Server v%s\n\n", appversion.Version)
		fmt.Fprintf(os.Stderr, "Usage: mcp-server [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  DEPLOYPILOT_DATABASE_TYPE    Database driver (default: sqlite)\n")
		fmt.Fprintf(os.Stderr, "  DEPLOYPILOT_DATABASE_DSN     Database DSN (default: ./data/deploypilot.db)\n")
		fmt.Fprintf(os.Stderr, "  DEPLOYPILOT_ENCRYPTION_KEY   AES-256 key: base64 (recommended, e.g. openssl rand -base64 32) or raw 32-byte string\n")
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("mcp-server %s\n", appversion.Version)
		os.Exit(0)
	}

	if err := run(*configPath, *dbDriver, *dbDSN); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(configFilePath, cliDriver, cliDSN string) error {
	// Load config (optional — falls back to defaults)
	cfg, err := config.Load(configFilePath)
	if err != nil {
		slog.Warn("config load failed, using defaults", "error", err)
		cfg = config.DefaultConfig()
	}

	// CLI flags override config/env
	if cliDriver != "" {
		cfg.Database.Type = cliDriver
	}
	if cliDSN != "" {
		cfg.Database.DSN = cliDSN
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

	// Seed (ignore "already exists" errors)
	if err := database.Seed(db); err != nil {
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			slog.Warn("database seed error", "error", err)
		}
	}

	// Create executor (local Docker by default)
	var executor deployer.CommandExecutor = &localExecutor{}

	// Load or generate encryption key
	encKey, err := crypto.LoadEncryptionKeyFromEnv()
	if err != nil {
		return fmt.Errorf("encryption key: %w", err)
	}
	if encKey == nil {
		encKey = crypto.NewEncryptionKey()
		slog.Warn("DEPLOYPILOT_ENCRYPTION_KEY not set, generated a temporary key (credentials will be lost on restart)")
	}
	bridge := service.NewBridge(db, executor, encKey, nil)
	// Create MCP server
	mcpServer := mcp.NewServer(bridge)

	// Start stdio transport
	slog.Info("DeployPilot MCP server starting", "version", appversion.Version, "transport", "stdio")
	slog.Info("database configured", "type", cfg.Database.Type, "dsn", cfg.Database.DSN)
	return server.ServeStdio(mcpServer)
}
