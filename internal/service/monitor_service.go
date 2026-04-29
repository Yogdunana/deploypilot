package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/healer"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/monitor"
)

// ---------- 41. GetContainerMetrics ----------

func (b *Bridge) GetContainerMetrics(ctx context.Context, containerName string) (interface{}, error) {
	m := b.getMonitor()
	return m.GetContainerMetrics(ctx, containerName)
}

// ---------- 42. GetSystemMetrics ----------

func (b *Bridge) GetSystemMetrics(ctx context.Context) (interface{}, error) {
	m := b.getMonitor()
	return m.GetSystemMetrics(ctx)
}

// ---------- 42b. GetRemoteSystemMetrics ----------

func (b *Bridge) GetRemoteSystemMetrics(ctx context.Context, serverID string) (interface{}, error) {
	remoteExec, err := b.getRemoteExecutor(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote executor for server %s: %w", serverID, err)
	}
	defer func() {
		if cerr := remoteExec.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	collector := monitor.NewCollector(remoteExec)
	return collector.CollectSystemMetrics(ctx)
}

// ---------- 43. ListAlerts ----------

func (b *Bridge) ListAlerts(ctx context.Context) (interface{}, error) {
	m := b.getMonitor()
	return m.GetAlerts(), nil
}

// ---------- 44. ListAlertRules ----------

func (b *Bridge) ListAlertRules(ctx context.Context) (interface{}, error) {
	m := b.getMonitor()
	return m.GetAlertRules(), nil
}

// getMonitor lazily initializes and returns the monitor.
func (b *Bridge) getMonitor() *monitor.Monitor {
	if b.Monitor == nil {
		b.Monitor = monitor.NewMonitor(b.Executor, b.getHealer())
		if b.DB != nil {
			b.Monitor.SetStore(monitor.NewMetricStore(b.DB))
		}
	}
	return b.Monitor
}

// getHealer lazily initializes and returns the healer.
func (b *Bridge) getHealer() *healer.Healer {
	if b.healer == nil {
		b.healer = healer.NewHealer(b.Executor, healer.DefaultHealingConfig())
	}
	return b.healer
}

// parseDuration parses a human-readable duration string (e.g., "1h", "24h", "7d", "30d").
func parseDuration(d string) time.Duration {
	switch d {
	case "1h":
		return 1 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	default:
		return 1 * time.Hour
	}
}

// ---------- 45. QueryMetricHistory ----------

func (b *Bridge) QueryMetricHistory(ctx context.Context, metricType string, duration string) (interface{}, error) {
	m := b.getMonitor()
	store := m.GetStore()
	if store == nil {
		return nil, fmt.Errorf("metric store not available (database not configured)")
	}

	dur := parseDuration(duration)
	opts := monitor.QueryOptions{
		StartTime:  time.Now().Add(-dur),
		EndTime:    time.Now(),
		MetricType: metricType,
		Limit:      1000,
	}

	return store.QueryMetrics(ctx, opts)
}

// ---------- 46. QueryAlertHistory ----------

func (b *Bridge) QueryAlertHistory(ctx context.Context, status string, limit int) (interface{}, error) {
	m := b.getMonitor()
	store := m.GetStore()
	if store == nil {
		return nil, fmt.Errorf("metric store not available (database not configured)")
	}

	if limit <= 0 {
		limit = 50
	}

	opts := monitor.AlertQueryOptions{
		Status: status,
		Limit:  limit,
	}

	return store.QueryAlerts(ctx, opts)
}

// ---------- Phase 3.10: Scheduled Task System ----------

// CreateScheduledTask creates a new scheduled task and registers it with the scheduler.
func (b *Bridge) CreateScheduledTask(ctx context.Context, name, cronExpr, taskType, command string, serverID string) (interface{}, error) {
	if b.Scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}

	task := model.ScheduledTask{
		ID:       fmt.Sprintf("stask-%d", time.Now().UnixNano()),
		Name:     name,
		CronExpr: cronExpr,
		TaskType: taskType,
		Command:  command,
		ServerID: serverID,
		Enabled:  true,
		Timeout:  300,
	}

	if err := b.Scheduler.AddTask(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// ListScheduledTasks returns all scheduled tasks.
func (b *Bridge) ListScheduledTasks(_ context.Context) (interface{}, error) {
	if b.Scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	return b.Scheduler.ListTasks()
}

// GetTaskExecutions returns execution history for a task.
func (b *Bridge) GetTaskExecutions(_ context.Context, taskID string, limit int) (interface{}, error) {
	if b.Scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	return b.Scheduler.GetTaskExecutions(taskID, limit)
}

// ToggleScheduledTask enables or disables a scheduled task.
func (b *Bridge) ToggleScheduledTask(ctx context.Context, taskID string, enabled bool) (interface{}, error) {
	if b.Scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	if err := b.Scheduler.ToggleTask(ctx, taskID, enabled); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"task_id": taskID,
		"enabled": enabled,
	}, nil
}

// DeleteScheduledTask removes a scheduled task.
func (b *Bridge) DeleteScheduledTask(ctx context.Context, taskID string) (interface{}, error) {
	if b.Scheduler == nil {
		return nil, fmt.Errorf("scheduler not initialized")
	}
	if err := b.Scheduler.RemoveTask(taskID); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"task_id": taskID,
		"deleted": true,
	}, nil
}
