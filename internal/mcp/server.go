package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// contextKeyRole is the context key for the user role.
type contextKeyRole struct{}

// ContextWithRole returns a new context carrying the given user role.
func ContextWithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, contextKeyRole{}, role)
}

// RoleFromContext extracts the user role from context.
// Returns empty string if no role is set, which will cause permission denial.
func RoleFromContext(ctx context.Context) string {
	if role, ok := ctx.Value(contextKeyRole{}).(string); ok && role != "" {
		return role
	}
	return ""
}

// withPermissionCheck wraps a tool handler with a permission check.
// If the user's role (from context) does not meet the minimum requirement,
// a permission denied error is returned.
func withPermissionCheck(toolName string, handler server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userRole := RoleFromContext(ctx)
		if !CheckPermission(toolName, userRole) {
			required := ToolPermissions[toolName]
			slog.Warn("permission denied for tool", "tool", toolName, "role", userRole, "required", RequiredRoleName(required))
			return mcp.NewToolResultError(fmt.Sprintf("permission denied: %s requires role %s or higher, current role: %s", toolName, RequiredRoleName(required), userRole)), nil
		}
		return handler(ctx, request)
	}
}

// NewServer creates a new MCP server with deploy tools registered.
func NewServer(deployer Deployer) *server.MCPServer {
	s := server.NewMCPServer(
		"DeployPilot",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	// Register tools by domain (each receives only its sub-interface, fixes #116)
	registerDeployTools(s, deployer)
	registerServerTools(s, deployer, deployer)
	registerCredentialTools(s, deployer)
	registerDNSTools(s, deployer)
	registerBackupTools(s, deployer)
	registerMonitorTools(s, deployer)
	registerLogTools(s, deployer)
	registerNotificationTools(s, deployer)
	registerTemplateTools(s, deployer)
	registerTaskTools(s, deployer)
	registerSSLTools(s, deployer)
	registerCICDTools(s, deployer)
	registerRegistryTools(s, deployer)
	registerPluginTools(s, deployer)
	registerK8sTools(s, deployer)
	registerSystemTools(s, deployer)
	registerPortForwardTools(s, deployer)
	registerSchedulerTools(s, deployer)

	return s
}
