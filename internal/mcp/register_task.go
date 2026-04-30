package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerTaskTools registers task tools.
func registerTaskTools(s *server.MCPServer, d TaskManager) {
	getTaskStatusTool := mcp.NewTool("get_task_status",
		mcp.WithDescription("Get status of an async task"),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID")),
	)
	s.AddTool(getTaskStatusTool, withPermissionCheck("get_task_status", withValidation("get_task_status", getTaskStatusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetTaskStatus(ctx, d, request)
	})))
	listTasksTool := mcp.NewTool("list_tasks",
		mcp.WithDescription("List recent tasks"),
		mcp.WithString("limit", mcp.Description("Maximum number of tasks to return (default: 20)")),
		mcp.WithString("status_filter", mcp.Description("Filter by status: running, completed, failed")),
	)
	s.AddTool(listTasksTool, withPermissionCheck("list_tasks", withValidation("list_tasks", listTasksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListTasks(ctx, d, request)
	})))

}
