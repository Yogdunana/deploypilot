package api

import (
	"github.com/gin-gonic/gin"
)

// APIVersionHandler returns current API version info.
// This endpoint is public and does not require authentication.
func APIVersionHandler(c *gin.Context) {
	respondSuccess(c, gin.H{
		"version":       "v1",
		"supported":     []string{"v1"},
		"status":        "stable",
		"documentation": "/api/v1/docs",
	})
}
