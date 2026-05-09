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
	sessionID := "default"
	session := contextManager.GetOrCreateSession(sessionID)
	entries := session.GetEntries()
	summary := session.GetSummary()

	type opEntry struct {
		Tool     string `json:"tool"`
		Success  bool   `json:"success"`
		Error    string `json:"error,omitempty"`
		Duration string `json:"duration,omitempty"`
		Time     string `json:"time"`
	}

	ops := make([]opEntry, 0, len(entries))
	for _, e := range entries {
		ops = append(ops, opEntry{
			Tool:     e.Tool,
			Success:  e.Success,
			Error:    e.Error,
			Duration: e.Duration,
			Time:     e.Time.Format("15:04:05"),
		})
	}

	result := map[string]interface{}{
		"session_id":   summary["session_id"],
		"total_ops":    len(entries),
		"memory_usage": summary["memory_usage"],
		"last_access":  fmt.Sprintf("%v", summary["last_access"]),
		"operations":   ops,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
