package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerServerTools registers server tools.
func registerServerTools(s *server.MCPServer, d Deployer) {
	listServersTool := mcp.NewTool("list_servers",
		mcp.WithDescription("List all registered servers"),
	)

	s.AddTool(listServersTool, withPermissionCheck("list_servers", withValidation("list_servers", listServersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListServers(ctx, d, request)
	})))
	addServerTool := mcp.NewTool("add_server",
		mcp.WithDescription("Register a new server for remote deployment. Connectivity is tested automatically."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Server name (e.g. production, staging)")),
		mcp.WithString("host", mcp.Required(), mcp.Description("Server hostname or IP address")),
		mcp.WithString("port", mcp.Description("SSH port. Default: 22. Cloud providers often use custom ports (e.g. 23196, 2222). Check your security group settings.")),
		mcp.WithString("user", mcp.Description("SSH username (default: root)")),
	)
	s.AddTool(addServerTool, withPermissionCheck("add_server", withValidation("add_server", addServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleAddServer(ctx, d, request)
	})))
	removeServerTool := mcp.NewTool("delete_server",
		mcp.WithDescription("Remove a registered server"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("Server ID to remove")),
	)
	s.AddTool(removeServerTool, withPermissionCheck("delete_server", withValidation("delete_server", removeServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRemoveServer(ctx, d, request)
	})))
	testServerTool := mcp.NewTool("test_server",
		mcp.WithDescription("Test SSH connectivity to a registered server. Returns latency and actionable suggestions if unreachable."),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("Server ID to test")),
	)
	s.AddTool(testServerTool, withPermissionCheck("test_server", withValidation("test_server", testServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleTestServer(ctx, d, request)
	})))
	updateSrvTool := mcp.NewTool("update_server",
		mcp.WithDescription("Update server configuration"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("Server ID")),
		mcp.WithString("config", mcp.Required(), mcp.Description("JSON string of config to update")),
	)
	s.AddTool(updateSrvTool, withPermissionCheck("update_server", withValidation("update_server", updateSrvTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateServer(ctx, d, request)
	})))
	detectPanelTool := mcp.NewTool("detect_panel",
		mcp.WithDescription("Detect which hosting panel (1Panel/BT-Panel) is installed on a server"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("Server ID to detect panel on")),
	)
	s.AddTool(detectPanelTool, withPermissionCheck("detect_panel", withValidation("detect_panel", detectPanelTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDetectPanel(ctx, d, request)
	})))
	doctorTool := mcp.NewTool("doctor",
		mcp.WithDescription("Check DeployPilot prerequisites: Docker availability, database connectivity, and SSH configuration."),
	)
	s.AddTool(doctorTool, withPermissionCheck("doctor", withValidation("doctor", doctorTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDoctor(ctx, d, request)
	})))
	execCommandTool := mcp.NewTool("exec_command",
		mcp.WithDescription("Execute a command on a server. Runs locally if server_id is omitted, or remotely via SSH if server_id is provided."),
		mcp.WithString("command", mcp.Required(), mcp.Description("Command to execute")),
		mcp.WithString("server_id", mcp.Description("Target server ID (omit for local execution)")),
		mcp.WithNumber("timeout", mcp.Description("Timeout in seconds (default: 30)")),
	)
	s.AddTool(execCommandTool, withPermissionCheck("exec_command", withValidation("exec_command", execCommandTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleExecCommand(ctx, d, request)
	})))

}
