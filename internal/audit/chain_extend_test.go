package audit

import (
	"errors"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// chainDB is a small in-memory fixture for the chain test family. It
// pre-creates the audit_logs and audit_hashes tables.
func chainDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newAuditDB(t)
}

// seedLog is a convenience for chain tests that need a log with a
// stable id, action and created_at.
func seedLog(t *testing.T, db *gorm.DB, id, action string, createdAt time.Time) {
	t.Helper()
	seedAuditLog(t, db, model.AuditLog{
		ID:        id,
		Action:    action,
		LogType:   "auth",
		CreatedAt: createdAt,
	})
}

// ---------- AppendHash / GetRecordHash ----------

// TestAppendHash_FirstRecord_PreviousHashEmpty locks in the genesis
// case: the very first record's PreviousHash must be empty.
func TestAppendHash_FirstRecord_PreviousHashEmpty(t *testing.T) {
	db := chainDB(t)
	seedLog(t, db, "a-1", "user.login", time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	ac := NewAuditChain(db, []byte("test-secret"))
	if err := ac.AppendHash("a-1", "deadbeef"); err != nil {
		t.Fatalf("AppendHash: %v", err)
	}
	got, err := ac.GetRecordHash("a-1")
	if err != nil {
		t.Fatalf("GetRecordHash: %v", err)
	}
	if got != "deadbeef" {
		t.Errorf("hash = %q, want deadbeef", got)
	}
	var entry model.AuditHash
	if err := db.Where("audit_id = ?", "a-1").First(&entry).Error; err != nil {
		t.Fatalf("query hash: %v", err)
	}
	if entry.PreviousHash != "" {
		t.Errorf("PreviousHash for genesis = %q, want empty", entry.PreviousHash)
	}
}

// TestAppendHash_ChainContinuity verifies that the n-th record's
// PreviousHash equals the (n-1)-th record's stored hash.
func TestAppendHash_ChainContinuity(t *testing.T) {
	db := chainDB(t)
	seedLog(t, db, "a-1", "user.login", time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	seedLog(t, db, "a-2", "user.logout", time.Date(2025, 1, 1, 12, 5, 0, 0, time.UTC))
	seedLog(t, db, "a-3", "user.update", time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC))

	ac := NewAuditChain(db, []byte("test-secret"))
	if err := ac.AppendHash("a-1", "h1"); err != nil {
		t.Fatalf("AppendHash a-1: %v", err)
	}
	if err := ac.AppendHash("a-2", "h2"); err != nil {
		t.Fatalf("AppendHash a-2: %v", err)
	}
	if err := ac.AppendHash("a-3", "h3"); err != nil {
		t.Fatalf("AppendHash a-3: %v", err)
	}

	var a2, a3 model.AuditHash
	if err := db.Where("audit_id = ?", "a-2").First(&a2).Error; err != nil {
		t.Fatalf("query a-2: %v", err)
	}
	if err := db.Where("audit_id = ?", "a-3").First(&a3).Error; err != nil {
		t.Fatalf("query a-3: %v", err)
	}
	if a2.PreviousHash != "h1" {
		t.Errorf("a-2.PreviousHash = %q, want h1", a2.PreviousHash)
	}
	if a3.PreviousHash != "h2" {
		t.Errorf("a-3.PreviousHash = %q, want h2", a3.PreviousHash)
	}
}

// TestGetRecordHash_NotFound asserts the documented error path: a
// missing record returns a non-nil error.
func TestGetRecordHash_NotFound(t *testing.T) {
	db := chainDB(t)
	ac := NewAuditChain(db, []byte("test-secret"))
	_, err := ac.GetRecordHash("no-such-id")
	if err == nil {
		t.Fatal("expected error for missing record, got nil")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// ---------- VerifyChain ----------

// TestVerifyChain_EmptyTable verifies the no-rows contract: the result
// is an empty slice and no error.
func TestVerifyChain_EmptyTable(t *testing.T) {
	db := chainDB(t)
	ac := NewAuditChain(db, []byte("test-secret"))
	results, err := ac.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results on empty table, got %d", len(results))
	}
}

// TestVerifyChain_SingleRecordValid covers the happy path: a single
// record with a correct hash must verify as Valid=true.
func TestVerifyChain_SingleRecordValid(t *testing.T) {
	db := chainDB(t)
	at := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	seedLog(t, db, "a-1", "user.login", at)

	ac := NewAuditChain(db, []byte("test-secret"))
	// Compute the hash that VerifyChain expects.
	var stored model.AuditLog
	db.First(&stored, "id = ?", "a-1")
	expected := ac.ComputeHash("", &stored)
	if err := ac.AppendHash("a-1", expected); err != nil {
		t.Fatalf("AppendHash: %v", err)
	}

	results, err := ac.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.Valid {
		t.Errorf("valid = false, expected true; diff: expected=%s actual=%s",
			r.ExpectedHash, r.ActualHash)
	}
	if r.RecordID != "a-1" {
		t.Errorf("RecordID = %q, want a-1", r.RecordID)
	}
	if r.ExpectedHash == "" || r.ActualHash == "" {
		t.Error("hash fields should be populated")
	}
}

// TestVerifyChain_MultipleRecordsAllValid builds a 3-link chain and
// verifies that all three records are reported Valid.
func TestVerifyChain_MultipleRecordsAllValid(t *testing.T) {
	db := chainDB(t)
	times := []time.Time{
		time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 12, 5, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC),
	}
	ids := []string{"a-1", "a-2", "a-3"}
	actions := []string{"user.login", "user.logout", "user.update"}
	for i, id := range ids {
		seedLog(t, db, id, actions[i], times[i])
	}

	ac := NewAuditChain(db, []byte("test-secret"))
	// Compute and append hashes using the actual chain (so the
	// expected hashes match what VerifyChain will recompute).
	prevHash := ""
	for _, id := range ids {
		var rec model.AuditLog
		if err := db.First(&rec, "id = ?", id).Error; err != nil {
			t.Fatalf("query %s: %v", id, err)
		}
		h := ac.ComputeHash(prevHash, &rec)
		if err := ac.AppendHash(id, h); err != nil {
			t.Fatalf("AppendHash %s: %v", id, err)
		}
		prevHash = h
	}

	results, err := ac.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Valid {
			t.Errorf("result %d (id=%s) not valid; diff: expected=%s actual=%s",
				i, r.RecordID, r.ExpectedHash, r.ActualHash)
		}
	}
}

// TestVerifyChain_TamperedHashDetected is the critical security test:
// flipping the stored hash of the second record must cause that
// record (and the next, since it depends on prevHash) to be reported
// as invalid.
func TestVerifyChain_TamperedHashDetected(t *testing.T) {
	db := chainDB(t)
	times := []time.Time{
		time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 12, 5, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC),
	}
	ids := []string{"a-1", "a-2", "a-3"}
	actions := []string{"user.login", "user.logout", "user.update"}
	for i, id := range ids {
		seedLog(t, db, id, actions[i], times[i])
	}

	ac := NewAuditChain(db, []byte("test-secret"))
	prevHash := ""
	for _, id := range ids {
		var rec model.AuditLog
		if err := db.First(&rec, "id = ?", id).Error; err != nil {
			t.Fatalf("query %s: %v", id, err)
		}
		h := ac.ComputeHash(prevHash, &rec)
		if err := ac.AppendHash(id, h); err != nil {
			t.Fatalf("AppendHash %s: %v", id, err)
		}
		prevHash = h
	}

	// Tamper: flip the last byte of the second record's stored hash.
	if err := db.Model(&model.AuditHash{}).
		Where("audit_id = ?", "a-2").
		Update("hash", "0" + prevHash[1:]).Error; err != nil {
		t.Fatalf("tamper: %v", err)
	}

	results, err := ac.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	// First record (genesis) is unaffected.
	if !results[0].Valid {
		t.Errorf("results[0] (a-1) should still be valid, got: expected=%s actual=%s",
			results[0].ExpectedHash, results[0].ActualHash)
	}
	// Second record: hash was tampered with.
	if results[1].Valid {
		t.Errorf("results[1] (a-2) should be invalid (tampered hash)")
	}
	// Third record: depends on the second's hash, so its stored hash
	// won't match the recomputed expected hash.
	if results[2].Valid {
		t.Errorf("results[2] (a-3) should be invalid (chain continuation broken)")
	}
}

// TestVerifyChain_MissingHashForRecord covers the case where a log row
// exists but its corresponding hash row does not. VerifyChain should
// report it as Invalid (hasHash=false → valid=false).
func TestVerifyChain_MissingHashForRecord(t *testing.T) {
	db := chainDB(t)
	seedLog(t, db, "a-1", "user.login", time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	seedLog(t, db, "a-2", "user.logout", time.Date(2025, 1, 1, 12, 5, 0, 0, time.UTC))
	// Insert a hash only for a-1, not for a-2.
	ac := NewAuditChain(db, []byte("test-secret"))
	var rec model.AuditLog
	db.First(&rec, "id = ?", "a-1")
	if err := ac.AppendHash("a-1", ac.ComputeHash("", &rec)); err != nil {
		t.Fatalf("AppendHash a-1: %v", err)
	}

	results, err := ac.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].Valid {
		t.Errorf("results[0] (a-1) should be valid, got invalid")
	}
	if results[1].Valid {
		t.Errorf("results[1] (a-2) should be invalid (no hash stored)")
	}
	if results[1].ActualHash != "" {
		t.Errorf("results[1].ActualHash = %q, want empty", results[1].ActualHash)
	}
}
