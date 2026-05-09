package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerSystemTools registers system tools.
func registerSystemTools(s *server.MCPServer, d SystemService) {
	checkSysUpdateTool := mcp.NewTool("check_system_update",
		mcp.WithDescription("Check if a newer version of DeployPilot is available"),
	)
	s.AddTool(checkSysUpdateTool, withPermissionCheck("check_system_update", withValidation("check_system_update", checkSysUpdateTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCheckSystemUpdate(ctx, d, request)
	})))
	getContextTool := mcp.NewTool("get_context",
		mcp.WithDescription("Get current MCP session context and operation history"),
	)
	s.AddTool(getContextTool, withPermissionCheck("get_context", withValidation("get_context", getContextTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetContext(ctx, request)
	})))

}
