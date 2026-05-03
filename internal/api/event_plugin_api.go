package api

import (
	"encoding/json"
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/gin-gonic/gin"
)

// globalEventPluginAPI is the package-level EventPluginAPI instance.
var globalEventPluginAPI *EventPluginAPI

// GetGlobalEventPluginAPI returns the global EventPluginAPI instance.
func GetGlobalEventPluginAPI() *EventPluginAPI { return globalEventPluginAPI }

// EventPluginAPI holds dependencies for event plugin API handlers.
type EventPluginAPI struct {
	pluginMgr *plugin.EventPluginManager
}

// NewEventPluginAPI creates a new EventPluginAPI.
func NewEventPluginAPI(pluginMgr *plugin.EventPluginManager) *EventPluginAPI {
	return &EventPluginAPI{pluginMgr: pluginMgr}
}

// ListEventPlugins lists all registered event plugins and their statuses.
// @Summary      List event plugins
// @Description  Retrieve all registered event-driven plugins with their status
// @Tags         Event Plugins
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (array of PluginInfo)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Router       /event-plugins [get]
func (a *EventPluginAPI) ListEventPlugins(c *gin.Context) {
	if a.pluginMgr == nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	plugins := a.pluginMgr.ListPlugins()
	respondSuccess(c, plugins)
}

// GetEventPlugin retrieves a single event plugin by name.
// @Summary      Get event plugin
// @Description  Retrieve a single event-driven plugin by name
// @Tags         Event Plugins
// @Produce      json
// @Security     BearerAuth
// @Param        name path string true "Plugin name"
// @Success      200 {object} map[string]interface{} "status, data (PluginInfo)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "not found"
// @Router       /event-plugins/{name} [get]
func (a *EventPluginAPI) GetEventPlugin(c *gin.Context) {
	if a.pluginMgr == nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	name := c.Param("name")
	p, ok := a.pluginMgr.GetPlugin(name)
	if !ok {
		respondErrori18n(c, http.StatusNotFound, "error.plugin.not_found")
		return
	}
	plugins := a.pluginMgr.ListPlugins()
	for _, info := range plugins {
		if info.Name == name {
			respondSuccess(c, info)
			return
		}
	}
	// Fallback: build info from the plugin itself
	respondSuccess(c, gin.H{
		"name":        p.Name(),
		"version":     p.Version(),
		"description": p.Description(),
	})
}

// UpdateEventPlugin updates an event plugin's config or enabled state.
// @Summary      Update event plugin
// @Description  Update an event plugin's configuration or enabled state
// @Tags         Event Plugins
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        name path string true "Plugin name"
// @Param        request body object{enabled=bool,config=map[string]interface{}} true "Plugin update request"
// @Success      200 {object} map[string]interface{} "status, data.message"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "not found"
// @Router       /event-plugins/{name} [put]
func (a *EventPluginAPI) UpdateEventPlugin(c *gin.Context) {
	if a.pluginMgr == nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	name := c.Param("name")

	var input struct {
		Enabled *bool                   `json:"enabled"`
		Config  map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
		return
	}

	if _, ok := a.pluginMgr.GetPlugin(name); !ok {
		respondErrori18n(c, http.StatusNotFound, "error.plugin.not_found")
		return
	}

	if input.Enabled != nil {
		if *input.Enabled {
			if err := a.pluginMgr.EnablePlugin(name); err != nil {
				respondErrori18n(c, http.StatusInternalServerError, "error.plugin.update_failed", err.Error())
				return
			}
		} else {
			if err := a.pluginMgr.DisablePlugin(name); err != nil {
				respondErrori18n(c, http.StatusInternalServerError, "error.plugin.update_failed", err.Error())
				return
			}
		}
	}

	if input.Config != nil {
		if err := a.pluginMgr.SetPluginConfig(name, input.Config); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.plugin.update_failed", err.Error())
			return
		}
	}

	// Also persist config to DB
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		tenantID = "tenant-default"
	}
	if input.Config != nil {
		configJSON, _ := json.Marshal(input.Config)
		_ = model.UpdatePluginConfigByTenant(tenantID, name, string(configJSON), input.Enabled)
	}

	respondSuccess(c, gin.H{"message": "plugin updated", "name": name})
}

// StartEventPlugin starts a specific event plugin.
// @Summary      Start event plugin
// @Description  Start a specific event-driven plugin
// @Tags         Event Plugins
// @Produce      json
// @Security     BearerAuth
// @Param        name path string true "Plugin name"
// @Success      200 {object} map[string]interface{} "status, data.message"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "not found"
// @Router       /event-plugins/{name}/start [post]
func (a *EventPluginAPI) StartEventPlugin(c *gin.Context) {
	if a.pluginMgr == nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	name := c.Param("name")
	if err := a.pluginMgr.EnablePlugin(name); err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.plugin.not_found")
		return
	}
	respondSuccess(c, gin.H{"message": "plugin started", "name": name})
}

// StopEventPlugin stops a specific event plugin.
// @Summary      Stop event plugin
// @Description  Stop a specific event-driven plugin
// @Tags         Event Plugins
// @Produce      json
// @Security     BearerAuth
// @Param        name path string true "Plugin name"
// @Success      200 {object} map[string]interface{} "status, data.message"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "not found"
// @Router       /event-plugins/{name}/stop [post]
func (a *EventPluginAPI) StopEventPlugin(c *gin.Context) {
	if a.pluginMgr == nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	name := c.Param("name")
	if err := a.pluginMgr.DisablePlugin(name); err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.plugin.not_found")
		return
	}
	respondSuccess(c, gin.H{"message": "plugin stopped", "name": name})
}
