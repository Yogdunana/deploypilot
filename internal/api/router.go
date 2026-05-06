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
		wsGroup.GET("/monitor", auth.AuthMiddleware(blacklist), gin.WrapH(http.HandlerFunc(globalMonitorAPI.MonitorWS)))
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

	// Public health check endpoint (no auth required for Docker healthcheck)
	api.GET("/system/health", SystemHealth(db))

	// Protected routes (JWT or API Key)
	protected := api.Group("")
	protected.Use(auth.APIKeyMiddleware(keySvc))
	protected.Use(auth.AuthMiddleware(blacklist))
	{

		// Register sub-route groups
		registerAppRoutes(protected, db, bridge)
		registerServerRoutes(protected, db, bridge)
		registerMonitorRoutes(protected, db, bridge)
		registerSettingsRoutes(protected, db, bridge, backupSvc)

		// Public endpoints (no auth required for heartbeat ping and status page)
		r.GET("/api/v1/heartbeat/ping/:token", globalMonitorAPI.PingHeartbeat)
		r.GET("/api/v1/status", globalMonitorAPI.GetStatusPage)

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
			registries.GET("/:id", auth.RequireResourceAccessCached(bridge, "registry", "id"), GetRegistry())
			registries.PUT("/:id", auth.RequireResourceAccessCached(bridge, "registry", "id"), UpdateRegistry())
			registries.DELETE("/:id", auth.RequireResourceAccessCached(bridge, "registry", "id"), DeleteRegistry())
		}

		// DNS (4 endpoints)
		dns := protected.Group("/dns")
		{
			dns.GET("/records", ListDNSRecords(bridge))
			dns.POST("/records", CreateDNSRecord(bridge))
			dns.PUT("/records/:id", UpdateDNSRecord(bridge))
			dns.DELETE("/records/:id", auth.RequireScope("delete"), DeleteDNSRecord(bridge))
		}

		// Providers (4 endpoints)
		providers := protected.Group("/providers")
		{
			providers.GET("", ListProviders(db))
			providers.POST("", CreateProvider(db))
			providers.PUT("/:id", UpdateProvider(db))
			providers.DELETE("/:id", auth.RequireScope("delete"), DeleteProvider(db))
		}

		// Notifications (4 endpoints)
		notifs := protected.Group("/notifications")
		{
			notifs.GET("", ListNotifications(db))
			notifs.POST("", CreateNotification(db))
			notifs.PUT("/:id", UpdateNotification(db))
			notifs.DELETE("/:id", auth.RequireScope("delete"), DeleteNotification(db))
		}

		// Templates (4 endpoints)
		templates := protected.Group("/templates")
		{
			templates.GET("", ListTemplates(bridge))
			templates.POST("", CreateTemplate(db))
			templates.PUT("/:id", UpdateTemplate(db))
			templates.DELETE("/:id", auth.RequireScope("delete"), DeleteTemplate(db))
		}

		// Users & Roles
		users := protected.Group("/users")
		{
			users.GET("/me", GetCurrentUser)
			users.GET("/me/onboarding", GetOnboardingStatus(db))
			users.PUT("/me/onboarding", CompleteOnboarding(db))
			users.POST("/me/demo", GenerateDemoData(db))
			users.GET("", auth.RoleRequired("owner", "admin"), ListUsers(db))
			users.DELETE("/:id", auth.RequireScope("delete"), auth.RoleRequired("owner"), DeleteUser(db))
			users.PUT("/:id/role", auth.RoleRequired("owner", "admin"), UpdateUserRole(db))
			users.POST("/:id/reset-2fa", auth.RoleRequired("owner", "admin"), ResetUser2FA(db, auditSvc))
		}
		roles := protected.Group("/roles")
		{
			roles.GET("", ListRoles(db))
		}

		// Audit logs (restricted to owner/admin)
		auditGroup := protected.Group("")
		auditGroup.Use(auth.RoleRequired("owner", "admin"))
		{
			auditGroup.GET("/audit-logs", ListAuditLogs(auditSvc))
			auditGroup.GET("/audit-logs/stats", GetAuditStats(auditSvc))
			auditGroup.GET("/audit-logs/export", ExportAuditLogs(auditSvc))
			auditGroup.POST("/audit-logs/archive", ArchiveAuditLogs(auditSvc))
			auditGroup.POST("/audit-logs/verify", VerifyAuditLogs(auditSvc))
			auditGroup.GET("/audit-logs/trace/:trace_id", GetAuditLogsByTraceID(auditSvc))
		}

		// Audit verification & compliance (Issue #155)
		auditVerGroup := protected.Group("/audit")
		auditVerGroup.Use(auth.RoleRequired("owner", "admin"))
		{
			auditVerGroup.GET("/verify", VerifyAuditChain)
			auditVerGroup.GET("/export", ExportAuditLogsV2)
			auditVerGroup.GET("/gdpr/export", GDPRExportUserData)
			auditVerGroup.DELETE("/gdpr/delete", auth.RequireScope("delete"), GDPRDeleteUserData)
			auditVerGroup.GET("/compliance", ComplianceReport)
		}
		protected.GET("/events", ListEventLogs(db))
		protected.GET("/events/stats", GetEventStats(db))

		// Alert silence management
		alerts := protected.Group("/alerts")
		{
			alerts.POST("/silences", CreateAlertSilence(db))
			alerts.GET("/silences", ListAlertSilences(db))
			alerts.DELETE("/silences/:id", auth.RequireScope("delete"), DeleteAlertSilence(db))
			alerts.POST("/escalations", CreateAlertEscalation(db))
			alerts.GET("/escalations", ListAlertEscalations(db))
			alerts.DELETE("/escalations/:id", auth.RequireScope("delete"), DeleteAlertEscalation(db))
			alerts.GET("/groups", ListAlertGroups(db))
			alerts.GET("/history", ListAlertHistory(db))
			alerts.GET("/history/:id", GetAlertHistory(db))
			alerts.GET("/stats", GetAlertStats(db))
			alerts.GET("/export", ExportAlertHistory(db))
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
			backups.GET("", ListBackups(backupSvc))
			backups.DELETE("/:backupId", auth.RequireScope("delete"), DeleteBackup(backupSvc))
		}

		// CI/CD (2 endpoints)
		cicd := protected.Group("/cicd")
		{
			cicd.POST("/trigger", auth.RequireScope("deploy"), TriggerCIBuild(bridge))
			cicd.GET("/status/:runID", GetCIBuildStatus(bridge))
		}

		// SSL (4 endpoints)
		ssl := protected.Group("/ssl")
		{
			ssl.GET("/certificates", ListSSLCertificates(db))
			ssl.POST("/certificates", RequestSSLCertificate(db))
			ssl.DELETE("/certificates/:id", auth.RequireScope("delete"), DeleteSSLCertificate(db))
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
			plugins.DELETE("/:id", auth.RequireScope("delete"), pluginHandler.DeletePlugin())
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
		protected.DELETE("/sessions/:token_id", auth.RequireScope("delete"), KickSession())
		protected.GET("/login-history", ListLoginHistory(auditSvc))

		// API Keys (3 endpoints)
		apiKeys := protected.Group("/api-keys")
		{
			apiKeys.GET("", ListAPIKeys(keySvc))
			apiKeys.POST("", CreateAPIKey(keySvc, auditSvc))
			apiKeys.GET("/:id", GetAPIKey(keySvc))
			apiKeys.PATCH("/:id", UpdateAPIKey(keySvc, auditSvc))
			apiKeys.DELETE("/:id", auth.RequireScope("delete"), DeleteAPIKey(keySvc, auditSvc))
			apiKeys.GET("/:id/stats", GetAPIKeyStats(keySvc))
		}

		// Per-user IP whitelist (3 endpoints)
		ipWhitelistSvc := service.NewIPWhitelistService(db)
		SetIPWhitelistAPI(NewIPWhitelistAPI(ipWhitelistSvc))
		ipWhitelist := protected.Group("/settings/ip-whitelist")
		{
			ipWhitelist.GET("", ListIPWhitelist)
			ipWhitelist.POST("", AddIPWhitelist)
			ipWhitelist.DELETE("/:id", auth.RequireScope("delete"), DeleteIPWhitelist)
			ipWhitelist.GET("/check", CheckIPAccess)
		}

		// Device binding (4 endpoints)
		deviceSvc := service.NewDeviceService(db)
		SetDeviceAPI(NewDeviceAPI(deviceSvc))
		devices := protected.Group("/devices")
		{
			devices.GET("", ListDevices)
			devices.GET("/current", CurrentDevice)
			devices.DELETE("/:id", auth.RequireScope("delete"), RevokeDevice)
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

		// License management (7 endpoints)
		license := protected.Group("/license")
		{
			license.GET("/status", GetLicenseStatusHandler(bridge))
			license.POST("/activate", ActivateLicenseHandler(bridge))
			license.POST("/deactivate", DeactivateLicenseHandler(bridge))
			license.POST("/issue", auth.RoleRequired("owner"), IssueLicenseHandler(bridge))
			license.GET("/list", auth.RoleRequired("owner"), ListLicensesHandler(bridge))
			license.POST("/:id/revoke", auth.RoleRequired("owner"), RevokeLicenseHandler(bridge))
			license.POST("/addon", PurchaseAddonHandler(bridge))

			// Key rotation (owner only)
			keys := license.Group("/keys")
			{
				keys.POST("/rotate", auth.RoleRequired("owner"), RotateLicenseKeysHandler(bridge))
				keys.GET("/list", auth.RoleRequired("owner"), ListLicenseKeysHandler(bridge))
				keys.GET("/version", GetKeyVersionHandler(bridge))
				keys.POST("/backup-shamir", auth.RoleRequired("owner"), BackupKeyShamirHandler(bridge))
			}
		}

		// Feature flags management (7 endpoints)
		ff := protected.Group("/feature-flags")
		{
			ff.GET("", ListFeatureFlagsHandler(bridge))
			ff.GET("/:key", GetFeatureFlagHandler(bridge))
			ff.PUT("/:key", auth.RoleRequired("owner"), UpdateFeatureFlagHandler(bridge))
			ff.POST("/:key/override", auth.RoleRequired("owner"), SetFeatureFlagOverrideHandler(bridge))
			ff.DELETE("/:key/override", auth.RequireScope("delete"), auth.RoleRequired("owner"), DeleteFeatureFlagOverrideHandler(bridge))
			ff.GET("/:key/overrides", auth.RoleRequired("owner"), ListFeatureFlagOverridesHandler(bridge))
			ff.GET("/tenant/:tenant_id", GetFeatureFlagsForTenantHandler(bridge))
		}

		// Trial period management (3 endpoints)
		trial := protected.Group("/trial")
		{
			trial.GET("/status", GetTrialStatusHandler(bridge))
			trial.POST("/extend", auth.RoleRequired("owner"), ExtendTrialHandler(bridge))
			trial.GET("/list", auth.RoleRequired("owner"), ListTrialPeriodsHandler(bridge))
		}

		// Degradation management (3 endpoints)
		degradation := protected.Group("/degradation")
		{
			degradation.GET("/status", GetDegradationStatusHandler(bridge))
			degradation.GET("/audits", auth.RoleRequired("owner"), ListDegradationAuditsHandler(bridge))
			degradation.GET("/export-summary", auth.RoleRequired("owner"), ExportDegradationSummaryHandler(bridge))
		}

		// Batch operations (1 endpoint)
		protected.POST("/batch-deploy", auth.RequireScope("deploy"), BatchDeployHandler(bridge))

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

