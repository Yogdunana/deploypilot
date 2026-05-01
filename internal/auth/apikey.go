package auth

import (
	"encoding/json"
	"log/slog"
	"net"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// APIKeyScopesKey is the context key for API key scopes.
const APIKeyScopesKey contextKey = "api_key_scopes"

// APIKeyIDKey is the context key for the API key ID.
const APIKeyIDKey contextKey = "api_key_id"

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

		// IP whitelist check
		if apiKey.AllowedIPs != "" {
			clientIP := c.ClientIP()
			if !isIPAllowed(clientIP, apiKey.AllowedIPs) {
				slog.Warn("API key IP not allowed", "key_prefix", apiKey.KeyPrefix, "ip", clientIP)
				c.JSON(403, gin.H{"status": "error", "message": "IP address not allowed for this API key"})
				c.Abort()
				return
			}
		}

		// Set user identity in context (same keys as JWT middleware)
		c.Set(string(UserIDKey), apiKey.UserID)
		c.Set(string(RoleKey), "dev") // API keys get dev-level access by default
		c.Set("auth_method", "api_key")
		c.Set(string(APIKeyIDKey), apiKey.ID)

		// Parse and set scopes
		scopes := keySvc.ParseScopes(apiKey.Scopes)
		c.Set(string(APIKeyScopesKey), scopes)

		c.Next()
	}
}

// isIPAllowed checks if a client IP is in the allowed IPs/CIDRs list.
func isIPAllowed(clientIP string, allowedIPsJSON string) bool {
	var allowedIPs []string
	if err := json.Unmarshal([]byte(allowedIPsJSON), &allowedIPs); err != nil {
		return true // if parse fails, allow (fail open)
	}
	if len(allowedIPs) == 0 {
		return true
	}

	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, allowed := range allowedIPs {
		// Try as CIDR
		if strings.Contains(allowed, "/") {
			_, cidr, err := net.ParseCIDR(allowed)
			if err == nil && cidr.Contains(ip) {
				return true
			}
			continue
		}
		// Try as plain IP
		if net.ParseIP(allowed) != nil && net.ParseIP(allowed).Equal(ip) {
			return true
		}
	}
	return false
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
