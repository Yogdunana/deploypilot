package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/api"
	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/i18n"
	"github.com/Yogdunana/deploypilot/internal/middleware"
	"github.com/Yogdunana/deploypilot/internal/service"
	webfs "github.com/Yogdunana/deploypilot/web"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Server wraps a Gin engine with application dependencies.
type Server struct {
	router *gin.Engine
	db     *gorm.DB
	bridge *service.Bridge
	addr   string
}

// New creates a new API server with the given address, database, bridge, and config.
func New(addr string, db *gorm.DB, bridge *service.Bridge, cfg *config.Config, blacklist auth.TokenBlacklist) *Server {
	r := gin.Default()

	// Security middleware
	r.Use(middleware.SecurityHeaders())
	r.Use(corsMiddleware(cfg.Server.CORSAllowedOrigins))

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

	// Audit logging
	auditSvc := service.NewAuditService(db)
	r.Use(middleware.AuditMiddleware(auditSvc))

	// WebSocket hub
	wsHub := api.NewWSHub()
	go wsHub.Run()

	api.RegisterRoutes(r, db, bridge, wsHub, auditSvc, nil, blacklist)

	// Serve embedded frontend static files
	serveStaticFiles(r)

	return &Server{router: r, db: db, bridge: bridge, addr: addr}
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
	r.GET("/vite.svg", func(c *gin.Context) {
		data, err := fs.ReadFile(distFS, "vite.svg")
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
	return s.router.Run(s.addr)
}

// Router returns the underlying Gin engine (useful for testing).
func (s *Server) Router() *gin.Engine {
	return s.router
}

// corsMiddleware adds CORS headers to all responses.
func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := false
		hasWildcard := false
		for _, o := range allowedOrigins {
			if o == "*" {
				hasWildcard = true
			}
			if o == "*" || o == origin {
				allowed = true
			}
		}
		if len(allowedOrigins) == 0 {
			// No origins configured — allow all
			c.Header("Access-Control-Allow-Origin", "*")
		} else if allowed {
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else if hasWildcard {
				c.Header("Access-Control-Allow-Origin", "*")
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
