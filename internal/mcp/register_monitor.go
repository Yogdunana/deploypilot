package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerMonitorTools registers monitor tools.
func registerMonitorTools(s *server.MCPServer, d Deployer) {
	detectEnvTool := mcp.NewTool("detect_environment",
		mcp.WithDescription("Detect server environment (OS, Docker, ports, services)"),
		mcp.WithString("level",
			mcp.Description("Detection level: 1=OS, 2=+Docker, 3=+Ports, 4=+Services (default: 2)"),
		),
		mcp.WithString("ports",
			mcp.Description("Comma-separated port list to check (e.g. 8080,3000)"),
		),
		mcp.WithString("services",
			mcp.Description("Comma-separated service URLs to check (e.g. tcp://localhost:3306)"),
		),
	)

	s.AddTool(detectEnvTool, withPermissionCheck("detect_environment", withValidation("detect_environment", detectEnvTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDetectEnv(ctx, d, request)
	})))
	healthCheckTool := mcp.NewTool("health_check",
		mcp.WithDescription("Check health of a deployed service"),
		mcp.WithString("target",
			mcp.Required(),
			mcp.Description("Health check target URL (e.g. http://localhost:8080/health or tcp://localhost:3306)"),
		),
		mcp.WithString("type",
			mcp.Description("Health check type: http or tcp (default: http)"),
		),
	)

	s.AddTool(healthCheckTool, withPermissionCheck("health_check", withValidation("health_check", healthCheckTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleHealthCheck(ctx, d, request)
	})))
	getContainerMetricsTool := mcp.NewTool("get_container_metrics",
		mcp.WithDescription("Get resource usage metrics (CPU, memory) for a specific container."),
		mcp.WithString("container_name", mcp.Required(), mcp.Description("Name of the container")),
	)
	s.AddTool(getContainerMetricsTool, withPermissionCheck("get_container_metrics", withValidation("get_container_metrics", getContainerMetricsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetContainerMetrics(ctx, d, request)
	})))
	getSystemMetricsTool := mcp.NewTool("get_system_metrics",
		mcp.WithDescription("Get system-level metrics (CPU, memory, disk usage)."),
	)
	s.AddTool(getSystemMetricsTool, withPermissionCheck("get_system_metrics", withValidation("get_system_metrics", getSystemMetricsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetSystemMetrics(ctx, d, request)
	})))
	listAlertsTool := mcp.NewTool("list_alerts",
		mcp.WithDescription("List all currently active (firing) alerts."),
	)
	s.AddTool(listAlertsTool, withPermissionCheck("list_alerts", withValidation("list_alerts", listAlertsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListAlerts(ctx, d, request)
	})))
	listAlertRulesTool := mcp.NewTool("list_alert_rules",
		mcp.WithDescription("List all configured alert rules."),
	)
	s.AddTool(listAlertRulesTool, withPermissionCheck("list_alert_rules", withValidation("list_alert_rules", listAlertRulesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListAlertRules(ctx, d, request)
	})))

}
