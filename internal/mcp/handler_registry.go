package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleRegistryLogin(ctx context.Context, d RegistryService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryID, err := request.RequireString("registry_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := map[string]interface{}{
		"registry_url": request.GetString("registry_url", ""),
		"username":     request.GetString("username", ""),
		"password":     request.GetString("password", ""),
	}

	result, err := d.RegistryOps(registryID, "login", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("registry login failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handlePushImage(ctx context.Context, d RegistryService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryID, err := request.RequireString("registry_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	localImage, err := request.RequireString("local_image")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := map[string]interface{}{
		"local_image": localImage,
		"remote_tag":  request.GetString("remote_tag", ""),
	}

	result, err := d.RegistryOps(registryID, "push", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("push image failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleListRegistryTags(ctx context.Context, d RegistryService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryID, err := request.RequireString("registry_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	repository, err := request.RequireString("repository")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := map[string]interface{}{
		"repository": repository,
	}

	result, err := d.RegistryOps(registryID, "list_tags", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list registry tags failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handlePingRegistry(ctx context.Context, d RegistryService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryID, err := request.RequireString("registry_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := d.RegistryOps(registryID, "ping", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ping registry failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
