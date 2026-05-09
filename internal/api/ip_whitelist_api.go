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

// getUserIDStr safely extracts user ID string from context
func getUserIDStr(c *gin.Context) (string, bool) {
	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		return "", false
	}
	userIDStr, ok := userID.(string)
	return userIDStr, ok
}

// ListIPWhitelist godoc
// @Summary      List IP whitelist
// @Description  Get all whitelist entries for the current user
// @Tags         IP Whitelist
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "list of IP whitelist entries"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /settings/ip-whitelist [get]
func ListIPWhitelist(c *gin.Context) {
	if globalIPWhitelistAPI == nil {
		respondError(c, http.StatusInternalServerError, "ip whitelist service not initialized")
		return
	}

	userIDStr, ok := getUserIDStr(c)
	if !ok {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	entries, err := globalIPWhitelistAPI.svc.List(userIDStr)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list IP whitelist entries")
		return
	}

	respondSuccess(c, entries)
}

// AddIPWhitelist godoc
// @Summary      Add IP whitelist entry
// @Description  Create a new whitelist entry for the current user
// @Tags         IP Whitelist
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{description=string,cidr=string} true "IP whitelist entry (cidr is required)"
// @Success      200 {object} map[string]interface{} "created whitelist entry"
// @Failure      400 {object} map[string]interface{} "invalid request or creation failed"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /settings/ip-whitelist [post]
func AddIPWhitelist(c *gin.Context) {
	if globalIPWhitelistAPI == nil {
		respondError(c, http.StatusInternalServerError, "ip whitelist service not initialized")
		return
	}

	userIDStr, ok := getUserIDStr(c)
	if !ok {
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
		userIDStr,
		input.Description,
		input.CIDR,
		"tenant-default",
		userIDStr,
	)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, entry)
}

// DeleteIPWhitelist godoc
// @Summary      Delete IP whitelist entry
// @Description  Remove a whitelist entry owned by the current user
// @Tags         IP Whitelist
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Whitelist entry ID"
// @Success      200 {object} map[string]interface{} "deletion confirmation"
// @Failure      400 {object} map[string]interface{} "invalid request or deletion failed"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /settings/ip-whitelist/{id} [delete]
func DeleteIPWhitelist(c *gin.Context) {
	if globalIPWhitelistAPI == nil {
		respondError(c, http.StatusInternalServerError, "ip whitelist service not initialized")
		return
	}

	userIDStr, ok := getUserIDStr(c)
	if !ok {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	id := c.Param("id")
	if id == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", "id is required")
		return
	}

	if err := globalIPWhitelistAPI.svc.Delete(id, userIDStr); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	respondSuccess(c, gin.H{"deleted": true})
}

// CheckIPAccess godoc
// @Summary      Check IP access
// @Description  Get the current user's IP and whitelist enforcement status
// @Tags         IP Whitelist
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "IP access status with enforcement info"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /settings/ip-whitelist/check [get]
func CheckIPAccess(c *gin.Context) {
	if globalIPWhitelistAPI == nil {
		respondError(c, http.StatusInternalServerError, "ip whitelist service not initialized")
		return
	}

	userIDStr, ok := getUserIDStr(c)
	if !ok {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.authentication_required")
		return
	}

	clientIP := c.ClientIP()
	enforced := globalIPWhitelistAPI.svc.IsEnforced(userIDStr)
	allowed := globalIPWhitelistAPI.svc.Check(clientIP, userIDStr)

	respondSuccess(c, gin.H{
		"ip":              clientIP,
		"enforced":        enforced,
		"allowed":         allowed,
		"whitelist_count": 0, // populated below
	})

	// Re-respond with full data including count
	entries, err := globalIPWhitelistAPI.svc.List(userIDStr)
	if err == nil {
		respondSuccess(c, gin.H{
			"ip":              clientIP,
			"enforced":        enforced,
			"allowed":         allowed,
			"whitelist_count": len(entries),
		})
	}
}
