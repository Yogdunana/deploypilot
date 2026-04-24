package server

import (
	"github.com/Yogdunana/deploypilot/internal/api"
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

// New creates a new API server with the given address, database, and bridge.
func New(addr string, db *gorm.DB, bridge *service.Bridge) *Server {
	r := gin.Default()
	r.Use(corsMiddleware())
	api.RegisterRoutes(r, db, bridge)
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
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
