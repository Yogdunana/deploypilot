package monitor

import (
	"context"
	"log"
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
				log.Println("[monitor] stopped")
				return
			case <-ticker.C:
				m.collectAndEvaluate(childCtx)
			}
		}
	}()

	log.Printf("[monitor] started with interval %v", interval)
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
		log.Printf("[monitor] failed to collect container metrics for %s: %v", containerName, err)
	} else {
		result.Metrics = metrics
	}

	// Collect container health
	healthMetric, err := m.collector.CollectContainerHealth(ctx, containerName)
	if err != nil {
		log.Printf("[monitor] failed to collect health for %s: %v", containerName, err)
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
			log.Printf("[monitor] healing failed for %s: %v", containerName, err)
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
		log.Printf("[monitor] system metric collection failed: %v", err)
		return
	}

	newAlerts := m.alertManager.Evaluate(metrics)
	for _, alert := range newAlerts {
		log.Printf("[monitor] alert fired: %s (severity: %s)", alert.Message, alert.Severity)
	}
}
