package service

import (
	"context"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupCheckerDB creates an in-memory SQLite DB for credential checker tests.
func setupCheckerDB(t *testing.T) (*gorm.DB, []byte) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create credentials table with lifecycle fields
	db.Exec(`CREATE TABLE IF NOT EXISTS credentials (
		id TEXT PRIMARY KEY, tenant_id TEXT, name TEXT NOT NULL,
		type TEXT NOT NULL, encrypted_value TEXT NOT NULL,
		expires_at DATETIME, last_rotated DATETIME, rotation_days INTEGER DEFAULT 90,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	encKey := crypto.NewEncryptionKey()
	return db, encKey
}

func TestCredentialChecker_FindsExpiringCredentials(t *testing.T) {
	db, encKey := setupCheckerDB(t)

	// Insert a credential expiring in 3 days
	threeDays := time.Now().Add(3 * 24 * time.Hour)
	encrypted, _ := crypto.Encrypt(encKey, "secret")
	db.Exec(`INSERT INTO credentials (id, tenant_id, name, type, encrypted_value, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"cred-expiring", "tenant-default", "expiring-cred", "ssh", encrypted, threeDays)

	// Insert a credential with no expiry
	encrypted2, _ := crypto.Encrypt(encKey, "secret2")
	db.Exec(`INSERT INTO credentials (id, tenant_id, name, type, encrypted_value)
		VALUES (?, ?, ?, ?, ?)`,
		"cred-never", "tenant-default", "never-expires", "ssh", encrypted2)

	// Insert a credential expiring in 30 days (should not be found)
	thirtyDays := time.Now().Add(30 * 24 * time.Hour)
	encrypted3, _ := crypto.Encrypt(encKey, "secret3")
	db.Exec(`INSERT INTO credentials (id, tenant_id, name, type, encrypted_value, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"cred-far", "tenant-default", "far-future", "ssh", encrypted3, thirtyDays)

	// Create checker and run a check
	bridge := &Bridge{DB: db, EncryptionKey: encKey}
	checker := NewCredentialChecker(db, bridge, 6*time.Hour)

	// Run check with a context that times out quickly
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// The check should find the expiring credential
	// We verify by checking ListExpiringCredentials directly
	creds, err := model.ListExpiringCredentials(db, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ListExpiringCredentials() error = %v", err)
	}

	if len(creds) != 1 {
		t.Errorf("found %d expiring credentials, want 1", len(creds))
	}

	if len(creds) > 0 && creds[0].Name != "expiring-cred" {
		t.Errorf("credential name = %q, want %q", creds[0].Name, "expiring-cred")
	}

	// Verify the check method runs without error
	// (It will try to send notifications but that's fine - bridge has no notifiers configured)
	checker.check(ctx)
}

func TestCredentialChecker_NoExpiringCredentials(t *testing.T) {
	db, encKey := setupCheckerDB(t)

	// Insert only credentials that don't expire soon
	thirtyDays := time.Now().Add(30 * 24 * time.Hour)
	encrypted, _ := crypto.Encrypt(encKey, "secret")
	db.Exec(`INSERT INTO credentials (id, tenant_id, name, type, encrypted_value, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"cred-far", "tenant-default", "far-future", "ssh", encrypted, thirtyDays)

	bridge := &Bridge{DB: db, EncryptionKey: encKey}
	checker := NewCredentialChecker(db, bridge, 6*time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	creds, err := model.ListExpiringCredentials(db, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ListExpiringCredentials() error = %v", err)
	}

	if len(creds) != 0 {
		t.Errorf("found %d expiring credentials, want 0", len(creds))
	}

	checker.check(ctx)
}

func TestCredentialChecker_AlreadyExpiredCredentials(t *testing.T) {
	db, encKey := setupCheckerDB(t)

	// Insert an already expired credential
	past := time.Now().Add(-24 * time.Hour)
	encrypted, _ := crypto.Encrypt(encKey, "secret")
	db.Exec(`INSERT INTO credentials (id, tenant_id, name, type, encrypted_value, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"cred-expired", "tenant-default", "already-expired", "ssh", encrypted, past)

	bridge := &Bridge{DB: db, EncryptionKey: encKey}
	checker := NewCredentialChecker(db, bridge, 6*time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	creds, err := model.ListExpiringCredentials(db, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ListExpiringCredentials() error = %v", err)
	}

	if len(creds) != 1 {
		t.Errorf("found %d expired credentials, want 1", len(creds))
	}

	if len(creds) > 0 {
		if !model.IsExpired(&creds[0]) {
			t.Error("credential should be expired")
		}
	}

	checker.check(ctx)
}

func TestNewCredentialChecker_DefaultInterval(t *testing.T) {
	db, encKey := setupCheckerDB(t)
	bridge := &Bridge{DB: db, EncryptionKey: encKey}

	checker := NewCredentialChecker(db, bridge, 0)
	if checker.checkInterval != 6*time.Hour {
		t.Errorf("checkInterval = %v, want 6h", checker.checkInterval)
	}
}

func TestNewCredentialChecker_CustomInterval(t *testing.T) {
	db, encKey := setupCheckerDB(t)
	bridge := &Bridge{DB: db, EncryptionKey: encKey}

	checker := NewCredentialChecker(db, bridge, 2*time.Hour)
	if checker.checkInterval != 2*time.Hour {
		t.Errorf("checkInterval = %v, want 2h", checker.checkInterval)
	}
}
