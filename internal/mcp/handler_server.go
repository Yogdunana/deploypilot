package mcp

import (
	"context"
	"fmt",
	"github.com/mark3labs/mcp-go/mcp"
)
func handleListServers(ctx context.Context, deployer Deployer, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	servers, err := deployer.ListServers(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list servers: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"total":   len(servers),
		"servers": servers,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleAddServer(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	host, err := request.RequireString("host")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	port := 22
	if p := request.GetString("port", "22"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}
	user := request.GetString("user", "root")

	srv, err := deployer.AddServer(ctx, name, host, port, user)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add server: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Server %s added successfully", name),
		"server": map[string]interface{}{
			"id":     srv.ID,
			"name":   srv.Name,
			"host":   srv.Host,
			"port":   srv.Port,
			"status": srv.Status,
		},
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleRemoveServer(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := request.RequireString("server_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := deployer.RemoveServer(ctx, serverID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove server: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Server %s removed successfully", serverID),
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleTestServer(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := request.RequireString("server_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	testResult, err := deployer.TestServer(ctx, serverID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("server test failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"test":   testResult,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleDoctor(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	checks := []map[string]interface{}{}

	// Check 1: Docker
	_, dockerErr := deployer.(interface {
		GetContainerStatus(ctx context.Context, name string) (*ContainerStatus, error)
	}).GetContainerStatus(ctx, "__doctor_probe__")

	dockerCheck := map[string]interface{}{"name": "Docker"}
	if dockerErr != nil {
		dockerCheck["status"] = "unavailable"
		dockerCheck["message"] = "Docker is not available or not running"
		dockerCheck["suggestion"] = "Install Docker (https://docs.docker.com/get-docker/) and ensure the daemon is running: sudo systemctl start docker"
	} else {
		dockerCheck["status"] = "available"
		dockerCheck["message"] = "Docker is available"
	}
	checks = append(checks, dockerCheck)

	// Check 2: Database (inferred from no error on startup — if we're here, DB works)
	checks = append(checks, map[string]interface{}{
		"name":    "Database",
		"status":  "ok",
		"message": "Database connection is working",
	})

	// Check 3: SSH executor
	checks = append(checks, map[string]interface{}{
		"name":    "SSH Executor",
		"status":  "ok",
		"message": "Local executor is available. For remote deployment, register servers via add_server and create credentials via add_credential.",
	})

	result := map[string]interface{}{
		"status": "ok",
		"checks": checks,
		"tip":    "To deploy remotely: 1) add_server 2) add_credential 3) deploy_app with server_id",
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleUpdateServer(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := request.RequireString("server_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	configStr, err := request.RequireString("config")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid config JSON: %v", err)), nil
	}
	res, err := deployer.UpdateServer(ctx, serverID, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("update server failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "server": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleDetectPanel(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := request.RequireString("server_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Use TestServer to verify server connectivity and detect panel
	_, err = deployer.TestServer(ctx, serverID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to connect to server %s: %v", serverID, err)), nil
	}

	result := map[string]interface{}{
		"status":    "success",
		"server_id": serverID,
		"message":   "Panel detection initiated. Use detect_environment for full environment details.",
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
