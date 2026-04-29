package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerSchedulerTools registers scheduled task tools.
func registerSchedulerTools(s *server.MCPServer, d Deployer) {
	createScheduledTaskTool := mcp.NewTool("create_scheduled_task",
		mcp.WithDescription("Create a new cron-based scheduled task. Supports shell commands, health checks, and log cleanup."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Task name"),
		),
		mcp.WithString("cron_expr",
			mcp.Required(),
			mcp.Description("Cron expression with seconds precision (e.g., '0 */6 * * *' for every 6 hours, '0 0 2 * * *' for daily at 2 AM)"),
		),
		mcp.WithString("task_type",
			mcp.Required(),
			mcp.Description("Task type: shell, health_check, or log_cleanup"),
		),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("Shell command to execute (for shell type), container name (for health_check), or ignored (for log_cleanup)"),
		),
		mcp.WithString("server_id",
			mcp.Description("Server ID to execute the command on (omit for local execution)"),
		),
	)
	s.AddTool(createScheduledTaskTool, withPermissionCheck("create_scheduled_task", withValidation("create_scheduled_task", createScheduledTaskTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateScheduledTask(ctx, d, request)
	})))

	listScheduledTasksTool := mcp.NewTool("list_scheduled_tasks",
		mcp.WithDescription("List all scheduled tasks with their status and configuration."),
	)
	s.AddTool(listScheduledTasksTool, withPermissionCheck("list_scheduled_tasks", withValidation("list_scheduled_tasks", listScheduledTasksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListScheduledTasks(ctx, d, request)
	})))

	getTaskExecutionsTool := mcp.NewTool("get_task_executions",
		mcp.WithDescription("Get execution history for a scheduled task."),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("Task ID to get execution history for"),
		),
		mcp.WithString("limit",
			mcp.Description("Maximum number of execution records to return (default: 20)"),
		),
	)
	s.AddTool(getTaskExecutionsTool, withPermissionCheck("get_task_executions", withValidation("get_task_executions", getTaskExecutionsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetTaskExecutions(ctx, d, request)
	})))

	toggleScheduledTaskTool := mcp.NewTool("toggle_scheduled_task",
		mcp.WithDescription("Enable or disable a scheduled task."),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("Task ID to toggle"),
		),
		mcp.WithString("enabled",
			mcp.Required(),
			mcp.Description("Set to 'true' to enable, 'false' to disable"),
		),
	)
	s.AddTool(toggleScheduledTaskTool, withPermissionCheck("toggle_scheduled_task", withValidation("toggle_scheduled_task", toggleScheduledTaskTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleToggleScheduledTask(ctx, d, request)
	})))

	deleteScheduledTaskTool := mcp.NewTool("delete_scheduled_task",
		mcp.WithDescription("Delete a scheduled task permanently."),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("Task ID to delete"),
		),
	)
	s.AddTool(deleteScheduledTaskTool, withPermissionCheck("delete_scheduled_task", withValidation("delete_scheduled_task", deleteScheduledTaskTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteScheduledTask(ctx, d, request)
	})))
}
