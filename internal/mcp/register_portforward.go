package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerPortForwardTools registers port forwarding tools.
func registerPortForwardTools(s *server.MCPServer, d PortForwardService) {
	portForwardTool := mcp.NewTool("port_forward",
		mcp.WithDescription("Manage SSH port forwards to servers. Actions: create, delete, list."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action to perform: 'create', 'delete', or 'list'")),
		mcp.WithString("server_id", mcp.Description("Target server ID (required for create/delete)")),
		mcp.WithNumber("local_port", mcp.Description("Local port to bind (required for create)")),
		mcp.WithNumber("remote_port", mcp.Description("Remote port to forward to (required for create)")),
		mcp.WithString("remote_host", mcp.Description("Remote host to connect to (default: 127.0.0.1)")),
	)
	s.AddTool(portForwardTool, withPermissionCheck("port_forward", withValidation("port_forward", portForwardTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlePortForward(ctx, d, request)
	})))
}
