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
)

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
}

// Handler returns an http.Handler that serves the Prometheus metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
