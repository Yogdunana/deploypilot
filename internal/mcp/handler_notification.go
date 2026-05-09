package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleSendNotification(ctx context.Context, d NotificationService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nType, err := request.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	appName, _ := request.RequireString("app")
	server, _ := request.RequireString("server")
	status, _ := request.RequireString("status")
	message := request.GetString("message", "")

	result, err := d.SendNotification(ctx, nType, appName, server, status, message)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("notification failed: %v", err)), nil
	}

	resp := map[string]interface{}{"status": "success", "notification": result}
	data, _ := json.MarshalIndent(resp, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
