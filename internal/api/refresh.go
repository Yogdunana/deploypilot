package api

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/gin-gonic/gin"
)

// refreshStore holds the global refresh token store.
var refreshStore auth.RefreshTokenStore

// SetRefreshTokenStore sets the global refresh token store.
func SetRefreshTokenStore(store auth.RefreshTokenStore) {
	refreshStore = store
}

// cookieConfig holds cookie settings.
type cookieConfig struct {
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
	Path     string
	Domain   string
	MaxAge   int // seconds; 0 = session cookie
}

var (
	defaultCookieConfig cookieConfig
	cookieConfigOnce   sync.Once
)

// SetCookieConfig updates the cookie configuration.
func SetCookieConfig(cfg cookieConfig) {
	cookieConfigOnce.Do(func() {
		defaultCookieConfig = cfg
	})
}

// setAuthCookies sets both access and refresh token cookies on the response.
func setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	cfg := defaultCookieConfig

	// Access token cookie (short-lived, httpOnly)
	c.SetSameSite(cfg.SameSite)
	c.SetCookie(auth.AccessTokenCookie, accessToken, cfg.MaxAge, cfg.Path, cfg.Domain, cfg.Secure, cfg.HTTPOnly)

	// Refresh token cookie (longer-lived, httpOnly)
	refreshMaxAge := 7 * 24 * 3600 // 7 days default
	if refreshMaxAge > 0 {
		c.SetCookie(auth.RefreshTokenCookie, refreshToken, refreshMaxAge, cfg.Path, cfg.Domain, cfg.Secure, cfg.HTTPOnly)
	}
}

// clearAuthCookies removes both auth cookies from the response.
func clearAuthCookies(c *gin.Context) {
	cfg := defaultCookieConfig
	c.SetSameSite(cfg.SameSite)
	c.SetCookie(auth.AccessTokenCookie, "", -1, cfg.Path, cfg.Domain, cfg.Secure, cfg.HTTPOnly)
	c.SetCookie(auth.RefreshTokenCookie, "", -1, cfg.Path, cfg.Domain, cfg.Secure, cfg.HTTPOnly)
}

// RefreshToken handles token refresh requests.
// POST /api/v1/auth/refresh
func RefreshToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get refresh token from cookie first, then from request body
		refreshToken := ""
		if cookie, err := c.Cookie(auth.RefreshTokenCookie); err == nil && cookie != "" {
			refreshToken = cookie
		}

		if refreshToken == "" {
			var input struct {
				RefreshToken string `json:"refresh_token"`
			}
			if c.ShouldBindJSON(&input) == nil && input.RefreshToken != "" {
				refreshToken = input.RefreshToken
			}
		}

		if refreshToken == "" {
			respondError(c, http.StatusUnauthorized, "refresh token is required")
			return
		}

		if refreshStore == nil {
			respondError(c, http.StatusInternalServerError, "refresh token store not configured")
			return
		}

		// Look up the refresh token
		entry, err := refreshStore.Retrieve(refreshToken)
		if err != nil {
			slog.Error("failed to validate refresh token", "error", err)
			respondError(c, http.StatusInternalServerError, "failed to validate refresh token")
			return
		}
		if entry == nil {
			clearAuthCookies(c)
			respondError(c, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}

		// Rotate: revoke old refresh token, issue new pair
		_ = refreshStore.Revoke(refreshToken)

		// Generate new access token
		accessToken, err := auth.GenerateToken(entry.UserID, entry.Role)
		if err != nil {
			slog.Error("failed to generate access token", "error", err)
			respondError(c, http.StatusInternalServerError, "failed to generate access token")
			return
		}

		// Generate new refresh token
		newRefreshID, err := auth.GenerateRefreshTokenID()
		if err != nil {
			slog.Error("failed to generate refresh token", "error", err)
			respondError(c, http.StatusInternalServerError, "failed to generate refresh token")
			return
		}

		newEntry := auth.RefreshTokenEntry{
			TokenID:    newRefreshID,
			UserID:     entry.UserID,
			Role:       entry.Role,
			DeviceInfo: c.GetHeader("User-Agent"),
			IPAddress:  c.ClientIP(),
			CreatedAt:  time.Now(),
			ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
		}
		if err := refreshStore.Store(newEntry); err != nil {
			slog.Error("failed to store refresh token", "error", err)
			respondError(c, http.StatusInternalServerError, "failed to store refresh token")
			return
		}

		// Set cookies
		setAuthCookies(c, accessToken, newRefreshID)

		respondSuccess(c, gin.H{
			"token":         accessToken,
			"refresh_token": newRefreshID,
			"expires_in":    86400, // 24h in seconds
		})
	}
}

// LogoutAllDevices revokes all refresh tokens for the authenticated user.
// POST /api/v1/auth/logout-all
func LogoutAllDevices() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
			respondError(c, http.StatusUnauthorized, "unauthorized")
			return
		}

		if refreshStore != nil {
			_ = refreshStore.RevokeAllForUser(uid)
		}

		clearAuthCookies(c)
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "logged out from all devices"})
	}
}
