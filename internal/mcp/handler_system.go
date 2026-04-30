package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleCheckSystemUpdate(ctx context.Context, d SystemService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	update, err := d.CheckSystemUpdate(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("check update failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "update": update}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleGetContext(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := "default" // In production, extract from session
	session := contextManager.GetOrCreateSession(sessionID)
	entries := session.GetEntries()
	summary := session.GetSummary()

	result := fmt.Sprintf("Session: %s\nEntries: %d\nMemory: %d bytes\nLast access: %s\n\nRecent operations:\n",
		summary["session_id"], summary["entries"], summary["memory_usage"], summary["last_access"])
	for i, e := range entries {
		result += fmt.Sprintf("%d. [%s] %s (%s)\n", i+1, e.Time.Format("15:04:05"), e.Tool, e.Duration)
	}

	return mcp.NewToolResultText(result), nil
}
