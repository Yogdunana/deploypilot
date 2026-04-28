package mcp

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleGetTaskStatus(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := request.RequireString("task_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	task, err := deployer.GetTaskStatus(ctx, taskID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get task status: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "task": task}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleListTasks(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := 20
	if l := request.GetString("limit", "20"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	statusFilter := request.GetString("status_filter", "")

	tasks, err := deployer.ListTasks(ctx, limit, statusFilter)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list tasks: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "tasks": tasks}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
