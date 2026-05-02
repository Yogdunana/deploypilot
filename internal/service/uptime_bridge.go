package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// getUptimeService lazily initializes and returns the MonitorService.
func (b *Bridge) getUptimeService() *MonitorService {
	if b.uptimeSvc == nil && b.DB != nil {
		b.uptimeSvc = NewMonitorService(b.DB)
	}
	return b.uptimeSvc
}

// CreateUptimeMonitor creates a new uptime monitor.
func (b *Bridge) CreateUptimeMonitor(ctx context.Context, name, monType, target string, interval, timeout int) (interface{}, error) {
	svc := b.getUptimeService()
	if svc == nil {
		return nil, fmt.Errorf("database not available")
	}
	mon := &Monitor{
		ID:       uuid.New().String(),
		Name:     name,
		Type:     monType,
		Target:   target,
		Interval: interval,
		Timeout:  timeout,
		Enabled:  true,
	}
	if err := svc.CreateMonitor(ctx, mon); err != nil {
		return nil, err
	}
	return mon, nil
}

// ListUptimeMonitors lists all uptime monitors.
func (b *Bridge) ListUptimeMonitors(ctx context.Context) (interface{}, error) {
	svc := b.getUptimeService()
	if svc == nil {
		return nil, fmt.Errorf("database not available")
	}
	return svc.ListMonitors(ctx, "")
}

// CheckUptimeMonitor triggers an immediate check.
func (b *Bridge) CheckUptimeMonitor(ctx context.Context, monitorID string) (interface{}, error) {
	svc := b.getUptimeService()
	if svc == nil {
		return nil, fmt.Errorf("database not available")
	}
	return svc.CheckMonitor(ctx, monitorID)
}

// GetMonitorSLA gets SLA metrics.
func (b *Bridge) GetMonitorSLA(ctx context.Context, monitorID string, days int) (interface{}, error) {
	svc := b.getUptimeService()
	if svc == nil {
		return nil, fmt.Errorf("database not available")
	}
	return svc.GetMonitorSLA(ctx, monitorID, days)
}

// DeleteUptimeMonitor deletes an uptime monitor.
func (b *Bridge) DeleteUptimeMonitor(ctx context.Context, monitorID string) error {
	svc := b.getUptimeService()
	if svc == nil {
		return fmt.Errorf("database not available")
	}
	return svc.DeleteMonitor(ctx, monitorID)
}

// CreateHeartbeat creates a heartbeat monitor.
func (b *Bridge) CreateHeartbeat(ctx context.Context, name string, interval, timeout int) (interface{}, error) {
	svc := b.getUptimeService()
	if svc == nil {
		return nil, fmt.Errorf("database not available")
	}
	hb := &Heartbeat{
		ID:       uuid.New().String(),
		Name:     name,
		Interval: interval,
		Timeout:  timeout,
		Enabled:  true,
	}
	if err := svc.CreateHeartbeat(ctx, hb); err != nil {
		return nil, err
	}
	return hb, nil
}

// ListHeartbeats lists all heartbeats.
func (b *Bridge) ListHeartbeats(ctx context.Context) (interface{}, error) {
	svc := b.getUptimeService()
	if svc == nil {
		return nil, fmt.Errorf("database not available")
	}
	return svc.ListHeartbeats(ctx, "")
}

// DeleteHeartbeat deletes a heartbeat.
func (b *Bridge) DeleteHeartbeat(ctx context.Context, heartbeatID string) error {
	svc := b.getUptimeService()
	if svc == nil {
		return fmt.Errorf("database not available")
	}
	return svc.DeleteHeartbeat(ctx, heartbeatID)
}
