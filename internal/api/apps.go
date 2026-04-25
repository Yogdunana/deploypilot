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
// @Summary      Create a new application
// @Description  Create a new application configuration with the provided details
// @Tags         Apps
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{name=string,repo_url=string,branch=string,domain=string,tech_stack=string,deploy_mode=string,server_id=string} true "Application creation request"
// @Success      200 {object} map[string]interface{} "status, data (App object)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps [post]
func CreateApp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input mcp.CreateAppConfig
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}
		if input.Name == "" || input.RepoURL == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.app.name_and_repo_required")
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
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, app)
	}
}

// ListApps lists all applications.
// @Summary      List all applications
// @Description  Retrieve a list of all registered applications
// @Tags         Apps
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (array of App)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps [get]
func ListApps(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var apps []model.App
		if err := db.Find(&apps).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		if apps == nil {
			apps = []model.App{}
		}
		respondSuccess(c, apps)
	}
}

// GetApp returns a single application by ID.
// @Summary      Get an application
// @Description  Retrieve details of a specific application by its ID
// @Tags         Apps
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Success      200 {object} map[string]interface{} "status, data (App object)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "app not found"
// @Router       /apps/{id} [get]
func GetApp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var app model.App
		if err := db.Where("id = ?", id).First(&app).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.app.not_found")
			return
		}
		respondSuccess(c, app)
	}
}

// UpdateApp updates an application by ID.
// @Summary      Update an application
// @Description  Update fields of an existing application by ID
// @Tags         Apps
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Param        request body map[string]interface{} true "Fields to update"
// @Success      200 {object} map[string]interface{} "status, data (updated App object)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id} [put]
func UpdateApp(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		if err := db.Model(&model.App{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
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
// @Summary      Delete an application
// @Description  Delete an application and its associated resources by ID
// @Tags         Apps
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id} [delete]
func DeleteApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := bridge.DeleteApp(c.Request.Context(), id); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{"message": "app deleted", "id": id})
	}
}

// DeployApp deploys an application.
// @Summary      Deploy an application
// @Description  Deploy an application using the provided deploy configuration. Set query param async=true for async deployment.
// @Tags         Apps
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Param        async query string false "Set to 'true' for async deployment" default(false)
// @Param        request body object{app_name=string,image=string,container_name=string,port=int,env_vars=map[string]string,volumes=[]string,server_id=string} false "Deploy configuration"
// @Success      200 {object} map[string]interface{} "status, data (ContainerStatus)"
// @Failure      400 {object} map[string]interface{} "invalid request or preflight failure"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id}/deploy [post]
func DeployApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check async query param
		if c.Query("async") == "true" {
			DeployAsyncHandler(bridge)(c)
			return
		}

		var cfg mcp.DeployConfig
		if err := c.ShouldBindJSON(&cfg); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
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
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, cs)
	}
}

// GetAppStatus returns the status of a deployed container.
// @Summary      Get application status
// @Description  Get the current deployment status of an application's container
// @Tags         Apps
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Success      200 {object} map[string]interface{} "status, data (ContainerStatus)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "app not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id}/status [get]
func GetAppStatus(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var app model.App
		if err := bridge.DB.Where("id = ?", id).First(&app).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.app.not_found")
			return
		}

		containerName := app.ContainerName
		if containerName == "" {
			containerName = app.Name
		}

		cs, err := bridge.GetContainerStatus(c.Request.Context(), containerName)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, cs)
	}
}

// RollbackApp rolls back an application to a previous image.
// @Summary      Rollback an application
// @Description  Roll back an application container to a previous Docker image
// @Tags         Apps
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Param        request body object{previous_image=string} true "Rollback request with previous image"
// @Success      200 {object} map[string]interface{} "status, data (ContainerStatus)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "app not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id}/rollback [post]
func RollbackApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			PreviousImage string `json:"previous_image"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.app.previous_image_required")
			return
		}

		var app model.App
		if err := bridge.DB.Where("id = ?", id).First(&app).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.app.not_found")
			return
		}

		containerName := app.ContainerName
		if containerName == "" {
			containerName = app.Name
		}

		cs, err := bridge.Rollback(c.Request.Context(), containerName, input.PreviousImage)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, cs)
	}
}

// GetContainerLogs returns container logs.
// @Summary      Get container logs
// @Description  Retrieve the most recent log lines from an application's container
// @Tags         Apps
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Param        tail query int false "Number of log lines to retrieve" default(100)
// @Success      200 {object} map[string]interface{} "status, data.container_name, data.tail, data.logs"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "app not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id}/logs/container [get]
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
			respondErrori18n(c, http.StatusNotFound, "error.app.not_found")
			return
		}

		containerName := app.ContainerName
		if containerName == "" {
			containerName = app.Name
		}

		logs, err := bridge.GetContainerLogs(c.Request.Context(), containerName, tail)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
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
// @Summary      Backup an application
// @Description  Create a backup snapshot of an application
// @Tags         Apps
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Success      200 {object} map[string]interface{} "status, data.backup_id, data.app_id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id}/backup [post]
func BackupApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		backupID, err := bridge.Backup(c.Request.Context(), id)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{"backup_id": backupID, "app_id": id})
	}
}

// RestoreApp restores an application from a backup.
// @Summary      Restore an application from backup
// @Description  Restore an application to a previous backup state
// @Tags         Apps
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Param        request body object{backup_id=string} true "Restore request with backup ID"
// @Success      200 {object} map[string]interface{} "status, data (ContainerStatus)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id}/restore [post]
func RestoreApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			BackupID string `json:"backup_id"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.BackupID == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.app.backup_id_required")
			return
		}

		cs, err := bridge.Restore(c.Request.Context(), input.BackupID)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, cs)
	}
}

// GetAppEnv returns environment variables for an application.
// @Summary      Get application environment variables
// @Description  Retrieve the environment variables configured for an application
// @Tags         Apps
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Success      200 {object} map[string]interface{} "status, data.app_id, data.env_vars"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "app not found"
// @Router       /apps/{id}/env [get]
func GetAppEnv(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var app model.App
		if err := db.Select("id, env_vars").Where("id = ?", id).First(&app).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.app.not_found")
			return
		}
		respondSuccess(c, gin.H{
			"app_id":   id,
			"env_vars": app.EnvVars,
		})
	}
}

// UpdateAppEnv updates environment variables for an application.
// @Summary      Update application environment variables
// @Description  Update the environment variables for an application
// @Tags         Apps
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Param        request body object{env_vars=string} true "Environment variables string"
// @Success      200 {object} map[string]interface{} "status, data.app_id, data.message"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id}/env [put]
func UpdateAppEnv(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			EnvVars string `json:"env_vars"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		if err := db.Model(&model.App{}).Where("id = ?", id).Update("env_vars", input.EnvVars).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
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
// @Summary      Build and deploy an application
// @Description  Build an application from git source and deploy it. Supports optional overrides for branch, tech stack, ports, env vars, and server.
// @Tags         Apps
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Param        request body object{branch=string,tech_stack=string,ports=string,env_vars=map[string]string,server_id=string} false "Build and deploy overrides"
// @Success      200 {object} map[string]interface{} "status, data (BuildResult)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "app not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id}/build [post]
func BuildAndDeployApp(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.Param("id")

		// Get app from DB
		var app model.App
		if err := bridge.DB.Where("id = ?", appID).First(&app).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.app.not_found")
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
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		respondSuccess(c, result)
	}
}
