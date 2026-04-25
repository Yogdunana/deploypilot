package healer

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// mockExecutor implements deployer.CommandExecutor for testing.
type mockExecutor struct {
	responses map[string]string // pattern substring -> output
	errs      map[string]error  // exact cmd -> error
}

func (m *mockExecutor) RunCommand(_ context.Context, cmd string) (string, error) {
	if err, ok := m.errs[cmd]; ok {
		return "", err
	}
	// Match by substring for flexible matching
	for pattern, output := range m.responses {
		if strings.Contains(cmd, pattern) {
			return output, nil
		}
	}
	return "", fmt.Errorf("no matching command: %s", cmd)
}

// healthyInspect returns inspect output for a healthy container.
func healthyInspect(name string) map[string]string {
	return map[string]string{
		"State.OOMKilled":     "false|0|false|1234|2024-01-01T00:00:00Z|0001-01-01T00:00:00Z|healthy",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": fmt.Sprintf("abc123|/%s|nginx:latest|running|2024-01-01T00:00:00Z", name),
		"docker restart":      "",
	}
}

// oomKilledInspect returns inspect output for an OOMKilled container.
func oomKilledInspect(name string) map[string]string {
	return map[string]string{
		"State.OOMKilled":     "true|137|false|0|2024-01-01T00:00:00Z|2024-01-01T00:01:00Z|none",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": fmt.Sprintf("abc123|/%s|nginx:latest|exited|2024-01-01T00:00:00Z", name),
		"docker restart":      "",
	}
}

// exitedInspect returns inspect output for a container that exited with non-zero code.
func exitedInspect(name string) map[string]string {
	return map[string]string{
		"State.OOMKilled":     "false|1|false|0|2024-01-01T00:00:00Z|2024-01-01T00:01:00Z|none",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": fmt.Sprintf("abc123|/%s|nginx:latest|exited|2024-01-01T00:00:00Z", name),
		"docker restart":      "",
	}
}

// restartingInspect returns inspect output for a container in restarting state.
func restartingInspect(name string) map[string]string {
	return map[string]string{
		"State.OOMKilled":     "false|0|true|0|2024-01-01T00:00:00Z|0001-01-01T00:00:00Z|none",
		"RestartCount":        "5",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": fmt.Sprintf("abc123|/%s|nginx:latest|restarting|2024-01-01T00:00:00Z", name),
		"docker restart":      "",
	}
}

func TestCheckAndHeal_HealthyContainer(t *testing.T) {
	exec := &mockExecutor{responses: healthyInspect("healthy-app")}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "healthy-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "none" {
		t.Errorf("expected action 'none', got %q", result.Action)
	}
	if result.NewState != "running" {
		t.Errorf("expected new state 'running', got %q", result.NewState)
	}
}

func TestCheckAndHeal_OOMKilled_Restarts(t *testing.T) {
	exec := &mockExecutor{responses: oomKilledInspect("oom-app")}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "oom-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "restarted" {
		t.Errorf("expected action 'restarted', got %q", result.Action)
	}
	if result.Reason != "container was OOMKilled" {
		t.Errorf("expected reason about OOMKilled, got %q", result.Reason)
	}
}

func TestCheckAndHeal_CrashLoop_RollsBack(t *testing.T) {
	exec := &mockExecutor{responses: restartingInspect("crash-app")}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "crash-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "rolled_back" {
		t.Errorf("expected action 'rolled_back', got %q", result.Action)
	}
}

func TestCheckAndHeal_Exited_Restarts(t *testing.T) {
	exec := &mockExecutor{responses: exitedInspect("exited-app")}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "exited-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "restarted" {
		t.Errorf("expected action 'restarted', got %q", result.Action)
	}
}

func TestNeedsRollback(t *testing.T) {
	h := NewHealer(&mockExecutor{}, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	if !h.needsRollback("test", 3) {
		t.Error("expected needsRollback to be true at max restarts")
	}
	if !h.needsRollback("test", 5) {
		t.Error("expected needsRollback to be true above max restarts")
	}
	if h.needsRollback("test", 2) {
		t.Error("expected needsRollback to be false below max restarts")
	}
}

func TestNeedsRollback_AutoRollbackDisabled(t *testing.T) {
	h := NewHealer(&mockExecutor{}, HealingConfig{
		AutoRestart:  true,
		AutoRollback: false,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	if h.needsRollback("test", 10) {
		t.Error("expected needsRollback to be false when auto_rollback is disabled")
	}
}

func TestRestartContainer(t *testing.T) {
	exec := &mockExecutor{
		responses: map[string]string{"docker restart": ""},
	}
	h := NewHealer(exec, DefaultHealingConfig())

	err := h.RestartContainer(context.Background(), "test-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestartContainer_Failure(t *testing.T) {
	exec := &mockExecutor{
		errs: map[string]error{"docker restart": fmt.Errorf("docker error")},
	}
	h := NewHealer(exec, DefaultHealingConfig())

	err := h.RestartContainer(context.Background(), "test-app")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCheckAndHeal_AutoRestartDisabled(t *testing.T) {
	exec := &mockExecutor{responses: oomKilledInspect("oom-app")}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  false,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "oom-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "notified" {
		t.Errorf("expected action 'notified' when auto_restart disabled, got %q", result.Action)
	}
}

func TestDefaultHealingConfig(t *testing.T) {
	cfg := DefaultHealingConfig()
	if !cfg.AutoRestart {
		t.Error("expected AutoRestart to be true")
	}
	if !cfg.AutoRollback {
		t.Error("expected AutoRollback to be true")
	}
	if cfg.MaxRestarts != 3 {
		t.Errorf("expected MaxRestarts 3, got %d", cfg.MaxRestarts)
	}
	if cfg.RestartWindow != 5*time.Minute {
		t.Errorf("expected RestartWindow 5m, got %v", cfg.RestartWindow)
	}
	if cfg.HealthCheckInterval != 30*time.Second {
		t.Errorf("expected HealthCheckInterval 30s, got %v", cfg.HealthCheckInterval)
	}
}

func TestRestartTracker(t *testing.T) {
	rt := newRestartTracker()

	// Record restarts
	rt.RecordRestart("app1")
	rt.RecordRestart("app1")
	rt.RecordRestart("app2")

	// Count within window
	count := rt.CountRecentRestarts("app1", 5*time.Minute)
	if count != 2 {
		t.Errorf("expected 2 restarts for app1, got %d", count)
	}

	count = rt.CountRecentRestarts("app2", 5*time.Minute)
	if count != 1 {
		t.Errorf("expected 1 restart for app2, got %d", count)
	}

	// Non-existent container
	count = rt.CountRecentRestarts("app3", 5*time.Minute)
	if count != 0 {
		t.Errorf("expected 0 restarts for app3, got %d", count)
	}
}

func TestRestartTracker_Reset(t *testing.T) {
	rt := newRestartTracker()
	rt.RecordRestart("app1")
	rt.RecordRestart("app1")

	rt.Reset("app1")
	count := rt.CountRecentRestarts("app1", 5*time.Minute)
	if count != 0 {
		t.Errorf("expected 0 restarts after reset, got %d", count)
	}
}

func TestRestartTracker_Expired(t *testing.T) {
	rt := newRestartTracker()
	// Record a restart in the past by manipulating the tracker directly
	rt.mu.Lock()
	rt.attempts["app1"] = []time.Time{time.Now().Add(-10 * time.Minute)}
	rt.mu.Unlock()

	count := rt.CountRecentRestarts("app1", 5*time.Minute)
	if count != 0 {
		t.Errorf("expected 0 restarts after expiry, got %d", count)
	}
}

func TestGetConfig(t *testing.T) {
	cfg := HealingConfig{
		AutoRestart:         true,
		AutoRollback:        true,
		MaxRestarts:         5,
		RestartWindow:       10 * time.Minute,
		HealthCheckInterval: 60 * time.Second,
	}
	h := NewHealer(&mockExecutor{}, cfg)
	got := h.GetConfig()
	if got.MaxRestarts != 5 {
		t.Errorf("expected MaxRestarts 5, got %d", got.MaxRestarts)
	}
	if got.RestartWindow != 10*time.Minute {
		t.Errorf("expected RestartWindow 10m, got %v", got.RestartWindow)
	}
}

func TestCheckAndHeal_Unhealthy_Restarts(t *testing.T) {
	responses := map[string]string{
		"State.OOMKilled":     "false|0|false|1234|2024-01-01T00:00:00Z|0001-01-01T00:00:00Z|unhealthy",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/unhealthy-app|nginx:latest|running|2024-01-01T00:00:00Z",
		"docker restart":      "",
	}
	exec := &mockExecutor{responses: responses}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "unhealthy-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "restarted" {
		t.Errorf("expected action 'restarted' for unhealthy container, got %q", result.Action)
	}
}

func TestCheckAndHeal_Unhealthy_MaxRestartsExceeded(t *testing.T) {
	responses := map[string]string{
		"State.OOMKilled":     "false|0|false|1234|2024-01-01T00:00:00Z|0001-01-01T00:00:00Z|unhealthy",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/unhealthy-app|nginx:latest|running|2024-01-01T00:00:00Z",
		"docker restart":      "",
	}
	exec := &mockExecutor{responses: responses}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	// Record 3 restarts to exceed the limit
	h.tracker.RecordRestart("unhealthy-app")
	h.tracker.RecordRestart("unhealthy-app")
	h.tracker.RecordRestart("unhealthy-app")

	result, err := h.CheckAndHeal(context.Background(), "unhealthy-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "rolled_back" {
		t.Errorf("expected action 'rolled_back' when max restarts exceeded, got %q", result.Action)
	}
}

func TestCheckAndHeal_ContainerNotFound(t *testing.T) {
	exec := &mockExecutor{responses: map[string]string{}}
	h := NewHealer(exec, DefaultHealingConfig())

	_, err := h.CheckAndHeal(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent container")
	}
}

func TestNewHealer_Defaults(t *testing.T) {
	h := NewHealer(&mockExecutor{}, HealingConfig{})
	if h.config.MaxRestarts != 3 {
		t.Errorf("expected default MaxRestarts 3, got %d", h.config.MaxRestarts)
	}
	if h.config.RestartWindow != 5*time.Minute {
		t.Errorf("expected default RestartWindow 5m, got %v", h.config.RestartWindow)
	}
	if h.config.HealthCheckInterval != 30*time.Second {
		t.Errorf("expected default HealthCheckInterval 30s, got %v", h.config.HealthCheckInterval)
	}
}

func TestCheckAndHeal_OOMKilled_MaxRestartsExceeded(t *testing.T) {
	responses := map[string]string{
		"State.OOMKilled":     "true|137|false|0|2024-01-01T00:00:00Z|2024-01-01T00:01:00Z|none",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/oom-app|nginx:latest|exited|2024-01-01T00:00:00Z",
		"docker restart":      "",
	}
	exec := &mockExecutor{responses: responses}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	// Record 3 restarts to exceed the limit
	h.tracker.RecordRestart("oom-app")
	h.tracker.RecordRestart("oom-app")
	h.tracker.RecordRestart("oom-app")

	result, err := h.CheckAndHeal(context.Background(), "oom-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "rolled_back" {
		t.Errorf("expected action 'rolled_back' when max restarts exceeded, got %q", result.Action)
	}
}

func TestCheckAndHeal_RestartFailure(t *testing.T) {
	responses := map[string]string{
		"State.OOMKilled":     "true|137|false|0|2024-01-01T00:00:00Z|2024-01-01T00:01:00Z|none",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/oom-app|nginx:latest|exited|2024-01-01T00:00:00Z",
	}
	exec := &mockExecutor{
		responses: responses,
		errs:      map[string]error{"docker restart oom-app": fmt.Errorf("docker daemon error")},
	}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "oom-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "notified" {
		t.Errorf("expected action 'notified' when restart fails, got %q", result.Action)
	}
}

func TestCheckAndHeal_Exited_MaxRestartsExceeded(t *testing.T) {
	responses := map[string]string{
		"State.OOMKilled":     "false|1|false|0|2024-01-01T00:00:00Z|2024-01-01T00:01:00Z|none",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/exited-app|nginx:latest|exited|2024-01-01T00:00:00Z",
		"docker restart":      "",
	}
	exec := &mockExecutor{responses: responses}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	// Record 3 restarts
	h.tracker.RecordRestart("exited-app")
	h.tracker.RecordRestart("exited-app")
	h.tracker.RecordRestart("exited-app")

	result, err := h.CheckAndHeal(context.Background(), "exited-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "rolled_back" {
		t.Errorf("expected action 'rolled_back', got %q", result.Action)
	}
}

func TestCheckAndHeal_RunningNone(t *testing.T) {
	// Container running with no health check (health=none) should be fine
	responses := map[string]string{
		"State.OOMKilled":     "false|0|false|1234|2024-01-01T00:00:00Z|0001-01-01T00:00:00Z|none",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/my-app|nginx:latest|running|2024-01-01T00:00:00Z",
	}
	exec := &mockExecutor{responses: responses}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "none" {
		t.Errorf("expected action 'none' for running container with no health check, got %q", result.Action)
	}
}

// Ensure mockExecutor implements CommandExecutor interface.
var _ deployer.CommandExecutor = (*mockExecutor)(nil)

// ===================== Additional Coverage =====================

func TestCheckAndHeal_Unhealthy_AutoRestartDisabled(t *testing.T) {
	responses := map[string]string{
		"State.OOMKilled":     "false|0|false|1234|2024-01-01T00:00:00Z|0001-01-01T00:00:00Z|unhealthy",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/unhealthy-no-auto|nginx:latest|running|2024-01-01T00:00:00Z",
		"docker restart":      "",
	}
	exec := &mockExecutor{responses: responses}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  false,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "unhealthy-no-auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "notified" {
		t.Errorf("expected action 'notified' when auto_restart disabled for unhealthy, got %q", result.Action)
	}
}

func TestCheckAndHeal_OOMKilled_AutoRestartDisabled(t *testing.T) {
	exec := &mockExecutor{responses: oomKilledInspect("oom-no-auto")}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  false,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "oom-no-auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "notified" {
		t.Errorf("expected action 'notified' when auto_restart disabled for OOMKilled, got %q", result.Action)
	}
}

func TestCheckAndHeal_Exited_AutoRestartDisabled(t *testing.T) {
	exec := &mockExecutor{responses: exitedInspect("exited-no-auto")}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  false,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "exited-no-auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "notified" {
		t.Errorf("expected action 'notified' when auto_restart disabled for exited, got %q", result.Action)
	}
}

func TestCheckAndHeal_Restarting_BelowMax(t *testing.T) {
	responses := map[string]string{
		"State.OOMKilled":     "false|0|true|0|2024-01-01T00:00:00Z|0001-01-01T00:00:00Z|none",
		"RestartCount":        "1",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/restart-below|nginx:latest|restarting|2024-01-01T00:00:00Z",
		"docker restart":      "",
	}
	exec := &mockExecutor{responses: responses}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  5,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "restart-below")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 1 (docker RestartCount) + 0 (tracker) = 1 < 5 max, so should NOT rollback
	if result.Action == "rolled_back" {
		t.Errorf("expected action other than rolled_back when below max restarts, got %q", result.Action)
	}
}

func TestCheckAndHeal_DefaultNoAction(t *testing.T) {
	// Container in some other state (e.g., "created") that doesn't match any condition
	responses := map[string]string{
		"State.OOMKilled":     "false|0|false|0|2024-01-01T00:00:00Z|0001-01-01T00:00:00Z|none",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/created-app|nginx:latest|created|2024-01-01T00:00:00Z",
		"docker restart":      "",
	}
	exec := &mockExecutor{responses: responses}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "created-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "none" {
		t.Errorf("expected action 'none' for created container, got %q", result.Action)
	}
}

func TestCheckAndHeal_OOMKilled_RestartFails(t *testing.T) {
	responses := map[string]string{
		"State.OOMKilled":     "true|137|false|0|2024-01-01T00:00:00Z|2024-01-01T00:01:00Z|none",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/oom-restart-fail|nginx:latest|exited|2024-01-01T00:00:00Z",
	}
	exec := &mockExecutor{
		responses: responses,
		errs:      map[string]error{"docker restart oom-restart-fail": fmt.Errorf("daemon error")},
	}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "oom-restart-fail")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "notified" {
		t.Errorf("expected action 'notified' when OOM restart fails, got %q", result.Action)
	}
}

func TestCheckAndHeal_Exited_RestartFails(t *testing.T) {
	responses := map[string]string{
		"State.OOMKilled":     "false|1|false|0|2024-01-01T00:00:00Z|2024-01-01T00:01:00Z|none",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/exited-restart-fail|nginx:latest|exited|2024-01-01T00:00:00Z",
	}
	exec := &mockExecutor{
		responses: responses,
		errs:      map[string]error{"docker restart exited-restart-fail": fmt.Errorf("daemon error")},
	}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "exited-restart-fail")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "notified" {
		t.Errorf("expected action 'notified' when exited restart fails, got %q", result.Action)
	}
}

func TestCheckAndHeal_Unhealthy_RestartFails(t *testing.T) {
	responses := map[string]string{
		"State.OOMKilled":     "false|0|false|1234|2024-01-01T00:00:00Z|0001-01-01T00:00:00Z|unhealthy",
		"RestartCount":        "0",
		".Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}": "abc123|/unhealthy-restart-fail|nginx:latest|running|2024-01-01T00:00:00Z",
	}
	exec := &mockExecutor{
		responses: responses,
		errs:      map[string]error{"docker restart unhealthy-restart-fail": fmt.Errorf("daemon error")},
	}
	h := NewHealer(exec, HealingConfig{
		AutoRestart:  true,
		AutoRollback: true,
		MaxRestarts:  3,
		RestartWindow: 5 * time.Minute,
	})

	result, err := h.CheckAndHeal(context.Background(), "unhealthy-restart-fail")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "notified" {
		t.Errorf("expected action 'notified' when unhealthy restart fails, got %q", result.Action)
	}
}
