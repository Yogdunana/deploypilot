package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerPluginTools registers plugin tools.
func registerPluginTools(s *server.MCPServer, d Deployer) {
	listPluginsTool := mcp.NewTool("list_plugins",
		mcp.WithDescription("List all registered plugins"),
		mcp.WithString("provider", mcp.Description("Filter by provider type (dns, notify, registry, cicd, server, ssl)")),
	)
	s.AddTool(listPluginsTool, withPermissionCheck("list_plugins", withValidation("list_plugins", listPluginsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListPlugins(ctx, d, request)
	})))
	managePluginTool := mcp.NewTool("manage_plugin",
		mcp.WithDescription("Enable, disable, or reload a plugin"),
		mcp.WithString("plugin_id", mcp.Required(), mcp.Description("Plugin ID")),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action to perform: enable, disable, reload")),
	)
	s.AddTool(managePluginTool, withPermissionCheck("manage_plugin", withValidation("manage_plugin", managePluginTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleManagePlugin(ctx, d, request)
	})))
	getPluginInfoTool := mcp.NewTool("get_plugin_info",
		mcp.WithDescription("Get detailed information about a plugin"),
		mcp.WithString("plugin_id", mcp.Required(), mcp.Description("Plugin ID")),
	)
	s.AddTool(getPluginInfoTool, withPermissionCheck("get_plugin_info", withValidation("get_plugin_info", getPluginInfoTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetPluginInfo(ctx, d, request)
	})))

}
