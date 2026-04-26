package api

import (
	"context"
	"os"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"
	"gorm.io/gorm"
)

// RegisterRoutes registers all API routes on the given Gin engine.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, bridge *service.Bridge, wsHub *WSHub, auditSvc *service.AuditService, pluginManager *plugin.Manager) {
	// Swagger documentation — only accessible in development mode.
	// In production, the endpoint is disabled to prevent information leakage.
	if os.Getenv("DEPLOYPILOT_ENV") == "development" || os.Getenv("GIN_MODE") == "debug" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	} else {
		// In production, require authentication to access Swagger docs
		r.GET("/swagger/*any", auth.AuthMiddleware(), ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := r.Group("/api/v1")

	// Store db in gin context for handlers that need it via context
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// WebSocket routes (auth via ticket or JWT query param)
	ticketStore := auth.NewWSTicketStore()
	go ticketStore.StartCleanup(context.Background(), 1*time.Minute)

	wsGroup := r.Group("/ws")
	{
		wsGroup.GET("/logs/:app_id", LogStreamWS(bridge, wsHub, ticketStore))
		wsGroup.GET("/terminal/:server_id", TerminalWS(bridge, wsHub, ticketStore))
		wsGroup.GET("/agent/:server_id", AgentTunnelWS(bridge, ticketStore))
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
		authGroup.POST("/ws-ticket", WSTicket(ticketStore, 30*time.Second))
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

		// Credentials (5 endpoints)
		creds := protected.Group("/credentials")
		{
			creds.GET("", ListCredentials(bridge))
			creds.POST("", CreateCredential(bridge))
			creds.PUT("/:id", UpdateCredential(bridge))
			creds.DELETE("/:id", DeleteCredential(bridge))
			creds.POST("/:id/rotate", RotateCredential(bridge, auditSvc))
		}

		// Clusters (6 endpoints)
		clusters := protected.Group("/clusters")
		{
			clusters.GET("", ListClusters(bridge))
			clusters.POST("", CreateCluster(bridge))
			clusters.GET("/:id", GetCluster(bridge))
			clusters.PUT("/:id", UpdateCluster(bridge))
			clusters.DELETE("/:id", DeleteCluster(bridge))
			clusters.POST("/:id/test", TestClusterConnection(bridge))
		}

		// Registries (5 endpoints)
		registries := protected.Group("/registries")
		{
			registries.GET("", ListRegistries())
			registries.POST("", CreateRegistry())
			registries.GET("/:id", GetRegistry())
			registries.PUT("/:id", UpdateRegistry())
			registries.DELETE("/:id", DeleteRegistry())
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

		// SSL (4 endpoints)
		ssl := protected.Group("/ssl")
		{
			ssl.GET("/certificates", ListSSLCertificates(db))
			ssl.POST("/certificates", RequestSSLCertificate(db))
			ssl.DELETE("/certificates/:id", DeleteSSLCertificate(db))
			ssl.POST("/certificates/:id/renew", RenewSSLCertificate(db))
		}

		// Plugins (8 endpoints)
		pluginHandler := NewPluginHandler(pluginManager)
		plugins := protected.Group("/plugins")
		{
			plugins.GET("", pluginHandler.ListPlugins())
			plugins.POST("", pluginHandler.CreatePlugin())
			plugins.GET("/:id", pluginHandler.GetPlugin())
			plugins.PUT("/:id", pluginHandler.UpdatePlugin())
			plugins.DELETE("/:id", pluginHandler.DeletePlugin())
			plugins.POST("/:id/enable", pluginHandler.EnablePlugin())
			plugins.POST("/:id/disable", pluginHandler.DisablePlugin())
			plugins.POST("/:id/reload", pluginHandler.ReloadPlugin())
		}
	}
}
