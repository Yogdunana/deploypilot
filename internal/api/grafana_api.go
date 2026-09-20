package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// globalGrafanaAPI is the package-level GrafanaAPI instance, accessible via GetGlobalGrafanaAPI().
var globalGrafanaAPI *GrafanaAPI

// GetGlobalGrafanaAPI returns the global GrafanaAPI instance.
func GetGlobalGrafanaAPI() *GrafanaAPI { return globalGrafanaAPI }

// GrafanaAPI provides HTTP handlers for Grafana integration management.
type GrafanaAPI struct {
	db  *gorm.DB
	cfg *config.GrafanaConfig
}

// NewGrafanaAPI creates a new GrafanaAPI.
func NewGrafanaAPI(db *gorm.DB, cfg *config.GrafanaConfig) *GrafanaAPI {
	return &GrafanaAPI{db: db, cfg: cfg}
}

// grafanaService returns a Grafana service using the handler DB.
// It writes an error response and returns nil when the DB is unavailable
// instead of panicking via MustGet.
func (a *GrafanaAPI) grafanaService(c *gin.Context) *service.GrafanaService {
	if a == nil || a.db == nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.user.database_not_available")
		return nil
	}
	return service.NewGrafanaService(a.db, a.cfg, nil)
}

// GetStatus returns the current Grafana integration status.
// GET /api/v1/grafana/status
func (a *GrafanaAPI) GetStatus(c *gin.Context) {
	svc := a.grafanaService(c)
	if svc == nil {
		return
	}
	status := svc.GetStatus()
	respondSuccess(c, status)
}

// TestConnection tests the connection to the Grafana server.
// POST /api/v1/grafana/test
func (a *GrafanaAPI) TestConnection(c *gin.Context) {
	svc := a.grafanaService(c)
	if svc == nil {
		return
	}
	result, err := svc.TestConnection()
	if err != nil {
		respondErrori18n(c, http.StatusServiceUnavailable, "error.common.internal_error")
		return
	}
	respondSuccess(c, result)
}

// SyncAll performs a full sync of datasources, folder, and dashboards.
// POST /api/v1/grafana/sync
func (a *GrafanaAPI) SyncAll(c *gin.Context) {
	svc := a.grafanaService(c)
	if svc == nil {
		return
	}
	count, err := svc.SyncAll()
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	respondSuccess(c, gin.H{"synced_dashboards": count})
}

// ListDashboards returns all Grafana dashboards.
// GET /api/v1/grafana/dashboards
func (a *GrafanaAPI) ListDashboards(c *gin.Context) {
	svc := a.grafanaService(c)
	if svc == nil {
		return
	}
	dashboards, err := svc.ListDashboards()
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	respondSuccess(c, dashboards)
}

// GetDashboard returns a single dashboard by ID.
// GET /api/v1/grafana/dashboards/:id
func (a *GrafanaAPI) GetDashboard(c *gin.Context) {
	svc := a.grafanaService(c)
	if svc == nil {
		return
	}
	id := c.Param("id")
	dashboard, err := svc.GetDashboard(id)
	if err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
		return
	}
	respondSuccess(c, dashboard)
}

// CreateDashboard creates a new custom Grafana dashboard.
// POST /api/v1/grafana/dashboards
func (a *GrafanaAPI) CreateDashboard(c *gin.Context) {
	svc := a.grafanaService(c)
	if svc == nil {
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
		JSON string `json:"json" binding:"required"`
		Tags string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	dashboard, err := svc.CreateCustomDashboard(req.Name, req.JSON, req.Tags)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   dashboard,
	})
}

// UpdateDashboard updates an existing Grafana dashboard.
// PUT /api/v1/grafana/dashboards/:id
func (a *GrafanaAPI) UpdateDashboard(c *gin.Context) {
	svc := a.grafanaService(c)
	if svc == nil {
		return
	}
	id := c.Param("id")

	var req struct {
		Name    string `json:"name"`
		JSON    string `json:"json"`
		Tags    string `json:"tags"`
		Enabled *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if err := svc.UpdateCustomDashboard(id, req.Name, req.JSON, req.Tags, enabled); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"id": id, "status": "updated"})
}

// DeleteDashboard deletes a Grafana dashboard.
// DELETE /api/v1/grafana/dashboards/:id
func (a *GrafanaAPI) DeleteDashboard(c *gin.Context) {
	svc := a.grafanaService(c)
	if svc == nil {
		return
	}
	id := c.Param("id")

	if err := svc.DeleteCustomDashboard(id); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"id": id, "status": "deleted"})
}

// ExportAll exports all dashboards and datasource configuration.
// GET /api/v1/grafana/export
func (a *GrafanaAPI) ExportAll(c *gin.Context) {
	svc := a.grafanaService(c)
	if svc == nil {
		return
	}
	result, err := svc.ExportAll()
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	respondSuccess(c, result)
}
