package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
)

// ListRegistries lists all registries for a tenant.
// @Summary      List registries
// @Description  Retrieve all container image registries for a tenant
// @Tags         Registries
// @Produce      json
// @Security     BearerAuth
// @Param        tenant_id query string false "Tenant ID" default("tenant-default")
// @Success      200 {object} map[string]interface{} "status, data (array of Registry)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /registries [get]
func ListRegistries() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Query("tenant_id")
		if tenantID == "" {
			tenantID = "tenant-default"
		}

		registries, err := model.ListRegistries(tenantID)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		if registries == nil {
			registries = []model.Registry{}
		}
		respondSuccess(c, registries)
	}
}

// CreateRegistry creates a new registry.
// @Summary      Create a registry
// @Description  Create a new container image registry configuration
// @Tags         Registries
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{tenant_id=string,name=string,provider=string,url=string,username=string,password=string} true "Registry creation request"
// @Success      200 {object} map[string]interface{} "status, data (Registry)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /registries [post]
func CreateRegistry() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			TenantID string `json:"tenant_id"`
			Name     string `json:"name" binding:"required"`
			Provider string `json:"provider" binding:"required"`
			URL      string `json:"url" binding:"required"`
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}
		if input.TenantID == "" {
			input.TenantID = "tenant-default"
		}
		if len(input.Name) > 255 || len(input.URL) > 2048 || len(input.Username) > 255 || len(input.Password) > 255 {
			respondErrori18n(c, http.StatusBadRequest, "error.common.input_too_long")
			return
		}

		registry, err := model.CreateRegistry(input.TenantID, input.Name, input.Provider, input.URL, input.Username, input.Password)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, registry)
	}
}

// GetRegistry retrieves a single registry by ID.
// @Summary      Get a registry
// @Description  Retrieve a container image registry by ID
// @Tags         Registries
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Registry ID"
// @Success      200 {object} map[string]interface{} "status, data (Registry)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /registries/{id} [get]
func GetRegistry() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		registry, err := model.GetRegistry(id)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.registry.not_found")
			return
		}
		respondSuccess(c, registry)
	}
}

// UpdateRegistry updates a registry.
// @Summary      Update a registry
// @Description  Update an existing container image registry configuration
// @Tags         Registries
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Registry ID"
// @Param        request body object{name=string,provider=string,url=string,username=string,password=string} true "Fields to update"
// @Success      200 {object} map[string]interface{} "status, data (updated Registry)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /registries/{id} [put]
func UpdateRegistry() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			Name     string `json:"name"`
			Provider string `json:"provider"`
			URL      string `json:"url"`
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		registry, err := model.UpdateRegistry(id, input.Name, input.Provider, input.URL, input.Username, input.Password)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, registry)
	}
}

// DeleteRegistry deletes a registry.
// @Summary      Delete a registry
// @Description  Delete a container image registry by ID
// @Tags         Registries
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Registry ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /registries/{id} [delete]
func DeleteRegistry() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := model.DeleteRegistry(id); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{"message": "registry deleted", "id": id})
	}
}
