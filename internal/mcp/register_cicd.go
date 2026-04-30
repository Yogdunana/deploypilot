package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerCICDTools registers cicd tools.
func registerCICDTools(s *server.MCPServer, d CICDService) {
	triggerCITool := mcp.NewTool("trigger_ci_build",
		mcp.WithDescription("Trigger a CI/CD build for a repository"),
		mcp.WithString("provider", mcp.Required(), mcp.Description("CI/CD provider type: github-actions")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name (e.g. my-project)")),
		mcp.WithString("branch", mcp.Required(), mcp.Description("Git branch to build (e.g. main)")),
	)
	s.AddTool(triggerCITool, withPermissionCheck("trigger_ci_build", withValidation("trigger_ci_build", triggerCITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleTriggerCIBuild(ctx, d, request)
	})))
	getCIStatusTool := mcp.NewTool("get_ci_build_status",
		mcp.WithDescription("Get the status of a CI/CD build"),
		mcp.WithString("provider", mcp.Required(), mcp.Description("CI/CD provider type: github-actions")),
		mcp.WithString("run_id", mcp.Required(), mcp.Description("Build run ID")),
	)
	s.AddTool(getCIStatusTool, withPermissionCheck("get_ci_build_status", withValidation("get_ci_build_status", getCIStatusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetCIBuildStatus(ctx, d, request)
	})))

}
