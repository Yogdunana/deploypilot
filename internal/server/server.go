package server

import (
	"github.com/Yogdunana/deploypilot/internal/api"
	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/middleware"
	"github.com/Yogdunana/deploypilot/internal/service"
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
func New(addr string, db *gorm.DB, bridge *service.Bridge, cfg *config.Config) *Server {
	r := gin.Default()

	// Security middleware
	r.Use(middleware.SecurityHeaders())
	r.Use(corsMiddleware(cfg.Server.CORSAllowedOrigins))

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

	api.RegisterRoutes(r, db, bridge, wsHub, auditSvc)
	return &Server{router: r, db: db, bridge: bridge, addr: addr}
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
