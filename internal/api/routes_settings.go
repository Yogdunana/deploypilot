package api

import (
	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/backup"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// registerSettingsRoutes registers all system settings and configuration routes.
func registerSettingsRoutes(protected *gin.RouterGroup, db *gorm.DB, bridge *service.Bridge, backupSvc *backup.Service) {
		system := protected.Group("/system")
		{
			system.GET("/version", GetVersion)
			system.GET("/update/check", CheckUpdate(bridge))
			system.POST("/update/perform", auth.RoleRequired("owner", "admin"), DoUpdate(bridge))
			system.GET("/sandbox", GetSandboxConfig(bridge))
			system.POST("/sandbox/validate", ValidateSandboxCommand(bridge))
			system.GET("/confirmations", ListConfirmations(bridge))
			system.POST("/confirmations/:id/confirm", auth.RoleRequired("owner", "admin"), ConfirmRequest(bridge))
			system.POST("/confirmations/:id/reject", auth.RoleRequired("owner", "admin"), RejectRequest(bridge))
			system.GET("/bruteforce", GetBruteForceStatus(bridge))
			system.POST("/bruteforce/accounts/:username/unlock", auth.RoleRequired("owner", "admin"), UnlockBruteForceAccount(bridge))
			system.POST("/bruteforce/ips/:ip/unlock", auth.RoleRequired("owner", "admin"), UnlockBruteForceIP(bridge))
			system.GET("/backup/status", GetBackupStatus(backupSvc))
			system.POST("/backup/trigger", auth.RoleRequired("owner", "admin"), TriggerBackup(backupSvc))
			system.GET("/backup/records", ListBackupRecords(backupSvc))
			system.DELETE("/backup/records/:id", auth.RoleRequired("owner", "admin"), DeleteBackupRecord(backupSvc))
			system.POST("/backup/cloud/download/:id", auth.RoleRequired("owner", "admin"), DownloadCloudBackup(backupSvc))
			system.POST("/backup/cloud/retention", auth.RoleRequired("owner", "admin"), ApplyCloudRetention(backupSvc))
			if bridge != nil {
				system.GET("/detect/all", DetectAllComponents(bridge.Executor))
				system.GET("/detect/databases", DetectDatabases(bridge.Executor))
				system.GET("/detect/webservers", DetectWebServers(bridge.Executor))
			}
			system.GET("/security/config", GetSecurityConfig())
			system.PUT("/security/config", auth.RoleRequired("owner", "admin"), UpdateSecurityConfig())
		}

}
