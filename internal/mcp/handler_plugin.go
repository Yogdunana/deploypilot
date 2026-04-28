package mcp

import (
	"context"
	"fmt",
	"github.com/mark3labs/mcp-go/mcp"
)
func handleListPlugins(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	provider := request.GetString("provider", "")

	result, err := deployer.ListPlugins(provider)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list plugins: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleManagePlugin(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pluginID, err := request.RequireString("plugin_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	action, err := request.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.PluginOps(pluginID, action)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("plugin %s failed: %v", action, err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleGetPluginInfo(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pluginID, err := request.RequireString("plugin_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.GetPluginInfo(pluginID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get plugin info: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
