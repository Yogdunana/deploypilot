package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds all CORS configuration parameters.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	ExposeHeaders    []string
	MaxAge           int // seconds
}

// CORS returns a gin middleware that handles Cross-Origin Resource Sharing.
//
// Security behavior:
//   - Empty AllowedOrigins: NO CORS headers are set (reject all cross-origin requests).
//     This is the most secure default. To allow all origins, explicitly set ["*"].
//   - When AllowCredentials is true, wildcard "*" origins are rejected per the CORS spec.
//   - Vary: Origin is always set when origins are explicitly configured.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	// Normalize defaults
	methods := cfg.AllowedMethods
	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	headers := cfg.AllowedHeaders
	if len(headers) == 0 {
		headers = []string{"Authorization", "Content-Type", "X-API-Key"}
	}
	maxAge := cfg.MaxAge
	if maxAge <= 0 {
		maxAge = 86400
	}

	// Pre-compute method/header strings
	methodsStr := strings.Join(methods, ", ")
	headersStr := strings.Join(headers, ", ")

	// Check for wildcard
	hasWildcard := false
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			hasWildcard = true
			break
		}
	}

	// Security: Reject wildcard with credentials configuration
	if hasWildcard && cfg.AllowCredentials {
		panic("CORS configuration error: AllowCredentials cannot be true when AllowedOrigins contains wildcard '*'")
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// No origins configured — do not set any CORS headers (secure default)
		if len(cfg.AllowedOrigins) == 0 {
			c.Next()
			return
		}

		// Determine if origin is allowed
		allowed := false
		for _, o := range cfg.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if !allowed {
			c.Next()
			return
		}

		// Set Allow-Origin
		if hasWildcard && !cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}

		// Set other CORS headers
		c.Header("Access-Control-Allow-Methods", methodsStr)
		c.Header("Access-Control-Allow-Headers", headersStr)
		c.Header("Access-Control-Max-Age", strconv.Itoa(maxAge))

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if len(cfg.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ", "))
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
