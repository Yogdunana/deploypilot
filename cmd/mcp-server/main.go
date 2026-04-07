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
	"gorm.io/gorm"
)

// version is set via -ldflags at build time.
var version = "dev"

// localExecutor runs commands on the local machine.
type localExecutor struct{}

func (e *localExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	out, err := c.CombinedOutput()
	return string(out), err
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load config (optional — falls back to defaults)
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

	// Seed (ignore "already exists" errors)
	if err := database.Seed(db); err != nil {
		// Seed may fail if data already exists, that's OK
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			log.Printf("warning: database seed: %v", err)
		}
	}

	// Create executor (local Docker by default)
	var executor deployer.CommandExecutor = &localExecutor{}

	// If SSH env vars are set, use SSH executor
	if sshHost := os.Getenv("DEPLOYPILOT_SSH_HOST"); sshHost != "" {
		sshPort := os.Getenv("DEPLOYPILOT_SSH_PORT")
		if sshPort == "" {
			sshPort = "22"
		}
		sshUser := os.Getenv("DEPLOYPILOT_SSH_USER")
		if sshUser == "" {
			sshUser = "root"
		}
		log.Printf("SSH mode: %s@%s:%s", sshUser, sshHost, sshPort)
		// SSH executor will be used when server_id is specified in deploy
		// For now, local executor is default; SSH is per-server
		_ = sshHost // used in future per-server routing
	}

	// Create bridge deployer
	bridge := service.NewBridge(db, executor)

	// Create MCP server
	mcpServer := mcp.NewServer(bridge)

	// Start stdio transport
	log.Printf("DeployPilot MCP server v%s starting (stdio)", version)
	return server.ServeStdio(mcpServer)
}
