package api

import (
	"encoding/json"
	"net/http"

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

// Verify2FA handles the second step of 2FA login.
func Verify2FA(db *gorm.DB, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			TwoFAToken string `json:"two_fa_token" binding:"required"`
			Code       string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
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
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(user.ID),
				Username:     user.Username,
				Action:       "auth.2fa_verify",
				ResourceType: "user",
				ResourceID:   user.ID,
			})
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

// Setup2FA generates a new TOTP secret and backup codes.
func Setup2FA(db *gorm.DB, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
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
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     user.Username,
				Action:       "auth.2fa_setup",
				ResourceType: "user",
				ResourceID:   uid,
			})
		}

		qrURL := auth.TOTPQRCodeURL(secret, user.Username)
		respondSuccess(c, gin.H{
			"secret":       secret,
			"qr_code_url": qrURL,
			"backup_codes": plaintextCodes,
		})
	}
}

// Confirm2FA enables 2FA after the user verifies a TOTP code.
func Confirm2FA(db *gorm.DB, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		var input struct {
			Code string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
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
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     user.Username,
				Action:       "auth.2fa_confirm",
				ResourceType: "user",
				ResourceID:   uid,
			})
		}

		respondSuccess(c, gin.H{"enabled": true})
	}
}

// Disable2FA turns off 2FA for the user.
func Disable2FA(db *gorm.DB, auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		var input struct {
			Code string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
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
			_ = auditSvc.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(uid),
				Username:     user.Username,
				Action:       "auth.2fa_disable",
				ResourceType: "user",
				ResourceID:   uid,
			})
		}

		respondSuccess(c, gin.H{"enabled": false})
	}
}
