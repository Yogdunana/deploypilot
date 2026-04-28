package service

import (
	"context"

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
