package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// mockMonitorExecutor implements deployer.CommandExecutor for monitor tests.
type mockMonitorExecutor struct {
	responses map[string]string
	errs      map[string]error
}

func (m *mockMonitorExecutor) RunCommand(_ context.Context, cmd string) (string, error) {
	if err, ok := m.errs[cmd]; ok {
		return "", err
	}
	for pattern, output := range m.responses {
		for i := 0; i <= len(cmd)-len(pattern); i++ {
			if cmd[i:i+len(pattern)] == pattern {
				return output, nil
			}
		}
	}
	return "", nil
}

func TestNewMonitor(t *testing.T) {
	exec := &mockMonitorExecutor{}
	m := NewMonitor(exec, nil)
	if m == nil {
		t.Fatal("expected non-nil monitor")
	}
	if m.collector == nil {
		t.Error("expected non-nil collector")
	}
	if m.alertManager == nil {
		t.Error("expected non-nil alertManager")
	}
}

func TestMonitor_StartStop(t *testing.T) {
	exec := &mockMonitorExecutor{
		responses: map[string]string{
			"cat /proc/stat":  "cpu  100 10 50 840 0 0 0 0 0 0",
			"free -m":         "Mem:   8000  4000  3000  100  1000  3500",
			"df -h /":         "/dev/sda1  100G  45G   50G  48% /",
		},
	}
	m := NewMonitor(exec, nil)

	if m.IsRunning() {
		t.Error("expected monitor to not be running initially")
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.Start(ctx, 100*time.Millisecond)

	// Give it a moment to start
	time.Sleep(150 * time.Millisecond)

	if !m.IsRunning() {
		t.Error("expected monitor to be running after Start")
	}

	m.Stop()

	if m.IsRunning() {
		t.Error("expected monitor to not be running after Stop")
	}

	cancel()
}

func TestMonitor_CheckApp(t *testing.T) {
	exec := &mockMonitorExecutor{
		responses: map[string]string{
			"docker stats":          "12.34%|50MiB / 1GiB|5.00%|1.2kB / 0B|0B / 0B",
			"State.Health.Status":   "healthy",
		},
	}
	m := NewMonitor(exec, nil)

	result, err := m.CheckApp(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ContainerName != "my-app" {
		t.Errorf("expected container name 'my-app', got %q", result.ContainerName)
	}
	if !result.Healthy {
		t.Error("expected healthy=true for healthy container")
	}
	if len(result.Metrics) == 0 {
		t.Error("expected metrics to be collected")
	}
}

func TestMonitor_CheckApp_Unhealthy(t *testing.T) {
	exec := &mockMonitorExecutor{
		responses: map[string]string{
			"docker stats":          "12.34%|50MiB / 1GiB|5.00%|1.2kB / 0B|0B / 0B",
			"State.Health.Status":   "unhealthy",
		},
	}
	m := NewMonitor(exec, nil)

	result, err := m.CheckApp(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Healthy {
		t.Error("expected healthy=false for unhealthy container")
	}
}

func TestMonitor_GetSystemMetrics(t *testing.T) {
	exec := &mockMonitorExecutor{
		responses: map[string]string{
			"cat /proc/stat":  "cpu  100 10 50 840 0 0 0 0 0 0",
			"free -m":         "Mem:   8000  4000  3000  100  1000  3500",
			"df -h /":         "/dev/sda1  100G  45G   50G  48% /",
		},
	}
	m := NewMonitor(exec, nil)

	metrics, err := m.GetSystemMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) == 0 {
		t.Error("expected system metrics")
	}
}

func TestMonitor_GetContainerMetrics(t *testing.T) {
	exec := &mockMonitorExecutor{
		responses: map[string]string{
			"docker stats": "12.34%|50MiB / 1GiB|5.00%|1.2kB / 0B|0B / 0B",
		},
	}
	m := NewMonitor(exec, nil)

	metrics, err := m.GetContainerMetrics(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(metrics) == 0 {
		t.Error("expected container metrics")
	}
}

func TestMonitor_GetAlerts(t *testing.T) {
	exec := &mockMonitorExecutor{}
	m := NewMonitor(exec, nil)

	alerts := m.GetAlerts()
	if alerts == nil {
		t.Error("expected non-nil alerts")
	}
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alerts))
	}
}

func TestMonitor_GetAlertRules(t *testing.T) {
	exec := &mockMonitorExecutor{}
	m := NewMonitor(exec, nil)

	rules := m.GetAlertRules()
	if len(rules) != 3 {
		t.Errorf("expected 3 alert rules, got %d", len(rules))
	}
}

func TestMonitor_GetAlertManager(t *testing.T) {
	exec := &mockMonitorExecutor{}
	m := NewMonitor(exec, nil)

	am := m.GetAlertManager()
	if am == nil {
		t.Error("expected non-nil alert manager")
	}
}

func TestMonitor_Start_Idempotent(t *testing.T) {
	exec := &mockMonitorExecutor{
		responses: map[string]string{
			"cat /proc/stat":  "cpu  100 10 50 840 0 0 0 0 0 0",
			"free -m":         "Mem:   8000  4000  3000  100  1000  3500",
			"df -h /":         "/dev/sda1  100G  45G   50G  48% /",
		},
	}
	m := NewMonitor(exec, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start twice should be idempotent
	m.Start(ctx, 100*time.Millisecond)
	m.Start(ctx, 100*time.Millisecond)

	if !m.IsRunning() {
		t.Error("expected monitor to be running")
	}

	m.Stop()
}

// Ensure mockMonitorExecutor implements CommandExecutor interface.
var _ deployer.CommandExecutor = (*mockMonitorExecutor)(nil)
