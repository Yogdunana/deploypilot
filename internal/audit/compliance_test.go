package audit

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupComplianceDB creates an in-memory DB with all the tables used by the
// audit compliance/export code, and seeds one audit log.
func setupComplianceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AuditLog{}, &model.AuditHash{}, &model.APIKey{}, &model.User{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	log := &model.AuditLog{
		ID:           "audit-1",
		UserID:       "user-1",
		Username:     "alice",
		Action:       "user.login",
		ResourceType: "session",
		LogType:      "auth",
		IPAddress:    "10.0.0.1",
		UserAgent:    "ua-1",
		CreatedAt:    time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	if err := db.Create(log).Error; err != nil {
		t.Fatalf("failed to insert log: %v", err)
	}
	return db
}

func TestExportUserData_ReturnsUserMetadata(t *testing.T) {
	db := setupComplianceDB(t)

	data, err := ExportUserData(db, "user-1")
	if err != nil {
		t.Fatalf("ExportUserData() error = %v", err)
	}
	if data["user_id"] != "user-1" {
		t.Errorf("user_id = %v, want user-1", data["user_id"])
	}
	if _, ok := data["exported_at"]; !ok {
		t.Error("exported_at field should be present")
	}

	// audit_log_count should match inserted logs.
	if count, _ := data["audit_log_count"].(int); count != 1 {
		t.Errorf("audit_log_count = %v, want 1", data["audit_log_count"])
	}

	// audit_logs should be a list.
	logs, ok := data["audit_logs"].([]model.AuditLog)
	if !ok {
		t.Fatalf("audit_logs type = %T, want []model.AuditLog", data["audit_logs"])
	}
	if len(logs) != 1 || logs[0].ID != "audit-1" {
		t.Errorf("audit_logs = %v, want one entry audit-1", logs)
	}
}

func TestExportUserData_NoLogs(t *testing.T) {
	db := setupComplianceDB(t)

	data, err := ExportUserData(db, "no-such-user")
	if err != nil {
		t.Fatalf("ExportUserData() error = %v", err)
	}
	count, _ := data["audit_log_count"].(int)
	if count != 0 {
		t.Errorf("audit_log_count for missing user = %d, want 0", count)
	}
	logs, ok := data["audit_logs"].([]model.AuditLog)
	if !ok {
		t.Fatalf("audit_logs type = %T, want []model.AuditLog", data["audit_logs"])
	}
	if len(logs) != 0 {
		t.Errorf("audit_logs for missing user = %v, want []", logs)
	}
}

func TestDeleteUserData_AnonymizesAuditLogs(t *testing.T) {
	db := setupComplianceDB(t)

	if err := DeleteUserData(db, "user-1"); err != nil {
		t.Fatalf("DeleteUserData() error = %v", err)
	}

	var got model.AuditLog
	if err := db.First(&got, "id = ?", "audit-1").Error; err != nil {
		t.Fatalf("failed to reload log: %v", err)
	}
	if got.Username != "[GDPR-DELETED]" {
		t.Errorf("Username = %q, want [GDPR-DELETED]", got.Username)
	}
	if got.IPAddress != "[GDPR-DELETED]" {
		t.Errorf("IPAddress = %q, want [GDPR-DELETED]", got.IPAddress)
	}
	if got.UserAgent != "[GDPR-DELETED]" {
		t.Errorf("UserAgent = %q, want [GDPR-DELETED]", got.UserAgent)
	}
	if got.UserID != "" {
		t.Errorf("UserID = %q, want empty after GDPR delete", got.UserID)
	}
	// Action should be preserved (audit trail integrity).
	if got.Action != "user.login" {
		t.Errorf("Action = %q, want preserved user.login", got.Action)
	}
}

func TestDeleteUserData_NoMatchIsNoop(t *testing.T) {
	db := setupComplianceDB(t)

	if err := DeleteUserData(db, "no-such-user"); err != nil {
		t.Errorf("DeleteUserData() for missing user returned error: %v", err)
	}
}

func TestDataRetentionPolicy_DeletesArchivedBeyondRetention(t *testing.T) {
	db := setupComplianceDB(t)

	old := &model.AuditLog{
		ID:        "old-archived",
		Action:    "user.login",
		Archived:  true,
		CreatedAt: time.Now().AddDate(0, 0, -200),
	}
	if err := db.Create(old).Error; err != nil {
		t.Fatalf("failed to insert old log: %v", err)
	}
	recent := &model.AuditLog{
		ID:        "recent-archived",
		Action:    "user.login",
		Archived:  true,
		CreatedAt: time.Now().AddDate(0, 0, -10),
	}
	if err := db.Create(recent).Error; err != nil {
		t.Fatalf("failed to insert recent log: %v", err)
	}
	// Old active log must NEVER be deleted (append-only protection).
	oldActive := &model.AuditLog{
		ID:        "old-active",
		Action:    "user.login",
		Archived:  false,
		CreatedAt: time.Now().AddDate(0, 0, -200),
	}
	if err := db.Create(oldActive).Error; err != nil {
		t.Fatalf("failed to insert old active log: %v", err)
	}

	cfg := &config.AuditConfig{RetentionDays: 90}
	if err := DataRetentionPolicy(db, cfg); err != nil {
		t.Fatalf("DataRetentionPolicy() error = %v", err)
	}

	var cnt int64
	if err := db.Model(&model.AuditLog{}).Where("id = ?", "old-archived").Count(&cnt).Error; err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if cnt != 0 {
		t.Errorf("old-archived should be deleted, count = %d", cnt)
	}

	if err := db.Model(&model.AuditLog{}).Where("id = ?", "recent-archived").Count(&cnt).Error; err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if cnt != 1 {
		t.Errorf("recent-archived should be kept, count = %d", cnt)
	}

	if err := db.Model(&model.AuditLog{}).Where("id = ?", "old-active").Count(&cnt).Error; err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if cnt != 1 {
		t.Errorf("old-active should be kept (append-only), count = %d", cnt)
	}
}

func TestDataRetentionPolicy_DefaultRetention(t *testing.T) {
	db := setupComplianceDB(t)

	old := &model.AuditLog{
		ID:        "very-old",
		Action:    "user.login",
		Archived:  true,
		CreatedAt: time.Now().AddDate(-1, 0, 0),
	}
	if err := db.Create(old).Error; err != nil {
		t.Fatalf("failed to insert log: %v", err)
	}

	cfg := &config.AuditConfig{RetentionDays: 0} // triggers default 90
	if err := DataRetentionPolicy(db, cfg); err != nil {
		t.Fatalf("DataRetentionPolicy() error = %v", err)
	}
	var cnt int64
	db.Model(&model.AuditLog{}).Where("id = ?", "very-old").Count(&cnt)
	if cnt != 0 {
		t.Errorf("very-old log should be deleted with default 90-day retention, count = %d", cnt)
	}
}

func TestGenerateComplianceReport_BasicCounts(t *testing.T) {
	db := setupComplianceDB(t)

	extra := &model.AuditLog{
		ID:        "audit-2",
		UserID:    "user-2",
		Username:  "bob",
		Action:    "deploy.start",
		LogType:   "operation",
		CreatedAt: time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
	}
	if err := db.Create(extra).Error; err != nil {
		t.Fatalf("failed to insert log: %v", err)
	}

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	report, err := GenerateComplianceReport(db, "", start, end)
	if err != nil {
		t.Fatalf("GenerateComplianceReport() error = %v", err)
	}
	if report.TotalRecords != 2 {
		t.Errorf("TotalRecords = %d, want 2", report.TotalRecords)
	}
	if got := report.RecordsByType["auth"]; got != 1 {
		t.Errorf("RecordsByType[auth] = %d, want 1", got)
	}
	if got := report.RecordsByType["operation"]; got != 1 {
		t.Errorf("RecordsByType[operation] = %d, want 1", got)
	}
	if report.UniqueUsers != 2 {
		t.Errorf("UniqueUsers = %d, want 2", report.UniqueUsers)
	}
	if report.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
}

func TestGenerateComplianceReport_RespectsTimeWindow(t *testing.T) {
	db := setupComplianceDB(t)

	futureStart := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	futureEnd := time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC)
	report, err := GenerateComplianceReport(db, "", futureStart, futureEnd)
	if err != nil {
		t.Fatalf("GenerateComplianceReport() error = %v", err)
	}
	if report.TotalRecords != 0 {
		t.Errorf("TotalRecords in future window = %d, want 0", report.TotalRecords)
	}
}

func TestMarshalComplianceReportJSON(t *testing.T) {
	report := ComplianceReport{
		TenantID:      "t-1",
		TotalRecords:  3,
		UniqueUsers:   2,
		ChainVerified: true,
		GeneratedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	data, err := MarshalComplianceReportJSON(report)
	if err != nil {
		t.Fatalf("MarshalComplianceReportJSON() error = %v", err)
	}
	if !json.Valid(data) {
		t.Errorf("output is not valid JSON: %s", string(data))
	}
	var got ComplianceReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.TenantID != "t-1" || got.TotalRecords != 3 || got.UniqueUsers != 2 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

// =====================================================================
// Tests for export.go (CSV and JSON exporters)
// =====================================================================

func TestExportCSV_ContainsHeaderAndRow(t *testing.T) {
	db := setupComplianceDB(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	reader, err := ExportCSV(db, "", start, end)
	if err != nil {
		t.Fatalf("ExportCSV() error = %v", err)
	}

	r := csv.NewReader(reader)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv.ReadAll() error = %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("expected at least 2 records (header + 1 row), got %d", len(records))
	}

	header := records[0]
	wantCols := []string{"ID", "UserID", "Action", "LogType", "HashVerified"}
	for _, w := range wantCols {
		if !containsStr(header, w) {
			t.Errorf("header missing %q: %v", w, header)
		}
	}

	row := records[1]
	if !containsStr(row, "audit-1") {
		t.Errorf("data row missing audit-1: %v", row)
	}
}

func TestExportCSV_EmptyResultReturnsHeaderOnly(t *testing.T) {
	db := setupComplianceDB(t)

	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2000, 12, 31, 0, 0, 0, 0, time.UTC)
	reader, err := ExportCSV(db, "", start, end)
	if err != nil {
		t.Fatalf("ExportCSV() error = %v", err)
	}
	r := csv.NewReader(reader)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv.ReadAll() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record (header only), got %d", len(records))
	}
}

func TestExportJSON_ValidJSONWithExpectedFields(t *testing.T) {
	db := setupComplianceDB(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	reader, err := ExportJSON(db, "", start, end)
	if err != nil {
		t.Fatalf("ExportJSON() error = %v", err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !json.Valid(body) {
		t.Fatalf("output is not valid JSON: %s", string(body))
	}
	var records []map[string]interface{}
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0]["id"] != "audit-1" {
		t.Errorf("record id = %v, want audit-1", records[0]["id"])
	}
	if _, ok := records[0]["hash_verified"]; !ok {
		t.Error("hash_verified field missing from exported record")
	}
	if _, ok := records[0]["chain_hash_valid"]; !ok {
		t.Error("chain_hash_valid field missing from exported record")
	}
}

func TestEnrichExportRecords_ComputesHashFlags(t *testing.T) {
	logs := []model.AuditLog{
		{ID: "with-hash", RecordHash: "abc123", CreatedAt: time.Now()},
		{ID: "without-hash", RecordHash: "", CreatedAt: time.Now()},
	}
	hashMap := map[string]model.AuditHash{
		"with-hash": {AuditID: "with-hash", Hash: "abc123"},
	}
	got := enrichExportRecords(logs, hashMap)
	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}
	if !got[0].HashVerified {
		t.Error("record with RecordHash should be HashVerified=true")
	}
	if !got[0].ChainHashValid {
		t.Error("record with matching AuditHash should be ChainHashValid=true")
	}
	if got[1].HashVerified {
		t.Error("record without RecordHash should be HashVerified=false")
	}
	if got[1].ChainHashValid {
		t.Error("record without AuditHash should be ChainHashValid=false")
	}
}

func containsStr(s []string, target string) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}
