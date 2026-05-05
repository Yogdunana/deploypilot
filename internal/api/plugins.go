package api

import (
	"log/slog"
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/gin-gonic/gin"
)

// PluginHandler holds dependencies for plugin API handlers.
type PluginHandler struct {
	lifecycleManager *plugin.Manager
}

// NewPluginHandler creates a new PluginHandler.
func NewPluginHandler(lifecycleManager *plugin.Manager) *PluginHandler {
	return &PluginHandler{lifecycleManager: lifecycleManager}
}

// ListPlugins lists all plugins for a tenant.
// @Summary      List plugins
// @Description  Retrieve all plugins for a tenant, optionally filtered by provider
// @Tags         Plugins
// @Produce      json
// @Security     BearerAuth
// @Param        tenant_id query string false "Tenant ID" default("tenant-default")
// @Param        provider query string false "Filter by provider (dns, notify, registry, cicd, server, ssl)"
// @Success      200 {object} map[string]interface{} "status, data (array of Plugin)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /plugins [get]
func (h *PluginHandler) ListPlugins() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Query("tenant_id")
		if tenantID == "" {
			tenantID = "tenant-default"
		}
		provider := c.Query("provider")

		plugins, err := model.ListPlugins(tenantID, provider)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		respondSuccess(c, plugins)
	}
}

// CreatePlugin creates a new plugin.
// @Summary      Create a plugin
// @Description  Create a new plugin configuration
// @Tags         Plugins
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{tenant_id=string,name=string,display_name=string,version=string,description=string,author=string,provider=string,type=string,config=string,enabled=bool,priority=int} true "Plugin creation request"
// @Success      200 {object} map[string]interface{} "status, data (Plugin)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /plugins [post]
func (h *PluginHandler) CreatePlugin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			TenantID    string `json:"tenant_id"`
			Name        string `json:"name" binding:"required"`
			DisplayName string `json:"display_name"`
			Version     string `json:"version"`
			Description string `json:"description"`
			Author      string `json:"author"`
			Provider    string `json:"provider" binding:"required"`
			Type        string `json:"type" binding:"required"`
			Config      string `json:"config"`
			Enabled     *bool  `json:"enabled"`
			Priority    int    `json:"priority"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}
		if input.TenantID == "" {
			input.TenantID = "tenant-default"
		}
		if input.Version == "" {
			input.Version = "1.0.0"
		}

		enabled := true
		if input.Enabled != nil {
			enabled = *input.Enabled
		}

		plug, err := model.CreatePlugin(
			input.TenantID, input.Name, input.DisplayName,
			input.Version, input.Description, input.Author,
			input.Provider, input.Type, input.Config,
			enabled, input.Priority,
		)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to create plugin", "error", err)
			respondErrori18n(c, http.StatusInternalServerError, "error.plugin.create_failed")
			return
		}

		respondSuccess(c, plug)
	}
}

// GetPlugin retrieves a plugin by ID.
// @Summary      Get a plugin
// @Description  Retrieve a plugin by ID
// @Tags         Plugins
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Plugin ID"
// @Success      200 {object} map[string]interface{} "status, data (Plugin)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /plugins/{id} [get]
func (h *PluginHandler) GetPlugin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		plug, err := model.GetPlugin(id)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.plugin.not_found")
			return
		}

		respondSuccess(c, plug)
	}
}

// UpdatePlugin updates a plugin.
// @Summary      Update a plugin
// @Description  Update a plugin's configuration
// @Tags         Plugins
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Plugin ID"
// @Param        request body object{display_name=string,version=string,description=string,author=string,config=string,enabled=bool,priority=int} true "Plugin update request"
// @Success      200 {object} map[string]interface{} "status, data (updated Plugin)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /plugins/{id} [put]
func (h *PluginHandler) UpdatePlugin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input map[string]interface{}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		plug, err := model.UpdatePlugin(id, input)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.plugin.not_found")
			return
		}

		respondSuccess(c, plug)
	}
}

// DeletePlugin deletes a plugin by ID.
// @Summary      Delete a plugin
// @Description  Delete a plugin by ID
// @Tags         Plugins
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Plugin ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /plugins/{id} [delete]
func (h *PluginHandler) DeletePlugin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Unload the plugin instance first
		if h.lifecycleManager != nil {
			_ = h.lifecycleManager.UnloadPlugin(id)
		}

		if err := model.DeletePlugin(id); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.plugin.not_found")
			return
		}

		respondSuccess(c, gin.H{"message": "plugin deleted", "id": id})
	}
}

// EnablePlugin enables a plugin and loads its instance.
// @Summary      Enable a plugin
// @Description  Enable a plugin and load its instance
// @Tags         Plugins
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Plugin ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /plugins/{id}/enable [post]
func (h *PluginHandler) EnablePlugin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if h.lifecycleManager == nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if err := h.lifecycleManager.EnablePlugin(c.Request.Context(), id); err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to enable plugin", "error", err)
			respondErrori18n(c, http.StatusInternalServerError, "error.plugin.instance_create_failed")
			return
		}

		respondSuccess(c, gin.H{"message": "plugin enabled", "id": id})
	}
}

// DisablePlugin disables a plugin and unloads its instance.
// @Summary      Disable a plugin
// @Description  Disable a plugin and unload its instance
// @Tags         Plugins
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Plugin ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /plugins/{id}/disable [post]
func (h *PluginHandler) DisablePlugin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if h.lifecycleManager == nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if err := h.lifecycleManager.DisablePlugin(c.Request.Context(), id); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.plugin.not_found")
			return
		}

		respondSuccess(c, gin.H{"message": "plugin disabled", "id": id})
	}
}

// ReloadPlugin reloads a plugin (unload + load).
// @Summary      Reload a plugin
// @Description  Reload a plugin by unloading and re-loading its instance
// @Tags         Plugins
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Plugin ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /plugins/{id}/reload [post]
func (h *PluginHandler) ReloadPlugin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if h.lifecycleManager == nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if err := h.lifecycleManager.ReloadPlugin(c.Request.Context(), id); err != nil {
			slog.ErrorContext(c.Request.Context(), "failed to reload plugin", "error", err)
			respondErrori18n(c, http.StatusInternalServerError, "error.plugin.instance_create_failed")
			return
		}

		respondSuccess(c, gin.H{"message": "plugin reloaded", "id": id})
	}
}
