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
	"time"

	"github.com/Yogdunana/deploypilot/internal/agent"
	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/server"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/redis/go-redis/v9"
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
	showVersion := flag.Bool("version", false, "print version and exit")
	dbDriver := flag.String("db-driver", "", "database driver: sqlite, postgres")
	dbDSN := flag.String("db-dsn", "", "database DSN")
	addr := flag.String("addr", "", "listen address (e.g. 0.0.0.0:8080)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("api-server %s\n", version)
		os.Exit(0)
	}

	if err := run(*dbDriver, *dbDSN, *addr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(cliDriver, cliDSN, cliAddr string) error {
	// Load config
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: config load failed, using defaults: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// Initialize structured logger
	config.InitLogger(cfg.Log)

	// CLI flags override config
	if cliDriver != "" {
		cfg.Database.Type = cliDriver
	}
	if cliDSN != "" {
		cfg.Database.DSN = cliDSN
	}

	// Ensure data directory exists
	dataDir := filepath.Dir(cfg.Database.DSN)
	if dataDir != "" && dataDir != "." {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("create data directory: %w", err)
		}
	}

	// Connect database
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

	// Create executor
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

	// Initialize event bus (Redis if available, otherwise in-memory)
	var eventBus service.EventBus
	if cfg.Redis.Addr != "" {
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			slog.Warn("Redis unavailable, falling back to in-memory event bus", "error", err)
			eventBus = service.NewInMemoryEventBus()
		} else {
			eventBus = service.NewRedisEventBus(rdb)
			slog.Info("using Redis Pub/Sub event bus")
		}
	} else {
		eventBus = service.NewInMemoryEventBus()
	}

	// Initialize agent tunnel manager
	tunnelManager := agent.NewTunnelManager()
	tunnelManager.StartCleanup(5 * time.Minute)

	bridge := service.NewBridge(db, executor, encKey, eventBus)
	bridge.TunnelManager = tunnelManager

	// Determine listen address
	listenAddr := cliAddr
	if listenAddr == "" {
		listenAddr = fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	}

	srv := server.New(listenAddr, db, bridge, cfg)

	slog.Info("DeployPilot API server starting", "version", version, "addr", listenAddr)
	slog.Info("database configured", "type", cfg.Database.Type, "dsn", cfg.Database.DSN)
	return srv.Run()
}
