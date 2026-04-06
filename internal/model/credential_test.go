package model

import (
	"testing"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/database"
)

func setupCredDB(t *testing.T) ([]byte, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	db, err := database.Connect("sqlite", tmpDir+"/test.db")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := database.Seed(db); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}
	encKey := crypto.NewEncryptionKey()
	InitDB(db, encKey)
	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
	return encKey, cleanup
}

func TestCreateCredential(t *testing.T) {
	encKey, cleanup := setupCredDB(t)
	defer cleanup()

	cred, err := CreateCredential(encKey, "tenant-default", "my-ssh-key", "ssh", "super-secret-value")
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}

	if cred.ID == "" {
		t.Error("ID should not be empty")
	}
	if cred.Name != "my-ssh-key" {
		t.Errorf("Name = %q, want %q", cred.Name, "my-ssh-key")
	}
	if cred.Type != "ssh" {
		t.Errorf("Type = %q, want %q", cred.Type, "ssh")
	}
	if cred.EncryptedValue == "super-secret-value" {
		t.Error("EncryptedValue should not equal plaintext")
	}
	if cred.EncryptedValue == "" {
		t.Error("EncryptedValue should not be empty")
	}
}

func TestGetCredential(t *testing.T) {
	encKey, cleanup := setupCredDB(t)
	defer cleanup()

	created, _ := CreateCredential(encKey, "tenant-default", "my-key", "ssh", "secret-value")

	got, err := GetCredential(encKey, created.ID)
	if err != nil {
		t.Fatalf("GetCredential() error = %v", err)
	}

	if got.Name != "my-key" {
		t.Errorf("Name = %q, want %q", got.Name, "my-key")
	}
	if got.PlainValue != "secret-value" {
		t.Errorf("PlainValue = %q, want %q", got.PlainValue, "secret-value")
	}
}

func TestGetCredentialNotFound(t *testing.T) {
	encKey, cleanup := setupCredDB(t)
	defer cleanup()

	_, err := GetCredential(encKey, "nonexistent-id")
	if err == nil {
		t.Error("GetCredential() should fail for nonexistent ID")
	}
}

func TestListCredentials(t *testing.T) {
	encKey, cleanup := setupCredDB(t)
	defer cleanup()

	CreateCredential(encKey, "tenant-default", "key-1", "ssh", "val-1")
	CreateCredential(encKey, "tenant-default", "key-2", "api_key", "val-2")

	creds, err := ListCredentials("tenant-default")
	if err != nil {
		t.Fatalf("ListCredentials() error = %v", err)
	}

	if len(creds) != 2 {
		t.Errorf("count = %d, want 2", len(creds))
	}
}

func TestListCredentialsByTenant(t *testing.T) {
	encKey, cleanup := setupCredDB(t)
	defer cleanup()

	CreateCredential(encKey, "tenant-default", "key-a", "ssh", "val-a")
	CreateCredential(encKey, "tenant-other", "key-b", "ssh", "val-b")

	creds, _ := ListCredentials("tenant-default")
	if len(creds) != 1 {
		t.Errorf("count = %d, want 1 (filtered by tenant)", len(creds))
	}
}

func TestDeleteCredential(t *testing.T) {
	encKey, cleanup := setupCredDB(t)
	defer cleanup()

	created, _ := CreateCredential(encKey, "tenant-default", "to-delete", "ssh", "val")

	err := DeleteCredential(created.ID)
	if err != nil {
		t.Fatalf("DeleteCredential() error = %v", err)
	}

	_, err = GetCredential(encKey, created.ID)
	if err == nil {
		t.Error("GetCredential() should fail after delete")
	}
}

func TestDeleteCredentialNotFound(t *testing.T) {
	_, cleanup := setupCredDB(t)
	defer cleanup()

	err := DeleteCredential("nonexistent-id")
	if err == nil {
		t.Error("DeleteCredential() should fail for nonexistent ID")
	}
}

func TestUpdateCredential(t *testing.T) {
	encKey, cleanup := setupCredDB(t)
	defer cleanup()

	created, _ := CreateCredential(encKey, "tenant-default", "my-key", "ssh", "old-value")

	updated, err := UpdateCredential(encKey, created.ID, "new-secret")
	if err != nil {
		t.Fatalf("UpdateCredential() error = %v", err)
	}

	if updated.PlainValue != "new-secret" {
		t.Errorf("PlainValue = %q, want %q", updated.PlainValue, "new-secret")
	}
}

func TestCredentialRoundTrip(t *testing.T) {
	encKey, cleanup := setupCredDB(t)
	defer cleanup()

	// Create
	cred, _ := CreateCredential(encKey, "tenant-default", "round-trip", "token", "my-api-token-12345")

	// Get and verify
	got, _ := GetCredential(encKey, cred.ID)
	if got.PlainValue != "my-api-token-12345" {
		t.Errorf("round-trip failed: got %q", got.PlainValue)
	}

	// Update
	UpdateCredential(encKey, cred.ID, "new-token-67890")

	// Get again
	got2, _ := GetCredential(encKey, cred.ID)
	if got2.PlainValue != "new-token-67890" {
		t.Errorf("round-trip update failed: got %q", got2.PlainValue)
	}

	// Delete
	DeleteCredential(cred.ID)

	// Verify deleted
	_, err := GetCredential(encKey, cred.ID)
	if err == nil {
		t.Error("should fail after delete in round-trip")
	}
}

func TestUpdateCredentialNotFound(t *testing.T) {
	encKey, cleanup := setupCredDB(t)
	defer cleanup()

	_, err := UpdateCredential(encKey, "nonexistent-id", "new-value")
	if err == nil {
		t.Error("UpdateCredential() should fail for nonexistent ID")
	}
}

func TestGenerateUUID(t *testing.T) {
	uuid := generateUUID()
	if uuid == "" {
		t.Error("generateUUID() should return non-empty string")
	}
	if len(uuid) != 32 {
		t.Errorf("generateUUID() length = %d, want 32", len(uuid))
	}
}

func TestGenerateUUIDUnique(t *testing.T) {
	uuid1 := generateUUID()
	uuid2 := generateUUID()
	if uuid1 == uuid2 {
		t.Error("generateUUID() should return unique values")
	}
}

func TestGetDBPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("getDB() should panic when db is not initialized")
		}
		msg, ok := r.(string)
		if !ok {
			t.Errorf("panic value should be string, got %T", r)
		}
		if msg != "model: database not initialized, call InitDB() first" {
			t.Errorf("panic message = %q", msg)
		}
	}()

	// Reset dbHolder to trigger panic
	dbHolder.db = nil
	getDB()
}

func TestGetEncKeyPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("getEncKey() should panic when encKey is not initialized")
		}
		msg, ok := r.(string)
		if !ok {
			t.Errorf("panic value should be string, got %T", r)
		}
		if msg != "model: encryption key not initialized, call InitDB() first" {
			t.Errorf("panic message = %q", msg)
		}
	}()

	// Reset encKey to trigger panic
	dbHolder.encKey = nil
	getEncKey()
}

func TestCredentialWithPlainStruct(t *testing.T) {
	cwp := &CredentialWithPlain{
		Credential: Credential{
			ID:             "cred-001",
			TenantID:       "tenant-001",
			Name:           "test-cred",
			Type:           "ssh",
			EncryptedValue: "encrypted",
		},
		PlainValue: "plain-text-value",
	}

	if cwp.ID != "cred-001" {
		t.Errorf("ID = %q", cwp.ID)
	}
	if cwp.PlainValue != "plain-text-value" {
		t.Errorf("PlainValue = %q", cwp.PlainValue)
	}
	if cwp.Name != "test-cred" {
		t.Errorf("Name = %q", cwp.Name)
	}
}

func TestCreateCredentialDifferentTypes(t *testing.T) {
	encKey, cleanup := setupCredDB(t)
	defer cleanup()

	types := []string{"ssh", "api_key", "token"}
	for _, credType := range types {
		t.Run(credType, func(t *testing.T) {
			cred, err := CreateCredential(encKey, "tenant-default", "cred-"+credType, credType, "value")
			if err != nil {
				t.Fatalf("CreateCredential(%s) error = %v", credType, err)
			}
			if cred.Type != credType {
				t.Errorf("Type = %q, want %q", cred.Type, credType)
			}
		})
	}
}

func TestListCredentialsEmpty(t *testing.T) {
	_, cleanup := setupCredDB(t)
	defer cleanup()

	// List for a tenant with no credentials
	creds, err := ListCredentials("tenant-nonexistent")
	if err != nil {
		t.Fatalf("ListCredentials() error = %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("count = %d, want 0", len(creds))
	}
}
