package api

import (
	"net/http"

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
