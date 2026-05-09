package api

import (
	"net/http"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ProcessAPI provides HTTP handlers for remote process management.
type ProcessAPI struct {
	procSvc *service.ProcessService
}

// NewProcessAPI creates a new ProcessAPI.
func NewProcessAPI(db *gorm.DB) *ProcessAPI {
	return &ProcessAPI{
		procSvc: service.NewProcessService(db),
	}
}

// ListProcesses lists processes on a remote server.
// GET /api/v1/servers/:server_id/processes?filter=nginx
func (p *ProcessAPI) ListProcesses(c *gin.Context) {
	serverID := c.Param("server_id")
	filter := c.Query("filter")

	stats, err := p.procSvc.ListProcesses(c.Request.Context(), serverID, filter)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, stats)
}

// GetProcess gets details of a specific process by PID.
// GET /api/v1/servers/:server_id/processes/:pid
func (p *ProcessAPI) GetProcess(c *gin.Context) {
	serverID := c.Param("server_id")
	pid, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	info, err := p.procSvc.GetProcess(c.Request.Context(), serverID, pid)
	if err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
		return
	}

	respondSuccess(c, info)
}

// KillProcess sends a signal to a process.
// POST /api/v1/servers/:server_id/processes/:pid/kill
func (p *ProcessAPI) KillProcess(c *gin.Context) {
	serverID := c.Param("server_id")
	pid, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	var req struct {
		Signal string `json:"signal"`
	}
	_ = c.ShouldBindJSON(&req) // signal is optional

	if err := p.procSvc.KillProcess(c.Request.Context(), serverID, pid, req.Signal); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"pid": pid, "signal": req.Signal, "status": "sent"})
}

// SearchProcesses searches for processes matching a pattern.
// GET /api/v1/servers/:server_id/processes/search?pattern=nginx
func (p *ProcessAPI) SearchProcesses(c *gin.Context) {
	serverID := c.Param("server_id")
	pattern := c.Query("pattern")
	if pattern == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	processes, err := p.procSvc.SearchProcesses(c.Request.Context(), serverID, pattern)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, processes)
}

// GetProcessTree returns the process tree.
// GET /api/v1/servers/:server_id/processes/tree
func (p *ProcessAPI) GetProcessTree(c *gin.Context) {
	serverID := c.Param("server_id")

	processes, err := p.procSvc.GetProcessTree(c.Request.Context(), serverID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, processes)
}

// GetSystemResources returns system resource usage.
// GET /api/v1/servers/:server_id/resources
func (p *ProcessAPI) GetSystemResources(c *gin.Context) {
	serverID := c.Param("server_id")

	resources, err := p.procSvc.SystemResources(c.Request.Context(), serverID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, resources)
}

// ========== Process Rules (CRUD) ==========

// CreateRule creates a new process monitoring rule.
// POST /api/v1/processes/rules
func (p *ProcessAPI) CreateRule(c *gin.Context) {
	var rule service.ProcessRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := p.procSvc.CreateRule(c.Request.Context(), &rule); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, rule)
}

// ListRules lists all process monitoring rules.
// GET /api/v1/processes/rules?server_id=xxx
func (p *ProcessAPI) ListRules(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	serverID := c.Query("server_id")

	rules, err := p.procSvc.ListRules(c.Request.Context(), tenantID, serverID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, rules)
}

// GetRule gets a process monitoring rule by ID.
// GET /api/v1/processes/rules/:id
func (p *ProcessAPI) GetRule(c *gin.Context) {
	id := c.Param("id")

	rule, err := p.procSvc.GetRule(c.Request.Context(), id)
	if err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
		return
	}

	respondSuccess(c, rule)
}

// UpdateRule updates a process monitoring rule.
// PUT /api/v1/processes/rules/:id
func (p *ProcessAPI) UpdateRule(c *gin.Context) {
	id := c.Param("id")

	var rule service.ProcessRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	rule.ID = id
	if err := p.procSvc.UpdateRule(c.Request.Context(), &rule); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, rule)
}

// DeleteRule deletes a process monitoring rule.
// DELETE /api/v1/processes/rules/:id
func (p *ProcessAPI) DeleteRule(c *gin.Context) {
	id := c.Param("id")

	if err := p.procSvc.DeleteRule(c.Request.Context(), id); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"id": id, "status": "deleted"})
}
