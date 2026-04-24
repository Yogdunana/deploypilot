package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateApp creates a new application.
func CreateApp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input mcp.CreateAppConfig
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}
		if input.Name == "" || input.RepoURL == "" {
			respondError(c, http.StatusBadRequest, "name and repo_url are required")
			return
		}

		id := uuid.New().String()
		tenantID := c.GetString(string(auth.UserIDKey))
		if tenantID == "" {
			tenantID = "tenant-default"
		}

		app := model.App{
			ID:        id,
			TenantID:  tenantID,
			Name:      input.Name,
			RepoURL:   input.RepoURL,
			Branch:    input.Branch,
			Domain:    input.Domain,
			TechStack: input.TechStack,
			DeployMode: input.DeployMode,
			ServerID:  input.ServerID,
			Status:    "created",
		}
		if app.Branch == "" {
			app.Branch = "main"
		}
		if app.TechStack == "" {
			app.TechStack = "docker"
		}
		if app.DeployMode == "" {
			app.DeployMode = "api"
		}

		if err := db.Create(&app).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, app)
	}
}

// ListApps lists all applications.
func ListApps(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var apps []model.App
		if err := db.Find(&apps).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if apps == nil {
			apps = []model.App{}
		}
		respondSuccess(c, apps)
	}
}

// GetApp returns a single application by ID.
func GetApp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var app model.App
		if err := db.Where("id = ?", id).First(&app).Error; err != nil {
			respondError(c, http.StatusNotFound, "app not found")
			return
		}
		respondSuccess(c, app)
	}
}

// UpdateApp updates an application by ID.
func UpdateApp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		if err := db.Model(&model.App{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}

		var app model.App
		if err := db.Where("id = ?", id).First(&app).Error; err != nil {
			respondSuccess(c, gin.H{"status": "updated", "id": id})
			return
		}
		respondSuccess(c, app)
	}
}

// DeleteApp deletes an application by ID.
func DeleteApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := bridge.DeleteApp(c.Request.Context(), id); err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, gin.H{"message": "app deleted", "id": id})
	}
}

// DeployApp deploys an application.
func DeployApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cfg mcp.DeployConfig
		if err := c.ShouldBindJSON(&cfg); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		cs, err := bridge.Deploy(c.Request.Context(), cfg)
		if err != nil {
			// Check for preflight error
			var pfErr *service.PreflightError
			if errors.As(err, &pfErr) {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  "preflight_failed",
					"code":    pfErr.Code,
					"message": pfErr.Message,
					"checks":  pfErr.Checks,
				})
				return
			}
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, cs)
	}
}

// GetAppStatus returns the status of a deployed container.
func GetAppStatus(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var app model.App
		if err := bridge.DB.Where("id = ?", id).First(&app).Error; err != nil {
			respondError(c, http.StatusNotFound, "app not found")
			return
		}

		containerName := app.ContainerName
		if containerName == "" {
			containerName = app.Name
		}

		cs, err := bridge.GetContainerStatus(c.Request.Context(), containerName)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, cs)
	}
}

// RollbackApp rolls back an application to a previous image.
func RollbackApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			PreviousImage string `json:"previous_image"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "previous_image is required")
			return
		}

		var app model.App
		if err := bridge.DB.Where("id = ?", id).First(&app).Error; err != nil {
			respondError(c, http.StatusNotFound, "app not found")
			return
		}

		containerName := app.ContainerName
		if containerName == "" {
			containerName = app.Name
		}

		cs, err := bridge.Rollback(c.Request.Context(), containerName, input.PreviousImage)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, cs)
	}
}

// GetContainerLogs returns container logs.
func GetContainerLogs(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		tail := 100
		if t := c.Query("tail"); t != "" {
			if parsed, err := strconv.Atoi(t); err == nil && parsed > 0 {
				tail = parsed
			}
		}

		var app model.App
		if err := bridge.DB.Where("id = ?", id).First(&app).Error; err != nil {
			respondError(c, http.StatusNotFound, "app not found")
			return
		}

		containerName := app.ContainerName
		if containerName == "" {
			containerName = app.Name
		}

		logs, err := bridge.GetContainerLogs(c.Request.Context(), containerName, tail)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, gin.H{
			"container_name": containerName,
			"tail":           tail,
			"logs":           logs,
		})
	}
}

// BackupApp creates a backup of an application.
func BackupApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		backupID, err := bridge.Backup(c.Request.Context(), id)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, gin.H{"backup_id": backupID, "app_id": id})
	}
}

// RestoreApp restores an application from a backup.
func RestoreApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			BackupID string `json:"backup_id"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.BackupID == "" {
			respondError(c, http.StatusBadRequest, "backup_id is required")
			return
		}

		cs, err := bridge.Restore(c.Request.Context(), input.BackupID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, cs)
	}
}

// GetAppEnv returns environment variables for an application.
func GetAppEnv(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var app model.App
		if err := db.Select("id, env_vars").Where("id = ?", id).First(&app).Error; err != nil {
			respondError(c, http.StatusNotFound, "app not found")
			return
		}
		respondSuccess(c, gin.H{
			"app_id":   id,
			"env_vars": app.EnvVars,
		})
	}
}

// UpdateAppEnv updates environment variables for an application.
func UpdateAppEnv(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			EnvVars string `json:"env_vars"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		if err := db.Model(&model.App{}).Where("id = ?", id).Update("env_vars", input.EnvVars).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, gin.H{"app_id": id, "message": "env vars updated"})
	}
}

// coalesce returns the first non-empty string.
func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// BuildAndDeployApp builds and deploys an application from git source.
func BuildAndDeployApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.Param("id")

		// Get app from DB
		var app model.App
		if err := bridge.DB.Where("id = ?", appID).First(&app).Error; err != nil {
			respondError(c, http.StatusNotFound, "app not found")
			return
		}

		// Parse optional overrides from request body
		var overrides struct {
			Branch    string            `json:"branch"`
			TechStack string            `json:"tech_stack"`
			Ports     string            `json:"ports"`
			EnvVars   map[string]string `json:"env_vars"`
			ServerID  string            `json:"server_id"`
		}
		_ = c.ShouldBindJSON(&overrides)

		cfg := mcp.BuildAndDeployConfig{
			RepoURL:   app.RepoURL,
			AppName:   app.Name,
			Branch:    coalesce(overrides.Branch, app.Branch),
			TechStack: coalesce(overrides.TechStack, app.TechStack),
			Ports:     coalesce(overrides.Ports, ""),
			EnvVars:   overrides.EnvVars,
			ServerID:  coalesce(overrides.ServerID, app.ServerID),
		}

		result, err := bridge.BuildAndDeploy(c.Request.Context(), cfg)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(c, result)
	}
}
