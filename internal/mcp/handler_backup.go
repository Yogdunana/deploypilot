package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleBackup(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	backupID, err := deployer.Backup(ctx, appID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("backup failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Backup created for app %s", appID),
		"backup": map[string]string{
			"id": backupID,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleRestore(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	backupID, err := request.RequireString("backup_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status, err := deployer.Restore(ctx, backupID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("restore failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Application restored from backup %s", backupID),
		"container": map[string]string{
			"id":     status.ID,
			"name":   status.Name,
			"image":  status.Image,
			"status": status.Status,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleBatchBackup(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appIDsStr, err := request.RequireString("app_ids")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var appIDs []string
	if err := json.Unmarshal([]byte(appIDsStr), &appIDs); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid app_ids JSON: %v", err)), nil
	}
	res, err := deployer.BatchBackup(ctx, appIDs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("batch backup failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "batch": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
