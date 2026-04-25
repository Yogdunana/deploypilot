package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// ListCredentials lists all credentials for a tenant.
// @Summary      List credentials
// @Description  Retrieve all encrypted credentials for a tenant
// @Tags         Credentials
// @Produce      json
// @Security     BearerAuth
// @Param        tenant_id query string false "Tenant ID" default("tenant-default")
// @Success      200 {object} map[string]interface{} "status, data (array of Credential)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /credentials [get]
func ListCredentials(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Query("tenant_id")
		if tenantID == "" {
			tenantID = "tenant-default"
		}

		creds, err := bridge.ListCredentials(c.Request.Context(), tenantID)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, creds)
	}
}

// CreateCredential creates a new encrypted credential.
// @Summary      Create a credential
// @Description  Create a new encrypted credential entry
// @Tags         Credentials
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{tenant_id=string,name=string,type=string,value=string} true "Credential creation request"
// @Success      200 {object} map[string]interface{} "status, data (Credential)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /credentials [post]
func CreateCredential(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			TenantID string `json:"tenant_id"`
			Name     string `json:"name" binding:"required"`
			Type     string `json:"type" binding:"required"`
			Value    string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}
		if input.TenantID == "" {
			input.TenantID = "tenant-default"
		}

		cred, err := bridge.CreateCredential(c.Request.Context(), input.TenantID, input.Name, input.Type, input.Value)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, cred)
	}
}

// UpdateCredential updates a credential's value.
// @Summary      Update a credential
// @Description  Update the value of an existing credential
// @Tags         Credentials
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Credential ID"
// @Param        request body object{value=string} true "New credential value"
// @Success      200 {object} map[string]interface{} "status, data (updated Credential)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /credentials/{id} [put]
func UpdateCredential(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			Value string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		result, err := bridge.UpdateCredential(c.Request.Context(), id, input.Value)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, result)
	}
}

// DeleteCredential deletes a credential.
// @Summary      Delete a credential
// @Description  Delete a credential by ID
// @Tags         Credentials
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Credential ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /credentials/{id} [delete]
func DeleteCredential(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := bridge.DeleteCredential(c.Request.Context(), id); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{"message": "credential deleted", "id": id})
	}
}
