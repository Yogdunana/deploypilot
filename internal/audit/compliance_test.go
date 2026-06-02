package audit

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newAuditDB spins up a fresh in-memory SQLite DB with the audit schema
// pre-migrated. It is the canonical test fixture for any audit_*_test.go
// in this package.
func newAuditDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.AuditLog{}, &model.AuditHash{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// The compliance & export code paths filter on `tenant_id`, but the
	// GORM `AuditLog` model does not declare that column. In production
	// the migration adds it; in the in-memory test fixture we add it
	// manually so the filter is exercised.
	if err := db.Exec("ALTER TABLE audit_logs ADD COLUMN tenant_id TEXT").Error; err != nil {
		t.Fatalf("add tenant_id column: %v", err)
	}
	return db
}

// seedAuditLog inserts a single audit log row.
func seedAuditLog(t *testing.T, db *gorm.DB, l model.AuditLog) {
	t.Helper()
	if err := db.Create(&l).Error; err != nil {
		t.Fatalf("create audit log: %v", err)
	}
}

// ---------- ExportUserData ----------

// TestExportUserData_IncludesAuditLogs confirms the user data export
// surfaces the user's audit logs and counts them.
func TestExportUserData_IncludesAuditLogs(t *testing.T) {
	db := newAuditDB(t)
	now := time.Now().UTC()
	seedAuditLog(t, db, model.AuditLog{
		ID: "a-1", UserID: "u-1", Username: "alice", Action: "user.login",
		IPAddress: "10.0.0.1", UserAgent: "ua", LogType: "auth", CreatedAt: now,
	})
	seedAuditLog(t, db, model.AuditLog{
		ID: "a-2", UserID: "u-1", Username: "alice", Action: "user.update",
		IPAddress: "10.0.0.1", UserAgent: "ua", LogType: "operation", CreatedAt: now,
	})
	seedAuditLog(t, db, model.AuditLog{
		ID: "a-3", UserID: "u-2", Username: "bob", Action: "user.login",
		IPAddress: "10.0.0.2", UserAgent: "ua", LogType: "auth", CreatedAt: now,
	})

	out, err := ExportUserData(db, "u-1")
	if err != nil {
		t.Fatalf("ExportUserData: %v", err)
	}
	if got := out["user_id"]; got != "u-1" {
		t.Errorf("user_id = %v, want u-1", got)
	}
	logs, ok := out["audit_logs"].([]model.AuditLog)
	if !ok {
		t.Fatalf("audit_logs missing or wrong type: %T", out["audit_logs"])
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 audit logs for u-1, got %d", len(logs))
	}
	if got := out["audit_log_count"]; got != 2 {
		t.Errorf("audit_log_count = %v, want 2", got)
	}
	if _, ok := out["exported_at"].(string); !ok {
		t.Error("exported_at should be a string")
	}
}

// TestExportUserData_NoDataReturnsEmptySlice asserts the "no rows"
// contract: even with zero matching records, the export must contain an
// empty slice (not nil) so the JSON consumer gets a stable shape.
func TestExportUserData_NoDataReturnsEmptySlice(t *testing.T) {
	db := newAuditDB(t)
	out, err := ExportUserData(db, "u-doesnotexist")
	if err != nil {
		t.Fatalf("ExportUserData: %v", err)
	}
	logs, ok := out["audit_logs"].([]model.AuditLog)
	if !ok {
		t.Fatalf("audit_logs missing or wrong type: %T", out["audit_logs"])
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 audit logs, got %d", len(logs))
	}
}

// ---------- DeleteUserData (GDPR) ----------

// TestDeleteUserData_AnonymizesAuditLogs confirms the GDPR right-to-be-
// forgotten path clears all PII fields while keeping the audit log
// itself intact (compliance / chain integrity).
func TestDeleteUserData_AnonymizesAuditLogs(t *testing.T) {
	db := newAuditDB(t)
	seedAuditLog(t, db, model.AuditLog{
		ID: "a-1", UserID: "u-1", Username: "alice", Action: "user.login",
		IPAddress: "10.0.0.1", UserAgent: "Mozilla/5.0", LogType: "auth",
	})
	seedAuditLog(t, db, model.AuditLog{
		ID: "a-2", UserID: "u-2", Username: "bob", Action: "user.login",
		IPAddress: "10.0.0.2", UserAgent: "curl/8", LogType: "auth",
	})

	if err := DeleteUserData(db, "u-1"); err != nil {
		t.Fatalf("DeleteUserData: %v", err)
	}

	var kept model.AuditLog
	if err := db.First(&kept, "id = ?", "a-1").Error; err != nil {
		t.Fatalf("audit log was deleted: %v", err)
	}
	if kept.UserID != "" {
		t.Errorf("UserID = %q, want empty", kept.UserID)
	}
	if kept.Username != "[GDPR-DELETED]" {
		t.Errorf("Username = %q, want [GDPR-DELETED]", kept.Username)
	}
	if kept.IPAddress != "[GDPR-DELETED]" {
		t.Errorf("IPAddress = %q, want [GDPR-DELETED]", kept.IPAddress)
	}
	if kept.UserAgent != "[GDPR-DELETED]" {
		t.Errorf("UserAgent = %q, want [GDPR-DELETED]", kept.UserAgent)
	}

	// Untouched user's row must remain intact.
	var bob model.AuditLog
	if err := db.First(&bob, "id = ?", "a-2").Error; err != nil {
		t.Fatalf("other user's audit log missing: %v", err)
	}
	if bob.Username != "bob" {
		t.Errorf("other user's username was modified: %q", bob.Username)
	}
	if bob.IPAddress != "10.0.0.2" {
		t.Errorf("other user's IP was modified: %q", bob.IPAddress)
	}
}

// ---------- DataRetentionPolicy ----------

// TestDataRetentionPolicy_OnlyDeletesArchived verifies the
// "append-only protection for active records" contract: only archived
// records older than the retention period are deleted; non-archived
// records are kept even if old.
func TestDataRetentionPolicy_OnlyDeletesArchived(t *testing.T) {
	db := newAuditDB(t)
	oldDate := time.Now().AddDate(-1, 0, 0) // 1 year old
	seedAuditLog(t, db, model.AuditLog{
		ID: "old-active", Action: "x", CreatedAt: oldDate, Archived: false,
	})
	seedAuditLog(t, db, model.AuditLog{
		ID: "old-archived", Action: "x", CreatedAt: oldDate, Archived: true,
	})
	seedAuditLog(t, db, model.AuditLog{
		ID: "new-archived", Action: "x", CreatedAt: time.Now().Add(-1 * time.Hour), Archived: true,
	})

	cfg := &config.AuditConfig{RetentionDays: 90}
	if err := DataRetentionPolicy(db, cfg); err != nil {
		t.Fatalf("DataRetentionPolicy: %v", err)
	}

	var rows []model.AuditLog
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	gotIDs := map[string]bool{}
	for _, r := range rows {
		gotIDs[r.ID] = true
	}
	if !gotIDs["old-active"] {
		t.Error("old active record was deleted – must be preserved")
	}
	if gotIDs["old-archived"] {
		t.Error("old archived record was not deleted")
	}
	if !gotIDs["new-archived"] {
		t.Error("new archived record was deleted – must be preserved")
	}
}

// TestDataRetentionPolicy_DefaultRetentionWhenZero covers the fallback
// when the operator has not configured a positive retention window.
func TestDataRetentionPolicy_DefaultRetentionWhenZero(t *testing.T) {
	db := newAuditDB(t)
	// An archived record from 10 years ago must be deleted under the
	// default 90-day retention policy.
	veryOld := time.Now().AddDate(-10, 0, 0)
	seedAuditLog(t, db, model.AuditLog{
		ID: "ancient", Action: "x", CreatedAt: veryOld, Archived: true,
	})

	cfg := &config.AuditConfig{RetentionDays: 0} // triggers default
	if err := DataRetentionPolicy(db, cfg); err != nil {
		t.Fatalf("DataRetentionPolicy: %v", err)
	}
	var rows []model.AuditLog
	db.Find(&rows)
	if len(rows) != 0 {
		t.Errorf("expected 0 records, got %d", len(rows))
	}
}

// TestDataRetentionPolicy_NegativeRetentionFallsBackToDefault locks in
// the "defensive default" contract: a misconfigured negative retention
// window must not accidentally retain or wipe everything.
func TestDataRetentionPolicy_NegativeRetentionFallsBackToDefault(t *testing.T) {
	db := newAuditDB(t)
	// Record from 5 years ago, archived.
	old := time.Now().AddDate(-5, 0, 0)
	seedAuditLog(t, db, model.AuditLog{
		ID: "ancient", Action: "x", CreatedAt: old, Archived: true,
	})

	cfg := &config.AuditConfig{RetentionDays: -1}
	if err := DataRetentionPolicy(db, cfg); err != nil {
		t.Fatalf("DataRetentionPolicy: %v", err)
	}
	var rows []model.AuditLog
	db.Find(&rows)
	if len(rows) != 0 {
		t.Errorf("expected 0 records (default retention applies), got %d", len(rows))
	}
}

// ---------- GenerateComplianceReport ----------

// TestGenerateComplianceReport_BasicAggregation verifies the report
// groups records by log_type and computes total/unique counts.
func TestGenerateComplianceReport_BasicAggregation(t *testing.T) {
	db := newAuditDB(t)
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		seedAuditLog(t, db, model.AuditLog{
			ID: "auth-" + string(rune('a'+i)),
			UserID: "u-1", Action: "user.login", LogType: "auth", CreatedAt: now,
		})
	}
	for i := 0; i < 2; i++ {
		seedAuditLog(t, db, model.AuditLog{
			ID: "op-" + string(rune('a'+i)),
			UserID: "u-1", Action: "app.deploy", LogType: "operation", CreatedAt: now,
		})
	}
	// One custom log type that does not appear in the report's bucket map.
	seedAuditLog(t, db, model.AuditLog{
		ID: "weird-1", UserID: "u-1", Action: "x", LogType: "weird_type", CreatedAt: now,
	})

	report, err := GenerateComplianceReport(db, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("GenerateComplianceReport: %v", err)
	}
	if report.TotalRecords != 6 {
		t.Errorf("TotalRecords = %d, want 6", report.TotalRecords)
	}
	if report.RecordsByType["auth"] != 3 {
		t.Errorf("auth = %d, want 3", report.RecordsByType["auth"])
	}
	if report.RecordsByType["operation"] != 2 {
		t.Errorf("operation = %d, want 2", report.RecordsByType["operation"])
	}
	if report.RecordsByType["other"] != 1 {
		t.Errorf("other = %d, want 1 (weird_type should fall in here)", report.RecordsByType["other"])
	}
	if report.UniqueUsers != 1 {
		t.Errorf("UniqueUsers = %d, want 1", report.UniqueUsers)
	}
	if report.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be populated")
	}
}

// TestGenerateComplianceReport_TenantFilter exercises the tenant-scoped
// branch: passing a tenantID should restrict all aggregations.
func TestGenerateComplianceReport_TenantFilter(t *testing.T) {
	db := newAuditDB(t)
	now := time.Now().UTC()
	seedAuditLog(t, db, model.AuditLog{
		ID: "t1-1", UserID: "u-1", Action: "x", LogType: "auth", CreatedAt: now,
	})
	seedAuditLog(t, db, model.AuditLog{
		ID: "t2-1", UserID: "u-1", Action: "x", LogType: "auth", CreatedAt: now,
	})
	// The GORM `AuditLog` model does not declare `TenantID`, so we
	// populate the column directly via raw SQL.
	if err := db.Exec("UPDATE audit_logs SET tenant_id = ? WHERE id = ?", "tenant-A", "t1-1").Error; err != nil {
		t.Fatalf("set tenant-A: %v", err)
	}
	if err := db.Exec("UPDATE audit_logs SET tenant_id = ? WHERE id = ?", "tenant-B", "t2-1").Error; err != nil {
		t.Fatalf("set tenant-B: %v", err)
	}

	report, err := GenerateComplianceReport(db, "tenant-A", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("GenerateComplianceReport: %v", err)
	}
	if report.TotalRecords != 1 {
		t.Errorf("TotalRecords = %d, want 1 (only tenant-A)", report.TotalRecords)
	}
}

// TestGenerateComplianceReport_TimeRange verifies the start/end filters.
func TestGenerateComplianceReport_TimeRange(t *testing.T) {
	db := newAuditDB(t)
	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	new := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	seedAuditLog(t, db, model.AuditLog{ID: "old", Action: "x", LogType: "auth", CreatedAt: old})
	seedAuditLog(t, db, model.AuditLog{ID: "mid", Action: "x", LogType: "auth", CreatedAt: mid})
	seedAuditLog(t, db, model.AuditLog{ID: "new", Action: "x", LogType: "auth", CreatedAt: new})

	report, err := GenerateComplianceReport(db, "",
		time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("GenerateComplianceReport: %v", err)
	}
	if report.TotalRecords != 1 {
		t.Errorf("TotalRecords = %d, want 1 (only mid)", report.TotalRecords)
	}
}

// TestMarshalComplianceReportJSON checks the JSON serialization.
func TestMarshalComplianceReportJSON(t *testing.T) {
	report := ComplianceReport{
		TenantID:    "t-1",
		PeriodStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		TotalRecords: 5,
		RecordsByType: map[string]int64{
			"auth": 3, "operation": 2,
		},
		UniqueUsers:   2,
		ChainVerified: true,
	}
	data, err := MarshalComplianceReportJSON(report)
	if err != nil {
		t.Fatalf("MarshalComplianceReportJSON: %v", err)
	}
	var round ComplianceReport
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round.TenantID != "t-1" || round.TotalRecords != 5 || round.UniqueUsers != 2 {
		t.Errorf("round-trip mismatch: %+v", round)
	}
	if round.RecordsByType["auth"] != 3 || round.RecordsByType["operation"] != 2 {
		t.Errorf("RecordsByType mismatch: %+v", round.RecordsByType)
	}
}

// ---------- ExportCSV / ExportJSON ----------

// TestExportCSV_HeaderAndRows checks the CSV has a stable header and
// the correct number of rows for N log records.
func TestExportCSV_HeaderAndRows(t *testing.T) {
	db := newAuditDB(t)
	now := time.Now().UTC()
	seedAuditLog(t, db, model.AuditLog{
		ID: "a-1", UserID: "u-1", Username: "alice", Action: "user.login",
		LogType: "auth", IPAddress: "10.0.0.1", UserAgent: "ua",
		RecordHash: "abc", CreatedAt: now,
	})
	seedAuditLog(t, db, model.AuditLog{
		ID: "a-2", UserID: "u-2", Username: "bob", Action: "user.logout",
		LogType: "auth", IPAddress: "10.0.0.2", UserAgent: "ua",
		RecordHash: "def", CreatedAt: now,
	})

	reader, err := ExportCSV(db, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	rows, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		t.Fatalf("csv read: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (1 header + 2 data), got %d", len(rows))
	}
	header := rows[0]
	for _, want := range []string{"ID", "UserID", "Username", "Action", "RecordHash", "HashVerified", "ChainHashValid"} {
		found := false
		for _, h := range header {
			if h == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("CSV header missing %q in %v", want, header)
		}
	}
	// First data row should be the alphabetically-first ID ("a-1").
	if rows[1][0] != "a-1" {
		t.Errorf("first data row ID = %q, want a-1 (sorted by id ASC)", rows[1][0])
	}
	// HashVerified should be "true" when RecordHash is set.
	if rows[1][11] != "true" {
		t.Errorf("HashVerified for a-1 = %q, want true", rows[1][11])
	}
	// ChainHashValid should be "false" when there is no AuditHash row.
	if rows[1][12] != "false" {
		t.Errorf("ChainHashValid for a-1 = %q, want false (no chain hash)", rows[1][12])
	}
}

// TestExportCSV_ChainHashValid confirms that AuditHash entries flip the
// ChainHashValid flag.
func TestExportCSV_ChainHashValid(t *testing.T) {
	db := newAuditDB(t)
	now := time.Now().UTC()
	seedAuditLog(t, db, model.AuditLog{
		ID: "a-1", UserID: "u-1", Action: "x", LogType: "auth",
		RecordHash: "abc", CreatedAt: now,
	})
	if err := db.Create(&model.AuditHash{AuditID: "a-1", Hash: "h1"}).Error; err != nil {
		t.Fatalf("create hash: %v", err)
	}

	reader, err := ExportCSV(db, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	rows, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		t.Fatalf("csv read: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[1][12] != "true" {
		t.Errorf("ChainHashValid = %q, want true", rows[1][12])
	}
}

// TestExportCSV_FilterByTenant ensures the tenant filter is honored.
func TestExportCSV_FilterByTenant(t *testing.T) {
	db := newAuditDB(t)
	seedAuditLog(t, db, model.AuditLog{ID: "t1", Action: "x", LogType: "auth"})
	seedAuditLog(t, db, model.AuditLog{ID: "t2", Action: "x", LogType: "auth"})
	if err := db.Exec("UPDATE audit_logs SET tenant_id = ? WHERE id = ?", "tenant-A", "t1").Error; err != nil {
		t.Fatalf("set tenant-A: %v", err)
	}
	if err := db.Exec("UPDATE audit_logs SET tenant_id = ? WHERE id = ?", "tenant-B", "t2").Error; err != nil {
		t.Fatalf("set tenant-B: %v", err)
	}

	reader, err := ExportCSV(db, "tenant-B", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	rows, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		t.Fatalf("csv read: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (1 header + 1 data), got %d", len(rows))
	}
	if rows[1][0] != "t2" {
		t.Errorf("exported id = %q, want t2 (tenant-B only)", rows[1][0])
	}
}

// TestExportJSON_ValidShape confirms the JSON output deserializes into
// the documented record type and contains the expected fields.
func TestExportJSON_ValidShape(t *testing.T) {
	db := newAuditDB(t)
	now := time.Now().UTC()
	seedAuditLog(t, db, model.AuditLog{
		ID: "a-1", UserID: "u-1", Username: "alice", Action: "user.login",
		LogType: "auth", IPAddress: "10.0.0.1", UserAgent: "ua",
		RecordHash: "abc", CreatedAt: now,
	})
	reader, err := ExportJSON(db, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ExportJSON: %v", err)
	}
	var records []AuditExportRecord
	if err := json.NewDecoder(reader).Decode(&records); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.ID != "a-1" || r.Username != "alice" || r.Action != "user.login" {
		t.Errorf("record content mismatch: %+v", r)
	}
	if !r.HashVerified {
		t.Error("HashVerified should be true (RecordHash is set)")
	}
	if r.ChainHashValid {
		t.Error("ChainHashValid should be false (no AuditHash row)")
	}
	if r.CreatedAt == "" {
		t.Error("CreatedAt should be populated as RFC3339 string")
	}
}

// TestExportCSV_EmptyResultReturnsHeaderOnly ensures an empty result
// set still produces a valid (header-only) CSV.
func TestExportCSV_EmptyResultReturnsHeaderOnly(t *testing.T) {
	db := newAuditDB(t)
	reader, err := ExportCSV(db, "no-such-tenant", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	rows, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		t.Fatalf("csv read: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row (header only), got %d", len(rows))
	}
}

// ---------- HTTP layer sanity (using httptest) ----------

// TestHTTPRoundTrip_SampleRequest is a small end-to-end smoke check:
// the audit package's exported types are JSON-serializable and can
// survive a round trip through an HTTP handler. This is a regression
// guard against accidental field-tag or type breakage.
func TestHTTPRoundTrip_SampleRequest(t *testing.T) {
	// A trivial handler that emits a ComplianceReport and decodes it
	// back on the client.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rep := ComplianceReport{
			TenantID:      "t-1",
			TotalRecords:  1,
			RecordsByType: map[string]int64{"auth": 1},
		}
		_ = json.NewEncoder(w).Encode(rep)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	var got ComplianceReport
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.TenantID != "t-1" || got.TotalRecords != 1 {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

// TestExportUserData_DBErrorSurfaces wraps a simple negative test:
// closing the underlying DB and re-exporting must return an error,
// not silently succeed.
func TestExportUserData_DBErrorSurfaces(t *testing.T) {
	db := newAuditDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlDB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	_, err = ExportUserData(db, "u-x")
	if err == nil {
		t.Fatal("expected error when DB is closed, got nil")
	}
}

// reference the context import to silence "imported and not used" in
// case a future test extension chooses to use it.
var _ = context.Background
var _ = os.Create
var _ = filepath.Join
var _ = rsa.GenerateKey
var _ = rand.Reader
var _ = strings.Repeat
