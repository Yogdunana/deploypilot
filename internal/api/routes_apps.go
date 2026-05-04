package api

import (
	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// registerAppRoutes registers all application-related routes.
func registerAppRoutes(protected *gin.RouterGroup, db *gorm.DB, bridge *service.Bridge) {
		apps := protected.Group("/apps")
		{
			apps.POST("", CreateApp(db))
			apps.GET("", ListApps(db))
			apps.GET("/:id", auth.RequireResourceAccessCached(bridge, "app", "id"), GetApp(db))
			apps.PUT("/:id", auth.RequireResourceAccessCached(bridge, "app", "id"), UpdateApp(db))
			apps.DELETE("/:id", auth.RequireResourceAccessCached(bridge, "app", "id"), DeleteApp(bridge))
			apps.POST("/:id/deploy", auth.RequireResourceAccessCached(bridge, "app", "id"), DeployApp(bridge))
			apps.POST("/:id/build", auth.RequireResourceAccessCached(bridge, "app", "id"), BuildAndDeployApp(bridge))
			apps.GET("/:id/status", auth.RequireResourceAccessCached(bridge, "app", "id"), GetAppStatus(bridge))
			apps.POST("/:id/rollback", auth.RequireResourceAccessCached(bridge, "app", "id"), RollbackApp(bridge))
			apps.GET("/:id/history", auth.RequireResourceAccessCached(bridge, "app", "id"), GetDeploymentHistory(bridge))
			apps.GET("/:id/logs/container", auth.RequireResourceAccessCached(bridge, "app", "id"), GetContainerLogs(bridge))
			apps.POST("/:id/backup", auth.RequireResourceAccessCached(bridge, "app", "id"), BackupApp(bridge))
			apps.POST("/:id/restore", auth.RequireResourceAccessCached(bridge, "app", "id"), RestoreApp(bridge))
			apps.GET("/:id/env", auth.RequireResourceAccessCached(bridge, "app", "id"), GetAppEnv(db))
			apps.PUT("/:id/env", auth.RequireResourceAccessCached(bridge, "app", "id"), UpdateAppEnv(db))
		}

}
