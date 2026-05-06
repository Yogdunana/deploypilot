package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/api"
	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/backup"
	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/i18n"
	"github.com/Yogdunana/deploypilot/internal/middleware"
	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/Yogdunana/deploypilot/internal/service"
	webfs "github.com/Yogdunana/deploypilot/web"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Server wraps a Gin engine with application dependencies.
type Server struct {
	router  *gin.Engine
	httpSrv *http.Server
	db      *gorm.DB
	bridge  *service.Bridge
	wsHub   *api.WSHub
	addr    string
}

// New creates a new API server with the given address, database, bridge, and config.
func New(addr string, db *gorm.DB, bridge *service.Bridge, cfg *config.Config, blacklist auth.TokenBlacklist, oauthSvc *service.OAuthService, rdb *redis.Client, backupSvc *backup.Service, eventPluginMgr *plugin.EventPluginManager) *Server {
	r := gin.Default()

	// Request tracing — must be first middleware
	r.Use(middleware.RequestTracing())

	// Security middleware
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   cfg.Server.CORSAllowedOrigins,
		AllowedMethods:   cfg.Server.CORSAllowedMethods,
		AllowedHeaders:   cfg.Server.CORSAllowedHeaders,
		AllowCredentials: cfg.Server.CORSAllowCredentials,
		ExposeHeaders:    cfg.Server.CORSExposeHeaders,
		MaxAge:           cfg.Server.CORSMaxAge,
	}))

	// Panel security hardening (Phase 4.0)
	r.Use(middleware.SecurityEntrance(cfg.Security.SecurityEntrance))
	r.Use(middleware.DomainBinding(cfg.Security.AllowedDomains))
	r.Use(middleware.IPWhitelist(cfg.Security.AllowedIPs))

	// i18n locale middleware (early, before auth and rate limiting)
	r.Use(i18n.LocaleMiddleware())

	// Rate limiting
	rateLimiter := middleware.NewRateLimiter(
		cfg.Security.RateLimitDefault,
		cfg.Security.RateLimitOwner,
		cfg.Security.RateLimitAdmin,
		cfg.Security.RateLimitDev,
		cfg.Security.RateLimitViewer,
	)
	r.Use(rateLimiter.Middleware())

	// Audit logging — optionally enable external file writer
	var auditSvc *service.AuditService
	if cfg.Audit.ExternalLogPath != "" {
		fileWriter, err := service.NewFileAuditWriter(cfg.Audit.ExternalLogPath)
		if err != nil {
			slog.Warn("failed to create external audit writer", "error", err)
			auditSvc = service.NewAuditService(db)
		} else {
			auditSvc = service.NewAuditService(db, fileWriter)
			slog.Info("external audit logging enabled", "path", cfg.Audit.ExternalLogPath)
		}
	} else {
		auditSvc = service.NewAuditService(db)
	}
	r.Use(middleware.AuditMiddleware(auditSvc))

	// WebSocket hub
	wsHub := api.NewWSHub(rdb)
	go wsHub.Run()

	// Register audit log real-time notification callback
	auditSvc.OnRecord(func(entry service.AuditEntry) {
		wsHub.Broadcast("audit", api.WSMessage{
			Type: "audit_log",
			Data: map[string]interface{}{
				"action":        entry.Action,
				"resource_type": entry.ResourceType,
				"resource_id":   entry.ResourceID,
				"username":      entry.Username,
				"log_type":      entry.LogType,
				"ip_address":    entry.IPAddress,
			},
		})
	})

	// API Key service
	keySvc := service.NewAPIKeyService(db)

	// Per-user IP whitelist service and middleware
	ipWhitelistSvc := service.NewIPWhitelistService(db)
	r.Use(middleware.UserIPWhitelistMiddleware(ipWhitelistSvc))

	// Device binding middleware (flags new devices, does not block)
	deviceSvc := service.NewDeviceService(db)
	r.Use(middleware.DeviceCheckMiddleware(deviceSvc))

	// Audit verification & compliance (Issue #155)
	auditSecretKey := []byte(cfg.Auth.JWTSecret)
	if len(auditSecretKey) == 0 {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			slog.Warn("failed to generate random audit key, using fallback")
			auditSecretKey = []byte("deploypilot-fallback-audit-key-change-me")
		} else {
			auditSecretKey = key
			slog.Warn("JWT_SECRET not set, using random audit chain key (audit chain will break on restart)")
		}
	}
	api.SetAuditVerificationAPI(api.NewAuditVerificationAPI(db, &cfg.Audit, auditSecretKey))

	// Set encryption key for 2FA TOTP secret encryption
	api.SetEncryptionKey(bridge.EncryptionKey)

	// Set password validator for registration and password changes
	api.SetPasswordValidator(middleware.NewPasswordValidator(cfg.Security))

	// Store security config for API access
	api.SetSecurityConfig(&cfg.Security)

	// Set audit service for auth event logging
	if auditSvc != nil {
		api.SetAuditServiceForAuth(auditSvc)
	}

	// Initialize refresh token store
	if rdb != nil {
		api.SetRefreshTokenStore(auth.NewRedisRefreshTokenStore(rdb))
	} else {
		memRefreshStore := auth.NewMemoryRefreshTokenStore()
		memRefreshStore.StartCleanup(context.Background(), 10*time.Minute)
		api.SetRefreshTokenStore(memRefreshStore)
	}

	api.RegisterRoutes(r, db, bridge, wsHub, auditSvc, nil, eventPluginMgr, blacklist, oauthSvc, backupSvc, keySvc, cfg.Monitor.MetricsPublic, &cfg.Grafana, &cfg.APIPlatform, rateLimiter)

	// Serve embedded frontend static files
	serveStaticFiles(r)

	return &Server{router: r, db: db, bridge: bridge, wsHub: wsHub, addr: addr}
}

// serveStaticFiles sets up serving of embedded frontend assets with SPA fallback.
func serveStaticFiles(r *gin.Engine) {
	// Get the sub-filesystem rooted at "dist"
	distFS, err := fs.Sub(webfs.DistFS, "dist")
	if err != nil {
		// If embed fails (e.g. no dist directory), skip static file serving
		return
	}

	// Serve static assets (js, css, images, etc.) from /assets/ path
	assetsFS, err := fs.Sub(distFS, "assets")
	if err == nil {
		r.StaticFS("/assets", http.FS(assetsFS))
	}

	// Serve other root-level static files (favicon, etc.)
	r.GET("/icon.svg", func(c *gin.Context) {
		data, err := fs.ReadFile(distFS, "icon.svg")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "image/svg+xml", data)
	})

	// SPA fallback: serve index.html for all non-API, non-WS, non-swagger routes
	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		return
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API, WebSocket, and Swagger routes (these should have been handled already,
		// but NoRoute catches everything that didn't match)
		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/ws/") ||
			strings.HasPrefix(path, "/swagger/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	s.httpSrv = &http.Server{
		Addr:              s.addr,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("HTTP server listening", "addr", s.addr)
	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
// It stops accepting new connections, closes the WebSocket hub,
// and waits for active requests to complete (up to the context deadline).
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("server shutting down...")

	// 1. Close WebSocket hub (close all WS connections)
	if s.wsHub != nil {
		s.wsHub.Close()
	}

	// 2. Shutdown HTTP server (stop accepting new requests, drain active ones)
	if s.httpSrv != nil {
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			return fmt.Errorf("HTTP server shutdown: %w", err)
		}
	}

	slog.Info("server shutdown complete")
	return nil
}

// WSHub returns the WebSocket hub (for external access if needed).
func (s *Server) WSHub() *api.WSHub {
	return s.wsHub
}

// Router returns the underlying Gin engine (useful for testing).
func (s *Server) Router() *gin.Engine {
	return s.router
}

