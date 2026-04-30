package auth

import (
	"log/slog"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// APIKeyMiddleware validates API keys from the X-API-Key header or
// Authorization: Bearer dp_xxx header. On success, it sets the same
// context keys as AuthMiddleware (UserIDKey, RoleKey) so downstream
// middleware works seamlessly.
func APIKeyMiddleware(keySvc *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawKey := extractAPIKey(c)
		if rawKey == "" {
			c.Next()
			return
		}

		// Only process keys with our prefix
		if !strings.HasPrefix(rawKey, "dp_") {
			c.Next()
			return
		}

		apiKey, err := keySvc.Validate(c.Request.Context(), rawKey)
		if err != nil {
			slog.Debug("API key validation failed", "error", err)
			c.Next()
			return
		}

		// Set user identity in context (same keys as JWT middleware)
		c.Set(string(UserIDKey), apiKey.UserID)
		c.Set(string(RoleKey), "dev") // API keys get dev-level access by default
		c.Set("auth_method", "api_key")
		c.Next()
	}
}

// extractAPIKey extracts the API key from X-API-Key header or Authorization header.
func extractAPIKey(c *gin.Context) string {
	// Try X-API-Key header first
	if key := c.GetHeader("X-API-Key"); key != "" {
		return strings.TrimSpace(key)
	}

	// Try Authorization: Bearer dp_xxx
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		key := strings.TrimSpace(parts[1])
		if strings.HasPrefix(key, "dp_") {
			return key
		}
	}

	return ""
}
