package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// encryptionKeyVal stores the application encryption key for TOTP secret encryption.
var encryptionKeyVal []byte

// SetEncryptionKey sets the encryption key used for TOTP secret encryption.
func SetEncryptionKey(key []byte) {
	encryptionKeyVal = key
}

// encryptionKey returns the current encryption key.
func encryptionKey() []byte {
	return encryptionKeyVal
}

// Verify2FA godoc
// @Summary      Verify 2FA
// @Description  Handle the second step of 2FA login by verifying TOTP code or backup code
// @Tags         2FA
// @Accept       json
// @Produce      json
// @Param        request body object{two_fa_token=string,code=string} true "2FA verification request"
// @Success      200 {object} map[string]interface{} "user info and JWT token"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "invalid token or 2FA code"
// @Failure      429 {object} map[string]interface{} "too many attempts"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /auth/2fa/verify [post]
func Verify2FA(db *gorm.DB, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			TwoFAToken string `json:"two_fa_token" binding:"required"`
			Code       string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		claims, err := auth.ParseToken(input.TwoFAToken)
		if err != nil {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_token")
			return
		}

		var user model.User
		if err := db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_token")
			return
		}

		secret, err := crypto.Decrypt(encryptionKey(), user.TOTPSecret)
		if err != nil || !auth.TOTPValidate(secret, input.Code) {
			idx := auth.VerifyBackupCode(user.BackupCodes, input.Code)
			if idx < 0 {
				respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_2fa_code")
				return
			}
			user.BackupCodes = auth.RemoveBackupCode(user.BackupCodes, idx)
			db.Save(&user)
		}

		token, err := auth.GenerateToken(user.ID, claims.Role)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.auth.failed_to_generate_token")
			return
		}

		if auditSvc != nil {
			if err := auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(user.ID),
				Username:     user.Username,
				Action:       "auth.2fa_verify",
				ResourceType: "user",
				ResourceID:   user.ID,
			}); err != nil {
				slog.WarnContext(c.Request.Context(), "failed to record audit log", "error", err)
			}
		}

		respondSuccess(c, gin.H{
			"user": model.User{
				ID:       user.ID,
				TenantID: user.TenantID,
				RoleID:   user.RoleID,
				Username: user.Username,
				Email:    user.Email,
			},
			"token": token,
		})
	}
}

// Setup2FA godoc
// @Summary      Setup 2FA
// @Description  Generate a new TOTP secret and backup codes for the authenticated user
// @Tags         2FA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "TOTP secret, QR code URL, and backup codes"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "user not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /2fa/setup [post]
func Setup2FA(db *gorm.DB, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		var user model.User
		if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		secret, err := auth.TOTPSecret()
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		encryptedSecret, err := crypto.Encrypt(encryptionKey(), secret)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		plaintextCodes, hashedCodes, err := auth.GenerateBackupCodes()
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		codesJSON, _ := json.Marshal(hashedCodes)

		user.TOTPSecret = encryptedSecret
		user.BackupCodes = string(codesJSON)
		if err := db.Save(&user).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if auditSvc != nil {
			if err := auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     user.Username,
				Action:       "auth.2fa_setup",
				ResourceType: "user",
				ResourceID:   uid,
			}); err != nil {
				slog.WarnContext(c.Request.Context(), "failed to record audit log", "error", err)
			}
		}

		qrURL := auth.TOTPQRCodeURL(secret, user.Username)
		respondSuccess(c, gin.H{
			"secret":       secret,
			"qr_code_url": qrURL,
			"backup_codes": plaintextCodes,
		})
	}
}

// Confirm2FA godoc
// @Summary      Confirm 2FA
// @Description  Enable 2FA after the user verifies a TOTP code
// @Tags         2FA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{code=string} true "TOTP code confirmation request"
// @Success      200 {object} map[string]interface{} "2FA enabled confirmation"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized or invalid 2FA code"
// @Failure      404 {object} map[string]interface{} "user not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /2fa/confirm [post]
func Confirm2FA(db *gorm.DB, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		var input struct {
			Code string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		var user model.User
		if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		secret, err := crypto.Decrypt(encryptionKey(), user.TOTPSecret)
		if err != nil || !auth.TOTPValidate(secret, input.Code) {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_2fa_code")
			return
		}

		user.TOTPEnabled = true
		if err := db.Save(&user).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if auditSvc != nil {
			if err := auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     user.Username,
				Action:       "auth.2fa_confirm",
				ResourceType: "user",
				ResourceID:   uid,
			}); err != nil {
				slog.WarnContext(c.Request.Context(), "failed to record audit log", "error", err)
			}
		}

		respondSuccess(c, gin.H{"enabled": true})
	}
}

// Disable2FA godoc
// @Summary      Disable 2FA
// @Description  Turn off 2FA for the authenticated user (requires current TOTP code)
// @Tags         2FA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{code=string} true "TOTP code to disable 2FA"
// @Success      200 {object} map[string]interface{} "2FA disabled confirmation"
// @Failure      400 {object} map[string]interface{} "invalid request or 2FA not enabled"
// @Failure      401 {object} map[string]interface{} "unauthorized or invalid 2FA code"
// @Failure      404 {object} map[string]interface{} "user not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /2fa/disable [post]
func Disable2FA(db *gorm.DB, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		var input struct {
			Code string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		var user model.User
		if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		if !user.TOTPEnabled {
			respondErrori18n(c, http.StatusBadRequest, "error.auth.2fa_not_enabled")
			return
		}

		secret, err := crypto.Decrypt(encryptionKey(), user.TOTPSecret)
		if err != nil || !auth.TOTPValidate(secret, input.Code) {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_2fa_code")
			return
		}

		user.TOTPEnabled = false
		user.TOTPSecret = ""
		user.BackupCodes = ""
		if err := db.Save(&user).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if auditSvc != nil {
			if err := auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     user.Username,
				Action:       "auth.2fa_disable",
				ResourceType: "user",
				ResourceID:   uid,
			}); err != nil {
				slog.WarnContext(c.Request.Context(), "failed to record audit log", "error", err)
			}
		}

		respondSuccess(c, gin.H{"enabled": false})
	}
}

// RegenerateBackupCodes godoc
// @Summary      Regenerate backup codes
// @Description  Generate new backup codes for the authenticated user (requires current TOTP code)
// @Tags         2FA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{code=string} true "Current TOTP code"
// @Success      200 {object} map[string]interface{} "new backup codes"
// @Failure      400 {object} map[string]interface{} "invalid request or 2FA not enabled"
// @Failure      401 {object} map[string]interface{} "unauthorized or invalid 2FA code"
// @Failure      404 {object} map[string]interface{} "user not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /2fa/regenerate-backup-codes [post]
func RegenerateBackupCodes(db *gorm.DB, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		var input struct {
			Code string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		var user model.User
		if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		if !user.TOTPEnabled {
			respondErrori18n(c, http.StatusBadRequest, "error.auth.2fa_not_enabled")
			return
		}

		secret, err := crypto.Decrypt(encryptionKey(), user.TOTPSecret)
		if err != nil || !auth.TOTPValidate(secret, input.Code) {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_2fa_code")
			return
		}

		plaintextCodes, hashedCodes, err := auth.GenerateBackupCodes()
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		codesJSON, _ := json.Marshal(hashedCodes)
		user.BackupCodes = string(codesJSON)
		if err := db.Save(&user).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if auditSvc != nil {
			if err := auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     user.Username,
				Action:       "auth.2fa_regenerate_codes",
				ResourceType: "user",
				ResourceID:   uid,
			}); err != nil {
				slog.WarnContext(c.Request.Context(), "failed to record audit log", "error", err)
			}
		}

		respondSuccess(c, gin.H{"backup_codes": plaintextCodes})
	}
}

// Get2FAStatus godoc
// @Summary      Get 2FA status
// @Description  Get the current 2FA status and backup code count for the authenticated user
// @Tags         2FA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "2FA status and backup code count"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "user not found"
// @Router       /2fa/status [get]
func Get2FAStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, ok := userID.(string)
		if !ok || uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		var user model.User
		if err := db.Where("id = ?", uid).First(&user).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		backupCodeCount := 0
		if user.BackupCodes != "" {
			var codes []string
			if json.Unmarshal([]byte(user.BackupCodes), &codes) == nil {
				backupCodeCount = len(codes)
			}
		}

		respondSuccess(c, gin.H{
			"enabled":          user.TOTPEnabled,
			"backup_code_count": backupCodeCount,
		})
	}
}

// ResetUser2FA godoc
// @Summary      Reset user 2FA
// @Description  Allow an admin or owner to reset another user's 2FA (requires owner or admin role)
// @Tags         2FA
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Target user ID"
// @Success      200 {object} map[string]interface{} "reset confirmation"
// @Failure      400 {object} map[string]interface{} "invalid request or 2FA not enabled"
// @Failure      404 {object} map[string]interface{} "user not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /users/{id}/reset-2fa [post]
func ResetUser2FA(db *gorm.DB, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetUserID := c.Param("id")
		if targetUserID == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", "user id required")
			return
		}

		var user model.User
		if err := db.Where("id = ?", targetUserID).First(&user).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		if !user.TOTPEnabled && user.TOTPSecret == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.auth.2fa_not_enabled")
			return
		}

		user.TOTPEnabled = false
		user.TOTPSecret = ""
		user.BackupCodes = ""
		if err := db.Save(&user).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		if auditSvc != nil {
			adminID, _ := c.Get(string(auth.UserIDKey))
			adminUID, _ := adminID.(string)
			if err := auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(adminUID),
				Username:     "",
				Action:       "auth.2fa_admin_reset",
				ResourceType: "user",
				ResourceID:   targetUserID,
				Detail:       "admin reset 2FA for user " + user.Username,
			}); err != nil {
				slog.WarnContext(c.Request.Context(), "failed to record audit log", "error", err)
			}
		}

		slog.Info("admin reset 2FA for user", "target_user", user.Username, "target_id", targetUserID)
		respondSuccess(c, gin.H{"message": "2FA has been reset for user " + user.Username})
	}
}

// twoFARateLimiter provides per-IP rate limiting for 2FA verification attempts.
type twoFARateLimiter struct {
	mu          sync.Mutex
	attempts    map[string]*twoFAAttempt
	maxAttempts int
	window      time.Duration
	stopCh      chan struct{}
}

type twoFAAttempt struct {
	count    int
	expireAt time.Time
}

// newTwoFARateLimiter creates a rate limiter for 2FA verification.
func newTwoFARateLimiter(maxAttempts int, window time.Duration) *twoFARateLimiter {
	rl := &twoFARateLimiter{
		attempts:    make(map[string]*twoFAAttempt),
		maxAttempts: maxAttempts,
		window:      window,
		stopCh:      make(chan struct{}),
	}
	// Background cleanup
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.cleanup()
			case <-rl.stopCh:
				return
			}
		}
	}()
	return rl
}

// Stop terminates the background cleanup goroutine.
func (rl *twoFARateLimiter) Stop() {
	close(rl.stopCh)
}

// Allow checks if an IP is allowed to attempt 2FA verification.
func (rl *twoFARateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	a, ok := rl.attempts[ip]
	if !ok || now.After(a.expireAt) {
		rl.attempts[ip] = &twoFAAttempt{count: 1, expireAt: now.Add(rl.window)}
		return true
	}
	a.count++
	return a.count <= rl.maxAttempts
}

// Remaining returns the remaining attempts for an IP.
func (rl *twoFARateLimiter) Remaining(ip string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	a, ok := rl.attempts[ip]
	if !ok || now.After(a.expireAt) {
		return rl.maxAttempts
	}
	remaining := rl.maxAttempts - a.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (rl *twoFARateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, a := range rl.attempts {
		if now.After(a.expireAt) {
			delete(rl.attempts, ip)
		}
	}
}

// Global 2FA rate limiter: max 10 attempts per IP per 5 minutes.
var TwoFARL = newTwoFARateLimiter(10, 5*time.Minute)

// Check2FARateLimit godoc
// @Summary      Check 2FA rate limit
// @Description  Middleware that rate-limits 2FA verification attempts (max 10 per IP per 5 minutes)
// @Tags         2FA
// @Produce      json
// @Failure      429 {object} map[string]interface{} "too many 2FA attempts"
func Check2FARateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !TwoFARL.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": "too many 2FA attempts, please try again later",
				"retry_after_seconds": 300,
			})
			c.Abort()
			return
		}
		c.Header("X-2FA-Attempts-Remaining", strconv.Itoa(TwoFARL.Remaining(ip)))
		c.Next()
	}
}
