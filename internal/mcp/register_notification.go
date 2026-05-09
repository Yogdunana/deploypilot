package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerNotificationTools registers notification tools.
func registerNotificationTools(s *server.MCPServer, d NotificationService) {
	sendNotifyTool := mcp.NewTool("send_notification",
		mcp.WithDescription("Send a deployment notification"),
		mcp.WithString("type", mcp.Required(), mcp.Description("Notification type: deploy_success, deploy_failed, health_check, rollback")),
		mcp.WithString("app", mcp.Required(), mcp.Description("Application name")),
		mcp.WithString("server", mcp.Required(), mcp.Description("Target server")),
		mcp.WithString("status", mcp.Required(), mcp.Description("Status: success, failed, warning")),
		mcp.WithString("message", mcp.Description("Notification message")),
	)
	s.AddTool(sendNotifyTool, withPermissionCheck("send_notification", withValidation("send_notification", sendNotifyTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSendNotification(ctx, d, request)
	})))

}
