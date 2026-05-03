package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LicenseCheckMiddleware checks if the license is valid before allowing access.
func LicenseCheckMiddleware(engine interface{ Validate() error }) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := engine.Validate(); err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "license error",
				"message": err.Error(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
