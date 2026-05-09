package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleListTemplates(ctx context.Context, d TemplateService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	templates, err := d.ListTemplates(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list templates failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "templates": templates}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleGetTemplate(ctx context.Context, d TemplateService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tmplType, err := request.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tmpl, err := d.GetTemplate(ctx, tmplType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get template failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "template": tmpl}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleListEnvTemplates(ctx context.Context, d TemplateService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	templates, err := d.ListEnvTemplates(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list env templates failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "env_templates": templates}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleGetEnvTemplate(ctx context.Context, d TemplateService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceType, err := request.RequireString("service_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tmpl, err := d.GetEnvTemplate(ctx, serviceType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get env template failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "env_template": tmpl}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
