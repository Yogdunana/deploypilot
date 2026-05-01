package monitor

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/engine/healer"
)

// CheckResult holds the result of a monitoring cycle.
type CheckResult struct {
	ContainerName string         `json:"container_name"`
	Healthy       bool           `json:"healthy"`
	Metrics       []Metric       `json:"metrics"`
	Alerts        []*Alert       `json:"alerts"`
	Healing       *healer.HealingResult `json:"healing,omitempty"`
	CheckedAt     time.Time      `json:"checked_at"`
}

// Monitor orchestrates periodic health checks and metric collection.
type Monitor struct {
	collector    *Collector
	alertManager *AlertManager
	healer       *healer.Healer
	executor     deployer.CommandExecutor
	store        *MetricStore
	alertHandler AlertHandler // optional: called when alerts fire/resolve
	cancel       context.CancelFunc
	mu           sync.Mutex
	running      bool
}

// NewMonitor creates a new Monitor with the given executor and optional healer.
func NewMonitor(executor deployer.CommandExecutor, h *healer.Healer) *Monitor {
	return &Monitor{
		collector:    NewCollector(executor),
		alertManager: NewAlertManager(),
		healer:       h,
		executor:     executor,
	}
}

// SetStore sets the metric store for persistence.
func (m *Monitor) SetStore(store *MetricStore) {
	m.store = store
}

// SetAlertHandler sets the alert handler for processing fired/resolved alerts.
func (m *Monitor) SetAlertHandler(handler AlertHandler) {
	m.alertHandler = handler
}

// GetStore returns the current metric store (may be nil).
func (m *Monitor) GetStore() *MetricStore {
	return m.store
}

// Start begins periodic monitoring in a background goroutine.
func (m *Monitor) Start(ctx context.Context, interval time.Duration) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	childCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-childCtx.Done():
				slog.Info("monitor stopped")
				return
			case <-ticker.C:
				m.collectAndEvaluate(childCtx)
			}
		}
	}()

	slog.Info("monitor started", "interval", interval)
}

// Stop stops the monitor.
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.running = false
}

// IsRunning returns whether the monitor is currently running.
func (m *Monitor) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// CheckApp performs a full health check cycle for one app.
func (m *Monitor) CheckApp(ctx context.Context, containerName string) (*CheckResult, error) {
	result := &CheckResult{
		ContainerName: containerName,
		CheckedAt:     time.Now(),
	}

	// Collect container metrics
	metrics, err := m.collector.CollectContainerMetrics(ctx, containerName)
	if err != nil {
		slog.Error("failed to collect container metrics", "container", containerName, "error", err)
	} else {
		result.Metrics = metrics
	}

	// Collect container health
	healthMetric, err := m.collector.CollectContainerHealth(ctx, containerName)
	if err != nil {
		slog.Error("failed to collect container health", "container", containerName, "error", err)
		result.Healthy = false
	} else {
		result.Healthy = healthMetric.Value == 1
		if healthMetric.Labels != nil {
			result.Metrics = append(result.Metrics, *healthMetric)
		}
	}

	// Evaluate metrics against alert rules
	newAlerts := m.alertManager.Evaluate(result.Metrics)
	if len(newAlerts) > 0 {
		result.Alerts = newAlerts
	}

	// Run self-healing if healer is configured
	if m.healer != nil && !result.Healthy {
		healingResult, err := m.healer.CheckAndHeal(ctx, containerName)
		if err != nil {
			slog.Error("healing failed", "container", containerName, "error", err)
		} else {
			result.Healing = healingResult
		}
	}

	return result, nil
}

// GetSystemMetrics collects and returns system-level metrics.
func (m *Monitor) GetSystemMetrics(ctx context.Context) ([]Metric, error) {
	return m.collector.CollectSystemMetrics(ctx)
}

// GetContainerMetrics collects and returns container-specific metrics.
func (m *Monitor) GetContainerMetrics(ctx context.Context, containerName string) ([]Metric, error) {
	return m.collector.CollectContainerMetrics(ctx, containerName)
}

// GetAlerts returns all currently active alerts.
func (m *Monitor) GetAlerts() []*Alert {
	return m.alertManager.GetActiveAlerts()
}

// GetAlertRules returns all configured alert rules.
func (m *Monitor) GetAlertRules() []AlertRule {
	return m.alertManager.GetRules()
}

// GetAlertManager returns the underlying AlertManager for direct access.
func (m *Monitor) GetAlertManager() *AlertManager {
	return m.alertManager
}

// collectAndEvaluate performs a system-level metric collection and evaluation cycle.
func (m *Monitor) collectAndEvaluate(ctx context.Context) {
	metrics, err := m.collector.CollectSystemMetrics(ctx)
	if err != nil {
		slog.Error("system metric collection failed", "error", err)
		return
	}

	// Persist metrics if store is configured
	if m.store != nil {
		if err := m.store.SaveMetrics(ctx, metrics); err != nil {
			slog.Warn("failed to persist metrics", "error", err)
		}
	}

	newAlerts := m.alertManager.Evaluate(metrics)
	for _, alert := range newAlerts {
		if m.alertHandler != nil {
			m.alertHandler.OnAlert(alert)
		} else {
			slog.Warn("alert fired", "message", alert.Message, "severity", alert.Severity)
		}
	}
}
