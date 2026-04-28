package mcp

import (
	"context"
	"fmt",
	"github.com/mark3labs/mcp-go/mcp"
)
func handleGetAppLogs(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tail := 100
	if t := request.GetString("tail", ""); t != "" {
		_, _ = fmt.Sscanf(t, "%d", &tail)
	}

	logs, err := deployer.GetContainerLogs(ctx, containerName, tail)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get logs: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"container_name": containerName,
		"tail":  tail,
		"logs":  logs,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleSearchAppLogs(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	keyword, err := request.RequireString("keyword")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := 50
	if l := request.GetString("limit", "50"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	res, err := deployer.SearchAppLogs(ctx, appID, keyword, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search logs failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "search": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
