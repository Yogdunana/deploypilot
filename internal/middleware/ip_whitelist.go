package middleware

import (
	"log/slog"
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// UserIPWhitelistMiddleware enforces per-user IP whitelist rules.
// If a user has whitelist entries configured, only IPs matching those entries are allowed.
// If a user has NO whitelist entries, all IPs are allowed (backward compatible).
func UserIPWhitelistMiddleware(svc *service.IPWhitelistService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get(string(auth.UserIDKey))
		if !exists {
			// No authenticated user — let downstream auth middleware handle it
			c.Next()
			return
		}

		userID, ok := userIDVal.(string)
		if !ok || userID == "" {
			c.Next()
			return
		}

		// If user has no whitelist entries, skip enforcement (backward compatible)
		if !svc.IsEnforced(userID) {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		if !svc.Check(clientIP, userID) {
			slog.Warn("user IP whitelist blocked",
				"user_id", userID,
				"ip", clientIP,
				"path", c.Request.URL.Path,
			)
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "access denied: your IP address is not in the whitelist for this account",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
