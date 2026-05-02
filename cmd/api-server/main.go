// @title           DeployPilot API
// @version         1.0
// @description     AI-native deployment platform REST API
// @termsOfService  http://swagger.io/terms/

// @contact.name   DeployPilot
// @contact.url    https://github.com/Yogdunana/deploypilot

// @license.name  MIT
// @license.url   https://github.com/Yogdunana/deploypilot/blob/main/LICENSE

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token with "Bearer " prefix

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	_ "github.com/Yogdunana/deploypilot/docs/swagger"
	"github.com/Yogdunana/deploypilot/internal/agent"
	"github.com/Yogdunana/deploypilot/internal/api"
	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/backup"
	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/database"
	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/metrics"
	"github.com/Yogdunana/deploypilot/internal/server"
	"github.com/Yogdunana/deploypilot/internal/service"
	appversion "github.com/Yogdunana/deploypilot/internal/version"
	"github.com/redis/go-redis/v9"
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
	showVersion := flag.Bool("version", false, "print version and exit")
	configPath := flag.String("config", "", "path to config.yaml file")
	dbDriver := flag.String("db-driver", "", "database driver: sqlite, postgres")
	dbDSN := flag.String("db-dsn", "", "database DSN")
	addr := flag.String("addr", "", "listen address (e.g. 0.0.0.0:8080)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("api-server %s\n", appversion.Version)
		os.Exit(0)
	}

	if err := run(*configPath, *dbDriver, *dbDSN, *addr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(configFilePath, cliDriver, cliDSN, cliAddr string) error {
	// Load config
	cfg, err := config.Load(configFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: config load failed, using defaults: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// Inject JWT secret from config into environment for jwt.go's os.Getenv("JWT_SECRET")
	if cfg.Auth.JWTSecret != "" && os.Getenv("JWT_SECRET") == "" {
		if len(cfg.Auth.JWTSecret) < 16 {
			return fmt.Errorf("auth.jwt_secret in config must be at least 16 characters, current length: %d", len(cfg.Auth.JWTSecret))
		}
		_ = os.Setenv("JWT_SECRET", cfg.Auth.JWTSecret)
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
		if err := os.MkdirAll(dataDir, 0750); err != nil {
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

	// Initialize event bus, token blacklist, and cache (Redis if available, otherwise in-memory)
	var eventBus service.EventBus
	var typedBus service.TypedEventBus
	var tokenBlacklist auth.TokenBlacklist
	var cache service.Cache
	var rdb *redis.Client
	if cfg.Redis.Addr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			slog.Warn("Redis unavailable, falling back to in-memory implementations", "error", err)
			eventBus = service.NewInMemoryEventBus()
			typedBus = service.NewInMemoryTypedEventBus()
			tokenBlacklist = auth.NewMemoryTokenBlacklist()
			cache = service.NewMemoryCache("dp:")
		} else {
			eventBus = service.NewRedisEventBus(rdb)
			typedBus = service.NewRedisTypedEventBus(rdb, "")
			tokenBlacklist = auth.NewRedisTokenBlacklist(rdb)
			cache = service.NewRedisCache(rdb, "dp:")
			slog.Info("using Redis for event bus, typed event bus, token blacklist, and cache")
		}
	} else {
		eventBus = service.NewInMemoryEventBus()
		typedBus = service.NewInMemoryTypedEventBus()
		tokenBlacklist = auth.NewMemoryTokenBlacklist()
		cache = service.NewMemoryCache("dp:")
	}

	// Initialize agent tunnel manager
	tunnelManager := agent.NewTunnelManager()
	tunnelManager.StartCleanup(5 * time.Minute)

	bridge := service.NewBridge(db, executor, encKey, eventBus)
	bridge.SetCache(cache)
	bridge.SetTypedBus(typedBus)
	bridge.TunnelManager = tunnelManager
	bridge.UpgradeSvc = service.NewUpgradeService("")

	// Initialize scheduler
	scheduler := service.NewScheduler(db, bridge)
	bridge.Scheduler = scheduler
	scheduler.Start(context.Background())

	// Initialize monitor scheduler for uptime/heartbeat checks (Phase 6.1-6.2)
	monitorSvc := service.NewMonitorService(db)
	monitorScheduler := service.NewMonitorScheduler(monitorSvc)
	monitorScheduler.Start(context.Background())

	// Connect monitor scheduler to WebSocket hub for real-time broadcasting
	monitorScheduler.SetUpdateCallback(func(checkType string, data interface{}) {
		if hub := api.GetGlobalMonitorAPI(); hub != nil {
			hub.GetMonitorHub().Broadcast(map[string]interface{}{
				"type": checkType,
				"data": data,
			})
		}
	})

	// Apply brute-force config from configuration file
	bf := cfg.BruteForce
	bridge.SetBruteForceConfig(service.BruteForceConfigFromMap(
		bf.MaxAttempts, bf.IPMaxAttempts,
		bf.LockoutDuration, bf.WindowDuration,
		bf.BaseDelay, bf.MaxDelay, bf.IPLockoutDuration,
		bf.ProgressiveDelay,
	))

	// Determine listen address
	listenAddr := cliAddr
	if listenAddr == "" {
		listenAddr = fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	}

	// Initialize OAuth service (if providers are configured)
	var oauthSvc *service.OAuthService
	if len(cfg.Auth.OAuthProviders) > 0 {
		oauthSvc = service.NewOAuthService(db, cfg.Auth.OAuthProviders)
		slog.Info("OAuth service initialized", "providers", len(cfg.Auth.OAuthProviders))
	}

	// Initialize backup service with optional cloud storage
	backupInterval, _ := time.ParseDuration(cfg.Backup.Interval)
	if backupInterval == 0 {
		backupInterval = 6 * time.Hour
	}
	backupSvc := backup.New(backup.Config{
		Enabled:        cfg.Backup.Enabled,
		Interval:       backupInterval,
		RetentionCount: cfg.Backup.RetentionCount,
		RetentionDays:  cfg.Backup.RetentionDays,
		BackupDir:      cfg.Backup.BackupDir,
	}, db, cfg.Database.Type, cfg.Database.DSN)

	// Configure cloud storage if enabled
	if cfg.Backup.CloudEnabled {
		cloudStorage, err := backup.NewS3Storage(backup.StorageConfig{
			Enabled:   true,
			Type:      cfg.Backup.CloudType,
			Endpoint:  cfg.Backup.CloudEndpoint,
			Region:    cfg.Backup.CloudRegion,
			Bucket:    cfg.Backup.CloudBucket,
			Prefix:    cfg.Backup.CloudPrefix,
			AccessKey: cfg.Backup.CloudAccessKey,
			SecretKey: cfg.Backup.CloudSecretKey,
			UseSSL:    cfg.Backup.CloudUseSSL,
			Encrypt:   cfg.Backup.CloudEncrypt,
		})
		if err != nil {
			slog.Warn("failed to initialize cloud storage, local backup only", "error", err)
		} else {
			backupSvc.SetStorage(cloudStorage)
			if cfg.Backup.CloudEncrypt && encKey != nil {
				backupSvc.SetEncryptionKey(encKey)
			}
			slog.Info("cloud backup storage enabled", "type", cfg.Backup.CloudType, "bucket", cfg.Backup.CloudBucket)
		}
	}
	backupSvc.Start()

	srv := server.New(listenAddr, db, bridge, cfg, tokenBlacklist, oauthSvc, rdb, backupSvc)

	// Initialize and start Prometheus metrics server
	metrics.Init()
	metricsServer := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Monitor.MetricsPort),
		Handler:      metrics.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		slog.Info("starting metrics server", "port", cfg.Monitor.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	// Start the API server in a goroutine so we can handle shutdown signals
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Run()
	}()

	slog.Info("DeployPilot API server starting", "version", appversion.Version, "addr", listenAddr)
	slog.Info("database configured", "type", cfg.Database.Type, "dsn", cfg.Database.DSN)

	// Wait for shutdown signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		slog.Info("received shutdown signal", "signal", sig.String())
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	slog.Info("starting graceful shutdown (timeout: 15s)...")

	// 1. Shutdown metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.Warn("metrics server shutdown error", "error", err)
	}

	// 2. Shutdown main API server (includes WebSocket hub closure)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("API server shutdown error", "error", err)
	}

	// 3. Stop monitor scheduler
	monitorScheduler.Stop()

	// 4. Stop scheduler
	if bridge.Scheduler != nil {
		bridge.Scheduler.Stop()
	}

	// 5. Stop monitor if running
	if bridge.Monitor != nil {
		bridge.Monitor.Stop()
	}
	if backupSvc != nil {
		backupSvc.Stop()
	}

	// 6. Close event bus
	if eventBus != nil {
		if err := eventBus.Close(); err != nil {
			slog.Warn("event bus close error", "error", err)
		}
	}
	if typedBus != nil {
		if err := typedBus.Close(); err != nil {
			slog.Warn("typed event bus close error", "error", err)
		}
	}

	// 7. Close cache (in-memory cache only; Redis cache is closed via rdb)
	if cache != nil && rdb == nil {
		if err := cache.Close(); err != nil {
			slog.Warn("cache close error", "error", err)
		}
	}

	// 8. Close database connection
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			slog.Warn("database close error", "error", err)
		}
	}

	slog.Info("graceful shutdown complete")
	return nil
}
