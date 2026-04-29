package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Yogdunana/deploypilot/internal/engine/healer"
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
