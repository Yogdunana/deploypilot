package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS audit_logs (
		id TEXT PRIMARY KEY, user_id TEXT, username TEXT,
		action TEXT, resource_type TEXT, resource_id TEXT, detail TEXT,
		log_type TEXT DEFAULT 'operation',
		ip_address TEXT, user_agent TEXT, record_hash TEXT,
		trace_id TEXT,
		archived BOOLEAN DEFAULT false,
		archived_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	return db
}

func TestAuditService_Record(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	err := svc.Record(context.TODO(), AuditEntry{
		UserID:       "user-1",
		Username:     "testuser",
		Action:       "app.create",
		ResourceType: "app",
		ResourceID:   "app-123",
		Detail:       map[string]string{"name": "myapp"},
		IPAddress:    "192.168.1.1",
		UserAgent:    "test-agent",
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	logs, total, err := svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(logs) != 1 {
		t.Fatalf("len(logs) = %d, want 1", len(logs))
	}
	if logs[0].Username != "testuser" {
		t.Errorf("username = %q, want %q", logs[0].Username, "testuser")
	}
	if logs[0].Action != "app.create" {
		t.Errorf("action = %q, want %q", logs[0].Action, "app.create")
	}
	if logs[0].ResourceType != "app" {
		t.Errorf("resource_type = %q, want %q", logs[0].ResourceType, "app")
	}
	if logs[0].ResourceID != "app-123" {
		t.Errorf("resource_id = %q, want %q", logs[0].ResourceID, "app-123")
	}
	if logs[0].IPAddress != "192.168.1.1" {
		t.Errorf("ip_address = %q, want %q", logs[0].IPAddress, "192.168.1.1")
	}
	if logs[0].Detail == "" {
		t.Error("detail should not be empty")
	}
}

func TestAuditService_List(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	// Create multiple entries
	for i := uint(0); i < 5; i++ {
		_ = svc.Record(context.TODO(), AuditEntry{
			UserID:   fmt.Sprintf("user-%d", i),
			Username: "user",
			Action:   "app.create",
		})
	}

	logs, total, err := svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 3})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(logs) != 3 {
		t.Errorf("len(logs) = %d, want 3 (page size)", len(logs))
	}
}

func TestAuditService_ListWithFilter(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	_ = svc.Record(context.TODO(), AuditEntry{UserID: "user-1", Action: "app.create", ResourceType: "app"})
	_ = svc.Record(context.TODO(), AuditEntry{UserID: "user-2", Action: "server.create", ResourceType: "server"})
	_ = svc.Record(context.TODO(), AuditEntry{UserID: "user-1", Action: "app.delete", ResourceType: "app"})

	// Filter by user_id
	_, total, err := svc.List(context.TODO(), AuditFilter{UserID: "user-1", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 2 {
		t.Errorf("total for user 1 = %d, want 2", total)
	}

	// Filter by action
	logs, actionTotal, err := svc.List(context.TODO(), AuditFilter{Action: "server.create", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if actionTotal != 1 {
		t.Errorf("total for server.create = %d, want 1", actionTotal)
	}
	if len(logs) != 1 {
		t.Fatalf("len(logs) = %d, want 1", len(logs))
	}
	if logs[0].ResourceType != "server" {
		t.Errorf("resource_type = %q, want %q", logs[0].ResourceType, "server")
	}

	// Filter by resource_type
	_, total, err = svc.List(context.TODO(), AuditFilter{ResourceType: "server", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total for server resource = %d, want 1", total)
	}
}

func TestAuditService_ListPagination(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	for i := 0; i < 15; i++ {
		_ = svc.Record(context.TODO(), AuditEntry{Action: "app.create"})
	}

	// Page 1
	logs, total, _ := svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 10})
	if total != 15 {
		t.Errorf("total = %d, want 15", total)
	}
	if len(logs) != 10 {
		t.Errorf("page 1 len = %d, want 10", len(logs))
	}

	// Page 2
	logs, _, _ = svc.List(context.TODO(), AuditFilter{Page: 2, PageSize: 10})
	if len(logs) != 5 {
		t.Errorf("page 2 len = %d, want 5", len(logs))
	}
}

func TestAuditService_ListEmpty(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	logs, total, err := svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if logs == nil {
		t.Error("logs should be empty slice, not nil")
	}
}

func TestAuditService_RecordWithNilDetail(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	err := svc.Record(context.TODO(), AuditEntry{
		UserID: "1",
		Action: "app.create",
		Detail: nil,
	})
	if err != nil {
		t.Fatalf("Record() with nil detail error = %v", err)
	}

	logs, _, _ := svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 10})
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Detail != "" {
		t.Errorf("detail = %q, want empty string for nil detail", logs[0].Detail)
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		want          string
	}{
		{"x-forwarded-for", "192.168.1.1:12345", "10.0.0.1, 172.16.0.1", "10.0.0.1"},
		{"no forwarded", "192.168.1.1:12345", "", "192.168.1.1"},
		{"no forwarded no port", "192.168.1.1", "", "192.168.1.1"},
		{"invalid forwarded", "192.168.1.1:12345", "not-an-ip", "192.168.1.1"},
		{"empty remote", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClientIP(tt.remoteAddr, tt.xForwardedFor)
			if got != tt.want {
				t.Errorf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAuditLogTableName(t *testing.T) {
	log := model.AuditLog{}
	if log.TableName() != "audit_logs" {
		t.Errorf("TableName() = %q, want %q", log.TableName(), "audit_logs")
	}
}

func TestAuditService_ListPageSizeBounds(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	for i := 0; i < 5; i++ {
		_ = svc.Record(context.TODO(), AuditEntry{Action: "app.create"})
	}

	// pageSize 0 should default to 20
	logs, _, _ := svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 0})
	if len(logs) != 5 {
		t.Errorf("with pageSize=0, got %d logs, want 5", len(logs))
	}

	// pageSize > 100 should be capped to 100
	logs, _, _ = svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 200})
	if len(logs) != 5 {
		t.Errorf("with pageSize=200, got %d logs, want 5", len(logs))
	}

	// page 0 should default to 1
	logs, _, _ = svc.List(context.TODO(), AuditFilter{Page: 0, PageSize: 10})
	if len(logs) != 5 {
		t.Errorf("with page=0, got %d logs, want 5", len(logs))
	}
}

func TestAuditService_VerifyRecord(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	_ = svc.Record(context.TODO(), AuditEntry{
		UserID: "1", Username: "test", Action: "app.create",
		ResourceType: "app", ResourceID: "1", IPAddress: "1.2.3.4",
	})

	logs, _, _ := svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 1})
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	// Intact record should pass
	if err := svc.VerifyRecord(logs[0]); err != nil {
		t.Errorf("VerifyRecord() should pass for intact record, got: %v", err)
	}

	// Tampered record should fail
	logs[0].Username = "tampered"
	if err := svc.VerifyRecord(logs[0]); err == nil {
		t.Error("VerifyRecord() should fail for tampered record")
	}
}

func TestAuditService_VerifyRecords(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	for i := 0; i < 5; i++ {
		_ = svc.Record(context.TODO(), AuditEntry{UserID: fmt.Sprintf("user-%d", i), Action: "app.create"})
	}

	logs, _, _ := svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 10})
	failed := svc.VerifyRecords(logs)
	if len(failed) != 0 {
		t.Errorf("VerifyRecords() failed = %v, want empty", failed)
	}

	// Tamper one record
	logs[2].Action = "tampered"
	failed = svc.VerifyRecords(logs)
	if len(failed) != 1 || failed[0] != logs[2].ID {
		t.Errorf("VerifyRecords() should report 1 failed record, got %v", failed)
	}
}

func TestAuditService_ListWithUsername(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	_ = svc.Record(context.TODO(), AuditEntry{Username: "alice", Action: "app.create"})
	_ = svc.Record(context.TODO(), AuditEntry{Username: "bob", Action: "app.create"})
	_ = svc.Record(context.TODO(), AuditEntry{Username: "alice", Action: "app.delete"})

	_, total, _ := svc.List(context.TODO(), AuditFilter{Username: "alice", Page: 1, PageSize: 10})
	if total != 2 {
		t.Errorf("total for username 'alice' = %d, want 2", total)
	}

	_, total, _ = svc.List(context.TODO(), AuditFilter{Username: "bob", Page: 1, PageSize: 10})
	if total != 1 {
		t.Errorf("total for username 'bob' = %d, want 1", total)
	}
}

func TestAuditService_ListWithTimeRange(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	_ = svc.Record(context.TODO(), AuditEntry{Action: "app.create"})
	_ = svc.Record(context.TODO(), AuditEntry{Action: "app.create"})

	// Query all — should get 2
	_, total, _ := svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 10})
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}

	// Query with future start time — should get 0
	future := time.Now().Add(24 * time.Hour)
	_, total, _ = svc.List(context.TODO(), AuditFilter{StartTime: future, Page: 1, PageSize: 10})
	if total != 0 {
		t.Errorf("total with future start = %d, want 0", total)
	}

	// Query with past end time — should get 0
	past := time.Now().Add(-24 * time.Hour)
	_, total, _ = svc.List(context.TODO(), AuditFilter{EndTime: past, Page: 1, PageSize: 10})
	if total != 0 {
		t.Errorf("total with past end = %d, want 0", total)
	}
}

func TestAuditService_Cleanup(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)

	// Record is created "now" — archive then cleanup with 0 days retention should delete it
	_ = svc.Record(context.TODO(), AuditEntry{Action: "app.create"})
	_, total, _ := svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 10})
	if total != 1 {
		t.Fatalf("total before cleanup = %d, want 1", total)
	}

	// Archive first (Cleanup only deletes archived records)
	_, _ = svc.Archive(context.TODO(), 0)

	deleted, err := svc.Cleanup(context.TODO(), 0)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	_, total, _ = svc.List(context.TODO(), AuditFilter{Page: 1, PageSize: 10})
	if total != 0 {
		t.Errorf("total after cleanup = %d, want 0", total)
	}

	// Cleanup with 365 days retention should not delete anything
	_ = svc.Record(context.TODO(), AuditEntry{Action: "app.create"})
	deleted, err = svc.Cleanup(context.TODO(), 365)
	if err != nil {
		t.Fatalf("Cleanup(365) error = %v", err)
	}
	if deleted != 0 {
		t.Errorf("deleted with 365 days = %d, want 0", deleted)
	}
}
