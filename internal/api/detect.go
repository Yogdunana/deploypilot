package api

import (
	"log/slog"
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/detector"
	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/gin-gonic/gin"
)

// DetectAllComponents detects all system components (databases + web servers).
// GET /api/v1/system/detect/all
func DetectAllComponents(executor deployer.CommandExecutor) gin.HandlerFunc {
	return func(c *gin.Context) {
		if executor == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": []interface{}{}, "message": "no executor configured"})
			return
		}

		d := detector.New(executor)
		components, err := d.DetectAll(c.Request.Context())
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to detect all components", "error", err)
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"data":    components,
			"summary": detector.GetDetectionSummary(components),
		})
	}
}

// DetectDatabases detects database instances (MySQL, PostgreSQL, Redis, MongoDB).
// GET /api/v1/system/detect/databases
func DetectDatabases(executor deployer.CommandExecutor) gin.HandlerFunc {
	return func(c *gin.Context) {
		if executor == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": []interface{}{}, "message": "no executor configured"})
			return
		}

		d := detector.New(executor)
		components, err := d.DetectDatabases(c.Request.Context())
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to detect databases", "error", err)
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"data":    components,
			"summary": detector.GetDetectionSummary(components),
		})
	}
}

// DetectWebServers detects web server instances (Nginx, Apache, OpenResty).
// GET /api/v1/system/detect/webservers
func DetectWebServers(executor deployer.CommandExecutor) gin.HandlerFunc {
	return func(c *gin.Context) {
		if executor == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": []interface{}{}, "message": "no executor configured"})
			return
		}

		d := detector.New(executor)
		components, err := d.DetectWebServers(c.Request.Context())
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to detect web servers", "error", err)
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"data":    components,
			"summary": detector.GetDetectionSummary(components),
		})
	}
}
