package mcp

import "context"

// UptimeService stubs for mockDeployer (ai-guide #25).
// These satisfy the UptimeService sub-interface added in Phase 6.1-6.2.

func (m *mockDeployer) CreateUptimeMonitor(_ context.Context, _ string, _ string, _ string, _ int, _ int) (interface{}, error) {
	return map[string]interface{}{"id": "mon-new-001", "name": "test-monitor", "status": "up"}, nil
}

func (m *mockDeployer) ListUptimeMonitors(_ context.Context) (interface{}, error) {
	return []interface{}{
		map[string]interface{}{"id": "mon-001", "name": "Google", "type": "http", "target": "https://google.com", "status": "up", "uptime": 99.9},
		map[string]interface{}{"id": "mon-002", "name": "GitHub", "type": "http", "target": "https://github.com", "status": "up", "uptime": 99.5},
	}, nil
}

func (m *mockDeployer) CheckUptimeMonitor(_ context.Context, _ string) (interface{}, error) {
	return map[string]interface{}{"status": "up", "latency": 45.2, "status_code": 200}, nil
}

func (m *mockDeployer) GetMonitorSLA(_ context.Context, _ string, _ int) (interface{}, error) {
	return map[string]interface{}{"uptime_pct": "99.9500", "avg_latency": "45.2", "total_checks": 1000, "up_checks": 999}, nil
}

func (m *mockDeployer) DeleteUptimeMonitor(_ context.Context, _ string) error {
	return nil
}

func (m *mockDeployer) CreateHeartbeat(_ context.Context, _ string, _ int, _ int) (interface{}, error) {
	return map[string]interface{}{"id": "hb-new-001", "name": "test-heartbeat", "token": "mock-token-123", "status": "up"}, nil
}

func (m *mockDeployer) ListHeartbeats(_ context.Context) (interface{}, error) {
	return []interface{}{
		map[string]interface{}{"id": "hb-001", "name": "cron-job", "status": "up", "interval": 60},
	}, nil
}

func (m *mockDeployer) DeleteHeartbeat(_ context.Context, _ string) error {
	return nil
}
