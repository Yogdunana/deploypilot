package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerRegistryTools registers registry tools.
func registerRegistryTools(s *server.MCPServer, d RegistryService) {
	registryLoginTool := mcp.NewTool("registry_login",
		mcp.WithDescription("Authenticate with a container registry"),
		mcp.WithString("registry_id", mcp.Required(), mcp.Description("Registry ID to authenticate with")),
	)
	s.AddTool(registryLoginTool, withPermissionCheck("registry_login", withValidation("registry_login", registryLoginTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRegistryLogin(ctx, d, request)
	})))
	pushImageTool := mcp.NewTool("push_image",
		mcp.WithDescription("Push a Docker image to a container registry"),
		mcp.WithString("registry_id", mcp.Required(), mcp.Description("Registry ID to push to")),
		mcp.WithString("local_image", mcp.Required(), mcp.Description("Local image name (e.g. myapp:latest)")),
		mcp.WithString("remote_tag", mcp.Description("Remote tag for the image (e.g. registry.example.com/myapp:v1)")),
	)
	s.AddTool(pushImageTool, withPermissionCheck("push_image", withValidation("push_image", pushImageTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlePushImage(ctx, d, request)
	})))
	listRegistryTagsTool := mcp.NewTool("list_registry_tags",
		mcp.WithDescription("List tags for a repository in a container registry"),
		mcp.WithString("registry_id", mcp.Required(), mcp.Description("Registry ID")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository name (e.g. myuser/myapp)")),
	)
	s.AddTool(listRegistryTagsTool, withPermissionCheck("list_registry_tags", withValidation("list_registry_tags", listRegistryTagsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListRegistryTags(ctx, d, request)
	})))
	pingRegistryTool := mcp.NewTool("ping_registry",
		mcp.WithDescription("Check if a container registry is accessible"),
		mcp.WithString("registry_id", mcp.Required(), mcp.Description("Registry ID to ping")),
	)
	s.AddTool(pingRegistryTool, withPermissionCheck("ping_registry", withValidation("ping_registry", pingRegistryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlePingRegistry(ctx, d, request)
	})))

}
