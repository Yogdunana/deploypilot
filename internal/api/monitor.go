package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// GetSystemMetrics returns system-level metrics (CPU, memory, disk).
// @Summary      Get system metrics
// @Description  Retrieve system-level resource metrics including CPU, memory, and disk usage
// @Tags         Monitor
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (SystemMetrics)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /monitor/system [get]
func GetSystemMetrics(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := bridge.GetSystemMetrics(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, data)
	}
}

// GetContainerMetrics returns resource usage metrics for a specific container.
// @Summary      Get container metrics
// @Description  Retrieve resource usage metrics for a specific container
// @Tags         Monitor
// @Produce      json
// @Security     BearerAuth
// @Param        name path string true "Container name"
// @Success      200 {object} map[string]interface{} "status, data (ContainerMetrics)"
// @Failure      400 {object} map[string]interface{} "container name is required"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /monitor/container/{name} [get]
func GetContainerMetrics(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.monitor.container_name_required")
			return
		}
		data, err := bridge.GetContainerMetrics(c.Request.Context(), name)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, data)
	}
}

// ListAlerts returns all currently active (firing) alerts.
// @Summary      List active alerts
// @Description  Retrieve all currently active (firing) alerts
// @Tags         Monitor
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (array of Alert)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /monitor/alerts [get]
func ListAlerts(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := bridge.ListAlerts(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, data)
	}
}

// ListAlertRules returns all configured alert rules.
// @Summary      List alert rules
// @Description  Retrieve all configured alert rules
// @Tags         Monitor
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (array of AlertRule)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /monitor/alert-rules [get]
func ListAlertRules(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := bridge.ListAlertRules(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, data)
	}
}

// HealContainer triggers self-healing for a container.
// @Summary      Heal a container
// @Description  Trigger self-healing actions for a container that is not running correctly
// @Tags         Monitor
// @Produce      json
// @Security     BearerAuth
// @Param        name path string true "Container name"
// @Success      200 {object} map[string]interface{} "status, data (heal result)"
// @Failure      400 {object} map[string]interface{} "container name is required"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /monitor/heal/{name} [post]
func HealContainer(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.monitor.container_name_required")
			return
		}
		data, err := bridge.HealContainer(c.Request.Context(), name)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, data)
	}
}

// CheckContainerHealth runs a health check for a specific container.
// @Summary      Check container health
// @Description  Retrieve the current status of a specific container (read-only health check, does not trigger self-healing)
// @Tags         Monitor
// @Produce      json
// @Security     BearerAuth
// @Param        name path string true "Container name"
// @Success      200 {object} map[string]interface{} "status, data (container status)"
// @Failure      400 {object} map[string]interface{} "container name is required"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /monitor/check/{name} [post]
func CheckContainerHealth(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.monitor.container_name_required")
			return
		}
		data, err := bridge.GetContainerStatus(c.Request.Context(), name)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, data)
	}
}
