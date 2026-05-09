package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// DeployTotal counts total deployments by app, server, and status (success/failed).
	DeployTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "deploypilot_deploy_total",
			Help: "Total number of deployments",
		},
		[]string{"app", "server", "status"},
	)

	// DeployDuration tracks deployment duration in seconds.
	DeployDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "deploypilot_deploy_duration_seconds",
			Help:    "Duration of deployments in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// ActiveContainers tracks the number of currently active containers.
	ActiveContainers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "deploypilot_active_containers",
			Help: "Number of currently active containers",
		},
	)

	// WSConnections tracks the number of active WebSocket connections.
	WSConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "deploypilot_ws_connections",
			Help: "Number of active WebSocket connections",
		},
	)

	// APILatency tracks API request latency in seconds.
	APILatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "deploypilot_api_request_duration_seconds",
			Help:    "API request latency in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	// CredentialExpiryDays tracks days until credential expiry.
	CredentialExpiryDays = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "deploypilot_credential_expiry_days",
			Help: "Days until credential expiry",
		},
		[]string{"name"},
	)

	// MonitorUp indicates whether a monitor target is up (1) or down (0).
	MonitorUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "deploypilot_monitor_up",
			Help: "Monitor target up status (1=up, 0=down)",
		},
		[]string{"name", "type", "target"},
	)

	// MonitorLatencyMs tracks the average latency of a monitor target in milliseconds.
	MonitorLatencyMs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "deploypilot_monitor_latency_ms",
			Help: "Monitor target average latency in milliseconds",
		},
		[]string{"name", "type", "target"},
	)

	// MonitorUptimePct tracks the uptime percentage of a monitor target.
	MonitorUptimePct = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "deploypilot_monitor_uptime_pct",
			Help: "Monitor target uptime percentage",
		},
		[]string{"name", "type", "target"},
	)

	// HeartbeatUp indicates whether a heartbeat target is up (1) or down (0).
	HeartbeatUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "deploypilot_heartbeat_up",
			Help: "Heartbeat up status (1=up, 0=down)",
		},
		[]string{"name"},
	)
)

// MonitorMetric is a local struct representing monitor data for gauge updates.
// It avoids circular imports with the service package.
type MonitorMetric struct {
	Name       string
	Type       string
	Target     string
	Status     string
	AvgLatency float64
	Uptime     float64
}

// HeartbeatMetric is a local struct representing heartbeat data for gauge updates.
// It avoids circular imports with the service package.
type HeartbeatMetric struct {
	Name   string
	Status string
}

// Init registers all Prometheus metrics. It is safe to call multiple times;
// subsequent calls after the first will panic if the registry already
// contains these collectors.
func Init() {
	prometheus.MustRegister(DeployTotal)
	prometheus.MustRegister(DeployDuration)
	prometheus.MustRegister(ActiveContainers)
	prometheus.MustRegister(WSConnections)
	prometheus.MustRegister(APILatency)
	prometheus.MustRegister(CredentialExpiryDays)
	prometheus.MustRegister(MonitorUp)
	prometheus.MustRegister(MonitorLatencyMs)
	prometheus.MustRegister(MonitorUptimePct)
	prometheus.MustRegister(HeartbeatUp)
}

// Handler returns an http.Handler that serves the Prometheus metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

// UpdateMonitorGauges iterates over the given monitors and updates the
// MonitorUp, MonitorLatencyMs, and MonitorUptimePct gauges.
func UpdateMonitorGauges(monitors []MonitorMetric) {
	for _, m := range monitors {
		labels := prometheus.Labels{
			"name":   m.Name,
			"type":   m.Type,
			"target": m.Target,
		}
		if m.Status == "up" {
			MonitorUp.With(labels).Set(1.0)
		} else {
			MonitorUp.With(labels).Set(0.0)
		}
		MonitorLatencyMs.With(labels).Set(m.AvgLatency)
		MonitorUptimePct.With(labels).Set(m.Uptime)
	}
}

// UpdateHeartbeatGauges iterates over the given heartbeats and updates the
// HeartbeatUp gauge.
func UpdateHeartbeatGauges(heartbeats []HeartbeatMetric) {
	for _, h := range heartbeats {
		labels := prometheus.Labels{
			"name": h.Name,
		}
		if h.Status == "up" {
			HeartbeatUp.With(labels).Set(1.0)
		} else {
			HeartbeatUp.With(labels).Set(0.0)
		}
	}
}
