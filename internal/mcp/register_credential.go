package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerCredentialTools registers credential tools.
func registerCredentialTools(s *server.MCPServer, d Deployer) {
	createCredTool := mcp.NewTool("add_credential",
		mcp.WithDescription("Create an encrypted credential. The value is encrypted with AES-256-GCM before storage — plaintext never touches the database."),
		mcp.WithString("tenant_id", mcp.Required(), mcp.Description("Tenant ID")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Credential name")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Credential type: ssh, api_key, token, password")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Plain credential value (will be encrypted)")),
	)
	s.AddTool(createCredTool, withPermissionCheck("add_credential", withValidation("add_credential", createCredTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateCredential(ctx, deployer, request)
	})))
	listCredsTool := mcp.NewTool("list_credentials",
		mcp.WithDescription("List all credentials for a tenant"),
		mcp.WithString("tenant_id", mcp.Required(), mcp.Description("Tenant ID")),
	)
	s.AddTool(listCredsTool, withPermissionCheck("list_credentials", withValidation("list_credentials", listCredsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListCredentials(ctx, deployer, request)
	})))
	deleteCredTool := mcp.NewTool("delete_credential",
		mcp.WithDescription("Delete a credential"),
		mcp.WithString("credential_id", mcp.Required(), mcp.Description("Credential ID to delete")),
	)
	s.AddTool(deleteCredTool, withPermissionCheck("delete_credential", withValidation("delete_credential", deleteCredTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteCredential(ctx, deployer, request)
	})))
	updateCredTool := mcp.NewTool("update_credential",
		mcp.WithDescription("Update a credential value"),
		mcp.WithString("credential_id", mcp.Required(), mcp.Description("Credential ID")),
		mcp.WithString("value", mcp.Required(), mcp.Description("New credential value")),
	)
	s.AddTool(updateCredTool, withPermissionCheck("update_credential", withValidation("update_credential", updateCredTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateCredential(ctx, deployer, request)
	})))

}
