package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/model"
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

		// Enrich with lifecycle info
		rows, ok := creds.([]map[string]interface{})
		if !ok {
			respondSuccess(c, creds)
			return
		}

		enriched := make([]map[string]interface{}, 0, len(rows))
		for _, r := range rows {
			entry := r

			// Parse expires_at and compute lifecycle fields
			var expiresAt *time.Time
			if eaStr, ok := entry["expires_at"].(string); ok && eaStr != "" {
				t, err := time.Parse(time.RFC3339, eaStr)
				if err == nil {
					expiresAt = &t
				}
			}

			if expiresAt != nil {
				cred := &model.Credential{ExpiresAt: expiresAt}
				entry["is_expired"] = model.IsExpired(cred)
				entry["days_until_expiry"] = model.DaysUntilExpiry(cred)
			} else {
				entry["is_expired"] = false
				entry["days_until_expiry"] = -1
			}

			enriched = append(enriched, entry)
		}

		respondSuccess(c, enriched)
	}
}

// CreateCredential creates a new encrypted credential.
// @Summary      Create a credential
// @Description  Create a new encrypted credential entry
// @Tags         Credentials
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{tenant_id=string,name=string,type=string,value=string,expires_in_days=int} true "Credential creation request"
// @Success      200 {object} map[string]interface{} "status, data (Credential)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /credentials [post]
func CreateCredential(bridge *service.Bridge, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			TenantID      string `json:"tenant_id"`
			Name          string `json:"name" binding:"required"`
			Type          string `json:"type" binding:"required"`
			Value         string `json:"value" binding:"required"`
			ExpiresInDays int    `json:"expires_in_days"` // 0 = never expires
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}
		if input.TenantID == "" {
			input.TenantID = "tenant-default"
		}

		cred, err := bridge.CreateCredentialWithExpiry(c.Request.Context(), input.TenantID, input.Name, input.Type, input.Value, input.ExpiresInDays)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		// Record audit log
		if auditSvc != nil {
			userID, _ := c.Get(string(auth.UserIDKey))
			uid, _ := userID.(string)
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     uid,
				Action:       "credential.create",
				ResourceType: "credential",
				ResourceID:   fmt.Sprintf("%v", cred),
				Detail:       map[string]string{"name": input.Name, "type": input.Type},
			})
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
func UpdateCredential(bridge *service.Bridge, auditSvc *service.AuditService) gin.HandlerFunc {
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
		// Record audit log
		if auditSvc != nil {
			userID, _ := c.Get(string(auth.UserIDKey))
			uid, _ := userID.(string)
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     uid,
				Action:       "credential.update",
				ResourceType: "credential",
				ResourceID:   id,
			})
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
func DeleteCredential(bridge *service.Bridge, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := bridge.DeleteCredential(c.Request.Context(), id); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		// Record audit log
		if auditSvc != nil {
			userID, _ := c.Get(string(auth.UserIDKey))
			uid, _ := userID.(string)
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     uid,
				Action:       "credential.delete",
				ResourceType: "credential",
				ResourceID:   id,
			})
		}

		respondSuccess(c, gin.H{"message": "credential deleted", "id": id})
	}
}

// RotateCredential rotates a credential's value and records an audit log.
// @Summary      Rotate a credential
// @Description  Rotate a credential's encrypted value
// @Tags         Credentials
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Credential ID"
// @Param        request body object{value=string} true "New credential value"
// @Success      200 {object} map[string]interface{} "status, data (rotated Credential)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /credentials/{id}/rotate [post]
func RotateCredential(bridge *service.Bridge, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			Value string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		result, err := bridge.RotateCredential(c.Request.Context(), id, input.Value)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.credential.rotate_failed")
			return
		}

		// Record audit log
		if auditSvc != nil {
			userID, _ := c.Get(string(auth.UserIDKey))
			uid, _ := userID.(string)
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     uid,
				Action:       "credential.rotate",
				ResourceType: "credential",
				ResourceID:   id,
				Detail:       map[string]string{"name": fmt.Sprintf("%v", result)},
			})
		}

		respondSuccess(c, gin.H{
			"id":          id,
			"status":      "rotated",
			"message":     "credential value rotated",
			"last_rotated": time.Now().Format(time.RFC3339),
		})
	}
}

// parseUserID converts a string user ID to uint for audit logging.
func parseUserID(s string) uint {
	n, _ := strconv.ParseUint(s, 10, 64)
	return uint(n)
}
