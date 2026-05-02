package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// MonitorUpdateCallback is called when a monitor check completes.
type MonitorUpdateCallback func(checkType string, data interface{})

// MonitorScheduler runs periodic uptime and heartbeat checks in the background.
type MonitorScheduler struct {
	svc       *MonitorService
	mu        sync.Mutex
	cancel    context.CancelFunc
	callback  MonitorUpdateCallback
	checkInterval time.Duration
}

// NewMonitorScheduler creates a new MonitorScheduler.
func NewMonitorScheduler(svc *MonitorService) *MonitorScheduler {
	return &MonitorScheduler{
		svc:           svc,
		checkInterval: 30 * time.Second,
	}
}

// SetUpdateCallback sets a callback for real-time updates (e.g., WebSocket push).
func (s *MonitorScheduler) SetUpdateCallback(cb MonitorUpdateCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callback = cb
}

// Start begins the background check loops.
func (s *MonitorScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()

	slog.Info("monitor scheduler started", "check_interval", s.checkInterval)

	go s.runMonitorChecks(ctx)
	go s.runHeartbeatChecks(ctx)
}

// Stop gracefully stops the scheduler.
func (s *MonitorScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	slog.Info("monitor scheduler stopped")
}

// runMonitorChecks periodically checks all enabled monitors.
func (s *MonitorScheduler) runMonitorChecks(ctx context.Context) {
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			results, err := s.svc.CheckAllMonitors(ctx)
			if err != nil {
				slog.Warn("monitor check cycle failed", "error", err)
				continue
			}
			s.notify("monitor_check", map[string]interface{}{
				"results": results,
				"count":   len(results),
			})
		}
	}
}

// runHeartbeatChecks periodically checks all heartbeats for timeouts.
func (s *MonitorScheduler) runHeartbeatChecks(ctx context.Context) {
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			timedOut, err := s.svc.CheckHeartbeats(ctx)
			if err != nil {
				slog.Warn("heartbeat check cycle failed", "error", err)
				continue
			}
			if len(timedOut) > 0 {
				s.notify("heartbeat_timeout", map[string]interface{}{
					"timed_out": timedOut,
					"count":     len(timedOut),
				})
			}
		}
	}
}

// notify sends an update via the callback if set.
func (s *MonitorScheduler) notify(checkType string, data interface{}) {
	s.mu.Lock()
	cb := s.callback
	s.mu.Unlock()
	if cb != nil {
		cb(checkType, data)
	}
}
