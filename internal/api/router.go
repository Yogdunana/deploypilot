package api

import (
	"context"
	"os"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/backup"
	"github.com/Yogdunana/deploypilot/internal/bruteforce"
	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/Yogdunana/deploypilot/internal/sandbox"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"
	"gorm.io/gorm"
)

// RegisterRoutes registers all API routes on the given Gin engine.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, bridge *service.Bridge, wsHub *WSHub, auditSvc *service.AuditService, pluginManager *plugin.Manager, blacklist auth.TokenBlacklist, oauthSvc *service.OAuthService, backupSvc *backup.Service, keySvc *service.APIKeyService) {
	// Swagger documentation — only accessible in development mode.
	// In production, the endpoint is disabled to prevent information leakage.
	if os.Getenv("DEPLOYPILOT_ENV") == "development" || os.Getenv("GIN_MODE") == "debug" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	} else {
		// In production, require authentication to access Swagger docs
		r.GET("/swagger/*any", auth.AuthMiddleware(blacklist), ginSwagger.WrapHandler(swaggerFiles.Handler))
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
	sseGroup.Use(auth.AuthMiddleware(blacklist))
	{
		sseGroup.GET("/deploy/:app_id", DeploySSE(bridge))
		sseGroup.GET("/alerts", AlertSSE(bridge))
	}

	// Public routes
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", Register(db))
		authGroup.POST("/login", Login(db, func() *bruteforce.Protector {
			if bridge != nil {
				return bridge.BFProtector
			}
			return nil
		}()))
		authGroup.POST("/ws-ticket", WSTicket(ticketStore, 30*time.Second))
		authGroup.POST("/revoke", RevokeToken(blacklist))
		authGroup.POST("/2fa/verify", Check2FARateLimit(), Verify2FA(db, auditSvc))
		authGroup.POST("/refresh", RefreshToken())
		if oauthSvc != nil {
			stateStore := auth.NewMemoryStateStore()
			go stateStore.StartCleanup(context.Background(), 5*time.Minute)
			authGroup.GET("/oauth/:provider", OAuthLogin(oauthSvc, stateStore))
			authGroup.GET("/oauth/:provider/callback", OAuthCallback(oauthSvc, stateStore))
		}
	}

	// Protected routes (JWT or API Key)
	protected := api.Group("")
	protected.Use(auth.APIKeyMiddleware(keySvc))
	protected.Use(auth.AuthMiddleware(blacklist))
	{
		// Apps (12 endpoints)
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

		// Servers (7 endpoints)
		servers := protected.Group("/servers")
		{
			servers.POST("", AddServer(bridge))
			servers.GET("", ListServers(bridge))
			servers.PUT("/:id", auth.RequireResourceAccessCached(bridge, "server", "id"), UpdateServer(bridge))
			servers.DELETE("/:id", auth.RequireResourceAccessCached(bridge, "server", "id"), DeleteServer(bridge))
			servers.POST("/:id/detect", auth.RequireResourceAccessCached(bridge, "server", "id"), DetectEnvironment(bridge))
			servers.GET("/:id/environment", auth.RequireResourceAccessCached(bridge, "server", "id"), GetServerEnvironment(bridge))
			servers.POST("/:id/test", auth.RequireResourceAccessCached(bridge, "server", "id"), TestServer(bridge))

			// File management
			fileAPI := NewFileManagerAPI(db, sandbox.New(sandbox.DefaultConfig()))
			servers.GET("/:id/files", fileAPI.ListFiles)
			servers.GET("/:id/files/read", fileAPI.ReadFile)
			servers.PUT("/:id/files/write", fileAPI.WriteFile)
			servers.DELETE("/:id/files", fileAPI.DeleteFile)
			servers.POST("/:id/files/mkdir", fileAPI.CreateDirectory)
			servers.POST("/:id/files/move", fileAPI.MoveFile)
			servers.GET("/:id/files/disk-usage", fileAPI.GetDiskUsage)
			servers.GET("/:id/files/info", fileAPI.GetFileInfo)
			servers.GET("/:id/files/search", fileAPI.SearchFiles)

			// Firewall management
			fwAPI := NewFirewallAPI(db, sandbox.New(sandbox.DefaultConfig()))
			servers.GET("/:id/firewall", fwAPI.GetFirewallStatus)
			servers.GET("/:id/firewall/detect", fwAPI.DetectFirewall)
			servers.POST("/:id/firewall/ports/open", fwAPI.OpenPort)
			servers.POST("/:id/firewall/ports/close", fwAPI.ClosePort)
			servers.POST("/:id/firewall/blocks", fwAPI.BlockIP)
			servers.DELETE("/:id/firewall/blocks/:ip", fwAPI.UnblockIP)
			servers.POST("/:id/firewall/common-ports", fwAPI.AllowCommonPorts)

			// SSH management
			servers.GET("/:id/ssh/authorizations", NewSSHAPI(db).ListServerAuthorizations)
		}

		// SSH key management (top-level)
		sshAPI := NewSSHAPI(db)
		sshGroup := protected.Group("/ssh")
		{
			sshGroup.POST("/keys/generate", sshAPI.GenerateKeyPair)
			sshGroup.POST("/keys/import", sshAPI.ImportPublicKey)
			sshGroup.GET("/keys", sshAPI.ListKeyPairs)
			sshGroup.GET("/keys/:id", sshAPI.GetKeyPair)
			sshGroup.DELETE("/keys/:id", sshAPI.DeleteKeyPair)
			sshGroup.GET("/keys/:id/authorizations", sshAPI.ListKeyAuthorizations)
			sshGroup.POST("/authorize", sshAPI.AuthorizeKey)
			sshGroup.POST("/revoke", sshAPI.RevokeKey)
		}

		// Credentials (5 endpoints)
		creds := protected.Group("/credentials")
		{
			creds.GET("", ListCredentials(bridge))
			creds.POST("", CreateCredential(bridge, auditSvc))
			creds.PUT("/:id", auth.RequireResourceAccessCached(bridge, "credential", "id"), UpdateCredential(bridge, auditSvc))
			creds.DELETE("/:id", auth.RequireResourceAccessCached(bridge, "credential", "id"), DeleteCredential(bridge, auditSvc))
			creds.POST("/:id/rotate", auth.RequireResourceAccessCached(bridge, "credential", "id"), RotateCredential(bridge, auditSvc))
		}

		// Clusters (6 endpoints)
		clusters := protected.Group("/clusters")
		{
			clusters.GET("", ListClusters(bridge))
			clusters.POST("", CreateCluster(bridge))
			clusters.GET("/:id", auth.RequireResourceAccessCached(bridge, "cluster", "id"), GetCluster(bridge))
			clusters.PUT("/:id", auth.RequireResourceAccessCached(bridge, "cluster", "id"), UpdateCluster(bridge))
			clusters.DELETE("/:id", auth.RequireResourceAccessCached(bridge, "cluster", "id"), DeleteCluster(bridge))
			clusters.POST("/:id/test", auth.RequireResourceAccessCached(bridge, "cluster", "id"), TestClusterConnection(bridge))
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
			users.GET("/me/onboarding", GetOnboardingStatus(db))
			users.PUT("/me/onboarding", CompleteOnboarding(db))
			users.POST("/me/demo", GenerateDemoData(db))
			users.GET("", auth.RoleRequired("owner", "admin"), ListUsers(db))
			users.DELETE("/:id", auth.RoleRequired("owner"), DeleteUser(db))
			users.PUT("/:id/role", auth.RoleRequired("owner", "admin"), UpdateUserRole(db))
			users.POST("/:id/reset-2fa", auth.RoleRequired("owner", "admin"), ResetUser2FA(db, auditSvc))
		}
		roles := protected.Group("/roles")
		{
			roles.GET("", ListRoles(db))
		}

		// Audit logs
		protected.GET("/audit-logs", ListAuditLogs(auditSvc))
		protected.GET("/audit-logs/stats", GetAuditStats(auditSvc))
		protected.GET("/audit-logs/export", ExportAuditLogs(auditSvc))
		protected.POST("/audit-logs/archive", ArchiveAuditLogs(auditSvc))
		protected.POST("/audit-logs/verify", VerifyAuditLogs(auditSvc))
		protected.GET("/audit-logs/trace/:trace_id", GetAuditLogsByTraceID(auditSvc))
		protected.GET("/events", ListEventLogs(db))
		protected.GET("/events/stats", GetEventStats(db))

		// Alert silence management
		alerts := protected.Group("/alerts")
		{
			alerts.POST("/silences", CreateAlertSilence(db))
			alerts.GET("/silences", ListAlertSilences(db))
			alerts.DELETE("/silences/:id", DeleteAlertSilence(db))
			alerts.POST("/escalations", CreateAlertEscalation(db))
			alerts.GET("/escalations", ListAlertEscalations(db))
			alerts.DELETE("/escalations/:id", DeleteAlertEscalation(db))
			alerts.GET("/groups", ListAlertGroups(db))
		}

		// System (4 endpoints)
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

	// Public health check endpoint (no auth required for Docker healthcheck)
	api.GET("/system/health", SystemHealth(db))

		// Deployments (2 endpoints)
		deployments := protected.Group("/deployments")
		{
			deployments.GET("", ListDeployments(db))
			deployments.GET("/:id", GetDeployment(db))
		}

		// Backups (2 endpoints)
		backups := protected.Group("/apps/:id/backups")
		{
			backups.GET("", ListBackups(backupSvc))
			backups.DELETE("/:backupId", DeleteBackup(backupSvc))
		}

		// Monitor (6 endpoints)
		mon := protected.Group("/monitor")
		{
			mon.GET("/system", GetSystemMetrics(bridge))
			mon.GET("/container/:name", GetContainerMetrics(bridge))
			mon.GET("/alerts", ListAlerts(bridge))
			mon.GET("/alert-rules", ListAlertRules(bridge))
			mon.POST("/alert-rules", CreateAlertRule(db))
			mon.GET("/alert-rules/:id", GetAlertRule(db))
			mon.PUT("/alert-rules/:id", UpdateAlertRule(db))
			mon.DELETE("/alert-rules/:id", DeleteAlertRule(db))
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

		// 2FA management (requires authentication)
		twofa := protected.Group("/2fa")
		{
			twofa.GET("/status", Get2FAStatus(db))
			twofa.POST("/setup", Setup2FA(db, auditSvc))
			twofa.POST("/confirm", Confirm2FA(db, auditSvc))
			twofa.POST("/disable", Disable2FA(db, auditSvc))
			twofa.POST("/regenerate-backup-codes", RegenerateBackupCodes(db, auditSvc))
		}

		// Session management
		protected.POST("/auth/logout-all", LogoutAllDevices())
		protected.GET("/sessions", ListSessions())
		protected.DELETE("/sessions/:token_id", KickSession())
		protected.GET("/login-history", ListLoginHistory(auditSvc))

		// API Keys (3 endpoints)
		apiKeys := protected.Group("/api-keys")
		{
			apiKeys.GET("", ListAPIKeys(keySvc))
		apiKeys.POST("", CreateAPIKey(keySvc, auditSvc))
		apiKeys.GET("/:id", GetAPIKey(keySvc))
		apiKeys.PATCH("/:id", UpdateAPIKey(keySvc, auditSvc))
		apiKeys.DELETE("/:id", DeleteAPIKey(keySvc, auditSvc))
		apiKeys.GET("/:id/stats", GetAPIKeyStats(keySvc))
		}
	}
}
