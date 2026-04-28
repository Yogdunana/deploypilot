package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerTemplateTools registers template tools.
func registerTemplateTools(s *server.MCPServer, d Deployer) {
	listTmplTool := mcp.NewTool("list_templates",
		mcp.WithDescription("List all available application templates"),
	)
	s.AddTool(listTmplTool, withPermissionCheck("list_templates", withValidation("list_templates", listTmplTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListTemplates(ctx, deployer, request)
	})))
	getTmplTool := mcp.NewTool("get_template",
		mcp.WithDescription("Get details of a specific application template"),
		mcp.WithString("type", mcp.Required(), mcp.Description("Template type: node, python, go, java, php, ruby, rust, static, docker")),
	)
	s.AddTool(getTmplTool, withPermissionCheck("get_template", withValidation("get_template", getTmplTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetTemplate(ctx, deployer, request)
	})))

}
