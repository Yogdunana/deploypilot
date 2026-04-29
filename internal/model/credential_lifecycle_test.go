package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupLifecycleDB creates an in-memory SQLite DB with credentials table for lifecycle tests.
func setupLifecycleDB(t *testing.T) (*gorm.DB, []byte, func()) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.AutoMigrate(&Tenant{}, &User{}, &Role{}, &Server{}, &App{}, &Credential{}, &Cluster{}, &DeploymentRecord{}, &AuditLog{}, &SSLCertificate{}, &Provider{})
	encKey := crypto.NewEncryptionKey()
	InitDB(db, encKey)

	// Ensure new columns exist (AutoMigrate handles this, but be safe)
	db.AutoMigrate(&Credential{})

	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
	return db, encKey, cleanup
}

func TestIsExpired_NilExpiresAt(t *testing.T) {
	cred := &Credential{}
	if IsExpired(cred) {
		t.Error("IsExpired() should return false when ExpiresAt is nil")
	}
}

func TestIsExpired_FutureDate(t *testing.T) {
	future := time.Now().Add(30 * 24 * time.Hour)
	cred := &Credential{ExpiresAt: &future}
	if IsExpired(cred) {
		t.Error("IsExpired() should return false for future ExpiresAt")
	}
}

func TestIsExpired_PastDate(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	cred := &Credential{ExpiresAt: &past}
	if !IsExpired(cred) {
		t.Error("IsExpired() should return true for past ExpiresAt")
	}
}

func TestIsExpired_ExactlyNow(t *testing.T) {
	// Due to time.Now() precision, exact equality cannot be reliably tested.
	// Instead, verify that a credential expiring slightly in the future is NOT expired.
	future := time.Now().Add(time.Minute)
	cred := &Credential{ExpiresAt: &future}
	if IsExpired(cred) {
		t.Error("IsExpired() should return false for ExpiresAt in the near future")
	}
}

func TestDaysUntilExpiry_NilExpiresAt(t *testing.T) {
	cred := &Credential{}
	days := DaysUntilExpiry(cred)
	if days != -1 {
		t.Errorf("DaysUntilExpiry() = %d, want -1 for nil ExpiresAt", days)
	}
}

func TestDaysUntilExpiry_FutureDate(t *testing.T) {
	future := time.Now().Add(10 * 24 * time.Hour)
	cred := &Credential{ExpiresAt: &future}
	days := DaysUntilExpiry(cred)
	if days < 9 || days > 10 {
		t.Errorf("DaysUntilExpiry() = %d, want ~10", days)
	}
}

func TestDaysUntilExpiry_PastDate(t *testing.T) {
	past := time.Now().Add(-5 * 24 * time.Hour)
	cred := &Credential{ExpiresAt: &past}
	days := DaysUntilExpiry(cred)
	if days >= 0 {
		t.Errorf("DaysUntilExpiry() = %d, want negative for past ExpiresAt", days)
	}
}

func TestDaysUntilExpiry_VerySoon(t *testing.T) {
	soon := time.Now().Add(12 * time.Hour)
	cred := &Credential{ExpiresAt: &soon}
	days := DaysUntilExpiry(cred)
	if days != 0 {
		t.Errorf("DaysUntilExpiry() = %d, want 0 for less than 24 hours", days)
	}
}

func TestListExpiringCredentials_NoExpiring(t *testing.T) {
	db, encKey, cleanup := setupLifecycleDB(t)
	defer cleanup()

	// Create a credential that expires far in the future
	farFuture := time.Now().Add(365 * 24 * time.Hour)
	_, err := CreateCredentialWithExpiry(encKey, "tenant-default", "far-future", "ssh", "secret", farFuture)
	if err != nil {
		t.Fatalf("CreateCredentialWithExpiry() error = %v", err)
	}

	creds, err := ListExpiringCredentials(db, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ListExpiringCredentials() error = %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("count = %d, want 0 (no credentials expiring within 7 days)", len(creds))
	}
}

func TestListExpiringCredentials_WithExpiring(t *testing.T) {
	db, encKey, cleanup := setupLifecycleDB(t)
	defer cleanup()

	// Create a credential expiring in 3 days
	nearFuture := time.Now().Add(3 * 24 * time.Hour)
	_, err := CreateCredentialWithExpiry(encKey, "tenant-default", "near-expiry", "ssh", "secret", nearFuture)
	if err != nil {
		t.Fatalf("CreateCredentialWithExpiry() error = %v", err)
	}

	// Create a credential expiring in 30 days (should not be returned)
	farFuture := time.Now().Add(30 * 24 * time.Hour)
	_, err = CreateCredentialWithExpiry(encKey, "tenant-default", "far-expiry", "ssh", "secret", farFuture)
	if err != nil {
		t.Fatalf("CreateCredentialWithExpiry() error = %v", err)
	}

	creds, err := ListExpiringCredentials(db, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ListExpiringCredentials() error = %v", err)
	}
	if len(creds) != 1 {
		t.Errorf("count = %d, want 1", len(creds))
	}
	if creds[0].Name != "near-expiry" {
		t.Errorf("Name = %q, want %q", creds[0].Name, "near-expiry")
	}
}

func TestListExpiringCredentials_AlreadyExpired(t *testing.T) {
	db, encKey, cleanup := setupLifecycleDB(t)
	defer cleanup()

	// Create a credential that already expired
	past := time.Now().Add(-24 * time.Hour)
	_, err := CreateCredentialWithExpiry(encKey, "tenant-default", "expired", "ssh", "secret", past)
	if err != nil {
		t.Fatalf("CreateCredentialWithExpiry() error = %v", err)
	}

	creds, err := ListExpiringCredentials(db, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ListExpiringCredentials() error = %v", err)
	}
	if len(creds) != 1 {
		t.Errorf("count = %d, want 1 (expired credential should be included)", len(creds))
	}
}

func TestListExpiringCredentials_NeverExpires(t *testing.T) {
	db, encKey, cleanup := setupLifecycleDB(t)
	defer cleanup()

	// Create a credential with no expiry
	_, err := CreateCredential(encKey, "tenant-default", "never-expires", "ssh", "secret")
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}

	creds, err := ListExpiringCredentials(db, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ListExpiringCredentials() error = %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("count = %d, want 0 (credential with nil ExpiresAt should be excluded)", len(creds))
	}
}

func TestRotateCredential_Success(t *testing.T) {
	db, encKey, cleanup := setupLifecycleDB(t)
	defer cleanup()

	// Create a credential with an expiry
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	created, err := CreateCredentialWithExpiry(encKey, "tenant-default", "rotate-me", "ssh", "old-value", expiresAt)
	if err != nil {
		t.Fatalf("CreateCredentialWithExpiry() error = %v", err)
	}

	// Rotate
	rotated, err := RotateCredential(db, encKey, created.ID, "new-value")
	if err != nil {
		t.Fatalf("RotateCredential() error = %v", err)
	}

	// Verify LastRotated is set
	if rotated.LastRotated == nil {
		t.Error("LastRotated should be set after rotation")
	}

	// Verify ExpiresAt is preserved
	if rotated.ExpiresAt == nil {
		t.Error("ExpiresAt should be preserved after rotation")
	}

	// Verify the encrypted value changed
	if rotated.EncryptedValue == created.EncryptedValue {
		t.Error("EncryptedValue should change after rotation")
	}
}

func TestRotateCredential_NotFound(t *testing.T) {
	db, encKey, cleanup := setupLifecycleDB(t)
	defer cleanup()

	_, err := RotateCredential(db, encKey, "nonexistent-id", "new-value")
	if err == nil {
		t.Error("RotateCredential() should fail for nonexistent ID")
	}
}

func TestRotateCredential_UpdatesEncryptedValue(t *testing.T) {
	db, encKey, cleanup := setupLifecycleDB(t)
	defer cleanup()

	created, err := CreateCredential(encKey, "tenant-default", "encrypt-test", "ssh", "original-value")
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}

	rotated, err := RotateCredential(db, encKey, created.ID, "rotated-value")
	if err != nil {
		t.Fatalf("RotateCredential() error = %v", err)
	}

	// Decrypt and verify the new value
	plainValue, err := crypto.Decrypt(encKey, rotated.EncryptedValue)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if plainValue != "rotated-value" {
		t.Errorf("decrypted value = %q, want %q", plainValue, "rotated-value")
	}
}

// CreateCredentialWithExpiry is a helper that creates a credential with a specific ExpiresAt.
func CreateCredentialWithExpiry(encKey []byte, tenantID, name, credType, plainValue string, expiresAt time.Time) (*CredentialWithPlain, error) {
	encrypted, err := crypto.Encrypt(encKey, plainValue)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential value: %w", err)
	}

	cred := &Credential{
		ID:             "cred-" + generateUUID()[:8],
		TenantID:       tenantID,
		Name:           name,
		Type:           credType,
		EncryptedValue: encrypted,
		ExpiresAt:      &expiresAt,
	}

	result := getDB().Create(cred)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create credential: %w", result.Error)
	}

	return &CredentialWithPlain{
		Credential: *cred,
		PlainValue: plainValue,
	}, nil
}
