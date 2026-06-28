package audit

import (
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupComplianceTestDB(t *testing.T) *gorm.DB {
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

func TestExportUserData(t *testing.T) {
	db := setupComplianceTestDB(t)

	userID := "user-123"
	now := time.Now().UTC()

	err := db.Create(&model.AuditLog{
		ID:         "audit-1",
		UserID:     userID,
		Username:   "testuser",
		Action:     "user.login",
		LogType:    "auth",
		IPAddress:  "192.168.1.1",
		UserAgent:  "test-agent",
		CreatedAt:  now,
	}).Error
	if err != nil {
		t.Fatalf("failed to create audit log: %v", err)
	}

	result, err := ExportUserData(db, userID)
	if err != nil {
		t.Fatalf("ExportUserData failed: %v", err)
	}

	if result["user_id"] != userID {
		t.Errorf("expected user_id %q, got %v", userID, result["user_id"])
	}

	auditLogs, ok := result["audit_logs"].([]model.AuditLog)
	if !ok {
		t.Fatal("expected audit_logs to be []model.AuditLog")
	}
	if len(auditLogs) != 1 {
		t.Errorf("expected 1 audit log, got %d", len(auditLogs))
	}
	if auditLogs[0].ID != "audit-1" {
		t.Errorf("expected audit ID 'audit-1', got %q", auditLogs[0].ID)
	}
}

func TestExportUserData_NoLogs(t *testing.T) {
	db := setupComplianceTestDB(t)

	result, err := ExportUserData(db, "nonexistent-user")
	if err != nil {
		t.Fatalf("ExportUserData failed: %v", err)
	}

	auditLogs, ok := result["audit_logs"].([]model.AuditLog)
	if !ok {
		t.Fatal("expected audit_logs to be []model.AuditLog")
	}
	if len(auditLogs) != 0 {
		t.Errorf("expected 0 audit logs, got %d", len(auditLogs))
	}
}

func TestDeleteUserData(t *testing.T) {
	db := setupComplianceTestDB(t)

	userID := "user-123"
	err := db.Create(&model.AuditLog{
		ID:         "audit-1",
		UserID:     userID,
		Username:   "testuser",
		Action:     "user.login",
		IPAddress:  "192.168.1.1",
		UserAgent:  "test-agent",
		CreatedAt:  time.Now().UTC(),
	}).Error
	if err != nil {
		t.Fatalf("failed to create audit log: %v", err)
	}

	err = DeleteUserData(db, userID)
	if err != nil {
		t.Fatalf("DeleteUserData failed: %v", err)
	}

	var log model.AuditLog
	err = db.Where("id = ?", "audit-1").First(&log).Error
	if err != nil {
		t.Fatalf("failed to find audit log: %v", err)
	}

	if log.Username != "[GDPR-DELETED]" {
		t.Errorf("expected username '[GDPR-DELETED]', got %q", log.Username)
	}
	if log.IPAddress != "[GDPR-DELETED]" {
		t.Errorf("expected ip_address '[GDPR-DELETED]', got %q", log.IPAddress)
	}
	if log.UserAgent != "[GDPR-DELETED]" {
		t.Errorf("expected user_agent '[GDPR-DELETED]', got %q", log.UserAgent)
	}
	if log.UserID != "" {
		t.Errorf("expected user_id to be empty, got %q", log.UserID)
	}
}

func TestDataRetentionPolicy(t *testing.T) {
	db := setupComplianceTestDB(t)

	now := time.Now().UTC()

	err := db.Create(&model.AuditLog{
		ID:         "audit-old",
		UserID:     "user-1",
		Action:     "test.action",
		Archived:   true,
		CreatedAt:  now.AddDate(0, 0, -100),
	}).Error
	if err != nil {
		t.Fatalf("failed to create old audit log: %v", err)
	}

	err = db.Create(&model.AuditLog{
		ID:         "audit-recent",
		UserID:     "user-1",
		Action:     "test.action",
		Archived:   true,
		CreatedAt:  now.AddDate(0, 0, -10),
	}).Error
	if err != nil {
		t.Fatalf("failed to create recent audit log: %v", err)
	}

	err = db.Create(&model.AuditLog{
		ID:         "audit-unarchived",
		UserID:     "user-1",
		Action:     "test.action",
		Archived:   false,
		CreatedAt:  now.AddDate(0, 0, -100),
	}).Error
	if err != nil {
		t.Fatalf("failed to create unarchived audit log: %v", err)
	}

	cfg := &config.AuditConfig{RetentionDays: 90}
	err = DataRetentionPolicy(db, cfg)
	if err != nil {
		t.Fatalf("DataRetentionPolicy failed: %v", err)
	}

	var count int64
	err = db.Model(&model.AuditLog{}).Count(&count).Error
	if err != nil {
		t.Fatalf("failed to count audit logs: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 remaining logs (recent + unarchived), got %d", count)
	}

	var exists bool
	err = db.Model(&model.AuditLog{}).Select("count(*) > 0").Where("id = ?", "audit-old").Scan(&exists).Error
	if err != nil {
		t.Fatalf("failed to check old log: %v", err)
	}
	if exists {
		t.Error("expected old archived log to be deleted")
	}

	err = db.Model(&model.AuditLog{}).Select("count(*) > 0").Where("id = ?", "audit-recent").Scan(&exists).Error
	if err != nil {
		t.Fatalf("failed to check recent log: %v", err)
	}
	if !exists {
		t.Error("expected recent archived log to still exist")
	}
}

func TestDataRetentionPolicy_DefaultRetention(t *testing.T) {
	db := setupComplianceTestDB(t)

	cfg := &config.AuditConfig{RetentionDays: 0}
	err := DataRetentionPolicy(db, cfg)
	if err != nil {
		t.Fatalf("DataRetentionPolicy with default failed: %v", err)
	}
}

func TestGenerateComplianceReport(t *testing.T) {
	db := setupComplianceTestDB(t)

	now := time.Now().UTC()
	startTime := now.Add(-7 * 24 * time.Hour)
	endTime := now

	err := db.Create(&model.AuditLog{
		ID:         "audit-1",
		UserID:     "user-1",
		Action:     "user.login",
		LogType:    "auth",
		CreatedAt:  now.Add(-24 * time.Hour),
	}).Error
	if err != nil {
		t.Fatalf("failed to create audit log: %v", err)
	}

	err = db.Create(&model.AuditLog{
		ID:         "audit-2",
		UserID:     "user-2",
		Action:     "deploy.create",
		LogType:    "operation",
		CreatedAt:  now.Add(-48 * time.Hour),
	}).Error
	if err != nil {
		t.Fatalf("failed to create audit log: %v", err)
	}

	report, err := GenerateComplianceReport(db, "", startTime, endTime)
	if err != nil {
		t.Fatalf("GenerateComplianceReport failed: %v", err)
	}

	if report.TotalRecords != 2 {
		t.Errorf("expected total records 2, got %d", report.TotalRecords)
	}
	if report.UniqueUsers != 2 {
		t.Errorf("expected unique users 2, got %d", report.UniqueUsers)
	}
	if report.RecordsByType["auth"] != 1 {
		t.Errorf("expected auth records 1, got %d", report.RecordsByType["auth"])
	}
	if report.RecordsByType["operation"] != 1 {
		t.Errorf("expected operation records 1, got %d", report.RecordsByType["operation"])
	}
}

func TestGenerateComplianceReport_NoLogs(t *testing.T) {
	db := setupComplianceTestDB(t)

	now := time.Now().UTC()
	startTime := now.Add(-7 * 24 * time.Hour)
	endTime := now

	report, err := GenerateComplianceReport(db, "", startTime, endTime)
	if err != nil {
		t.Fatalf("GenerateComplianceReport failed: %v", err)
	}

	if report.TotalRecords != 0 {
		t.Errorf("expected total records 0, got %d", report.TotalRecords)
	}
}

func TestMarshalComplianceReportJSON(t *testing.T) {
	report := ComplianceReport{
		TenantID:    "tenant-1",
		PeriodStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		TotalRecords: 100,
		RecordsByType: map[string]int64{
			"auth":      20,
			"operation": 80,
		},
		ChainVerified: true,
		GeneratedAt:   time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
	}

	data, err := MarshalComplianceReportJSON(report)
	if err != nil {
		t.Fatalf("MarshalComplianceReportJSON failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}