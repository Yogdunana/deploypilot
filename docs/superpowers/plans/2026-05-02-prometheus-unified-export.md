# Phase 6.3: Prometheus Unified Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify Prometheus metrics export into a single `/metrics` endpoint on the main API port, eliminating the separate port 9091 and migrating hand-crafted metrics to native Prometheus types.

**Architecture:** All Prometheus metrics (existing deploy/container/WS/API + new monitor/heartbeat gauges) are registered in `internal/metrics/metrics.go` using native `prometheus` client types. The `/metrics` endpoint is served via `promhttp.Handler()` on the main Gin router. MonitorScheduler updates gauges after each check cycle. Users can toggle public access via a config flag.

**Tech Stack:** Go 1.23, `github.com/prometheus/client_golang` v1.23.2 (already in go.mod), Gin web framework, Vue 3 + TypeScript

**Branch:** `feat/phase-6.3-prometheus-unified`
**PR Title:** `[v1.6] Phase 6.3: Unify Prometheus metrics export to single /metrics endpoint`

---

### Task 1: Add monitor/heartbeat Gauge metrics to internal/metrics

**Files:**
- Modify: `internal/metrics/metrics.go`

- [ ] **Step 1: Add 4 new GaugeVec variables**

Add after the existing `CredentialExpiryDays` variable block:

```go
// MonitorUp indicates whether a monitor target is up (1) or down (0).
MonitorUp = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "deploypilot_monitor_up",
		Help: "Whether a monitor target is up (1) or down (0)",
	},
	[]string{"name", "type", "target"},
)

// MonitorLatencyMs reports the latest check latency in milliseconds.
MonitorLatencyMs = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "deploypilot_monitor_latency_ms",
		Help: "Latest monitor check latency in milliseconds",
	},
	[]string{"name", "type", "target"},
)

// MonitorUptimePct reports the SLA uptime percentage.
MonitorUptimePct = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "deploypilot_monitor_uptime_pct",
		Help: "Monitor uptime percentage (SLA)",
	},
	[]string{"name", "type", "target"},
)

// HeartbeatUp indicates whether a heartbeat source is alive (1) or timed out (0).
HeartbeatUp = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "deploypilot_heartbeat_up",
		Help: "Whether a heartbeat source is alive (1) or timed out (0)",
	},
	[]string{"name"},
)
```

- [ ] **Step 2: Register new metrics in Init()**

Add these lines at the end of the `Init()` function, before the closing `}`:

```go
	prometheus.MustRegister(MonitorUp)
	prometheus.MustRegister(MonitorLatencyMs)
	prometheus.MustRegister(MonitorUptimePct)
	prometheus.MustRegister(HeartbeatUp)
```

- [ ] **Step 3: Add UpdateMonitorGauges and UpdateHeartbeatGauges functions**

Add after the `Handler()` function:

```go
// UpdateMonitorGauges updates monitor-related Prometheus gauges from check results.
// It accepts a slice of Monitor structs (from CheckAllMonitors).
func UpdateMonitorGauges(monitors []service.Monitor) {
	for _, mon := range monitors {
		labels := prometheus.Labels{"name": mon.Name, "type": mon.Type, "target": mon.Target}
		up := 0.0
		if mon.Status == "up" {
			up = 1.0
		}
		MonitorUp.With(labels).Set(up)
		MonitorLatencyMs.With(labels).Set(mon.AvgLatency)
		MonitorUptimePct.With(labels).Set(mon.Uptime)
	}
}

// UpdateHeartbeatGauges updates heartbeat-related Prometheus gauges.
func UpdateHeartbeatGauges(heartbeats []service.Heartbeat) {
	for _, hb := range heartbeats {
		labels := prometheus.Labels{"name": hb.Name}
		up := 0.0
		if hb.Status == "up" {
			up = 1.0
		}
		HeartbeatUp.With(labels).Set(up)
	}
}
```

**IMPORTANT:** These functions import `service` package types. To avoid circular imports (`metrics` → `service` → `metrics`), define interface types locally in metrics.go instead:

```go
// MonitorMetric represents monitor data for Prometheus gauge updates.
type MonitorMetric struct {
	Name      string
	Type      string
	Target    string
	Status    string
	AvgLatency float64
	Uptime    float64
}

// HeartbeatMetric represents heartbeat data for Prometheus gauge updates.
type HeartbeatMetric struct {
	Name   string
	Status string
}

// UpdateMonitorGauges updates monitor-related Prometheus gauges.
func UpdateMonitorGauges(monitors []MonitorMetric) {
	for _, mon := range monitors {
		labels := prometheus.Labels{"name": mon.Name, "type": mon.Type, "target": mon.Target}
		up := 0.0
		if mon.Status == "up" {
			up = 1.0
		}
		MonitorUp.With(labels).Set(up)
		MonitorLatencyMs.With(labels).Set(mon.AvgLatency)
		MonitorUptimePct.With(labels).Set(mon.Uptime)
	}
}

// UpdateHeartbeatGauges updates heartbeat-related Prometheus gauges.
func UpdateHeartbeatGauges(heartbeats []HeartbeatMetric) {
	for _, hb := range heartbeats {
		labels := prometheus.Labels{"name": hb.Name}
		up := 0.0
		if hb.Status == "up" {
			up = 1.0
		}
		HeartbeatUp.With(labels).Set(up)
	}
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./internal/metrics/`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/metrics/metrics.go
git commit -m "feat(metrics): add monitor and heartbeat Prometheus gauges"
```

---

### Task 2: Connect MonitorScheduler to Prometheus gauges

**Files:**
- Modify: `internal/service/monitor_scheduler.go`

- [ ] **Step 1: Add metrics import**

Add to the import block:

```go
	"github.com/Yogdunana/deploypilot/internal/metrics"
```

- [ ] **Step 2: Update runMonitorChecks to feed Prometheus gauges**

In `runMonitorChecks()`, after the `s.notify(...)` call, add Prometheus gauge update. The `results` variable from `s.svc.CheckAllMonitors(ctx)` returns `[]Monitor`. Convert to `[]metrics.MonitorMetric` and call `metrics.UpdateMonitorGauges()`.

Replace the body of the `case <-ticker.C:` block in `runMonitorChecks`:

```go
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
			// Update Prometheus gauges
			metricMonitors := make([]metrics.MonitorMetric, len(results))
			for i, r := range results {
				metricMonitors[i] = metrics.MonitorMetric{
					Name:       r.Name,
					Type:       r.Type,
					Target:     r.Target,
					Status:     r.Status,
					AvgLatency: r.AvgLatency,
					Uptime:     r.Uptime,
				}
			}
			metrics.UpdateMonitorGauges(metricMonitors)
```

- [ ] **Step 3: Update runHeartbeatChecks to feed Prometheus gauges**

In `runHeartbeatChecks()`, after the heartbeat timeout check, also update gauges for ALL heartbeats (not just timed-out ones). We need to fetch all enabled heartbeats and update their gauges.

Replace the body of the `case <-ticker.C:` block in `runHeartbeatChecks`:

```go
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
			// Update Prometheus gauges for all heartbeats
			allHB, err := s.svc.ListHeartbeats(ctx)
			if err == nil {
				metricHBs := make([]metrics.HeartbeatMetric, len(allHB))
				for i, hb := range allHB {
					metricHBs[i] = metrics.HeartbeatMetric{
						Name:   hb.Name,
						Status: hb.Status,
					}
				}
				metrics.UpdateHeartbeatGauges(metricHBs)
			}
```

**IMPORTANT:** Verify that `MonitorService` has a `ListHeartbeats` method. If not, check for an equivalent method name (e.g., `GetHeartbeats`, `ListAllHeartbeats`). The monitor_api.go handler `ListHeartbeats` likely calls such a method. Read the handler to find the correct method name.

- [ ] **Step 4: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./internal/service/`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/service/monitor_scheduler.go
git commit -m "feat(monitor): connect scheduler to Prometheus gauges"
```

---

### Task 3: Update MonitorConfig — remove MetricsPort, add MetricsPublic

**Files:**
- Modify: `internal/config/config.go`
- Modify: `configs/config.yaml.example`

- [ ] **Step 1: Modify MonitorConfig struct**

In `internal/config/config.go`, find the `MonitorConfig` struct (around line 169) and change it to:

```go
// MonitorConfig holds monitoring settings.
type MonitorConfig struct {
	Enabled       bool `mapstructure:"enabled"`
	MetricsPublic bool `mapstructure:"metrics_public"`
}
```

- [ ] **Step 2: Update default values**

Find the Monitor defaults section (around line 303-305) and change to:

```go
	// Monitor
	v.SetDefault("monitor.enabled", true)
	v.SetDefault("monitor.metrics_public", false)
```

- [ ] **Step 3: Update config example**

In `configs/config.yaml.example`, change the monitor section to:

```yaml
monitor:
  enabled: true
  metrics_public: false
```

- [ ] **Step 4: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./internal/config/`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go configs/config.yaml.example
git commit -m "feat(config): replace MetricsPort with MetricsPublic in MonitorConfig"
```

---

### Task 4: Remove standalone metrics server from main.go and serve.go

**Files:**
- Modify: `cmd/api-server/main.go`
- Modify: `cmd/deploypilot/serve.go`

- [ ] **Step 1: Remove metrics server from cmd/api-server/main.go**

Find and remove the entire standalone metrics server block (around lines 276-289):

```go
	// DELETE THIS BLOCK:
	// Initialize and start Prometheus metrics server
	metrics.Init()
	metricsServer := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Monitor.MetricsPort),
		Handler:      metrics.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		slog.Info("starting metrics server", "port", cfg.Monitor.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
		}
	}()
```

Replace with just the Init call (metrics still need to be registered):

```go
	// Initialize Prometheus metrics (served via /metrics on the main API server)
	metrics.Init()
```

Also remove the `"strconv"` import if it's no longer used elsewhere in the file. Check with `go build` — if `strconv` is still used by other code, keep it.

- [ ] **Step 2: Remove metrics server from cmd/deploypilot/serve.go**

Find and remove the standalone metrics server block (around lines 99-113):

```go
	// DELETE THIS BLOCK:
	metrics.Init()
	go func() {
		metricsPort := strconv.Itoa(cfg.Monitor.MetricsPort)
		slog.Info("starting metrics server", "port", metricsPort)
		metricsServer := &http.Server{
			Addr:         ":" + metricsPort,
			Handler:      metrics.Handler(),
			ReadTimeout:  10 * time.Second,
			WriteTimeout:  10 * time.Second,
		}
		if err := metricsServer.ListenAndServe(); err != nil {
			slog.Error("metrics server failed", "error", err)
		}
	}()
```

Replace with:

```go
	// Initialize Prometheus metrics (served via /metrics on the main API server)
	metrics.Init()
```

Also remove the `"strconv"` import if no longer needed.

- [ ] **Step 3: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./cmd/api-server/ && go build ./cmd/deploypilot/`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add cmd/api-server/main.go cmd/deploypilot/serve.go
git commit -m "refactor: remove standalone metrics server, keep metrics.Init() only"
```

---

### Task 5: Update router — remove old /api/v1/metrics, add new /metrics

**Files:**
- Modify: `internal/api/router.go`

- [ ] **Step 1: Remove old /api/v1/metrics route**

Find and delete this line (around line 240):

```go
		r.GET("/api/v1/metrics", globalMonitorAPI.GetPrometheusMetrics)
```

- [ ] **Step 2: Add /metrics route with JWT auth (in protected group)**

Inside the `protected` group (after the existing route registrations), add:

```go
		// Prometheus metrics (JWT authenticated by default)
		protected.GET("/metrics", gin.WrapH(metrics.Handler()))
```

**IMPORTANT:** `gin.WrapH` wraps an `http.Handler` into a Gin handler. You need to import `"github.com/Yogdunana/deploypilot/internal/metrics"` in router.go.

- [ ] **Step 3: Add public /metrics route (conditional on MetricsPublic config)**

The `RegisterRoutes` function needs access to `MonitorConfig` to decide whether to register a public `/metrics` endpoint. Currently, `RegisterRoutes` doesn't receive config. Two options:

**Option A (simpler):** Always register the public `/metrics` at the root level, and let the handler check auth internally. But this duplicates the route.

**Option B (cleaner):** Pass `cfg.Monitor.MetricsPublic` as a parameter to `RegisterRoutes`. If true, also register a public version.

Since `RegisterRoutes` already has many parameters, use Option B. Add a `metricsPublic bool` parameter:

Update the function signature:
```go
func RegisterRoutes(r *gin.Engine, db *gorm.DB, bridge *service.Bridge, wsHub *WSHub, auditSvc *service.AuditService, pluginManager *plugin.Manager, blacklist auth.TokenBlacklist, oauthSvc *service.OAuthService, backupSvc *backup.Service, keySvc *service.APIKeyService, metricsPublic bool) {
```

Then after the protected metrics route, add:
```go
	// Public metrics endpoint (when enabled in config)
	if metricsPublic {
		r.GET("/metrics", gin.WrapH(metrics.Handler()))
	}
```

**IMPORTANT:** This also requires updating ALL callers of `RegisterRoutes`:
- `internal/server/server.go` — `New()` function
- Any test files that call `RegisterRoutes`

Read `internal/server/server.go` to find how `RegisterRoutes` is called, then update the call site to pass the new parameter.

- [ ] **Step 4: Update server.go to pass metricsPublic**

In `internal/server/server.go`, find the `RegisterRoutes` call and add the new parameter. The `Server` struct or `New()` function likely has access to config. Pass `cfg.Monitor.MetricsPublic` as the last argument.

- [ ] **Step 5: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./...`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add internal/api/router.go internal/server/server.go
git commit -m "feat(router): add /metrics endpoint with JWT auth, remove /api/v1/metrics"
```

---

### Task 6: Remove GetPrometheusMetrics from MonitorService and MonitorAPI

**Files:**
- Modify: `internal/service/monitor_service.go`
- Modify: `internal/api/monitor_api.go` (or wherever `GetPrometheusMetrics` handler lives)

- [ ] **Step 1: Remove GetPrometheusMetrics from MonitorService**

In `internal/service/monitor_service.go`, delete the entire `GetPrometheusMetrics` method (around lines 438-472) and the section header comment `// ========== Prometheus Metrics ==========`.

Also check if `boolToInt` helper function is used only by `GetPrometheusMetrics`. If so, delete it too. If it's used elsewhere, keep it.

- [ ] **Step 2: Remove GetPrometheusMetrics handler from MonitorAPI**

Find the `GetPrometheusMetrics` method on `MonitorAPI` (likely in `internal/api/monitor_api.go` or a similar file). Delete the entire method.

- [ ] **Step 3: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./...`
Expected: No errors (the route was already removed in Task 5, so no dangling references)

- [ ] **Step 4: Commit**

```bash
git add internal/service/monitor_service.go internal/api/monitor_api.go
git commit -m "refactor: remove hand-crafted GetPrometheusMetrics, replaced by native Prometheus gauges"
```

---

### Task 7: Add MetricsPublic toggle to frontend MonitorSettings

**Files:**
- Modify: `web/src/views/MonitorSettings.vue`
- Create: `web/src/api/modules/monitor_settings.ts` (if not exists)

- [ ] **Step 1: Create monitor settings API module**

Create `web/src/api/modules/monitor_settings.ts`:

```typescript
import request from '@/utils/request'

export interface MonitorSettings {
  check_interval: number
  timeout: number
  retries: number
  heartbeat_timeout: number
  scheduler_enabled: boolean
  metrics_public: boolean
}

export function getMonitorSettings() {
  return request.get<MonitorSettings>('/monitor/settings')
}

export function updateMonitorSettings(data: Partial<MonitorSettings>) {
  return request.put('/monitor/settings', data)
}
```

**NOTE:** Check if `@/utils/request` is the correct import path by reading an existing API module (e.g., `web/src/api/modules/monitor_query.ts`).

- [ ] **Step 2: Add metrics_public toggle to MonitorSettings.vue**

Add a new `metricsPublic` ref and a new section card in the template. Add after the Scheduler section:

```vue
    <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
      <h3 class="text-base font-semibold mb-4">Prometheus Metrics</h3>
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium">Public Metrics Endpoint</p>
          <p class="text-xs text-gray-500 mt-0.5">Allow unauthenticated access to /metrics for Prometheus scraping. When disabled, JWT authentication is required.</p>
        </div>
        <button class="relative w-11 h-6 rounded-full transition-colors" :class="metricsPublic ? 'bg-blue-600' : 'bg-gray-300'" @click="metricsPublic = !metricsPublic">
          <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform" :class="metricsPublic ? 'translate-x-5' : ''"></span>
        </button>
      </div>
    </div>
```

Add the ref in the script section:

```typescript
const metricsPublic = ref(false)
```

Update `resetDefaults()` to include:

```typescript
metricsPublic.value = false
```

- [ ] **Step 3: Verify frontend builds**

Run: `cd /data/user/work/deploypilot-dev/web && npm run build`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add web/src/views/MonitorSettings.vue web/src/api/modules/monitor_settings.ts
git commit -m "feat(frontend): add Prometheus public metrics toggle to MonitorSettings"
```

---

### Task 8: Write unit tests for metrics package

**Files:**
- Create: `internal/metrics/metrics_test.go`

- [ ] **Step 1: Write tests for UpdateMonitorGauges and UpdateHeartbeatGauges**

```go
package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestUpdateMonitorGauges(t *testing.T) {
	monitors := []MonitorMetric{
		{Name: "google", Type: "http", Target: "https://google.com", Status: "up", AvgLatency: 45.5, Uptime: 99.99},
		{Name: "db", Type: "tcp", Target: "db:5432", Status: "down", AvgLatency: 0, Uptime: 95.5},
	}

	UpdateMonitorGauges(monitors)

	// Verify MonitorUp
	if n := testutil.CollectAndCount(MonitorUp); n != 2 {
		t.Errorf("expected 2 MonitorUp metrics, got %d", n)
	}

	// Verify MonitorLatencyMs
	if n := testutil.CollectAndCount(MonitorLatencyMs); n != 2 {
		t.Errorf("expected 2 MonitorLatencyMs metrics, got %d", n)
	}

	// Verify MonitorUptimePct
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
```

- [ ] **Step 2: Run tests**

Run: `cd /data/user/work/deploypilot-dev && go test ./internal/metrics/ -v`
Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add internal/metrics/metrics_test.go
git commit -m "test(metrics): add unit tests for monitor and heartbeat gauge updates"
```

---

### Task 9: Run full test suite and lint

- [ ] **Step 1: Run Go tests**

Run: `cd /data/user/work/deploypilot-dev && go test ./... -count=1`
Expected: All tests pass

- [ ] **Step 2: Run linter**

Run: `cd /data/user/work/deploypilot-dev && golangci-lint run ./...`
Expected: No errors (fix any errcheck or other lint issues)

- [ ] **Step 3: Run frontend build**

Run: `cd /data/user/work/deploypilot-dev/web && npm run build`
Expected: No errors

- [ ] **Step 4: Fix any issues found**

Address any test failures, lint errors, or build issues.

- [ ] **Step 5: Final commit if any fixes needed**

```bash
git add -A
git commit -m "fix: address lint and test issues for Prometheus unified export"
```

---

### Task 10: Push and create PR

- [ ] **Step 1: Push branch**

```bash
git push -u origin feat/phase-6.3-prometheus-unified
```

- [ ] **Step 2: Create PR**

Create PR with title: `[v1.6] Phase 6.3: Unify Prometheus metrics export to single /metrics endpoint`

Body should reference: `Closes #10` (the Prometheus metrics export issue)

Include summary of changes:
- Removed standalone metrics server on port 9091
- Migrated uptime/heartbeat metrics to native Prometheus Gauge types
- Added `/metrics` endpoint on main API port with JWT auth
- Added `metrics_public` config toggle for optional public access
- MonitorScheduler now updates Prometheus gauges after each check cycle
- Added unit tests for new gauge update functions

- [ ] **Step 3: Monitor CI**

Wait for all CI checks to pass. If any fail, fix and push again.

- [ ] **Step 4: Squash merge**

Once CI passes, squash merge the PR using:
```bash
curl -X PUT \
  -H "Authorization: token <PAT>" \
  -H "Accept: application/vnd.github+json" \
  https://api.github.com/repos/Yogdunana/deploypilot/pulls/<PR_NUMBER>/merge \
  -d '{"merge_method":"squash"}'
```
