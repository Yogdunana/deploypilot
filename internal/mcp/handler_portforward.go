package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

func handlePortForward(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action, err := request.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate action
	if action != "create" && action != "delete" && action != "list" {
		return mcp.NewToolResultError("invalid action: must be 'create', 'delete', or 'list'"), nil
	}

	serverID := request.GetString("server_id", "")
	localPort := 0
	if lp := request.GetString("local_port", ""); lp != "" {
		if v, err := strconv.Atoi(lp); err == nil && v > 0 {
			localPort = v
		}
	}
	remotePort := 0
	if rp := request.GetString("remote_port", ""); rp != "" {
		if v, err := strconv.Atoi(rp); err == nil && v > 0 {
			remotePort = v
		}
	}
	remoteHost := request.GetString("remote_host", "127.0.0.1")

	output, err := deployer.PortForward(ctx, action, serverID, localPort, remotePort, remoteHost)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("port forward operation failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"action": action,
		"output": output,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
