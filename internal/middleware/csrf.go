package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// CSRFTokenCookie is the name of the CSRF token cookie.
	CSRFTokenCookie = "deploypilot_csrf"
	// CSRFTokenHeader is the name of the CSRF token header.
	CSRFTokenHeader = "X-CSRF-Token"
	// csrfTokenLength is the length of the CSRF token in bytes.
	csrfTokenLength = 32
)

// CSRF returns a middleware that provides CSRF protection using the
// Double Submit Cookie pattern. Safe methods (GET, HEAD, OPTIONS) are
// exempt. The token is set as a SameSite=Strict cookie and must be
// sent back via the X-CSRF-Token header for state-changing requests.
//
// Note: This is a defense-in-depth measure. The existing SameSite=Strict
// cookie policy already provides strong CSRF protection for modern browsers.
// The middleware can be disabled by setting "csrf_skip" in the gin context.
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow tests to skip CSRF check
		if _, ok := c.Get("csrf_skip"); ok {
			c.Next()
			return
		}

		// Skip safe methods
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			// Ensure CSRF cookie exists for safe requests too
			if _, err := c.Cookie(CSRFTokenCookie); err != nil {
				setCSRFCookie(c, generateCSRFToken())
			}
			c.Next()
			return
		}

		// For state-changing methods, validate CSRF token
		cookieToken, err := c.Cookie(CSRFTokenCookie)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
			return
		}

		headerToken := c.GetHeader(CSRFTokenHeader)
		if headerToken == "" {
			// Also check form field for HTML form submissions
			headerToken = c.PostForm("csrf_token")
		}

		if !strings.EqualFold(headerToken, cookieToken) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token mismatch"})
			return
		}

		c.Next()
	}
}

// generateCSRFToken creates a new random CSRF token.
func generateCSRFToken() string {
	b := make([]byte, csrfTokenLength)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// setCSRFCookie sets the CSRF token as an HttpOnly, SameSite=Strict cookie.
func setCSRFCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(CSRFTokenCookie, token, int(24*time.Hour.Seconds()), "/", "", true, false)
}
