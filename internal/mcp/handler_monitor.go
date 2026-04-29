package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleDetectEnv(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	level := 2
	if l := request.GetString("level", "2"); l != "" {
		_, _ = fmt.Sscanf(l, "%d", &level)
	}

	var ports []int
	if p := request.GetString("ports", ""); p != "" {
		for _, ps := range strings.Split(p, ",") {
			ps = strings.TrimSpace(ps)
			var port int
			if _, err := fmt.Sscanf(ps, "%d", &port); err == nil {
				ports = append(ports, port)
			}
		}
	}

	var services []string
	if s := request.GetString("services", ""); s != "" {
		services = strings.Split(s, ",")
		for i := range services {
			services[i] = strings.TrimSpace(services[i])
		}
	}

	env, err := deployer.DetectEnv(ctx, level, ports, services)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("environment detection failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":      "success",
		"environment": env,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleHealthCheck(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	target, err := request.RequireString("target")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	healthType := request.GetString("type", "http")

	health, err := deployer.HealthCheck(ctx, target, healthType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("health check failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"health": health,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleHealContainer(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.HealContainer(ctx, containerName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("heal failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleGetContainerMetrics(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.GetContainerMetrics(ctx, containerName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get container metrics: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleGetSystemMetrics(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID := request.GetString("server_id", "")

	var result interface{}
	var err error

	if serverID != "" {
		result, err = deployer.GetRemoteSystemMetrics(ctx, serverID)
	} else {
		result, err = deployer.GetSystemMetrics(ctx)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get system metrics: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleListAlerts(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := deployer.ListAlerts(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list alerts: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleListAlertRules(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := deployer.ListAlertRules(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list alert rules: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleQueryMetricHistory(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	metricType := request.GetString("metric_type", "")
	duration := request.GetString("duration", "1h")

	result, err := deployer.QueryMetricHistory(ctx, metricType, duration)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query metric history: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleQueryAlertHistory(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := request.GetString("status", "")
	limit := 50
	if l := request.GetString("limit", ""); l != "" {
		_, _ = fmt.Sscanf(l, "%d", &limit)
	}

	result, err := deployer.QueryAlertHistory(ctx, status, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query alert history: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
