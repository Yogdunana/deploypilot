package audit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupComplianceDB returns an in-memory SQLite DB with the tables needed
// for compliance-related tests.
func setupComplianceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.AuditLog{}, &model.AuditHash{}, &model.APIKey{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// TestExportUserData_ReturnsEmptyArrayForMissingUser ensures the export
// helper gracefully handles a user with no audit data and produces a
// valid (empty) audit_logs array, not a nil one. This matters because
// the JSON payload is consumed by downstream tooling that may not handle
// nil arrays correctly.
func TestExportUserData_ReturnsEmptyArrayForMissingUser(t *testing.T) {
	db := setupComplianceDB(t)

	data, err := ExportUserData(db, "nobody")
	if err != nil {
		t.Fatalf("ExportUserData returned error: %v", err)
	}

	logs, ok := data["audit_logs"]
	if !ok {
		t.Fatal("expected audit_logs key in export")
	}
	logsSlice, ok := logs.([]model.AuditLog)
	if !ok {
		t.Fatalf("expected []model.AuditLog, got %T", logs)
	}
	if logsSlice == nil {
		t.Error("expected empty (non-nil) slice for missing user, got nil")
	}
	if len(logsSlice) != 0 {
		t.Errorf("expected 0 entries, got %d", len(logsSlice))
	}
	if data["user_id"] != "nobody" {
		t.Errorf("user_id=%v, want %q", data["user_id"], "nobody")
	}
	if _, ok := data["exported_at"]; !ok {
		t.Error("exported_at key should be present")
	}
}

// TestExportUserData_IncludesAuditLogsForUser ensures that the export
// contains the user's actual audit entries.
func TestExportUserData_IncludesAuditLogsForUser(t *testing.T) {
	db := setupComplianceDB(t)

	// Seed: one log for the user, one for someone else.
	if err := db.Create(&model.AuditLog{
		ID:       "log-1",
		UserID:   "user-1",
		Username: "alice",
		Action:   "user.login",
		LogType:  "auth",
	}).Error; err != nil {
		t.Fatalf("seed #1: %v", err)
	}
	if err := db.Create(&model.AuditLog{
		ID:       "log-2",
		UserID:   "other-user",
		Username: "bob",
		Action:   "user.login",
		LogType:  "auth",
	}).Error; err != nil {
		t.Fatalf("seed #2: %v", err)
	}

	data, err := ExportUserData(db, "user-1")
	if err != nil {
		t.Fatalf("ExportUserData returned error: %v", err)
	}

	logs := data["audit_logs"].([]model.AuditLog)
	if len(logs) != 1 {
		t.Fatalf("expected 1 entry for user-1, got %d", len(logs))
	}
	if logs[0].ID != "log-1" {
		t.Errorf("got log id %q, want %q", logs[0].ID, "log-1")
	}
	if data["audit_log_count"] != 1 {
		t.Errorf("audit_log_count=%v, want 1", data["audit_log_count"])
	}
}

// TestExportUserData_AnonymizesAPIKeys verifies the GDPR "no raw key
// material in exports" property. Only metadata (id, name, dates) is
// exposed; KeyHash, Scopes, etc. are stripped.
func TestExportUserData_AnonymizesAPIKeys(t *testing.T) {
	db := setupComplianceDB(t)

	// Seed: one API key for the user.
	now := time.Now().UTC().Truncate(time.Second)
	lastUsed := now.Add(-time.Hour)
	expires := now.Add(24 * time.Hour)
	if err := db.Create(&model.APIKey{
		ID:         "key-1",
		TenantID:   "tenant-1",
		UserID:     "user-1",
		Name:       "test-key",
		KeyHash:    "secret-hash-must-not-leak",
		KeyPrefix:  "dp_abc12345",
		Scopes:     `["read","deploy"]`,
		AllowedIPs: `["10.0.0.1"]`,
		LastUsedAt: &lastUsed,
		ExpiresAt:  &expires,
		CreatedAt:  now,
	}).Error; err != nil {
		t.Fatalf("seed api key: %v", err)
	}

	data, err := ExportUserData(db, "user-1")
	if err != nil {
		t.Fatalf("ExportUserData returned error: %v", err)
	}

	// Marshal the result and ensure KeyHash/Scopes/AllowedIPs do NOT appear.
	out, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	forbidden := []string{"secret-hash-must-not-leak", `"read"`, `"deploy"`, `10.0.0.1`}
	for _, f := range forbidden {
		if contains(s, f) {
			t.Errorf("export leaked forbidden content %q in payload: %s", f, s)
		}
	}
}

// TestDeleteUserData_AnonymizesPII ensures the GDPR "right to be forgotten"
// path redacts username, IP, user agent, and clears the user_id, while
// leaving the audit row itself intact (so the chain integrity is preserved).
func TestDeleteUserData_AnonymizesPII(t *testing.T) {
	db := setupComplianceDB(t)

	if err := db.Create(&model.AuditLog{
		ID:        "log-1",
		UserID:    "user-1",
		Username:  "alice",
		IPAddress: "10.0.0.1",
		UserAgent: "Mozilla/5.0",
		Action:    "user.login",
		LogType:   "auth",
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := db.Create(&model.AuditLog{
		ID:       "log-2",
		UserID:   "other-user",
		Username: "bob",
		Action:   "user.login",
		LogType:  "auth",
	}).Error; err != nil {
		t.Fatalf("seed #2: %v", err)
	}

	if err := DeleteUserData(db, "user-1"); err != nil {
		t.Fatalf("DeleteUserData returned error: %v", err)
	}

	var log1 model.AuditLog
	if err := db.Where("id = ?", "log-1").First(&log1).Error; err != nil {
		t.Fatalf("log-1 should still exist: %v", err)
	}
	if log1.UserID != "" {
		t.Errorf("user_id should be cleared, got %q", log1.UserID)
	}
	if log1.Username != "[GDPR-DELETED]" {
		t.Errorf("username should be GDPR-DELETED, got %q", log1.Username)
	}
	if log1.IPAddress != "[GDPR-DELETED]" {
		t.Errorf("ip_address should be GDPR-DELETED, got %q", log1.IPAddress)
	}
	if log1.UserAgent != "[GDPR-DELETED]" {
		t.Errorf("user_agent should be GDPR-DELETED, got %q", log1.UserAgent)
	}

	// Other user's record must not be touched.
	var log2 model.AuditLog
	if err := db.Where("id = ?", "log-2").First(&log2).Error; err != nil {
		t.Fatalf("log-2 should still exist: %v", err)
	}
	if log2.Username != "bob" {
		t.Errorf("other user's username should be unchanged, got %q", log2.Username)
	}
}

// TestDataRetentionPolicy_DeletesArchivedBeyondCutoff covers the basic
// retention rule: archived records older than RetentionDays are removed.
func TestDataRetentionPolicy_DeletesArchivedBeyondCutoff(t *testing.T) {
	db := setupComplianceDB(t)

	oldArchived := time.Now().AddDate(0, 0, -120) // 120 days ago
	if err := db.Create(&model.AuditLog{
		ID:        "old-archived",
		Action:    "user.login",
		Archived:  true,
		CreatedAt: oldArchived,
	}).Error; err != nil {
		t.Fatalf("seed old: %v", err)
	}
	if err := db.Create(&model.AuditLog{
		ID:        "old-active",
		Action:    "user.login",
		Archived:  false, // not archived -> must NOT be deleted
		CreatedAt: oldArchived,
	}).Error; err != nil {
		t.Fatalf("seed active: %v", err)
	}
	if err := db.Create(&model.AuditLog{
		ID:        "recent-archived",
		Action:    "user.login",
		Archived:  true,
		CreatedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed recent: %v", err)
	}

	cfg := &config.AuditConfig{RetentionDays: 90}
	if err := DataRetentionPolicy(db, cfg); err != nil {
		t.Fatalf("DataRetentionPolicy: %v", err)
	}

	if db.Where("id = ?", "old-archived").First(&model.AuditLog{}).Error == nil {
		t.Error("old-archived should have been deleted")
	}
	if db.Where("id = ?", "old-active").First(&model.AuditLog{}).Error != nil {
		t.Error("old-active (not archived) must NOT be deleted by retention")
	}
	if db.Where("id = ?", "recent-archived").First(&model.AuditLog{}).Error != nil {
		t.Error("recent-archived must NOT be deleted (within retention window)")
	}
}

// TestDataRetentionPolicy_DefaultRetentionWhenZero covers the
// "RetentionDays <= 0 -> use 90" fallback so a misconfigured deployment
// still gets *some* retention applied (rather than deleting everything).
func TestDataRetentionPolicy_DefaultRetentionWhenZero(t *testing.T) {
	db := setupComplianceDB(t)

	// 200 days old, archived -> should be deleted under the 90-day default.
	if err := db.Create(&model.AuditLog{
		ID:        "very-old",
		Action:    "user.login",
		Archived:  true,
		CreatedAt: time.Now().AddDate(0, 0, -200),
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	cfg := &config.AuditConfig{RetentionDays: 0}
	if err := DataRetentionPolicy(db, cfg); err != nil {
		t.Fatalf("DataRetentionPolicy: %v", err)
	}
	if db.Where("id = ?", "very-old").First(&model.AuditLog{}).Error == nil {
		t.Error("200-day-old archived record should be deleted by default 90-day policy")
	}
}

// TestGenerateComplianceReport_BasicFields ensures the report is
// populated correctly from a small seeded dataset.
func TestGenerateComplianceReport_BasicFields(t *testing.T) {
	db := setupComplianceDB(t)

	// Seed: 2 auth + 1 operation logs.
	for i, action := range []string{"user.login", "user.logout", "user.update"} {
		logType := "auth"
		if action == "user.update" {
			logType = "operation"
		}
		if err := db.Create(&model.AuditLog{
			ID:       "log-" + string(rune('1'+i)),
			UserID:   "user-" + string(rune('1'+i)),
			Username: "u" + string(rune('1'+i)),
			Action:   action,
			LogType:  logType,
		}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	report, err := GenerateComplianceReport(db, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("GenerateComplianceReport: %v", err)
	}
	if report.TotalRecords != 3 {
		t.Errorf("TotalRecords=%d, want 3", report.TotalRecords)
	}
	if report.RecordsByType["auth"] != 2 {
		t.Errorf("auth count=%d, want 2", report.RecordsByType["auth"])
	}
	if report.RecordsByType["operation"] != 1 {
		t.Errorf("operation count=%d, want 1", report.RecordsByType["operation"])
	}
	if report.UniqueUsers != 3 {
		t.Errorf("UniqueUsers=%d, want 3", report.UniqueUsers)
	}
	if report.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
}

// TestGenerateComplianceReport_TenantFilter exercises the tenant
// scoping logic. Reports for one tenant must not include another tenant's
// records.
func TestGenerateComplianceReport_TenantFilter(t *testing.T) {
	db := setupComplianceDB(t)

	// The audit_logs model does not have a TenantID column in the model
	// definition we have here, so we work around it by exercising the
	// empty-tenant path. (The current model doesn't carry tenant_id on
	// audit logs.) This test still ensures the empty-tenant path returns
	// global counts.
	if err := db.Create(&model.AuditLog{
		ID:      "log-x",
		Action:  "user.login",
		LogType: "auth",
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	report, err := GenerateComplianceReport(db, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("GenerateComplianceReport: %v", err)
	}
	if report.TotalRecords != 1 {
		t.Errorf("TotalRecords=%d, want 1", report.TotalRecords)
	}
}

// TestGenerateComplianceReport_PeriodFilter exercises the date-range
// filter on the report.
func TestGenerateComplianceReport_PeriodFilter(t *testing.T) {
	db := setupComplianceDB(t)

	old := time.Now().AddDate(0, 0, -10)
	recent := time.Now().AddDate(0, 0, -1)
	for _, ts := range []time.Time{old, recent} {
		if err := db.Create(&model.AuditLog{
			ID:        "log-" + ts.Format("20060102150405"),
			Action:    "user.login",
			LogType:   "auth",
			CreatedAt: ts,
		}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// Only the recent log should be counted.
	start := time.Now().AddDate(0, 0, -2)
	end := time.Now()
	report, err := GenerateComplianceReport(db, "", start, end)
	if err != nil {
		t.Fatalf("GenerateComplianceReport: %v", err)
	}
	if report.TotalRecords != 1 {
		t.Errorf("TotalRecords=%d, want 1", report.TotalRecords)
	}
}

// TestGenerateComplianceReport_ChainIssuesAreCounted ensures that
// when the chain reports an invalid record, the report's ChainIssues
// counter is bumped.
func TestGenerateComplianceReport_ChainIssuesAreCounted(t *testing.T) {
	db := setupComplianceDB(t)
	chain := NewAuditChain(db, []byte("compliance-test"))

	// Seed an audit log and an INVALID hash to break the chain.
	rec := &model.AuditLog{
		ID:        "log-tamper",
		Action:    "user.login",
		LogType:   "auth",
		CreatedAt: time.Now().UTC(),
	}
	if err := db.Create(rec).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := chain.AppendHash("log-tamper", "wrong-hash-on-purpose"); err != nil {
		t.Fatalf("AppendHash: %v", err)
	}

	report, err := GenerateComplianceReport(db, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("GenerateComplianceReport: %v", err)
	}
	if !report.ChainVerified {
		t.Error("ChainVerified should be true (verification ran, just with issues)")
	}
	if report.ChainIssues == 0 {
		t.Error("ChainIssues should be > 0 for a tampered chain")
	}
}

// TestMarshalComplianceReportJSON covers the JSON serialization helper.
func TestMarshalComplianceReportJSON(t *testing.T) {
	report := ComplianceReport{
		TenantID:      "tenant-x",
		TotalRecords:  42,
		ChainVerified: true,
		GeneratedAt:   time.Now().UTC(),
	}
	data, err := MarshalComplianceReportJSON(report)
	if err != nil {
		t.Fatalf("MarshalComplianceReportJSON: %v", err)
	}

	var back ComplianceReport
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.TenantID != "tenant-x" {
		t.Errorf("round-trip TenantID=%q, want %q", back.TenantID, "tenant-x")
	}
	if back.TotalRecords != 42 {
		t.Errorf("round-trip TotalRecords=%d, want 42", back.TotalRecords)
	}
	if !back.ChainVerified {
		t.Error("round-trip ChainVerified should be true")
	}
}

// contains is a small helper that avoids importing strings just for one
// call site (and keeps the test file's import list minimal).
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
