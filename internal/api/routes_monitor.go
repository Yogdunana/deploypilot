package api

import (
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// registerMonitorRoutes registers all monitoring-related routes.
func registerMonitorRoutes(protected *gin.RouterGroup, db *gorm.DB, bridge *service.Bridge) {
	// Monitors
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
	// These are registered on the engine directly in router.go
	// r.GET("/api/v1/heartbeat/ping/:token", globalMonitorAPI.PingHeartbeat)
	// r.GET("/api/v1/status", globalMonitorAPI.GetStatusPage)

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
}
