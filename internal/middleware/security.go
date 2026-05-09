package middleware

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

// getCSPPolicy returns the Content-Security-Policy header value.
// Configurable via DEPLOYPILOT_CSP_POLICY env var. If empty, uses a secure default.
func getCSPPolicy() string {
	if policy := os.Getenv("DEPLOYPILOT_CSP_POLICY"); policy != "" {
		return policy
	}
	return "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; " +
		"connect-src 'self' ws: wss:; " +
		"font-src 'self'; " +
		"frame-ancestors 'none'; " +
		"report-uri /api/v1/system/csp-report"
}

// getHSTSMaxAge returns the HSTS max-age value in seconds.
// Configurable via DEPLOYPILOT_HSTS_MAX_AGE env var. Default: 31536000 (1 year).
// Set to 0 to disable (useful for development).
func getHSTSMaxAge() int {
	val := os.Getenv("DEPLOYPILOT_HSTS_MAX_AGE")
	if val == "" {
		return 31536000
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// SecurityHeaders adds common security headers to all responses.
func SecurityHeaders() gin.HandlerFunc {
	hstsMaxAge := getHSTSMaxAge()
	cspPolicy := getCSPPolicy()
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", cspPolicy)
		if hstsMaxAge > 0 {
			c.Header("Strict-Transport-Security",
				fmt.Sprintf("max-age=%d; includeSubDomains", hstsMaxAge))
		}
		c.Next()
	}
}
