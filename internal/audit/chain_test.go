package audit

import (
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuditTestDB(t *testing.T) (*gorm.DB, *AuditChain) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	chain := NewAuditChain(db, []byte("test-secret-key"))
	return db, chain
}

func TestComputeHash(t *testing.T) {
	_, chain := setupAuditTestDB(t)

	record := &model.AuditLog{
		ID:        1,
		Action:    "user.login",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	hash1 := chain.ComputeHash("", record)
	if hash1 == "" {
		t.Error("expected non-empty hash")
	}

	// Same input should produce same hash
	hash2 := chain.ComputeHash("", record)
	if hash1 != hash2 {
		t.Error("same input should produce same hash")
	}

	// Different prev hash should produce different hash
	hash3 := chain.ComputeHash("different-prev", record)
	if hash1 == hash3 {
		t.Error("different prev hash should produce different hash")
	}
}

func TestComputeHash_DifferentRecords(t *testing.T) {
	_, chain := setupAuditTestDB(t)

	record1 := &model.AuditLog{ID: 1, Action: "user.login", CreatedAt: time.Now()}
	record2 := &model.AuditLog{ID: 2, Action: "user.logout", CreatedAt: time.Now()}

	hash1 := chain.ComputeHash("prev", record1)
	hash2 := chain.ComputeHash("prev", record2)

	if hash1 == hash2 {
		t.Error("different records should produce different hashes")
	}
}

func TestNewAuditChain(t *testing.T) {
	db, _ := setupAuditTestDB(t)

	chain := NewAuditChain(db, []byte("secret"))
	if chain == nil {
		t.Error("expected non-nil chain")
	}
	if chain.secretKey == nil {
		t.Error("expected non-nil secretKey")
	}
}
