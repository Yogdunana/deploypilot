package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database and auto-migrates all
// uptime-related tables. Each test gets its own isolated instance.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(
		&Monitor{},
		&Heartbeat{},
		&MonitorCheckResult{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return db
}

// newTestMonitor returns a Monitor with sensible defaults for testing.
func newTestMonitor(name, monType, target string) *Monitor {
	return &Monitor{
		ID:        generateID(),
		TenantID:  "tenant-test",
		Name:      name,
		Type:      monType,
		Target:    target,
		Interval:  60,
		Timeout:   2,
		Retries:   3,
		Status:    "unknown",
		Enabled:   true,
		Uptime:    100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// newTestHeartbeat returns a Heartbeat with sensible defaults for testing.
func newTestHeartbeat(name string, timeout int) *Heartbeat {
	return &Heartbeat{
		ID:        generateID(),
		TenantID:  "tenant-test",
		Name:      name,
		Token:     "", // will be auto-generated
		Interval:  60,
		Timeout:   timeout,
		Status:    "unknown",
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ---------- 1. TestCreateMonitor ----------

func TestCreateMonitor(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(db)
	ctx := context.Background()

	mon := newTestMonitor("Google DNS", "tcp", "8.8.8.8:53")
	err := svc.CreateMonitor(ctx, mon)
	if err != nil {
		t.Fatalf("CreateMonitor() error = %v", err)
	}
	if mon.ID == "" {
		t.Fatal("monitor ID should not be empty after creation")
	}

	// Verify it can be retrieved.
	got, err := svc.GetMonitor(ctx, mon.ID)
	if err != nil {
		t.Fatalf("GetMonitor() error = %v", err)
	}
	if got.Name != mon.Name {
		t.Errorf("Name = %q, want %q", got.Name, mon.Name)
	}
	if got.Type != mon.Type {
		t.Errorf("Type = %q, want %q", got.Type, mon.Type)
	}
	if got.Target != mon.Target {
		t.Errorf("Target = %q, want %q", got.Target, mon.Target)
	}
	if !got.Enabled {
		t.Error("Enabled should be true")
	}
}

// ---------- 2. TestListMonitors ----------

func TestListMonitors(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(db)
	ctx := context.Background()

	// Create two monitors.
	m1 := newTestMonitor("Monitor-1", "http", "https://example.com")
	m2 := newTestMonitor("Monitor-2", "tcp", "10.0.0.1:80")
	if err := svc.CreateMonitor(ctx, m1); err != nil {
		t.Fatalf("CreateMonitor(m1) error = %v", err)
	}
	if err := svc.CreateMonitor(ctx, m2); err != nil {
		t.Fatalf("CreateMonitor(m2) error = %v", err)
	}

	// List all.
	list, err := svc.ListMonitors(ctx, "")
	if err != nil {
		t.Fatalf("ListMonitors() error = %v", err)
	}
	monitors, ok := list.([]Monitor)
	if !ok {
		t.Fatal("ListMonitors() should return []Monitor")
	}
	if len(monitors) != 2 {
		t.Errorf("len(monitors) = %d, want 2", len(monitors))
	}

	// Filter by tenant.
	list, err = svc.ListMonitors(ctx, "tenant-test")
	if err != nil {
		t.Fatalf("ListMonitors(tenant) error = %v", err)
	}
	monitors = list.([]Monitor)
	if len(monitors) != 2 {
		t.Errorf("len(monitors) for tenant-test = %d, want 2", len(monitors))
	}

	// Filter by non-existent tenant.
	list, err = svc.ListMonitors(ctx, "tenant-other")
	if err != nil {
		t.Fatalf("ListMonitors(other) error = %v", err)
	}
	monitors = list.([]Monitor)
	if len(monitors) != 0 {
		t.Errorf("len(monitors) for tenant-other = %d, want 0", len(monitors))
	}
}

// ---------- 3. TestGetMonitor ----------

func TestGetMonitor(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(db)
	ctx := context.Background()

	mon := newTestMonitor("Detailed Monitor", "http", "https://example.com/health")
	mon.Interval = 30
	mon.Timeout = 5
	mon.Retries = 5
	if err := svc.CreateMonitor(ctx, mon); err != nil {
		t.Fatalf("CreateMonitor() error = %v", err)
	}

	got, err := svc.GetMonitor(ctx, mon.ID)
	if err != nil {
		t.Fatalf("GetMonitor() error = %v", err)
	}
	if got.ID != mon.ID {
		t.Errorf("ID = %q, want %q", got.ID, mon.ID)
	}
	if got.Name != "Detailed Monitor" {
		t.Errorf("Name = %q, want 'Detailed Monitor'", got.Name)
	}
	if got.Type != "http" {
		t.Errorf("Type = %q, want 'http'", got.Type)
	}
	if got.Target != "https://example.com/health" {
		t.Errorf("Target = %q, want 'https://example.com/health'", got.Target)
	}
	if got.Interval != 30 {
		t.Errorf("Interval = %d, want 30", got.Interval)
	}
	if got.Timeout != 5 {
		t.Errorf("Timeout = %d, want 5", got.Timeout)
	}
	if got.Retries != 5 {
		t.Errorf("Retries = %d, want 5", got.Retries)
	}
	if got.TenantID != "tenant-test" {
		t.Errorf("TenantID = %q, want 'tenant-test'", got.TenantID)
	}

	// Non-existent ID should return error.
	_, err = svc.GetMonitor(ctx, "non-existent-id")
	if err == nil {
		t.Fatal("GetMonitor(non-existent) should return error")
	}
}

// ---------- 4. TestDeleteMonitor ----------

func TestDeleteMonitor(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(db)
	ctx := context.Background()

	mon := newTestMonitor("To Delete", "http", "https://example.com")
	if err := svc.CreateMonitor(ctx, mon); err != nil {
		t.Fatalf("CreateMonitor() error = %v", err)
	}

	// Verify it exists.
	_, err := svc.GetMonitor(ctx, mon.ID)
	if err != nil {
		t.Fatalf("GetMonitor() before delete error = %v", err)
	}

	// Delete.
	if err := svc.DeleteMonitor(ctx, mon.ID); err != nil {
		t.Fatalf("DeleteMonitor() error = %v", err)
	}

	// Verify it is gone.
	_, err = svc.GetMonitor(ctx, mon.ID)
	if err == nil {
		t.Fatal("GetMonitor() after delete should return error")
	}

	// Deleting non-existent should not error (GORM soft-delete behavior).
	err = svc.DeleteMonitor(ctx, "non-existent-id")
	if err != nil {
		t.Errorf("DeleteMonitor(non-existent) should not error, got %v", err)
	}
}

// ---------- 5. TestGetMonitorSLA ----------

func TestGetMonitorSLA(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(db)
	ctx := context.Background()

	mon := newTestMonitor("SLA Monitor", "http", "https://example.com")
	if err := svc.CreateMonitor(ctx, mon); err != nil {
		t.Fatalf("CreateMonitor() error = %v", err)
	}

	// Manually insert check results to avoid real HTTP calls.
	now := time.Now()
	results := []MonitorCheckResult{
		{ID: generateID(), MonitorID: mon.ID, Status: "up", StatusCode: 200, Latency: 50, Message: "OK", CreatedAt: now},
		{ID: generateID(), MonitorID: mon.ID, Status: "up", StatusCode: 200, Latency: 60, Message: "OK", CreatedAt: now.Add(-1 * time.Hour)},
		{ID: generateID(), MonitorID: mon.ID, Status: "up", StatusCode: 200, Latency: 70, Message: "OK", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: generateID(), MonitorID: mon.ID, Status: "down", StatusCode: 500, Latency: 100, Message: "500", CreatedAt: now.Add(-3 * time.Hour)},
		{ID: generateID(), MonitorID: mon.ID, Status: "up", StatusCode: 200, Latency: 55, Message: "OK", CreatedAt: now.Add(-4 * time.Hour)},
	}
	for i := range results {
		if err := db.Create(&results[i]).Error; err != nil {
			t.Fatalf("insert check result %d error = %v", i, err)
		}
	}

	// Query SLA for 30 days (covers all results).
	sla, err := svc.GetMonitorSLA(ctx, mon.ID, 30)
	if err != nil {
		t.Fatalf("GetMonitorSLA() error = %v", err)
	}
	m, ok := sla.(map[string]interface{})
	if !ok {
		t.Fatal("GetMonitorSLA() should return map[string]interface{}")
	}
	if m["monitor_id"] != mon.ID {
		t.Errorf("monitor_id = %v, want %q", m["monitor_id"], mon.ID)
	}
	if m["total_checks"].(int64) != 5 {
		t.Errorf("total_checks = %v, want 5", m["total_checks"])
	}
	if m["up_checks"].(int64) != 4 {
		t.Errorf("up_checks = %v, want 4", m["up_checks"])
	}
	// 4/5 = 80% uptime.
	uptimeStr := m["uptime"].(string)
	if uptimeStr != "80.00" {
		t.Errorf("uptime = %q, want '80.00'", uptimeStr)
	}

	// Query SLA for 1 day (only results within the last 24h).
	sla, err = svc.GetMonitorSLA(ctx, mon.ID, 1)
	if err != nil {
		t.Fatalf("GetMonitorSLA(1day) error = %v", err)
	}
	m = sla.(map[string]interface{})
	// Only the first result is within 1 day.
	if m["total_checks"].(int64) != 1 {
		t.Errorf("total_checks(1day) = %v, want 1", m["total_checks"])
	}
	if m["up_checks"].(int64) != 1 {
		t.Errorf("up_checks(1day) = %v, want 1", m["up_checks"])
	}
}

// ---------- 6. TestCreateHeartbeat ----------

func TestCreateHeartbeat(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(db)
	ctx := context.Background()

	hb := newTestHeartbeat("deploy-cron", 120)
	if err := svc.CreateHeartbeat(ctx, hb); err != nil {
		t.Fatalf("CreateHeartbeat() error = %v", err)
	}
	if hb.ID == "" {
		t.Fatal("heartbeat ID should not be empty")
	}
	if hb.Token == "" {
		t.Fatal("heartbeat Token should be auto-generated")
	}
	if len(hb.Token) != 32 {
		t.Errorf("Token length = %d, want 32 (hex-encoded 16 bytes)", len(hb.Token))
	}

	// Verify retrieval.
	list, err := svc.ListHeartbeats(ctx, "")
	if err != nil {
		t.Fatalf("ListHeartbeats() error = %v", err)
	}
	heartbeats, ok := list.([]Heartbeat)
	if !ok {
		t.Fatal("ListHeartbeats() should return []Heartbeat")
	}
	if len(heartbeats) != 1 {
		t.Fatalf("len(heartbeats) = %d, want 1", len(heartbeats))
	}
	if heartbeats[0].Name != "deploy-cron" {
		t.Errorf("Name = %q, want 'deploy-cron'", heartbeats[0].Name)
	}
}

// ---------- 7. TestPingHeartbeat ----------

func TestPingHeartbeat(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(db)
	ctx := context.Background()

	hb := newTestHeartbeat("backup-job", 300)
	if err := svc.CreateHeartbeat(ctx, hb); err != nil {
		t.Fatalf("CreateHeartbeat() error = %v", err)
	}

	token := hb.Token

	// Ping with the token.
	err := svc.PingHeartbeat(ctx, token)
	if err != nil {
		t.Fatalf("PingHeartbeat() error = %v", err)
	}

	// Verify LastBeat was updated and status is "up".
	list, err := svc.ListHeartbeats(ctx, "")
	if err != nil {
		t.Fatalf("ListHeartbeats() error = %v", err)
	}
	heartbeats := list.([]Heartbeat)
	if len(heartbeats) != 1 {
		t.Fatalf("len(heartbeats) = %d, want 1", len(heartbeats))
	}
	if heartbeats[0].LastBeat == nil {
		t.Fatal("LastBeat should be set after ping")
	}
	if heartbeats[0].Status != "up" {
		t.Errorf("Status = %q, want 'up'", heartbeats[0].Status)
	}

	// Ping with invalid token should error.
	err = svc.PingHeartbeat(ctx, "invalid-token-000000000000000")
	if err == nil {
		t.Fatal("PingHeartbeat(invalid) should return error")
	}
}

// ---------- 8. TestCheckHeartbeats ----------

func TestCheckHeartbeats(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(db)
	ctx := context.Background()

	// Create a heartbeat with a very short timeout and an old CreatedAt.
	hb := newTestHeartbeat("stale-job", 1) // 1 second timeout
	hb.CreatedAt = time.Now().Add(-5 * time.Minute)
	hb.UpdatedAt = time.Now().Add(-5 * time.Minute)
	if err := svc.CreateHeartbeat(ctx, hb); err != nil {
		t.Fatalf("CreateHeartbeat() error = %v", err)
	}

	// Check heartbeats. Since LastBeat is nil and CreatedAt is older than
	// the timeout, this heartbeat should be detected as timed out.
	timedOut, err := svc.CheckHeartbeats(ctx)
	if err != nil {
		t.Fatalf("CheckHeartbeats() error = %v", err)
	}
	timedOutList, ok := timedOut.([]Heartbeat)
	if !ok {
		t.Fatal("CheckHeartbeats() should return []Heartbeat")
	}
	if len(timedOutList) != 1 {
		t.Fatalf("len(timedOut) = %d, want 1", len(timedOutList))
	}
	if timedOutList[0].ID != hb.ID {
		t.Errorf("timedOut ID = %q, want %q", timedOutList[0].ID, hb.ID)
	}
	if timedOutList[0].Status != "down" {
		t.Errorf("timedOut Status = %q, want 'down'", timedOutList[0].Status)
	}

	// Verify the heartbeat was updated in the DB.
	list, err := svc.ListHeartbeats(ctx, "")
	if err != nil {
		t.Fatalf("ListHeartbeats() error = %v", err)
	}
	heartbeats := list.([]Heartbeat)
	if heartbeats[0].Status != "down" {
		t.Errorf("heartbeat Status in DB = %q, want 'down'", heartbeats[0].Status)
	}
}

// ---------- 9. TestGetStatusPage ----------

func TestGetStatusPage(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(db)
	ctx := context.Background()

	// Create monitors with different statuses.
	m1 := newTestMonitor("API Server", "http", "https://api.example.com")
	m1.Status = "up"
	m2 := newTestMonitor("Database", "tcp", "db.example.com:5432")
	m2.Status = "down"
	m3 := newTestMonitor("Cache", "tcp", "redis.example.com:6379")
	m3.Status = "up"

	for _, m := range []*Monitor{m1, m2, m3} {
		if err := svc.CreateMonitor(ctx, m); err != nil {
			t.Fatalf("CreateMonitor() error = %v", err)
		}
	}

	page, err := svc.GetStatusPage(ctx, "")
	if err != nil {
		t.Fatalf("GetStatusPage() error = %v", err)
	}
	pm, ok := page.(map[string]interface{})
	if !ok {
		t.Fatal("GetStatusPage() should return map[string]interface{}")
	}
	if pm["total"].(int) != 3 {
		t.Errorf("total = %v, want 3", pm["total"])
	}
	if pm["up"].(int) != 2 {
		t.Errorf("up = %v, want 2", pm["up"])
	}
	if pm["down"].(int) != 1 {
		t.Errorf("down = %v, want 1", pm["down"])
	}
	// Partial outage because some are up and some are down.
	if pm["overall_status"].(string) != "partial_outage" {
		t.Errorf("overall_status = %q, want 'partial_outage'", pm["overall_status"])
	}

	// Test with all monitors up.
	m2.Status = "up"
	db.Save(m2)

	page, err = svc.GetStatusPage(ctx, "")
	if err != nil {
		t.Fatalf("GetStatusPage() error = %v", err)
	}
	pm = page.(map[string]interface{})
	if pm["overall_status"].(string) != "operational" {
		t.Errorf("overall_status = %q, want 'operational'", pm["overall_status"])
	}

	// Test with tenant filter.
	page, err = svc.GetStatusPage(ctx, "tenant-other")
	if err != nil {
		t.Fatalf("GetStatusPage(other) error = %v", err)
	}
	pm = page.(map[string]interface{})
	if pm["total"].(int) != 0 {
		t.Errorf("total for other tenant = %v, want 0", pm["total"])
	}
}

// ---------- 10. TestGetPrometheusMetrics ----------

func TestGetPrometheusMetrics(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(db)
	ctx := context.Background()

	// Create monitors and heartbeats.
	m1 := newTestMonitor("Web Server", "http", "https://web.example.com")
	m1.Status = "up"
	m1.Uptime = 99.95
	m1.AvgLatency = 42.5
	m1.TotalChecks = 1000

	m2 := newTestMonitor("DB Replica", "tcp", "db.example.com:5432")
	m2.Status = "down"
	m2.Uptime = 85.0
	m2.AvgLatency = 120.0
	m2.TotalChecks = 500

	for _, m := range []*Monitor{m1, m2} {
		if err := svc.CreateMonitor(ctx, m); err != nil {
			t.Fatalf("CreateMonitor() error = %v", err)
		}
	}

	hb := newTestHeartbeat("cron-sync", 300)
	hb.Status = "up"
	if err := svc.CreateHeartbeat(ctx, hb); err != nil {
		t.Fatalf("CreateHeartbeat() error = %v", err)
	}

	metrics, err := svc.GetPrometheusMetrics(ctx)
	if err != nil {
		t.Fatalf("GetPrometheusMetrics() error = %v", err)
	}

	// Verify the output contains expected Prometheus metric lines.
	if !strings.Contains(metrics, "deploypilot_monitor_up{") {
		t.Error("metrics should contain deploypilot_monitor_up")
	}
	if !strings.Contains(metrics, fmt.Sprintf(`deploypilot_monitor_up{name="Web Server",type="http",target="https://web.example.com"} 1`)) {
		t.Error("metrics should contain Web Server up=1")
	}
	if !strings.Contains(metrics, fmt.Sprintf(`deploypilot_monitor_up{name="DB Replica",type="tcp",target="db.example.com:5432"} 0`)) {
		t.Error("metrics should contain DB Replica up=0")
	}
	if !strings.Contains(metrics, "deploypilot_monitor_uptime_percent{") {
		t.Error("metrics should contain deploypilot_monitor_uptime_percent")
	}
	if !strings.Contains(metrics, "deploypilot_monitor_avg_latency_ms{") {
		t.Error("metrics should contain deploypilot_monitor_avg_latency_ms")
	}
	if !strings.Contains(metrics, "deploypilot_monitor_total_checks{") {
		t.Error("metrics should contain deploypilot_monitor_total_checks")
	}
	if !strings.Contains(metrics, fmt.Sprintf(`deploypilot_heartbeat_up{name="cron-sync"} 1`)) {
		t.Error("metrics should contain heartbeat_up for cron-sync")
	}

	// Verify disabled monitors are excluded.
	m1.Enabled = false
	db.Save(m1)

	metrics, err = svc.GetPrometheusMetrics(ctx)
	if err != nil {
		t.Fatalf("GetPrometheusMetrics() after disable error = %v", err)
	}
	if strings.Contains(metrics, "Web Server") {
		t.Error("disabled monitor 'Web Server' should not appear in metrics")
	}
	// DB Replica should still be present.
	if !strings.Contains(metrics, "DB Replica") {
		t.Error("enabled monitor 'DB Replica' should still appear in metrics")
	}
}
