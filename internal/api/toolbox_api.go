package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ToolboxAPI provides HTTP handlers for the toolbox.
type ToolboxAPI struct {
	tbSvc *service.ToolboxService
}

// NewToolboxAPI creates a new ToolboxAPI.
func NewToolboxAPI(db *gorm.DB) *ToolboxAPI {
	return &ToolboxAPI{
		tbSvc: service.NewToolboxService(db),
	}
}

// DetectEnvironment detects the system environment of a remote server.
// GET /api/v1/servers/:server_id/toolbox/detect
func (t *ToolboxAPI) DetectEnvironment(c *gin.Context) {
	serverID := c.Param("server_id")

	info, err := t.tbSvc.DetectEnvironment(c.Request.Context(), serverID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, info)
}

// RunScript runs a custom script on a remote server.
// POST /api/v1/servers/:server_id/toolbox/run
func (t *ToolboxAPI) RunScript(c *gin.Context) {
	serverID := c.Param("server_id")

	var req struct {
		Script string `json:"script" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	result, err := t.tbSvc.RunScript(c.Request.Context(), serverID, req.Script)
	if err != nil {
		respondErrori18n(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, result)
}

// RunBuiltInScript runs a built-in script.
// POST /api/v1/servers/:server_id/toolbox/builtin
func (t *ToolboxAPI) RunBuiltInScript(c *gin.Context) {
	serverID := c.Param("server_id")

	var req struct {
		Category string `json:"category" binding:"required"`
		Name     string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	result, err := t.tbSvc.RunBuiltInScript(c.Request.Context(), serverID, req.Category, req.Name)
	if err != nil {
		respondErrori18n(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, result)
}

// ListBuiltInScripts lists all available built-in scripts.
// GET /api/v1/toolbox/scripts/builtin
func (t *ToolboxAPI) ListBuiltInScripts(c *gin.Context) {
	scripts := t.tbSvc.ListBuiltInScripts()
	respondSuccess(c, scripts)
}

// ========== Custom Script CRUD ==========

// CreateScript creates a new custom script.
// POST /api/v1/toolbox/scripts
func (t *ToolboxAPI) CreateScript(c *gin.Context) {
	var script service.ToolboxScript
	if err := c.ShouldBindJSON(&script); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := t.tbSvc.CreateScript(c.Request.Context(), &script); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, script)
}

// ListScripts lists custom scripts.
// GET /api/v1/toolbox/scripts?category=xxx
func (t *ToolboxAPI) ListScripts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	category := c.Query("category")

	scripts, err := t.tbSvc.ListScripts(c.Request.Context(), tenantID, category)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, scripts)
}

// GetScript gets a custom script by ID.
// GET /api/v1/servers/:id/toolbox/scripts/:script_id
func (t *ToolboxAPI) GetScript(c *gin.Context) {
	id := c.Param("script_id")

	script, err := t.tbSvc.GetScript(c.Request.Context(), id)
	if err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
		return
	}

	respondSuccess(c, script)
}

// UpdateScript updates a custom script.
// PUT /api/v1/servers/:id/toolbox/scripts/:script_id
func (t *ToolboxAPI) UpdateScript(c *gin.Context) {
	id := c.Param("script_id")

	var script service.ToolboxScript
	if err := c.ShouldBindJSON(&script); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	script.ID = id
	if err := t.tbSvc.UpdateScript(c.Request.Context(), &script); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, script)
}

// DeleteScript deletes a custom script.
// DELETE /api/v1/servers/:id/toolbox/scripts/:script_id
func (t *ToolboxAPI) DeleteScript(c *gin.Context) {
	id := c.Param("script_id")

	if err := t.tbSvc.DeleteScript(c.Request.Context(), id); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"id": id, "status": "deleted"})
}
