package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

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

// withPermissionCheck wraps a tool handler with a permission check and
// automatic operation history recording. Each tool invocation is recorded
// in the session context for later retrieval via list_recent_operations.
func withPermissionCheck(toolName string, handler server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userRole := RoleFromContext(ctx)
		if !CheckPermission(toolName, userRole) {
			required := ToolPermissions[toolName]
			slog.Warn("permission denied for tool", "tool", toolName, "role", userRole, "required", RequiredRoleName(required))
			return mcp.NewToolResultError(fmt.Sprintf("permission denied: %s requires role %s or higher, current role: %s", toolName, RequiredRoleName(required), userRole)), nil
		}

		// Record operation start time
		start := time.Now()

		// Extract arguments as JSON string for history
		args := ""
		if params := request.Params; params != nil && len(params.Arguments) > 0 {
			if argBytes, err := json.Marshal(params.Arguments); err == nil {
				args = string(argBytes)
			}
		}

		// Execute the actual handler
		result, err := handler(ctx, request)

		// Compute duration
		duration := time.Since(start)

		// Record the operation in session context
		recordOperation(toolName, args, result, err, duration)

		return result, err
	}
}

// recordOperation adds an operation entry to the session context.
func recordOperation(toolName, args string, result *mcp.CallToolResult, err error, duration time.Duration) {
	// Skip recording for context management tools to avoid noise
	switch toolName {
	case "list_recent_operations", "clear_context", "get_context":
		return
	}

	sessionID := "default"
	session := contextManager.GetOrCreateSession(sessionID)

	entry := ContextEntry{
		Tool:     toolName,
		Args:     args,
		Duration: duration.Round(time.Millisecond).String(),
		Time:     time.Now(),
		Success:  err == nil,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	if result != nil && result.IsError {
		entry.Success = false
		// Extract error text from result
		if content := result.Content; len(content) > 0 {
			if textContent, ok := content[0].(mcp.TextContent); ok {
				entry.Error = textContent.Text
			}
		}
	}

	// Truncate result to avoid excessive memory usage
	if result != nil && len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			resultText := textContent.Text
			if len(resultText) > 500 {
				resultText = resultText[:500] + "..."
			}
			entry.Result = resultText
		}
	}

	session.AddEntry(entry)
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
	registerContextTools(s)

	return s
}
