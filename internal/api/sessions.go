package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// sessionResponse is the API representation of an active session.
type sessionResponse struct {
	TokenID    string    `json:"token_id"`
	DeviceInfo string    `json:"device_info"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IsCurrent  bool      `json:"is_current"`
}

// ListSessions godoc
// @Summary      List sessions
// @Description  Get all active sessions for the authenticated user
// @Tags         Sessions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "list of active sessions"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /sessions [get]
func ListSessions() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
			respondError(c, http.StatusUnauthorized, "unauthorized")
			return
		}

		if refreshStore == nil {
			respondSuccess(c, gin.H{"sessions": []sessionResponse{}})
			return
		}

		entries, err := refreshStore.ListForUser(uid)
		if err != nil {
			slog.Error("failed to list sessions", "error", err)
			respondError(c, http.StatusInternalServerError, "failed to list sessions")
			return
		}

		// Determine current session (from cookie or Authorization header)
		currentTokenID := ""
		if cookie, err := c.Cookie(auth.AccessTokenCookie); err == nil && cookie != "" {
			// The access token itself isn't the refresh token, but we can't easily
			// match. Use the refresh token cookie if available.
			if rtCookie, err := c.Cookie(auth.RefreshTokenCookie); err == nil {
				currentTokenID = rtCookie
			}
		}

		sessions := make([]sessionResponse, 0, len(entries))
		for _, e := range entries {
			sessions = append(sessions, sessionResponse{
				TokenID:    e.TokenID,
				DeviceInfo: e.DeviceInfo,
				IPAddress:  e.IPAddress,
				CreatedAt:  e.CreatedAt,
				ExpiresAt:  e.ExpiresAt,
				IsCurrent:  e.TokenID == currentTokenID,
			})
		}

		respondSuccess(c, gin.H{
			"sessions": sessions,
			"count":    len(sessions),
		})
	}
}

// KickSession godoc
// @Summary      Kick session
// @Description  Revoke a single session (refresh token) for the authenticated user
// @Tags         Sessions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        token_id path string true "Refresh token ID"
// @Success      200 {object} map[string]interface{} "revocation confirmation"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      403 {object} map[string]interface{} "cannot kick another user's session"
// @Failure      404 {object} map[string]interface{} "session not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /sessions/{token_id} [delete]
func KickSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
			respondError(c, http.StatusUnauthorized, "unauthorized")
			return
		}

		tokenID := c.Param("token_id")
		if tokenID == "" {
			respondError(c, http.StatusBadRequest, "token_id is required")
			return
		}

		if refreshStore == nil {
			respondError(c, http.StatusInternalServerError, "session store not configured")
			return
		}

		// Verify the token belongs to the current user
		entry, err := refreshStore.Retrieve(tokenID)
		if err != nil {
			slog.Error("failed to retrieve session", "error", err)
			respondError(c, http.StatusInternalServerError, "failed to retrieve session")
			return
		}
		if entry == nil {
			respondError(c, http.StatusNotFound, "session not found")
			return
		}
		if entry.UserID != uid {
			respondError(c, http.StatusForbidden, "cannot kick another user's session")
			return
		}

		if err := refreshStore.Revoke(tokenID); err != nil {
			slog.Error("failed to revoke session", "error", err)
			respondError(c, http.StatusInternalServerError, "failed to revoke session")
			return
		}

		respondSuccess(c, gin.H{"message": "session revoked"})
	}
}

// ListLoginHistory godoc
// @Summary      List login history
// @Description  Get paginated login history for the authenticated user
// @Tags         Sessions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page number (default 1)"
// @Param        page_size query int false "Page size (default 20, max 100)"
// @Success      200 {object} map[string]interface{} "paginated login history"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /login-history [get]
func ListLoginHistory(auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
			respondError(c, http.StatusUnauthorized, "unauthorized")
			return
		}

		if auditSvc == nil {
			respondSuccess(c, gin.H{"history": []interface{}{}, "total": 0})
			return
		}

		// Parse pagination params
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		// Parse user ID to uint for audit filter
		uidUint := parseUserID(uid)

		logs, total, err := auditSvc.List(c.Request.Context(), service.AuditFilter{
			UserID:   uidUint,
			Action:   "user.login",
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			slog.Error("failed to query login history", "error", err)
			respondError(c, http.StatusInternalServerError, "failed to query login history")
			return
		}

		type loginEntry struct {
			ID        uint      `json:"id"`
			IPAddress string    `json:"ip_address"`
			UserAgent string    `json:"user_agent"`
			Detail    string    `json:"detail,omitempty"`
			CreatedAt time.Time `json:"created_at"`
		}

		history := make([]loginEntry, 0, len(logs))
		for _, log := range logs {
			history = append(history, loginEntry{
				ID:        log.ID,
				IPAddress: log.IPAddress,
				UserAgent: log.UserAgent,
				Detail:    log.Detail,
				CreatedAt: log.CreatedAt,
			})
		}

		respondSuccess(c, gin.H{
			"history":   history,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
	}
}
