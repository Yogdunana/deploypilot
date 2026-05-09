package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerUptimeTools registers uptime monitoring and heartbeat tools.
func registerUptimeTools(s *server.MCPServer, d UptimeService) {
	createTool := mcp.NewTool("create_uptime_monitor",
		mcp.WithDescription("Create an uptime monitor to periodically check a target URL, TCP port, or IP address."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Monitor name")),
		mcp.WithString("type", mcp.Description("Check type: http, tcp, ping (default: http)")),
		mcp.WithString("target", mcp.Required(), mcp.Description("Target URL, host:port, or IP address")),
		mcp.WithString("interval", mcp.Description("Check interval in seconds (default: 60)")),
		mcp.WithString("timeout", mcp.Description("Request timeout in seconds (default: 10)")),
	)
	s.AddTool(createTool, withPermissionCheck("create_uptime_monitor", withValidation("create_uptime_monitor", createTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateUptimeMonitor(ctx, d, request)
	})))

	listTool := mcp.NewTool("list_uptime_monitors",
		mcp.WithDescription("List all configured uptime monitors with their current status and statistics."),
	)

	s.AddTool(listTool, withPermissionCheck("list_uptime_monitors", withValidation("list_uptime_monitors", listTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListUptimeMonitors(ctx, d, request)
	})))

	checkTool := mcp.NewTool("check_uptime_monitor",
		mcp.WithDescription("Trigger an immediate check on a specific uptime monitor."),
		mcp.WithString("monitor_id", mcp.Required(), mcp.Description("Monitor ID to check")),
	)

	s.AddTool(checkTool, withPermissionCheck("check_uptime_monitor", withValidation("check_uptime_monitor", checkTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCheckUptimeMonitor(ctx, d, request)
	})))

	slaTool := mcp.NewTool("get_monitor_sla",
		mcp.WithDescription("Get SLA metrics (uptime percentage, avg latency) for a monitor over a time period."),
		mcp.WithString("monitor_id", mcp.Required(), mcp.Description("Monitor ID")),
		mcp.WithString("days", mcp.Description("Number of days to analyze (default: 30)")),
	)

	s.AddTool(slaTool, withPermissionCheck("get_monitor_sla", withValidation("get_monitor_sla", slaTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetMonitorSLA(ctx, d, request)
	})))

	deleteMonTool := mcp.NewTool("delete_uptime_monitor",
		mcp.WithDescription("Delete an uptime monitor."),
		mcp.WithString("monitor_id", mcp.Required(), mcp.Description("Monitor ID to delete")),
	)

	s.AddTool(deleteMonTool, withPermissionCheck("delete_uptime_monitor", withValidation("delete_uptime_monitor", deleteMonTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteUptimeMonitor(ctx, d, request)
	})))

	createHbTool := mcp.NewTool("create_heartbeat",
		mcp.WithDescription("Create a heartbeat monitor. Returns a unique ping URL for your application to call periodically."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Heartbeat name")),
		mcp.WithString("interval", mcp.Description("Expected ping interval in seconds (default: 60)")),
		mcp.WithString("timeout", mcp.Description("Alert after this many seconds without a ping (default: 120)")),
	)

	s.AddTool(createHbTool, withPermissionCheck("create_heartbeat", withValidation("create_heartbeat", createHbTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateHeartbeat(ctx, d, request)
	})))

	listHbTool := mcp.NewTool("list_heartbeats",
		mcp.WithDescription("List all heartbeat monitors with their current status."),
	)

	s.AddTool(listHbTool, withPermissionCheck("list_heartbeats", withValidation("list_heartbeats", listHbTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListHeartbeats(ctx, d, request)
	})))

	deleteHbTool := mcp.NewTool("delete_heartbeat",
		mcp.WithDescription("Delete a heartbeat monitor."),
		mcp.WithString("heartbeat_id", mcp.Required(), mcp.Description("Heartbeat ID to delete")),
	)

	s.AddTool(deleteHbTool, withPermissionCheck("delete_heartbeat", withValidation("delete_heartbeat", deleteHbTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteHeartbeat(ctx, d, request)
	})))
}
