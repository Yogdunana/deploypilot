package monitor

import (
	"fmt"
	"sync"
	"time"
)

// AlertSeverity defines alert severity levels.
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical" // P0
	SeverityWarning  AlertSeverity = "warning"  // P1
	SeverityInfo     AlertSeverity = "info"     // P2
)

// AlertRule defines a monitoring alert rule.
type AlertRule struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	MetricType MetricType    `json:"metric_type"`
	Condition  string        `json:"condition"` // gt, lt, eq, neq
	Threshold  float64       `json:"threshold"`
	Severity   AlertSeverity `json:"severity"`
	Enabled    bool          `json:"enabled"`
	Cooldown   time.Duration `json:"cooldown"`
}

// Alert represents a triggered alert.
type Alert struct {
	ID         string        `json:"id"`
	RuleID     string        `json:"rule_id"`
	RuleName   string        `json:"rule_name"`
	Severity   AlertSeverity `json:"severity"`
	Message    string        `json:"message"`
	Value      float64       `json:"value"`
	Threshold  float64       `json:"threshold"`
	Status     string        `json:"status"` // firing, resolved
	FiredAt    time.Time     `json:"fired_at"`
	ResolvedAt *time.Time    `json:"resolved_at,omitempty"`
}

// AlertManager evaluates metrics against rules and fires alerts.
type AlertManager struct {
	rules     []AlertRule
	alerts    map[string]*Alert // active alerts by rule ID
	cooldowns map[string]time.Time
	mu        sync.RWMutex
}

// NewAlertManager creates a new AlertManager with default rules.
func NewAlertManager() *AlertManager {
	am := &AlertManager{
		rules:     DefaultRules(),
		alerts:    make(map[string]*Alert),
		cooldowns: make(map[string]time.Time),
	}
	return am
}

// AddRule adds an alert rule to the manager.
func (am *AlertManager) AddRule(rule AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules = append(am.rules, rule)
}

// DefaultRules returns built-in alert rules.
func DefaultRules() []AlertRule {
	return []AlertRule{
		{
			ID:         "disk-space",
			Name:       "Disk Space Low",
			MetricType: MetricDisk,
			Condition:  "lt",
			Threshold:  5.0,
			Severity:   SeverityWarning,
			Enabled:    true,
			Cooldown:   30 * time.Minute,
		},
		{
			ID:         "memory-high",
			Name:       "Memory Usage High",
			MetricType: MetricMemory,
			Condition:  "gt",
			Threshold:  90.0,
			Severity:   SeverityWarning,
			Enabled:    true,
			Cooldown:   15 * time.Minute,
		},
		{
			ID:         "cpu-high",
			Name:       "CPU Usage High",
			MetricType: MetricCPU,
			Condition:  "gt",
			Threshold:  95.0,
			Severity:   SeverityWarning,
			Enabled:    true,
			Cooldown:   10 * time.Minute,
		},
	}
}

// Evaluate checks metrics against rules and returns new alerts.
func (am *AlertManager) Evaluate(metrics []Metric) []*Alert {
	am.mu.Lock()
	defer am.mu.Unlock()

	var newAlerts []*Alert
	now := time.Now()

	for _, metric := range metrics {
		for _, rule := range am.rules {
			if !rule.Enabled {
				continue
			}
			if rule.MetricType != metric.Type {
				continue
			}

			// Check cooldown
			if lastFire, ok := am.cooldowns[rule.ID]; ok && now.Sub(lastFire) < rule.Cooldown {
				continue
			}

			if am.evaluateCondition(metric.Value, rule.Condition, rule.Threshold) {
				// Check if already firing
				if existing, ok := am.alerts[rule.ID]; ok && existing.Status == "firing" {
					// Update value but don't create new alert
					existing.Value = metric.Value
					continue
				}

				alert := &Alert{
					ID:        fmt.Sprintf("alert-%d", now.UnixNano()),
					RuleID:    rule.ID,
					RuleName:  rule.Name,
					Severity:  rule.Severity,
					Message:   fmt.Sprintf("%s: %s = %.2f %s (threshold: %.2f)", rule.Name, metric.Name, metric.Value, metric.Unit, rule.Threshold),
					Value:     metric.Value,
					Threshold: rule.Threshold,
					Status:    "firing",
					FiredAt:   now,
				}
				am.alerts[rule.ID] = alert
				am.cooldowns[rule.ID] = now
				newAlerts = append(newAlerts, alert)
			} else {
				// Condition no longer met - resolve if firing
				if existing, ok := am.alerts[rule.ID]; ok && existing.Status == "firing" {
					resolvedAt := now
					existing.Status = "resolved"
					existing.ResolvedAt = &resolvedAt
				}
			}
		}
	}

	return newAlerts
}

// GetActiveAlerts returns all currently firing alerts.
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var active []*Alert
	for _, alert := range am.alerts {
		if alert.Status == "firing" {
			active = append(active, alert)
		}
	}
	if active == nil {
		active = []*Alert{}
	}
	return active
}

// GetRules returns all configured alert rules.
func (am *AlertManager) GetRules() []AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.rules
}

// ResolveAlert marks an alert as resolved.
func (am *AlertManager) ResolveAlert(ruleID string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	if alert, ok := am.alerts[ruleID]; ok && alert.Status == "firing" {
		now := time.Now()
		alert.Status = "resolved"
		alert.ResolvedAt = &now
	}
}

// evaluateCondition checks if a value matches a condition against a threshold.
func (am *AlertManager) evaluateCondition(value float64, condition string, threshold float64) bool {
	switch condition {
	case "gt":
		return value > threshold
	case "lt":
		return value < threshold
	case "eq":
		return value == threshold
	case "neq":
		return value != threshold
	case "gte":
		return value >= threshold
	case "lte":
		return value <= threshold
	default:
		return false
	}
}
