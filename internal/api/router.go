package api

import (
	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers all API routes on the given Gin engine.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, bridge *service.Bridge, wsHub *WSHub, auditSvc *service.AuditService) {
	api := r.Group("/api/v1")

	// Store db in gin context for handlers that need it via context
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// WebSocket routes (auth via query param token, not middleware)
	wsGroup := r.Group("/ws")
	{
		wsGroup.GET("/logs/:app_id", LogStreamWS(bridge, wsHub))
		wsGroup.GET("/terminal/:server_id", TerminalWS(bridge, wsHub))
		wsGroup.GET("/agent/:server_id", AgentTunnelWS(bridge))
	}

	// SSE routes (requires auth)
	sseGroup := api.Group("/sse")
	sseGroup.Use(auth.AuthMiddleware())
	{
		sseGroup.GET("/deploy/:app_id", DeploySSE(bridge))
	}

	// Public routes
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", Register(db))
		authGroup.POST("/login", Login(db))
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(auth.AuthMiddleware())
	{
		// Apps (12 endpoints)
		apps := protected.Group("/apps")
		{
			apps.POST("", CreateApp(db))
			apps.GET("", ListApps(db))
			apps.GET("/:id", GetApp(db))
			apps.PUT("/:id", UpdateApp(db))
			apps.DELETE("/:id", DeleteApp(bridge))
			apps.POST("/:id/deploy", DeployApp(bridge))
			apps.POST("/:id/build", BuildAndDeployApp(bridge))
			apps.GET("/:id/status", GetAppStatus(bridge))
			apps.POST("/:id/rollback", RollbackApp(bridge))
			apps.GET("/:id/logs/container", GetContainerLogs(bridge))
			apps.POST("/:id/backup", BackupApp(bridge))
			apps.POST("/:id/restore", RestoreApp(bridge))
			apps.GET("/:id/env", GetAppEnv(db))
			apps.PUT("/:id/env", UpdateAppEnv(db))
		}

		// Servers (7 endpoints)
		servers := protected.Group("/servers")
		{
			servers.POST("", AddServer(bridge))
			servers.GET("", ListServers(bridge))
			servers.PUT("/:id", UpdateServer(bridge))
			servers.DELETE("/:id", DeleteServer(bridge))
			servers.POST("/:id/detect", DetectEnvironment(bridge))
			servers.GET("/:id/environment", GetServerEnvironment(bridge))
			servers.POST("/:id/test", TestServer(bridge))
		}

		// Credentials (4 endpoints)
		creds := protected.Group("/credentials")
		{
			creds.GET("", ListCredentials(bridge))
			creds.POST("", CreateCredential(bridge))
			creds.PUT("/:id", UpdateCredential(bridge))
			creds.DELETE("/:id", DeleteCredential(bridge))
		}

		// DNS (4 endpoints)
		dns := protected.Group("/dns")
		{
			dns.GET("/records", ListDNSRecords(bridge))
			dns.POST("/records", CreateDNSRecord(bridge))
			dns.PUT("/records/:id", UpdateDNSRecord(bridge))
			dns.DELETE("/records/:id", DeleteDNSRecord(bridge))
		}

		// Providers (4 endpoints)
		providers := protected.Group("/providers")
		{
			providers.GET("", ListProviders(db))
			providers.POST("", CreateProvider(db))
			providers.PUT("/:id", UpdateProvider(db))
			providers.DELETE("/:id", DeleteProvider(db))
		}

		// Notifications (4 endpoints)
		notifs := protected.Group("/notifications")
		{
			notifs.GET("", ListNotifications(db))
			notifs.POST("", CreateNotification(db))
			notifs.PUT("/:id", UpdateNotification(db))
			notifs.DELETE("/:id", DeleteNotification(db))
		}

		// Templates (4 endpoints)
		templates := protected.Group("/templates")
		{
			templates.GET("", ListTemplates(bridge))
			templates.POST("", CreateTemplate(db))
			templates.PUT("/:id", UpdateTemplate(db))
			templates.DELETE("/:id", DeleteTemplate(db))
		}

		// Users & Roles
		users := protected.Group("/users")
		{
			users.GET("/me", GetCurrentUser)
			users.GET("", auth.RoleRequired("owner", "admin"), ListUsers(db))
			users.DELETE("/:id", auth.RoleRequired("owner"), DeleteUser(db))
			users.PUT("/:id/role", auth.RoleRequired("owner", "admin"), UpdateUserRole(db))
		}
		roles := protected.Group("/roles")
		{
			roles.GET("", ListRoles(db))
		}

		// Audit logs
		protected.GET("/audit-logs", ListAuditLogs(auditSvc))

		// System (3 endpoints)
		system := protected.Group("/system")
		{
			system.GET("/version", GetVersion)
			system.GET("/update/check", CheckUpdate(bridge))
			system.GET("/health", SystemHealth(db))
		}

		// Deployments (2 endpoints)
		deployments := protected.Group("/deployments")
		{
			deployments.GET("", ListDeployments(db))
			deployments.GET("/:id", GetDeployment(db))
		}

		// Backups (2 endpoints)
		backups := protected.Group("/apps/:id/backups")
		{
			backups.GET("", ListBackups(db))
			backups.DELETE("/:backupId", DeleteBackup(db))
		}

		// Monitor (6 endpoints)
		mon := protected.Group("/monitor")
		{
			mon.GET("/system", GetSystemMetrics(bridge))
			mon.GET("/container/:name", GetContainerMetrics(bridge))
			mon.GET("/alerts", ListAlerts(bridge))
			mon.GET("/alert-rules", ListAlertRules(bridge))
			mon.POST("/heal/:name", HealContainer(bridge))
			mon.POST("/check/:name", CheckContainerHealth(bridge))
		}

		// CI/CD (2 endpoints)
		cicd := protected.Group("/cicd")
		{
			cicd.POST("/trigger", TriggerCIBuild(bridge))
			cicd.GET("/status/:runID", GetCIBuildStatus(bridge))
		}
	}
}
