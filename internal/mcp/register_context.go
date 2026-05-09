package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerContextTools registers session context and operation history tools.
func registerContextTools(s *server.MCPServer) {
	listOpsTool := mcp.NewTool("list_recent_operations",
		mcp.WithDescription("List recent MCP operations in the current session. Returns operation history with tool name, arguments, result status, and timing. Useful for understanding what actions have been taken and providing context for follow-up commands like rollback."),
		mcp.WithString("tool_filter", mcp.Description("Optional filter: only show operations for this tool name (e.g., 'deploy_app', 'rollback_app'). If empty, shows all operations.")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of operations to return (default: 10, max: 50).")),
	)
	s.AddTool(listOpsTool, withPermissionCheck("list_recent_operations", withValidation("list_recent_operations", listOpsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListRecentOperations(request)
	})))

	clearContextTool := mcp.NewTool("clear_context",
		mcp.WithDescription("Clear the current MCP session's operation history. Useful for starting fresh in a long conversation."),
	)
	s.AddTool(clearContextTool, withPermissionCheck("clear_context", withValidation("clear_context", clearContextTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleClearContext(request)
	})))
}

// handleListRecentOperations returns recent operations from the session context.
func handleListRecentOperations(request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := "default"
	session := contextManager.GetOrCreateSession(sessionID)
	entries := session.GetEntries()

	// Apply optional tool filter
	toolFilter := request.GetString("tool_filter", "")
	if toolFilter != "" {
		filtered := make([]ContextEntry, 0)
		for _, e := range entries {
			if e.Tool == toolFilter {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	// Apply limit (default 10, max 50)
	limit := 10
	if limitVal := request.GetString("limit", ""); limitVal != "" {
		if n, err := fmt.Sscanf(limitVal, "%d", &limit); err == nil && n == 1 {
			if limit > 50 {
				limit = 50
			}
			if limit < 1 {
				limit = 1
			}
		}
	}

	// Return most recent first (reverse order)
	start := len(entries) - limit
	if start < 0 {
		start = 0
	}
	recent := make([]ContextEntry, 0, len(entries)-start)
	for i := len(entries) - 1; i >= start; i-- {
		recent = append(recent, entries[i])
	}

	// Build response
	type operationEntry struct {
		Index    int       `json:"index"`
		Tool     string    `json:"tool"`
		Args     string    `json:"args,omitempty"`
		Success  bool      `json:"success"`
		Error    string    `json:"error,omitempty"`
		Duration string    `json:"duration,omitempty"`
		Time     time.Time `json:"time"`
	}

	ops := make([]operationEntry, 0, len(recent))
	for i, e := range recent {
		ops = append(ops, operationEntry{
			Index:    len(recent) - i,
			Tool:     e.Tool,
			Args:     e.Args,
			Success:  e.Success,
			Error:    e.Error,
			Duration: e.Duration,
			Time:     e.Time,
		})
	}

	result := map[string]interface{}{
		"total":    len(entries),
		"returned": len(ops),
		"filter":   toolFilter,
		"operations": ops,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// handleClearContext clears the current session's operation history.
func handleClearContext(_ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := "default"
	session := contextManager.GetOrCreateSession(sessionID)
	session.Clear()
	return mcp.NewToolResultText("Session context cleared. Operation history has been reset."), nil
}
