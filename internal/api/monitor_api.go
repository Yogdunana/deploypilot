package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MonitorAPI provides HTTP handlers for monitoring and observability.
type MonitorAPI struct {
	monSvc *service.MonitorService
}

// NewMonitorAPI creates a new MonitorAPI.
func NewMonitorAPI(db *gorm.DB) *MonitorAPI {
	return &MonitorAPI{
		monSvc: service.NewMonitorService(db),
	}
}

// ========== Uptime Monitor CRUD ==========

// CreateMonitor creates a new uptime monitor.
// POST /api/v1/monitors
func (m *MonitorAPI) CreateMonitor(c *gin.Context) {
	var mon service.Monitor
	if err := c.ShouldBindJSON(&mon); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := m.monSvc.CreateMonitor(c.Request.Context(), &mon); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, mon)
}

// ListMonitors lists all monitors.
// GET /api/v1/monitors
func (m *MonitorAPI) ListMonitors(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	monitors, err := m.monSvc.ListMonitors(c.Request.Context(), tenantID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, monitors)
}

// GetMonitor gets a monitor by ID.
// GET /api/v1/monitors/:id
func (m *MonitorAPI) GetMonitor(c *gin.Context) {
	id := c.Param("id")

	mon, err := m.monSvc.GetMonitor(c.Request.Context(), id)
	if err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
		return
	}

	respondSuccess(c, mon)
}

// UpdateMonitor updates a monitor.
// PUT /api/v1/monitors/:id
func (m *MonitorAPI) UpdateMonitor(c *gin.Context) {
	id := c.Param("id")

	var mon service.Monitor
	if err := c.ShouldBindJSON(&mon); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	mon.ID = id
	if err := m.monSvc.UpdateMonitor(c.Request.Context(), &mon); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, mon)
}

// DeleteMonitor deletes a monitor.
// DELETE /api/v1/monitors/:id
func (m *MonitorAPI) DeleteMonitor(c *gin.Context) {
	id := c.Param("id")

	if err := m.monSvc.DeleteMonitor(c.Request.Context(), id); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"id": id, "status": "deleted"})
}

// ========== Monitor Checks ==========

// CheckMonitor triggers a check on a specific monitor.
// POST /api/v1/monitors/:id/check
func (m *MonitorAPI) CheckMonitor(c *gin.Context) {
	id := c.Param("id")

	result, err := m.monSvc.CheckMonitor(c.Request.Context(), id)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, result)
}

// CheckAllMonitors triggers checks on all enabled monitors.
// POST /api/v1/monitors/check-all
func (m *MonitorAPI) CheckAllMonitors(c *gin.Context) {
	results, err := m.monSvc.CheckAllMonitors(c.Request.Context())
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"results": results, "count": len(results)})
}

// GetMonitorResults gets recent check results.
// GET /api/v1/monitors/:id/results?limit=50
func (m *MonitorAPI) GetMonitorResults(c *gin.Context) {
	id := c.Param("id")
	limit := 50
	_ = c.ShouldBindQuery(&limit)

	results, err := m.monSvc.GetMonitorResults(c.Request.Context(), id, limit)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, results)
}

// GetMonitorSLA gets SLA metrics for a monitor.
// GET /api/v1/monitors/:id/sla?days=30
func (m *MonitorAPI) GetMonitorSLA(c *gin.Context) {
	id := c.Param("id")
	days := 30
	_ = c.ShouldBindQuery(&days)

	sla, err := m.monSvc.GetMonitorSLA(c.Request.Context(), id, days)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, sla)
}

// ========== Status Page ==========

// GetStatusPage returns the public status page data.
// GET /api/v1/status
func (m *MonitorAPI) GetStatusPage(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, err := m.monSvc.GetStatusPage(c.Request.Context(), tenantID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, page)
}

// ========== Heartbeat ==========

// CreateHeartbeat creates a new heartbeat monitor.
// POST /api/v1/heartbeats
func (m *MonitorAPI) CreateHeartbeat(c *gin.Context) {
	var hb service.Heartbeat
	if err := c.ShouldBindJSON(&hb); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := m.monSvc.CreateHeartbeat(c.Request.Context(), &hb); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, hb)
}

// ListHeartbeats lists all heartbeats.
// GET /api/v1/heartbeats
func (m *MonitorAPI) ListHeartbeats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	heartbeats, err := m.monSvc.ListHeartbeats(c.Request.Context(), tenantID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, heartbeats)
}

// DeleteHeartbeat deletes a heartbeat.
// DELETE /api/v1/heartbeats/:id
func (m *MonitorAPI) DeleteHeartbeat(c *gin.Context) {
	id := c.Param("id")

	if err := m.monSvc.DeleteHeartbeat(c.Request.Context(), id); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"id": id, "status": "deleted"})
}

// PingHeartbeat receives a heartbeat ping (public endpoint).
// GET /api/v1/heartbeat/ping/:token
func (m *MonitorAPI) PingHeartbeat(c *gin.Context) {
	token := c.Param("token")

	if err := m.monSvc.PingHeartbeat(c.Request.Context(), token); err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
		return
	}

	respondSuccess(c, gin.H{"status": "ok"})
}

// ========== Prometheus Metrics ==========

// GetPrometheusMetrics exports metrics in Prometheus format.
// GET /api/v1/metrics
func (m *MonitorAPI) GetPrometheusMetrics(c *gin.Context) {
	metrics, err := m.monSvc.GetPrometheusMetrics(c.Request.Context())
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(http.StatusOK, metrics)
}
