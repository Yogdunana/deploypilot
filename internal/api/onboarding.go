package api

import (
	"log/slog"
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CompleteOnboarding marks the current user's onboarding as completed.
// PUT /api/v1/users/me/onboarding
func CompleteOnboarding(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		result := db.Model(&model.User{}).
			Where("id = ?", uid).
			Update("onboarding_completed", true)
		if result.Error != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		respondSuccess(c, gin.H{"message": "onboarding completed"})
	}
}

// GetOnboardingStatus returns the current user's onboarding status and empty state hints.
// GET /api/v1/users/me/onboarding
func GetOnboardingStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		var user model.User
		if err := db.Select("id, tenant_id, onboarding_completed, last_login_at").
			Where("id = ?", uid).First(&user).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		// Check empty states for each resource type
		var serverCount, appCount, credCount int64
		db.Model(&model.Server{}).Where("tenant_id = ?", user.TenantID).Count(&serverCount)
		db.Model(&model.App{}).Where("tenant_id = ?", user.TenantID).Count(&appCount)
		db.Model(&model.Credential{}).Where("tenant_id = ?", user.TenantID).Count(&credCount)

		isFirstLogin := user.LastLoginAt == nil && !user.OnboardingCompleted

		respondSuccess(c, gin.H{
			"onboarding_completed": user.OnboardingCompleted,
			"is_first_login":       isFirstLogin,
			"empty_state": gin.H{
				"no_servers":      serverCount == 0,
				"no_apps":          appCount == 0,
				"no_credentials":   credCount == 0,
				"server_count":     serverCount,
				"app_count":        appCount,
				"credential_count": credCount,
			},
		})
	}
}

// GenerateDemoData creates sample demo resources (server, app, credential) for the current user.
// POST /api/v1/users/me/demo
func GenerateDemoData(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get(string(auth.UserIDKey))
		uid, _ := userID.(string)
		if uid == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.common.unauthorized")
			return
		}

		var user model.User
		if err := db.Select("id, tenant_id").Where("id = ?", uid).First(&user).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		// Check if demo data already exists
		var existingCount int64
		db.Model(&model.Server{}).Where("tenant_id = ? AND name LIKE ?", user.TenantID, "Demo%").Count(&existingCount)
		if existingCount > 0 {
			respondErrori18n(c, http.StatusConflict, "error.demo.already_exists")
			return
		}

		// Create demo credential
		credID := uuid.New().String()
		demoCred := model.Credential{
			ID:             credID,
			TenantID:       user.TenantID,
			Name:           "Demo SSH Key",
			Type:           "ssh",
			EncryptedValue: "demo-encrypted-key-content",
			RotationDays:   0,
		}
		if err := db.Create(&demoCred).Error; err != nil {
			slog.Error("failed to create demo credential", "error", err)
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		// Create demo server
		serverID := uuid.New().String()
		demoServer := model.Server{
			ID:           serverID,
			TenantID:     user.TenantID,
			CredentialID: credID,
			Name:         "Demo Server",
			Host:         "demo.example.com",
			Port:         22,
			Status:       "unknown",
			Tags:         `["demo","example"]`,
		}
		if err := db.Create(&demoServer).Error; err != nil {
			slog.Error("failed to create demo server", "error", err)
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		// Create demo app
		appID := uuid.New().String()
		demoApp := model.App{
			ID:        appID,
			TenantID:  user.TenantID,
			ServerID:  serverID,
			Name:      "Demo App",
			RepoURL:   "https://github.com/example/demo-app",
			Branch:    "main",
			TechStack: "docker",
			DeployMode: "api",
			Status:    "pending",
		}
		if err := db.Create(&demoApp).Error; err != nil {
			slog.Error("failed to create demo app", "error", err)
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		slog.Info("demo data generated", "user_id", uid, "tenant_id", user.TenantID)

		respondSuccess(c, gin.H{
			"message": "demo data generated",
			"created": gin.H{
				"credential": gin.H{"id": credID, "name": demoCred.Name},
				"server":     gin.H{"id": serverID, "name": demoServer.Name},
				"app":        gin.H{"id": appID, "name": demoApp.Name},
			},
		})
	}
}
