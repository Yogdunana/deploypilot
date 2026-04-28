package mcp

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleListTemplates(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	templates, err := deployer.ListTemplates(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list templates failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "templates": templates}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleGetTemplate(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tmplType, err := request.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tmpl, err := deployer.GetTemplate(ctx, tmplType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get template failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "template": tmpl}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
