package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleCreateCredential(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tenantID, err := request.RequireString("tenant_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	credType, err := request.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	value, err := request.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cred, err := deployer.CreateCredential(ctx, tenantID, name, credType, value)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create credential: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":     "success",
		"message":    fmt.Sprintf("Credential %s created successfully", name),
		"credential": cred,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleListCredentials(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tenantID, err := request.RequireString("tenant_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	creds, err := deployer.ListCredentials(ctx, tenantID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list credentials: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":      "success",
		"credentials": creds,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleDeleteCredential(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	credID, err := request.RequireString("credential_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := deployer.DeleteCredential(ctx, credID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete credential: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Credential %s deleted successfully", credID),
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleUpdateCredential(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	credID, err := request.RequireString("credential_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	value, err := request.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	res, err := deployer.UpdateCredential(ctx, credID, value)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("update credential failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "credential": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
