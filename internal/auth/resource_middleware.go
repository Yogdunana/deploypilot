package auth

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/i18n"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequireResourceAccess returns middleware that checks if the authenticated user
// has access to the specified resource.
func RequireResourceAccess(db *gorm.DB, resourceType string, idParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		resourceID := c.Param(idParam)
		if resourceID == "" {
			locale := i18n.GetLocaleFromContext(c)
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": i18n.T(locale, "error.common.invalid_request")})
			c.Abort()
			return
		}

		userID, _ := c.Get(string(UserIDKey))
		role, _ := c.Get(string(RoleKey))
		userIDStr, _ := userID.(string)
		roleStr, _ := role.(string)

		if !service.CheckResourceAccess(db, resourceType, resourceID, roleStr, userIDStr) {
			locale := i18n.GetLocaleFromContext(c)
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": i18n.T(locale, "error.auth.insufficient_permissions")})
			c.Abort()
			return
		}

		c.Next()
	}
}