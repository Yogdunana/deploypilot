package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DegradationChecker defines the interface for checking degradation status.
type DegradationChecker interface {
	CheckReadOnly(ctx interface{}) error
}

// ReadOnlyMiddleware blocks all mutating requests (POST, PUT, DELETE, PATCH)
// when the instance is in read-only mode.
func ReadOnlyMiddleware(checker DegradationChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH" {
			if err := checker.CheckReadOnly(c.Request.Context()); err != nil {
				slog.Warn("read-only mode: mutation blocked",
					"method", method,
					"path", c.Request.URL.Path,
					"reason", err.Error(),
				)
				c.JSON(http.StatusForbidden, gin.H{
					"status":  "error",
					"code":    "read_only_mode",
					"message": err.Error(),
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
