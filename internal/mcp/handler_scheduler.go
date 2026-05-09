package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func handleCreateScheduledTask(ctx context.Context, d SchedulerService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	cronExpr, err := request.RequireString("cron_expr")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	taskType, err := request.RequireString("task_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	command, err := request.RequireString("command")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	serverID := request.GetString("server_id", "")

	result, err := d.CreateScheduledTask(ctx, name, cronExpr, taskType, command, serverID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create scheduled task: %v", err)), nil
	}

	data, _ := json.MarshalIndent(map[string]interface{}{"status": "success", "task": result}, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListScheduledTasks(ctx context.Context, d SchedulerService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := d.ListScheduledTasks(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list scheduled tasks: %v", err)), nil
	}

	data, _ := json.MarshalIndent(map[string]interface{}{"status": "success", "tasks": result}, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetTaskExecutions(ctx context.Context, d SchedulerService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := request.RequireString("task_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := 20
	if l := request.GetString("limit", ""); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	result, err := d.GetTaskExecutions(ctx, taskID, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get task executions: %v", err)), nil
	}

	data, _ := json.MarshalIndent(map[string]interface{}{"status": "success", "task_id": taskID, "executions": result}, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleToggleScheduledTask(ctx context.Context, d SchedulerService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := request.RequireString("task_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	enabledStr, err := request.RequireString("enabled")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	enabled := strings.EqualFold(strings.TrimSpace(enabledStr), "true")

	result, err := d.ToggleScheduledTask(ctx, taskID, enabled)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to toggle scheduled task: %v", err)), nil
	}

	data, _ := json.MarshalIndent(map[string]interface{}{"status": "success", "result": result}, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDeleteScheduledTask(ctx context.Context, d SchedulerService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := request.RequireString("task_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := d.DeleteScheduledTask(ctx, taskID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete scheduled task: %v", err)), nil
	}

	data, _ := json.MarshalIndent(map[string]interface{}{"status": "success", "result": result}, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
