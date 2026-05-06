package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// ListAPIKeys godoc
// @Summary      List API keys
// @Description  Get all API keys for the authenticated user
// @Tags         API Keys
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "list of API keys"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /api-keys [get]
func ListAPIKeys(keySvc *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		keys, err := keySvc.List(c.Request.Context(), uid)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, keys)
	}
}

// CreateAPIKey godoc
// @Summary      Create API key
// @Description  Generate a new API key and return it (raw key shown only once)
// @Tags         API Keys
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{name=string,scopes=[]string,allowed_ips=[]string,expires_in_days=int} true "API key creation request"
// @Success      200 {object} map[string]interface{} "created API key with raw key"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /api-keys [post]
func CreateAPIKey(keySvc *service.APIKeyService, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		var input struct {
			Name          string   `json:"name" binding:"required"`
			Scopes        []string `json:"scopes"`
			AllowedIPs    []string `json:"allowed_ips"`
			ExpiresInDays int      `json:"expires_in_days"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		if len(input.Scopes) == 0 {
			input.Scopes = []string{"read"}
		} else {
			input.Scopes = auth.ValidateScopes(input.Scopes)
			if len(input.Scopes) == 0 {
				input.Scopes = []string{"read"}
			}
		}

		apiKey, rawKey, err := keySvc.Create(c.Request.Context(), uid, "tenant-default", input.Name, input.Scopes, input.ExpiresInDays)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		// Store allowed IPs if provided
		if len(input.AllowedIPs) > 0 {
			ipsJSON, _ := json.Marshal(input.AllowedIPs)
			_ = keySvc.Update(c.Request.Context(), apiKey.ID, uid, map[string]interface{}{
				"allowed_ips": string(ipsJSON),
			})
			apiKey.AllowedIPs = string(ipsJSON)
		}

		if auditSvc != nil {
			if err := auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     uid,
				Action:       "apikey.create",
				ResourceType: "apikey",
				ResourceID:   apiKey.ID,
				Detail:       map[string]string{"name": input.Name, "prefix": apiKey.KeyPrefix},
			}); err != nil {
				slog.WarnContext(c.Request.Context(), "failed to record audit log", "error", err)
			}
		}

		respondSuccess(c, gin.H{
			"id":         apiKey.ID,
			"name":       apiKey.Name,
			"key":        rawKey,
			"key_prefix": apiKey.KeyPrefix,
			"scopes":     input.Scopes,
			"expires_at": apiKey.ExpiresAt,
			"created_at": apiKey.CreatedAt,
		})
	}
}

// DeleteAPIKey godoc
// @Summary      Delete API key
// @Description  Revoke an API key by its ID
// @Tags         API Keys
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "API key ID"
// @Success      200 {object} map[string]interface{} "revocation confirmation"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /api-keys/{id} [delete]
func DeleteAPIKey(keySvc *service.APIKeyService, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		keyID := c.Param("id")
		if err := keySvc.Delete(c.Request.Context(), keyID, uid); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if auditSvc != nil {
			if err := auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     uid,
				Action:       "apikey.delete",
				ResourceType: "apikey",
				ResourceID:   keyID,
			}); err != nil {
				slog.WarnContext(c.Request.Context(), "failed to record audit log", "error", err)
			}
		}

		respondSuccess(c, gin.H{"message": "API key revoked", "id": keyID})
	}
}

// GetAPIKey godoc
// @Summary      Get API key
// @Description  Get details of a single API key by its ID
// @Tags         API Keys
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "API key ID"
// @Success      200 {object} map[string]interface{} "API key details"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "not found"
// @Router       /api-keys/{id} [get]
func GetAPIKey(keySvc *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		keyID := c.Param("id")
		apiKey, err := keySvc.GetByID(c.Request.Context(), keyID, uid)
		if err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		respondSuccess(c, apiKey)
	}
}

// UpdateAPIKey godoc
// @Summary      Update API key
// @Description  Update an API key's metadata (name, scopes, allowed IPs, expiration)
// @Tags         API Keys
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "API key ID"
// @Param        request body object{name=string,scopes=[]string,allowed_ips=[]string,expires_in_days=int} true "API key update request"
// @Success      200 {object} map[string]interface{} "update confirmation"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /api-keys/{id} [patch]
func UpdateAPIKey(keySvc *service.APIKeyService, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		keyID := c.Param("id")
		var input struct {
			Name          *string  `json:"name"`
			Scopes        []string `json:"scopes"`
			AllowedIPs    []string `json:"allowed_ips"`
			ExpiresInDays *int     `json:"expires_in_days"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		updates := make(map[string]interface{})
		if input.Name != nil {
			updates["name"] = *input.Name
		}
		if input.Scopes != nil {
			validatedScopes := auth.ValidateScopes(input.Scopes)
			if len(validatedScopes) == 0 {
				validatedScopes = []string{"read"}
			}
			scopesJSON, _ := json.Marshal(validatedScopes)
			updates["scopes"] = string(scopesJSON)
		}
		if input.AllowedIPs != nil {
			if len(input.AllowedIPs) == 0 {
				updates["allowed_ips"] = ""
			} else {
				ipsJSON, _ := json.Marshal(input.AllowedIPs)
				updates["allowed_ips"] = string(ipsJSON)
			}
		}
		if input.ExpiresInDays != nil {
			if *input.ExpiresInDays <= 0 {
				updates["expires_at"] = nil
			} else {
				expires := time.Now().AddDate(0, 0, *input.ExpiresInDays)
				updates["expires_at"] = expires
			}
		}

		if len(updates) == 0 {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", "no fields to update")
			return
		}

		if err := keySvc.Update(c.Request.Context(), keyID, uid, updates); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if auditSvc != nil {
			if err := auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     uid,
				Action:       "apikey.update",
				ResourceType: "apikey",
				ResourceID:   keyID,
			}); err != nil {
				slog.WarnContext(c.Request.Context(), "failed to record audit log", "error", err)
			}
		}

		respondSuccess(c, gin.H{"message": "API key updated", "id": keyID})
	}
}

// GetAPIKeyStats godoc
// @Summary      Get API key stats
// @Description  Get usage statistics for a specific API key
// @Tags         API Keys
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "API key ID"
// @Success      200 {object} map[string]interface{} "API key usage statistics"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "not found"
// @Router       /api-keys/{id}/stats [get]
func GetAPIKeyStats(keySvc *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		keyID := c.Param("id")
		apiKey, err := keySvc.GetByID(c.Request.Context(), keyID, uid)
		if err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		respondSuccess(c, gin.H{
			"id":          apiKey.ID,
			"name":        apiKey.Name,
			"key_prefix":  apiKey.KeyPrefix,
			"usage_count": apiKey.UsageCount,
			"last_used_at": apiKey.LastUsedAt,
			"created_at":  apiKey.CreatedAt,
		})
	}
}
