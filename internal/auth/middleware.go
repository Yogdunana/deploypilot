package auth

import (
	"net/http"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/i18n"
	"github.com/gin-gonic/gin"
)

// contextKey is the type for context keys used by auth middleware.
type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey contextKey = "userID"
	// RoleKey is the context key for the authenticated user's role.
	RoleKey contextKey = "role"
)

// roleHierarchy defines the permission levels for roles.
// Higher index = higher permission.
var roleHierarchy = map[string]int{
	"owner":  4,
	"admin":  3,
	"dev":    2,
	"viewer": 1,
}

// AuthMiddleware extracts and validates the Bearer token from the Authorization header.
// It sets the userID and role in the gin.Context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			locale := i18n.GetLocaleFromContext(c)
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": i18n.T(locale, "error.auth.authorization_header_required")})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			locale := i18n.GetLocaleFromContext(c)
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": i18n.T(locale, "error.auth.invalid_authorization_format")})
			c.Abort()
			return
		}

		claims, err := ParseToken(parts[1])
		if err != nil {
			locale := i18n.GetLocaleFromContext(c)
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": i18n.T(locale, "error.auth.invalid_or_expired_token")})
			c.Abort()
			return
		}

		c.Set(string(UserIDKey), claims.UserID)
		c.Set(string(RoleKey), claims.Role)
		c.Next()
	}
}

// RoleRequired returns middleware that checks if the authenticated user has one of the required roles.
// The role hierarchy is: owner > admin > dev > viewer.
// A user with a higher role automatically satisfies a lower role requirement.
func RoleRequired(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(string(RoleKey))
		if !exists {
			locale := i18n.GetLocaleFromContext(c)
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": i18n.T(locale, "error.auth.authentication_required")})
			c.Abort()
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			locale := i18n.GetLocaleFromContext(c)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": i18n.T(locale, "error.auth.invalid_role_in_context")})
			c.Abort()
			return
		}

		userLevel, ok := roleHierarchy[roleStr]
		if !ok {
			locale := i18n.GetLocaleFromContext(c)
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": i18n.T(locale, "error.auth.unknown_user_role")})
			c.Abort()
			return
		}

		for _, requiredRole := range roles {
			requiredLevel, ok := roleHierarchy[requiredRole]
			if !ok {
				continue
			}
			if userLevel >= requiredLevel {
				c.Next()
				return
			}
		}

		locale := i18n.GetLocaleFromContext(c)
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": i18n.T(locale, "error.auth.insufficient_permissions")})
		c.Abort()
	}
}

// OptionalAuth parses the JWT token if present but does not require it.
// If a valid token is found, it sets userID and role in the context.
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.Next()
			return
		}

		claims, err := ParseToken(parts[1])
		if err != nil {
			c.Next()
			return
		}

		c.Set(string(UserIDKey), claims.UserID)
		c.Set(string(RoleKey), claims.Role)
		c.Next()
	}
}
