package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// GetSystemMetrics returns system-level metrics (CPU, memory, disk).
func GetSystemMetrics(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := bridge.GetSystemMetrics(c.Request.Context())
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, data)
	}
}

// GetContainerMetrics returns resource usage metrics for a specific container.
func GetContainerMetrics(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			respondError(c, http.StatusBadRequest, "container name is required")
			return
		}
		data, err := bridge.GetContainerMetrics(c.Request.Context(), name)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, data)
	}
}

// ListAlerts returns all currently active (firing) alerts.
func ListAlerts(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := bridge.ListAlerts(c.Request.Context())
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, data)
	}
}

// ListAlertRules returns all configured alert rules.
func ListAlertRules(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := bridge.ListAlertRules(c.Request.Context())
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, data)
	}
}

// HealContainer triggers self-healing for a container.
func HealContainer(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			respondError(c, http.StatusBadRequest, "container name is required")
			return
		}
		data, err := bridge.HealContainer(c.Request.Context(), name)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, data)
	}
}

// CheckContainerHealth runs a health check for a specific container.
func CheckContainerHealth(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			respondError(c, http.StatusBadRequest, "container name is required")
			return
		}
		data, err := bridge.HealContainer(c.Request.Context(), name)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, data)
	}
}
