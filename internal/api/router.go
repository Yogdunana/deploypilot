package api

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/backup"
	"github.com/Yogdunana/deploypilot/internal/bruteforce"
	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/metrics"
	"github.com/Yogdunana/deploypilot/internal/middleware"
	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/Yogdunana/deploypilot/internal/sandbox"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"
	"gorm.io/gorm"
)

// globalMonitorAPI is the package-level MonitorAPI instance, accessible via GetGlobalMonitorAPI().
var globalMonitorAPI *MonitorAPI

// GetGlobalMonitorAPI returns the global MonitorAPI instance.
func GetGlobalMonitorAPI() *MonitorAPI { return globalMonitorAPI }

// RegisterRoutes registers all API routes on the given Gin engine.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, bridge *service.Bridge, wsHub *WSHub, auditSvc *service.AuditService, pluginManager *plugin.Manager, eventPluginMgr *plugin.EventPluginManager, blacklist auth.TokenBlacklist, oauthSvc *service.OAuthService, backupSvc *backup.Service, keySvc *service.APIKeyService, metricsPublic bool, grafanaCfg *config.GrafanaConfig, apiPlatformCfg *config.APIPlatformConfig) {
	// Swagger documentation — only accessible in development mode.
	// In production, the endpoint is disabled to prevent information leakage.
	if os.Getenv("DEPLOYPILOT_ENV") == "development" || os.Getenv("GIN_MODE") == "debug" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	} else {
		// In production, require authentication to access Swagger docs
		r.GET("/swagger/*any", auth.AuthMiddleware(blacklist), ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := r.Group("/api/v1")
	api.Use(middleware.APIVersionMiddleware())

	// Public version endpoint (no auth required)
	api.GET("/version", APIVersionHandler)

	// Store db in gin context for handlers that need it via context
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// WebSocket routes (auth via ticket or JWT query param)
	ticketStore := auth.NewWSTicketStore()
	go ticketStore.StartCleanup(context.Background(), 1*time.Minute)

	// Monitor API (created early for WebSocket hub)
	globalMonitorAPI = NewMonitorAPI(db)
	go globalMonitorAPI.GetMonitorHub().Run()

	// Outbound Webhook API
	globalWebhookAPI = NewOutboundWebhookAPI(db)

	// Grafana API
	globalGrafanaAPI = NewGrafanaAPI(db, grafanaCfg)

	// OAuth2 API (API Open Platform)
	if apiPlatformCfg != nil {
		globalOAuth2API = NewOAuth2API(db, apiPlatformCfg)
	}

	wsGroup := r.Group("/ws")
	{
		wsGroup.GET("/logs/:app_id", LogStreamWS(bridge, wsHub, ticketStore))
		wsGroup.GET("/terminal/:server_id", TerminalWS(bridge, wsHub, ticketStore))
		wsGroup.GET("/agent/:server_id", AgentTunnelWS(bridge, ticketStore))
		wsGroup.GET("/monitor", gin.WrapH(http.HandlerFunc(globalMonitorAPI.MonitorWS)))
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

		// Process daemon management
		procAPI := NewProcessAPI(db)
		servers.GET("/:id/processes", procAPI.ListProcesses)
		servers.GET("/:id/processes/tree", procAPI.GetProcessTree)
		servers.GET("/:id/processes/search", procAPI.SearchProcesses)
		servers.GET("/:id/processes/:pid", procAPI.GetProcess)
		servers.POST("/:id/processes/:pid/kill", procAPI.KillProcess)
		servers.GET("/:id/resources", procAPI.GetSystemResources)

		procGroup := protected.Group("/processes")
		{
			procGroup.GET("/rules", procAPI.ListRules)
			procGroup.GET("/rules/:id", procAPI.GetRule)
			procGroup.POST("/rules", procAPI.CreateRule)
			procGroup.PUT("/rules/:id", procAPI.UpdateRule)
			procGroup.DELETE("/rules/:id", procAPI.DeleteRule)
		}

		// System snapshot management
		snapAPI := NewSnapshotAPI(db)
		servers.GET("/:id/snapshots", snapAPI.ListSnapshots)
		servers.GET("/:id/snapshots/files", snapAPI.GetSnapshotFiles)
		servers.GET("/:id/snapshots/diff", snapAPI.DiffSnapshots)
		servers.GET("/:id/snapshots/:snap_id", snapAPI.GetSnapshot)
		servers.POST("/:id/snapshots", snapAPI.CreateSnapshot)
		servers.POST("/:id/snapshots/:snap_id/restore", snapAPI.RestoreSnapshot)
		servers.DELETE("/:id/snapshots/:snap_id", snapAPI.DeleteSnapshot)

		// Toolbox management
		tbAPI := NewToolboxAPI(db)
		servers.GET("/:id/toolbox/detect", tbAPI.DetectEnvironment)
		servers.POST("/:id/toolbox/run", tbAPI.RunScript)
		servers.POST("/:id/toolbox/builtin", tbAPI.RunBuiltInScript)

		tbGroup := protected.Group("/toolbox")
		{
			tbGroup.GET("/scripts/builtin", tbAPI.ListBuiltInScripts)
			tbGroup.GET("/scripts", tbAPI.ListScripts)
			tbGroup.GET("/scripts/:id", tbAPI.GetScript)
			tbGroup.POST("/scripts", tbAPI.CreateScript)
			tbGroup.PUT("/scripts/:id", tbAPI.UpdateScript)
			tbGroup.DELETE("/scripts/:id", tbAPI.DeleteScript)
		}

		// Monitoring & Observability
		monGroup := protected.Group("/monitors")
		{
			monGroup.GET("", globalMonitorAPI.ListMonitors)
			monGroup.GET("/overview", globalMonitorAPI.QueryMonitorOverview)
			monGroup.POST("", globalMonitorAPI.CreateMonitor)
			monGroup.GET("/:id", globalMonitorAPI.GetMonitor)
			monGroup.PUT("/:id", globalMonitorAPI.UpdateMonitor)
			monGroup.DELETE("/:id", globalMonitorAPI.DeleteMonitor)
			monGroup.POST("/:id/check", globalMonitorAPI.CheckMonitor)
			monGroup.GET("/:id/results", globalMonitorAPI.GetMonitorResults)
			monGroup.GET("/:id/sla", globalMonitorAPI.GetMonitorSLA)
			monGroup.GET("/:id/history", globalMonitorAPI.QueryMonitorHistory)
			monGroup.GET("/:id/export", ExportMonitorData(db))
			monGroup.POST("/check-all", globalMonitorAPI.CheckAllMonitors)
		}

		// Heartbeat
		hbGroup := protected.Group("/heartbeats")
		{
			hbGroup.GET("", globalMonitorAPI.ListHeartbeats)
			hbGroup.POST("", globalMonitorAPI.CreateHeartbeat)
			hbGroup.DELETE("/:id", globalMonitorAPI.DeleteHeartbeat)
		}

		// Public endpoints (no auth required for heartbeat ping and status page)
		r.GET("/api/v1/heartbeat/ping/:token", globalMonitorAPI.PingHeartbeat)
		r.GET("/api/v1/status", globalMonitorAPI.GetStatusPage)

		// Outbound Webhooks
		whGroup := protected.Group("/webhooks")
		{
			whGroup.GET("", globalWebhookAPI.ListWebhooks)
			whGroup.POST("", globalWebhookAPI.CreateWebhook)
			whGroup.GET("/:id", globalWebhookAPI.GetWebhook)
			whGroup.PUT("/:id", globalWebhookAPI.UpdateWebhook)
			whGroup.DELETE("/:id", globalWebhookAPI.DeleteWebhook)
			whGroup.POST("/:id/test", globalWebhookAPI.TestWebhook)
			whGroup.GET("/:id/deliveries", globalWebhookAPI.ListDeliveries)
			whGroup.GET("/:id/deliveries/:did", globalWebhookAPI.GetDelivery)
		}

		// Grafana Integration
		grafanaGroup := protected.Group("/grafana")
		{
			grafanaGroup.GET("/status", globalGrafanaAPI.GetStatus)
			grafanaGroup.POST("/test", globalGrafanaAPI.TestConnection)
			grafanaGroup.POST("/sync", globalGrafanaAPI.SyncAll)
			grafanaGroup.GET("/dashboards", globalGrafanaAPI.ListDashboards)
			grafanaGroup.GET("/dashboards/:id", globalGrafanaAPI.GetDashboard)
			grafanaGroup.POST("/dashboards", globalGrafanaAPI.CreateDashboard)
			grafanaGroup.PUT("/dashboards/:id", globalGrafanaAPI.UpdateDashboard)
			grafanaGroup.DELETE("/dashboards/:id", globalGrafanaAPI.DeleteDashboard)
			grafanaGroup.GET("/export", globalGrafanaAPI.ExportAll)
		}

		// OAuth2 client management (protected)
		if globalOAuth2API != nil {
			oauth2 := protected.Group("/oauth")
			{
				oauth2.GET("/clients", globalOAuth2API.ListClients)
				oauth2.POST("/clients", globalOAuth2API.CreateClient)
				oauth2.GET("/clients/:id", globalOAuth2API.GetClient)
				oauth2.PUT("/clients/:id", globalOAuth2API.UpdateClient)
				oauth2.DELETE("/clients/:id", globalOAuth2API.DeleteClient)
				oauth2.POST("/clients/:id/secret", globalOAuth2API.RegenerateSecret)
				oauth2.POST("/authorize", globalOAuth2API.Authorize)
			}
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

		// Audit verification & compliance (Issue #155)
		auditVerGroup := protected.Group("/audit")
		{
			auditVerGroup.GET("/verify", VerifyAuditChain)
			auditVerGroup.GET("/export", ExportAuditLogsV2)
			auditVerGroup.GET("/gdpr/export", GDPRExportUserData)
			auditVerGroup.DELETE("/gdpr/delete", auth.RoleRequired("owner", "admin"), GDPRDeleteUserData)
			auditVerGroup.GET("/compliance", ComplianceReport)
		}
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
			alerts.GET("/history", ListAlertHistory(db))
			alerts.GET("/history/:id", GetAlertHistory(db))
			alerts.GET("/stats", GetAlertStats(db))
			alerts.GET("/export", ExportAlertHistory(db))
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

		// Event Plugins (5 endpoints) — event-driven plugin system
		if eventPluginMgr != nil {
			globalEventPluginAPI = NewEventPluginAPI(eventPluginMgr)
			eventPlugins := protected.Group("/event-plugins")
			{
				eventPlugins.GET("", globalEventPluginAPI.ListEventPlugins)
				eventPlugins.GET("/:name", globalEventPluginAPI.GetEventPlugin)
				eventPlugins.PUT("/:name", globalEventPluginAPI.UpdateEventPlugin)
				eventPlugins.POST("/:name/start", globalEventPluginAPI.StartEventPlugin)
				eventPlugins.POST("/:name/stop", globalEventPluginAPI.StopEventPlugin)
			}
			// Register plugin-specific API routes for each plugin
			for _, p := range eventPluginMgr.ListPlugins() {
				pl, ok := eventPluginMgr.GetPlugin(p.Name)
				if !ok {
					continue
				}
				pluginGroup := eventPlugins.Group("/" + p.Name)
				pl.RegisterAPIRoutes(pluginGroup)
			}
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

		// Per-user IP whitelist (3 endpoints)
		ipWhitelistSvc := service.NewIPWhitelistService(db)
		SetIPWhitelistAPI(NewIPWhitelistAPI(ipWhitelistSvc))
		ipWhitelist := protected.Group("/settings/ip-whitelist")
		{
			ipWhitelist.GET("", ListIPWhitelist)
			ipWhitelist.POST("", AddIPWhitelist)
			ipWhitelist.DELETE("/:id", DeleteIPWhitelist)
			ipWhitelist.GET("/check", CheckIPAccess)
		}

		// Device binding (4 endpoints)
		deviceSvc := service.NewDeviceService(db)
		SetDeviceAPI(NewDeviceAPI(deviceSvc))
		devices := protected.Group("/devices")
		{
			devices.GET("", ListDevices)
			devices.GET("/current", CurrentDevice)
			devices.DELETE("/:id", RevokeDevice)
			devices.POST("/:id/trust", TrustDevice)
		}

		// Code signing (4 endpoints)
		globalSigningAPI = NewSigningAPI(db)
		signingGroup := protected.Group("/security/signing")
		{
			signingGroup.GET("/status", GetSigningStatus)
			signingGroup.POST("/verify", VerifySignature)
			signingGroup.POST("/keys/generate", auth.RoleRequired("owner", "admin"), GenerateKeys)
			signingGroup.POST("/keys/rotate", auth.RoleRequired("owner", "admin"), RotateKeys)
		}

		// License management (3 endpoints)
		SetLicenseAPI(&LicenseAPI{})
		license := protected.Group("/license")
		{
			license.GET("/status", GetLicenseAPI().GetLicenseStatus)
			license.POST("/activate", GetLicenseAPI().ActivateLicense)
			license.POST("/deactivate", GetLicenseAPI().DeactivateLicense)
		}

		// Prometheus metrics (JWT authenticated by default)
		protected.GET("/metrics", gin.WrapH(metrics.Handler()))
	}

	// Public metrics endpoint (when enabled in config)
	if metricsPublic {
		r.GET("/metrics", gin.WrapH(metrics.Handler()))
	}

	// OAuth2 token endpoint (public — uses client credentials)
	if globalOAuth2API != nil {
		oauth2Public := api.Group("/oauth")
		{
			oauth2Public.POST("/token", globalOAuth2API.Token)
			oauth2Public.POST("/token/refresh", globalOAuth2API.RefreshToken)
			oauth2Public.POST("/token/revoke", globalOAuth2API.RevokeToken)
		}
	}
}
