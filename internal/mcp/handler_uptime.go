package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func handleCreateUptimeMonitor(ctx context.Context, d UptimeService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	monType := request.GetString("type", "http")
	target, err := request.RequireString("target")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	interval := 60
	if i := request.GetString("interval", ""); i != "" {
		_, _ = fmt.Sscanf(i, "%d", &interval)
	}
	timeout := 10
	if t := request.GetString("timeout", ""); t != "" {
		_, _ = fmt.Sscanf(t, "%d", &timeout)
	}
	result, err := d.CreateUptimeMonitor(ctx, name, monType, target, interval, timeout)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create monitor: %v", err)), nil
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListUptimeMonitors(ctx context.Context, d UptimeService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := d.ListUptimeMonitors(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list monitors: %v", err)), nil
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleCheckUptimeMonitor(ctx context.Context, d UptimeService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	monitorID, err := request.RequireString("monitor_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, err := d.CheckUptimeMonitor(ctx, monitorID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("monitor check failed: %v", err)), nil
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetMonitorSLA(ctx context.Context, d UptimeService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	monitorID, err := request.RequireString("monitor_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	days := 30
	if dStr := request.GetString("days", ""); dStr != "" {
		_, _ = fmt.Sscanf(dStr, "%d", &days)
	}
	result, err := d.GetMonitorSLA(ctx, monitorID, days)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get SLA: %v", err)), nil
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDeleteUptimeMonitor(ctx context.Context, d UptimeService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	monitorID, err := request.RequireString("monitor_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := d.DeleteUptimeMonitor(ctx, monitorID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete monitor: %v", err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf(`{"status":"deleted","id":"%s"}`, monitorID)), nil
}

func handleCreateHeartbeat(ctx context.Context, d UptimeService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	interval := 60
	if i := request.GetString("interval", ""); i != "" {
		_, _ = fmt.Sscanf(i, "%d", &interval)
	}
	timeout := 120
	if t := request.GetString("timeout", ""); t != "" {
		_, _ = fmt.Sscanf(t, "%d", &timeout)
	}
	result, err := d.CreateHeartbeat(ctx, name, interval, timeout)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create heartbeat: %v", err)), nil
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListHeartbeats(ctx context.Context, d UptimeService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := d.ListHeartbeats(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list heartbeats: %v", err)), nil
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDeleteHeartbeat(ctx context.Context, d UptimeService, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	heartbeatID, err := request.RequireString("heartbeat_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := d.DeleteHeartbeat(ctx, heartbeatID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete heartbeat: %v", err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf(`{"status":"deleted","id":"%s"}`, heartbeatID)), nil
}
