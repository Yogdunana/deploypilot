package audit

import (
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
)

func TestAppendHash_FirstEntry(t *testing.T) {
	_, chain := setupAuditTestDB(t)

	record := &model.AuditLog{
		ID:        "audit-1",
		Action:    "user.login",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	hash := chain.ComputeHash("", record)

	if err := chain.AppendHash("audit-1", hash); err != nil {
		t.Fatalf("AppendHash() error = %v", err)
	}

	got, err := chain.GetRecordHash("audit-1")
	if err != nil {
		t.Fatalf("GetRecordHash() error = %v", err)
	}
	if got != hash {
		t.Errorf("GetRecordHash() = %q, want %q", got, hash)
	}
}

func TestAppendHash_ChainsFromPrevious(t *testing.T) {
	_, chain := setupAuditTestDB(t)

	// Insert two records in order, with the second chaining off the first.
	prev := &model.AuditLog{ID: "audit-A", Action: "user.login", CreatedAt: time.Now()}
	hashPrev := chain.ComputeHash("", prev)
	if err := chain.AppendHash("audit-A", hashPrev); err != nil {
		t.Fatalf("AppendHash(prev) error = %v", err)
	}

	curr := &model.AuditLog{ID: "audit-B", Action: "user.logout", CreatedAt: time.Now()}
	hashCurr := chain.ComputeHash(hashPrev, curr)
	if err := chain.AppendHash("audit-B", hashCurr); err != nil {
		t.Fatalf("AppendHash(curr) error = %v", err)
	}

	// GetRecordHash should return each record's stored hash.
	gotA, err := chain.GetRecordHash("audit-A")
	if err != nil {
		t.Fatalf("GetRecordHash(A) error = %v", err)
	}
	if gotA != hashPrev {
		t.Errorf("audit-A hash = %q, want %q", gotA, hashPrev)
	}

	gotB, err := chain.GetRecordHash("audit-B")
	if err != nil {
		t.Fatalf("GetRecordHash(B) error = %v", err)
	}
	if gotB != hashCurr {
		t.Errorf("audit-B hash = %q, want %q", gotB, hashCurr)
	}
}

func TestGetRecordHash_NotFound(t *testing.T) {
	_, chain := setupAuditTestDB(t)

	_, err := chain.GetRecordHash("nonexistent")
	if err == nil {
		t.Error("GetRecordHash() with nonexistent ID should return error")
	}
}

func TestVerifyChain_Empty(t *testing.T) {
	_, chain := setupAuditTestDB(t)

	results, err := chain.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("VerifyChain() on empty db returned %d results, want 0", len(results))
	}
}

func TestVerifyChain_AllValid(t *testing.T) {
	db, chain := setupAuditTestDB(t)

	// Insert audit logs and matching hashes, in order.
	logs := []model.AuditLog{
		{ID: "log-1", Action: "user.login", CreatedAt: time.Now()},
		{ID: "log-2", Action: "user.action", CreatedAt: time.Now()},
		{ID: "log-3", Action: "user.logout", CreatedAt: time.Now()},
	}
	prevHash := ""
	for i := range logs {
		if err := db.Create(&logs[i]).Error; err != nil {
			t.Fatalf("failed to insert log: %v", err)
		}
		hash := chain.ComputeHash(prevHash, &logs[i])
		if err := chain.AppendHash(logs[i].ID, hash); err != nil {
			t.Fatalf("AppendHash(%s) error = %v", logs[i].ID, err)
		}
		prevHash = hash
	}

	results, err := chain.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain() error = %v", err)
	}
	if len(results) != len(logs) {
		t.Fatalf("expected %d results, got %d", len(logs), len(results))
	}
	for i, r := range results {
		if !r.Valid {
			t.Errorf("result[%d] (id=%s) should be valid, got expected=%q actual=%q",
				i, r.RecordID, r.ExpectedHash, r.ActualHash)
		}
	}
}

func TestVerifyChain_DetectsMissingHash(t *testing.T) {
	db, chain := setupAuditTestDB(t)

	// Insert a log without creating a matching hash entry.
	log := model.AuditLog{ID: "log-no-hash", Action: "user.login", CreatedAt: time.Now()}
	if err := db.Create(&log).Error; err != nil {
		t.Fatalf("failed to insert log: %v", err)
	}

	results, err := chain.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Valid {
		t.Error("result should be invalid when no hash entry exists")
	}
	if results[0].ActualHash != "" {
		t.Errorf("ActualHash should be empty for missing hash, got %q", results[0].ActualHash)
	}
}

func TestVerifyChain_DetectsTamperedHash(t *testing.T) {
	db, chain := setupAuditTestDB(t)

	log := model.AuditLog{ID: "log-tampered", Action: "user.login", CreatedAt: time.Now()}
	if err := db.Create(&log).Error; err != nil {
		t.Fatalf("failed to insert log: %v", err)
	}
	// Insert a wrong hash.
	if err := db.Create(&model.AuditHash{
		AuditID:      "log-tampered",
		Hash:         "wrong-hash",
		PreviousHash: "",
	}).Error; err != nil {
		t.Fatalf("failed to insert hash: %v", err)
	}

	results, err := chain.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Valid {
		t.Error("result should be invalid when stored hash doesn't match expected")
	}
	if results[0].ActualHash != "wrong-hash" {
		t.Errorf("ActualHash should be the stored value %q, got %q", "wrong-hash", results[0].ActualHash)
	}
}
