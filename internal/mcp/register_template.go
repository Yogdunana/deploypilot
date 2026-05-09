package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerTemplateTools registers template tools.
func registerTemplateTools(s *server.MCPServer, d TemplateService) {
	listTmplTool := mcp.NewTool("list_templates",
		mcp.WithDescription("List all available application templates"),
	)
	s.AddTool(listTmplTool, withPermissionCheck("list_templates", withValidation("list_templates", listTmplTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListTemplates(ctx, d, request)
	})))
	getTmplTool := mcp.NewTool("get_template",
		mcp.WithDescription("Get details of a specific application template"),
		mcp.WithString("type", mcp.Required(), mcp.Description("Template type: node, python, go, java, php, ruby, rust, static, docker")),
	)
	s.AddTool(getTmplTool, withPermissionCheck("get_template", withValidation("get_template", getTmplTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetTemplate(ctx, d, request)
	})))
	listEnvTmplTool := mcp.NewTool("list_env_templates",
		mcp.WithDescription("List all available environment variable templates for common services"),
	)
	s.AddTool(listEnvTmplTool, withPermissionCheck("list_env_templates", withValidation("list_env_templates", listEnvTmplTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListEnvTemplates(ctx, d, request)
	})))
	getEnvTmplTool := mcp.NewTool("get_env_template",
		mcp.WithDescription("Get environment variable template for a specific service type"),
		mcp.WithString("service_type", mcp.Required(), mcp.Description("Service type: mysql, redis, postgresql, mongodb, nginx")),
	)
	s.AddTool(getEnvTmplTool, withPermissionCheck("get_env_template", withValidation("get_env_template", getEnvTmplTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetEnvTemplate(ctx, d, request)
	})))

}
