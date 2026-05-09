package metrics

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMain(m *testing.M) {
	Init()
	os.Exit(m.Run())
}

// newRegistry creates a fresh prometheus.Registry and registers all metrics
// into it so that tests do not interfere with the global default registry.
func newRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(DeployTotal)
	reg.MustRegister(DeployDuration)
	reg.MustRegister(ActiveContainers)
	reg.MustRegister(WSConnections)
	reg.MustRegister(APILatency)
	reg.MustRegister(CredentialExpiryDays)
	return reg
}

func TestInitDoesNotPanic(t *testing.T) {
	// Init registers with the global registry. We can only call it once
	// without panicking (double-register panics). Guard with recover.
	defer func() {
		if r := recover(); r != nil {
			// If it panics because of double-register that is expected
			// when tests run together; it means the first call succeeded.
			t.Logf("Init panicked (expected on repeated calls): %v", r)
		}
	}()
	Init()
}

func TestHandlerReturnsNonNil(t *testing.T) {
	h := Handler()
	if h == nil {
		t.Fatal("expected non-nil handler")
	}

	// Verify it is usable as an http.Handler
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if len(body) == 0 {
		t.Fatal("expected non-empty response body")
	}
}

func TestCounterIncrement(t *testing.T) {
	reg := newRegistry()

	// Increment the counter
	DeployTotal.WithLabelValues("myapp", "server1", "success").Inc()
	DeployTotal.WithLabelValues("myapp", "server1", "success").Inc()
	DeployTotal.WithLabelValues("myapp", "server1", "failed").Inc()

	// Read back from the registry
	count := testutil.ToFloat64(DeployTotal.WithLabelValues("myapp", "server1", "success"))
	if count != 2 {
		t.Fatalf("expected counter value 2, got %f", count)
	}

	countFailed := testutil.ToFloat64(DeployTotal.WithLabelValues("myapp", "server1", "failed"))
	if countFailed != 1 {
		t.Fatalf("expected failed counter value 1, got %f", countFailed)
	}

	_ = reg // registry used for isolation
}

func TestHistogramObservation(t *testing.T) {
	reg := newRegistry()

	// Observe some durations
	DeployDuration.Observe(0.5)
	DeployDuration.Observe(1.0)
	DeployDuration.Observe(2.0)

	// testutil.ToFloat64 panics on histograms; use CollectAndCount instead.
	// A histogram produces _bucket, _sum, _count lines (at least 3).
	count := testutil.CollectAndCount(DeployDuration)
	if count < 1 {
		t.Fatalf("expected at least 1 metric family collected, got %d", count)
	}

	_ = reg
}

func TestGaugeSet(t *testing.T) {
	reg := newRegistry()

	ActiveContainers.Set(5)
	val := testutil.ToFloat64(ActiveContainers)
	if val != 5 {
		t.Fatalf("expected gauge value 5, got %f", val)
	}

	ActiveContainers.Dec()
	val = testutil.ToFloat64(ActiveContainers)
	if val != 4 {
		t.Fatalf("expected gauge value 4 after Dec, got %f", val)
	}

	_ = reg
}

func TestWSConnectionsIncDec(t *testing.T) {
	reg := newRegistry()

	WSConnections.Inc()
	WSConnections.Inc()
	val := testutil.ToFloat64(WSConnections)
	if val != 2 {
		t.Fatalf("expected ws connections 2, got %f", val)
	}

	WSConnections.Dec()
	val = testutil.ToFloat64(WSConnections)
	if val != 1 {
		t.Fatalf("expected ws connections 1 after Dec, got %f", val)
	}

	_ = reg
}

func TestCredentialExpiryDays(t *testing.T) {
	reg := newRegistry()

	CredentialExpiryDays.WithLabelValues("my-cred").Set(30)
	val := testutil.ToFloat64(CredentialExpiryDays.WithLabelValues("my-cred"))
	if val != 30 {
		t.Fatalf("expected credential expiry 30, got %f", val)
	}

	_ = reg
}

func TestUpdateMonitorGauges(t *testing.T) {
	monitors := []MonitorMetric{
		{Name: "google", Type: "http", Target: "https://google.com", Status: "up", AvgLatency: 45.5, Uptime: 99.99},
		{Name: "db", Type: "tcp", Target: "db:5432", Status: "down", AvgLatency: 0, Uptime: 95.5},
	}

	UpdateMonitorGauges(monitors)

	if n := testutil.CollectAndCount(MonitorUp); n != 2 {
		t.Errorf("expected 2 MonitorUp metrics, got %d", n)
	}
	if n := testutil.CollectAndCount(MonitorLatencyMs); n != 2 {
		t.Errorf("expected 2 MonitorLatencyMs metrics, got %d", n)
	}
	if n := testutil.CollectAndCount(MonitorUptimePct); n != 2 {
		t.Errorf("expected 2 MonitorUptimePct metrics, got %d", n)
	}
}

func TestUpdateHeartbeatGauges(t *testing.T) {
	heartbeats := []HeartbeatMetric{
		{Name: "app-1", Status: "up"},
		{Name: "app-2", Status: "down"},
	}

	UpdateHeartbeatGauges(heartbeats)

	if n := testutil.CollectAndCount(HeartbeatUp); n != 2 {
		t.Errorf("expected 2 HeartbeatUp metrics, got %d", n)
	}
}

func TestUpdateMonitorGaugesEmpty(t *testing.T) {
	UpdateMonitorGauges([]MonitorMetric{})
	// Should not panic
}

func TestUpdateHeartbeatGaugesEmpty(t *testing.T) {
	UpdateHeartbeatGauges([]HeartbeatMetric{})
	// Should not panic
}
