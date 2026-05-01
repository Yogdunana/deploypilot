package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// ListAPIKeys returns all API keys for the authenticated user.
func ListAPIKeys(keySvc *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
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

// CreateAPIKey generates a new API key and returns it (raw key shown only once).
func CreateAPIKey(keySvc *service.APIKeyService, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
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
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		if len(input.Scopes) == 0 {
			input.Scopes = []string{"read"}
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
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     uid,
				Action:       "apikey.create",
				ResourceType: "apikey",
				ResourceID:   apiKey.ID,
				Detail:       map[string]string{"name": input.Name, "prefix": apiKey.KeyPrefix},
			})
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

// DeleteAPIKey revokes an API key.
func DeleteAPIKey(keySvc *service.APIKeyService, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		keyID := c.Param("id")
		if err := keySvc.Delete(c.Request.Context(), keyID, uid); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if auditSvc != nil {
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     uid,
				Action:       "apikey.delete",
				ResourceType: "apikey",
				ResourceID:   keyID,
			})
		}

		respondSuccess(c, gin.H{"message": "API key revoked", "id": keyID})
	}
}

// GetAPIKey returns a single API key's details.
// GET /api/v1/api-keys/:id
func GetAPIKey(keySvc *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
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

// UpdateAPIKey updates an API key's metadata (name, scopes, allowed_ips, expires_at).
// PATCH /api/v1/api-keys/:id
func UpdateAPIKey(keySvc *service.APIKeyService, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
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
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		updates := make(map[string]interface{})
		if input.Name != nil {
			updates["name"] = *input.Name
		}
		if input.Scopes != nil {
			scopesJSON, _ := json.Marshal(input.Scopes)
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
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     uid,
				Action:       "apikey.update",
				ResourceType: "apikey",
				ResourceID:   keyID,
			})
		}

		respondSuccess(c, gin.H{"message": "API key updated", "id": keyID})
	}
}

// GetAPIKeyStats returns usage statistics for a specific API key.
// GET /api/v1/api-keys/:id/stats
func GetAPIKeyStats(keySvc *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
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
