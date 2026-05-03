package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// globalIPWhitelistAPI is the package-level IPWhitelistAPI instance.
var globalIPWhitelistAPI *IPWhitelistAPI

// IPWhitelistAPI handles per-user IP whitelist HTTP endpoints.
type IPWhitelistAPI struct {
	svc *service.IPWhitelistService
}

// NewIPWhitelistAPI creates a new IPWhitelistAPI.
func NewIPWhitelistAPI(svc *service.IPWhitelistService) *IPWhitelistAPI {
	return &IPWhitelistAPI{svc: svc}
}

// SetIPWhitelistAPI sets the global IPWhitelistAPI instance.
func SetIPWhitelistAPI(api *IPWhitelistAPI) {
	globalIPWhitelistAPI = api
}

// ListIPWhitelist returns all whitelist entries for the current user.
// GET /api/v1/settings/ip-whitelist
func ListIPWhitelist(c *gin.Context) {
	if globalIPWhitelistAPI == nil {
		respondError(c, http.StatusInternalServerError, "ip whitelist service not initialized")
		return
	}

	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	entries, err := globalIPWhitelistAPI.svc.List(userID.(string))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list IP whitelist entries")
		return
	}

	respondSuccess(c, entries)
}

// AddIPWhitelist creates a new whitelist entry for the current user.
// POST /api/v1/settings/ip-whitelist
func AddIPWhitelist(c *gin.Context) {
	if globalIPWhitelistAPI == nil {
		respondError(c, http.StatusInternalServerError, "ip whitelist service not initialized")
		return
	}

	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	var input struct {
		Description string `json:"description"`
		CIDR        string `json:"cidr" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", "cidr is required")
		return
	}

	entry, err := globalIPWhitelistAPI.svc.Create(
		userID.(string),
		input.Description,
		input.CIDR,
		"tenant-default",
		userID.(string),
	)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, entry)
}

// DeleteIPWhitelist removes a whitelist entry owned by the current user.
// DELETE /api/v1/settings/ip-whitelist/:id
func DeleteIPWhitelist(c *gin.Context) {
	if globalIPWhitelistAPI == nil {
		respondError(c, http.StatusInternalServerError, "ip whitelist service not initialized")
		return
	}

	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	id := c.Param("id")
	if id == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", "id is required")
		return
	}

	if err := globalIPWhitelistAPI.svc.Delete(id, userID.(string)); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, gin.H{"deleted": true})
}

// CheckIPAccess returns the current user's IP and whitelist status.
// GET /api/v1/settings/ip-whitelist/check
func CheckIPAccess(c *gin.Context) {
	if globalIPWhitelistAPI == nil {
		respondError(c, http.StatusInternalServerError, "ip whitelist service not initialized")
		return
	}

	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	clientIP := c.ClientIP()
	enforced := globalIPWhitelistAPI.svc.IsEnforced(userID.(string))
	allowed := globalIPWhitelistAPI.svc.Check(clientIP, userID.(string))

	respondSuccess(c, gin.H{
		"ip":              clientIP,
		"enforced":        enforced,
		"allowed":         allowed,
		"whitelist_count": 0, // populated below
	})

	// Re-respond with full data including count
	entries, err := globalIPWhitelistAPI.svc.List(userID.(string))
	if err == nil {
		respondSuccess(c, gin.H{
			"ip":              clientIP,
			"enforced":        enforced,
			"allowed":         allowed,
			"whitelist_count": len(entries),
		})
	}
}
