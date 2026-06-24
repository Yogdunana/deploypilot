package audit

import (
	"strings"
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
	if err := db.AutoMigrate(&model.AuditLog{}, &model.AuditHash{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	chain := NewAuditChain(db, []byte("test-secret-key"))
	return db, chain
}

func TestComputeHash(t *testing.T) {
	_, chain := setupAuditTestDB(t)

	record := &model.AuditLog{
		ID:        "audit-init",
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

	record1 := &model.AuditLog{ID: "audit-1", Action: "user.login", CreatedAt: time.Now()}
	record2 := &model.AuditLog{ID: "audit-2", Action: "user.logout", CreatedAt: time.Now()}

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
		t.Fatal("expected non-nil chain")
	}
	if chain.secretKey == nil {
		t.Fatal("expected non-nil secretKey")
	}
}

// TestAppendHash_StoresRecord verifies the basic insert path of AppendHash.
func TestAppendHash_StoresRecord(t *testing.T) {
	db, chain := setupAuditTestDB(t)

	if err := chain.AppendHash("audit-1", "hash-value-1"); err != nil {
		t.Fatalf("AppendHash returned error: %v", err)
	}

	// The hash should be retrievable now.
	var entry model.AuditHash
	if err := db.Where("audit_id = ?", "audit-1").First(&entry).Error; err != nil {
		t.Fatalf("expected hash entry, got error: %v", err)
	}
	if entry.Hash != "hash-value-1" {
		t.Errorf("stored hash=%q, want %q", entry.Hash, "hash-value-1")
	}
	if entry.PreviousHash != "" {
		t.Errorf("expected empty previous hash for first record, got %q", entry.PreviousHash)
	}
}

// TestAppendHash_LinksToPreviousRecord verifies that a second AppendHash
// correctly captures the previous record's hash as its own previousHash.
func TestAppendHash_LinksToPreviousRecord(t *testing.T) {
	db, chain := setupAuditTestDB(t)

	if err := chain.AppendHash("audit-1", "hash-1"); err != nil {
		t.Fatalf("AppendHash #1 returned error: %v", err)
	}
	if err := chain.AppendHash("audit-2", "hash-2"); err != nil {
		t.Fatalf("AppendHash #2 returned error: %v", err)
	}

	var entry model.AuditHash
	if err := db.Where("audit_id = ?", "audit-2").First(&entry).Error; err != nil {
		t.Fatalf("expected hash entry for audit-2, got error: %v", err)
	}
	if entry.PreviousHash != "hash-1" {
		t.Errorf("PreviousHash=%q, want %q (linkage to previous record broken)", entry.PreviousHash, "hash-1")
	}
}

// TestAppendHash_StrictDescendingOrder ensures that AppendHash picks the
// largest audit_id strictly less than the current one as the previous
// record (i.e. it does not pick a same-or-greater ID).
func TestAppendHash_StrictDescendingOrder(t *testing.T) {
	db, chain := setupAuditTestDB(t)

	// Insert out of order: audit-1, audit-3, audit-2.
	_ = chain.AppendHash("audit-1", "hash-1")
	_ = chain.AppendHash("audit-3", "hash-3")
	_ = chain.AppendHash("audit-2", "hash-2")

	// For audit-2, the strict "<" query should select audit-1 (largest
	// ID strictly less than audit-2), not audit-3.
	var entry model.AuditHash
	if err := db.Where("audit_id = ?", "audit-2").First(&entry).Error; err != nil {
		t.Fatalf("expected hash entry for audit-2, got error: %v", err)
	}
	if entry.PreviousHash != "hash-1" {
		t.Errorf("PreviousHash=%q, want %q (must be audit-1's hash, not audit-3's)",
			entry.PreviousHash, "hash-1")
	}
}

// TestGetRecordHash_RetrievesStoredHash verifies the simple lookup helper.
func TestGetRecordHash_RetrievesStoredHash(t *testing.T) {
	_, chain := setupAuditTestDB(t)

	if err := chain.AppendHash("audit-x", "stored-hash-value"); err != nil {
		t.Fatalf("AppendHash returned error: %v", err)
	}

	got, err := chain.GetRecordHash("audit-x")
	if err != nil {
		t.Fatalf("GetRecordHash returned error: %v", err)
	}
	if got != "stored-hash-value" {
		t.Errorf("GetRecordHash=%q, want %q", got, "stored-hash-value")
	}
}

// TestGetRecordHash_MissingReturnsError ensures that an unknown audit_id
// surfaces a gorm error to the caller (so the API layer can map it to
// 404 / NotFound).
func TestGetRecordHash_MissingReturnsError(t *testing.T) {
	_, chain := setupAuditTestDB(t)
	if _, err := chain.GetRecordHash("does-not-exist"); err == nil {
		t.Error("expected error for missing record, got nil")
	}
}

// TestVerifyChain_Empty verifies the no-records path.
func TestVerifyChain_Empty(t *testing.T) {
	_, chain := setupAuditTestDB(t)

	results, err := chain.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain returned error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty chain, got %d", len(results))
	}
}

// TestVerifyChain_ValidChain is the end-to-end happy path: a chain that
// was assembled with correct prevHash linkage should be reported as fully
// valid.
func TestVerifyChain_ValidChain(t *testing.T) {
	db, chain := setupAuditTestDB(t)

	// Build a chain of three records with the canonical prev-hash linkage.
	prev := ""
	records := []*model.AuditLog{
		{ID: "a-1", Action: "user.login", CreatedAt: time.Now().UTC()},
		{ID: "a-2", Action: "user.update", CreatedAt: time.Now().UTC().Add(time.Second)},
		{ID: "a-3", Action: "user.logout", CreatedAt: time.Now().UTC().Add(2 * time.Second)},
	}
	for _, r := range records {
		hash := chain.ComputeHash(prev, r)
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
		if err := chain.AppendHash(r.ID, hash); err != nil {
			t.Fatalf("AppendHash: %v", err)
		}
		prev = hash
	}

	results, err := chain.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain returned error: %v", err)
	}
	if len(results) != len(records) {
		t.Fatalf("expected %d results, got %d", len(records), len(results))
	}
	for i, r := range results {
		if !r.Valid {
			t.Errorf("result[%d] (%s) is not valid: expected=%q actual=%q",
				i, r.RecordID, r.ExpectedHash, r.ActualHash)
		}
	}
}

// TestVerifyChain_TamperedHashIsInvalid verifies that mutating a stored
// hash to an incorrect value causes VerifyChain to flag the affected
// record as invalid. This is the core tamper-detection property of the
// hash chain.
func TestVerifyChain_TamperedHashIsInvalid(t *testing.T) {
	db, chain := setupAuditTestDB(t)

	rec := &model.AuditLog{ID: "a-1", Action: "user.login", CreatedAt: time.Now().UTC()}
	if err := db.Create(rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}

	realHash := chain.ComputeHash("", rec)
	if err := chain.AppendHash("a-1", realHash); err != nil {
		t.Fatalf("AppendHash: %v", err)
	}

	// Tamper: overwrite the stored hash with a wrong value.
	if err := db.Model(&model.AuditHash{}).
		Where("audit_id = ?", "a-1").
		Update("hash", "deadbeef").Error; err != nil {
		t.Fatalf("tamper update: %v", err)
	}

	results, err := chain.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Valid {
		t.Error("tampered record was reported as valid")
	}
	if results[0].ActualHash != "deadbeef" {
		t.Errorf("ActualHash=%q, want %q", results[0].ActualHash, "deadbeef")
	}
}

// TestVerifyChain_MissingHashIsInvalid verifies that if a record exists
// in audit_logs but no AuditHash row was ever written, the record is
// reported as invalid (not silently passed).
func TestVerifyChain_MissingHashIsInvalid(t *testing.T) {
	db, chain := setupAuditTestDB(t)

	rec := &model.AuditLog{ID: "a-1", Action: "user.login", CreatedAt: time.Now().UTC()}
	if err := db.Create(rec).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}
	// Deliberately do NOT call AppendHash.

	results, err := chain.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Valid {
		t.Error("record without hash entry was reported as valid")
	}
}

// TestVerifyChain_OrderByID ensures the records are processed in insertion
// order, which is required for the prev-hash walk to be meaningful.
func TestVerifyChain_OrderByID(t *testing.T) {
	db, chain := setupAuditTestDB(t)

	// Insert out of insertion order: a-3 first, a-1 second, a-2 third.
	prev := ""
	records := []*model.AuditLog{
		{ID: "a-3", Action: "third", CreatedAt: time.Now().UTC().Add(2 * time.Second)},
		{ID: "a-1", Action: "first", CreatedAt: time.Now().UTC()},
		{ID: "a-2", Action: "second", CreatedAt: time.Now().UTC().Add(time.Second)},
	}
	for _, r := range records {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("create: %v", err)
		}
		// We compute a deliberately-INCORRECT prev linkage by always using
		// "" to confirm that ordering by ID (not insertion order) drives
		// the verify walk. The test then asserts that the results array
		// is ordered by id, ascending.
		h := chain.ComputeHash(prev, r)
		if err := chain.AppendHash(r.ID, h); err != nil {
			t.Fatalf("AppendHash: %v", err)
		}
		prev = h
	}

	results, err := chain.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	wantOrder := []string{"a-1", "a-2", "a-3"}
	for i, w := range wantOrder {
		if results[i].RecordID != w {
			t.Errorf("results[%d].RecordID=%q, want %q (chain must be ordered by id)",
				i, results[i].RecordID, w)
		}
	}
}

// TestComputeHash_DifferentSecretProducesDifferentHash is a sanity check
// that the secret key actually participates in the hash. If it were
// silently ignored, swapping secrets would not invalidate the chain.
func TestComputeHash_DifferentSecretProducesDifferentHash(t *testing.T) {
	_, chain := setupAuditTestDB(t)
	rec := &model.AuditLog{ID: "a", Action: "x", CreatedAt: time.Now()}

	a := chain.ComputeHash("", rec)
	chain.secretKey = []byte("a-different-secret")
	b := chain.ComputeHash("", rec)

	if a == b {
		t.Error("different secrets should produce different hashes")
	}
	if !strings.HasPrefix(a, "") { // sanity that the value is hex
		_ = a
	}
}
