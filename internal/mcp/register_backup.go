package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerBackupTools registers backup tools.
func registerBackupTools(s *server.MCPServer, d Deployer) {
	backupTool := mcp.NewTool("backup_database",
		mcp.WithDescription("Create a backup of an application"),
		mcp.WithString("app_id",
			mcp.Required(),
			mcp.Description("ID of the application to backup"),
		),
	)

	s.AddTool(backupTool, withPermissionCheck("backup_database", withValidation("backup_database", backupTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBackup(ctx, deployer, request)
	})))
	restoreTool := mcp.NewTool("restore_database",
		mcp.WithDescription("Restore an application from a backup"),
		mcp.WithString("backup_id",
			mcp.Required(),
			mcp.Description("ID of the backup to restore from"),
		),
	)

	s.AddTool(restoreTool, withPermissionCheck("restore_database", withValidation("restore_database", restoreTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRestore(ctx, deployer, request)
	})))
	batchBackupTool := mcp.NewTool("batch_backup",
		mcp.WithDescription("Backup multiple applications at once"),
		mcp.WithString("app_ids", mcp.Required(), mcp.Description("JSON array of app IDs: [\"id1\", \"id2\"]")),
	)
	s.AddTool(batchBackupTool, withPermissionCheck("batch_backup", withValidation("batch_backup", batchBackupTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBatchBackup(ctx, deployer, request)
	})))

}
