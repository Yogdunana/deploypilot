package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleTriggerCIBuild(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	provider, err := request.RequireString("provider")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	repo, err := request.RequireString("repo")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	branch, err := request.RequireString("branch")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.TriggerCIBuild(ctx, provider, repo, branch)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to trigger CI build: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleGetCIBuildStatus(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	provider, err := request.RequireString("provider")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	runID, err := request.RequireString("run_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.GetCIBuildStatus(ctx, provider, runID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get CI build status: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
