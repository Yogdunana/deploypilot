package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerLogTools registers log tools.
func registerLogTools(s *server.MCPServer, d Deployer) {
	getLogsTool := mcp.NewTool("get_app_logs",
		mcp.WithDescription("Get logs from a deployed container"),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name of the container"),
		),
		mcp.WithString("tail",
			mcp.Description("Number of lines to retrieve (default: 100)"),
		),
	)

	s.AddTool(getLogsTool, withPermissionCheck("get_app_logs", withValidation("get_app_logs", getLogsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAppLogs(ctx, deployer, request)
	})))
	searchLogsTool := mcp.NewTool("search_app_logs",
		mcp.WithDescription("Search container logs by keyword"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("Application ID")),
		mcp.WithString("keyword", mcp.Required(), mcp.Description("Search keyword")),
		mcp.WithString("limit", mcp.Description("Max results (default: 50)")),
	)
	s.AddTool(searchLogsTool, withPermissionCheck("search_app_logs", withValidation("search_app_logs", searchLogsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSearchAppLogs(ctx, deployer, request)
	})))

}
