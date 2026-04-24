package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// ListCredentials lists all credentials for a tenant.
func ListCredentials(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Query("tenant_id")
		if tenantID == "" {
			tenantID = "tenant-default"
		}

		creds, err := bridge.ListCredentials(c.Request.Context(), tenantID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, creds)
	}
}

// CreateCredential creates a new encrypted credential.
func CreateCredential(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			TenantID string `json:"tenant_id"`
			Name     string `json:"name" binding:"required"`
			Type     string `json:"type" binding:"required"`
			Value    string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}
		if input.TenantID == "" {
			input.TenantID = "tenant-default"
		}

		cred, err := bridge.CreateCredential(c.Request.Context(), input.TenantID, input.Name, input.Type, input.Value)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, cred)
	}
}

// UpdateCredential updates a credential's value.
func UpdateCredential(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			Value string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		result, err := bridge.UpdateCredential(c.Request.Context(), id, input.Value)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// DeleteCredential deletes a credential.
func DeleteCredential(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := bridge.DeleteCredential(c.Request.Context(), id); err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, gin.H{"message": "credential deleted", "id": id})
	}
}
