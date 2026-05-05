package service

import (
	"context"
	"strings"
)

// ========== MonitorService interface (stubs) ==========

func (b *Bridge) GetSystemMetrics(ctx context.Context) (interface{}, error) {
	if b.Monitor != nil {
		return b.Monitor.GetSystemMetrics(ctx)
	}
	return map[string]interface{}{
		"cpu": "0%", "memory": "0MB", "disk": "0MB", "load": "0.0 0.0 0.0",
	}, nil
}

func (b *Bridge) GetContainerMetrics(ctx context.Context, name string) (interface{}, error) {
	if b.Monitor != nil {
		return b.Monitor.GetContainerMetrics(ctx, name)
	}
	return map[string]interface{}{
		"name": name, "cpu": "0%", "memory": "0MB",
	}, nil
}

func (b *Bridge) ListAlerts(ctx context.Context) (interface{}, error) {
	if b.Monitor != nil {
		return b.Monitor.ListAlerts(ctx)
	}
	return []interface{}{}, nil
}

func (b *Bridge) ListAlertRules(ctx context.Context) (interface{}, error) {
	if b.Monitor != nil {
		return b.Monitor.ListAlertRules(ctx)
	}
	return []interface{}{}, nil
}

func (b *Bridge) GetRemoteSystemMetrics(ctx context.Context, serverID string) (interface{}, error) {
	exec, err := b.getRemoteExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	result := make(map[string]interface{})
	if output, err := exec.RunCommand(ctx, "free -m 2>/dev/null"); err == nil {
		result["memory"] = strings.TrimSpace(output)
	}
	if output, err := exec.RunCommand(ctx, "df -h --total 2>/dev/null | tail -1"); err == nil {
		result["disk"] = strings.TrimSpace(output)
	}
	if output, err := exec.RunCommand(ctx, "cat /proc/loadavg 2>/dev/null"); err == nil {
		result["load_average"] = strings.TrimSpace(output)
	}
	if output, err := exec.RunCommand(ctx, "uptime -p 2>/dev/null || uptime 2>/dev/null"); err == nil {
		result["uptime"] = strings.TrimSpace(output)
	}
	return result, nil
}

func (b *Bridge) QueryMetricHistory(ctx context.Context, metricType string, duration string) (interface{}, error) {
	return b.Monitor.QueryMetricHistory(ctx, metricType, duration)
}

func (b *Bridge) QueryAlertHistory(ctx context.Context, status string, limit int) (interface{}, error) {
	return b.Monitor.QueryAlertHistory(ctx, status, limit)
}
