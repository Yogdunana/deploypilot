package service

import (
	"context"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAPIKeyTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&model.APIKey{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestAPIKeyService_Create(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	svc := NewAPIKeyService(db, "test-salt-for-unit-tests")

	apiKey, rawKey, err := svc.Create(context.TODO(), "user-1", "tenant-default", "test-key", []string{"read", "write"}, 30)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rawKey == "" {
		t.Fatal("rawKey is empty")
	}
	if len(rawKey) != 35 { // "dp_" + 32 hex chars
		t.Errorf("rawKey length = %d, want 35", len(rawKey))
	}
	if apiKey.KeyPrefix != rawKey[:10] {
		t.Errorf("KeyPrefix = %q, want %q", apiKey.KeyPrefix, rawKey[:10])
	}
	if apiKey.KeyHash == "" {
		t.Error("KeyHash is empty")
	}
	if apiKey.Name != "test-key" {
		t.Errorf("Name = %q, want 'test-key'", apiKey.Name)
	}
	if apiKey.ExpiresAt == nil {
		t.Error("ExpiresAt should be set for 30-day expiry")
	}
}

func TestAPIKeyService_CreateNoExpiry(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	svc := NewAPIKeyService(db, "test-salt-for-unit-tests")

	apiKey, _, err := svc.Create(context.TODO(), "user-1", "tenant-default", "permanent-key", []string{"read"}, 0)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if apiKey.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil for no-expiry key")
	}
}

func TestAPIKeyService_Validate(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	svc := NewAPIKeyService(db, "test-salt-for-unit-tests")

	_, rawKey, err := svc.Create(context.TODO(), "user-1", "tenant-default", "test-key", []string{"read"}, 0)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Valid key should pass
	apiKey, err := svc.Validate(context.TODO(), rawKey)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if apiKey.UserID != "user-1" {
		t.Errorf("UserID = %q, want 'user-1'", apiKey.UserID)
	}

	// Invalid key should fail
	_, err = svc.Validate(context.TODO(), "dp_invalidkey123456789012345678")
	if err == nil {
		t.Error("Validate() should fail for invalid key")
	}
}

func TestAPIKeyService_List(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	svc := NewAPIKeyService(db, "test-salt-for-unit-tests")

	for i := 0; i < 3; i++ {
		_, _, err := svc.Create(context.TODO(), "user-1", "tenant-default", "key-"+string(rune('A'+i)), []string{"read"}, 0)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	// Create key for different user
	_, _, _ = svc.Create(context.TODO(), "user-2", "tenant-default", "other-key", []string{"read"}, 0)

	keys, err := svc.List(context.TODO(), "user-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("len(keys) = %d, want 3", len(keys))
	}
}

func TestAPIKeyService_Delete(t *testing.T) {
	db := setupAPIKeyTestDB(t)
	svc := NewAPIKeyService(db, "test-salt-for-unit-tests")

	apiKey, _, err := svc.Create(context.TODO(), "user-1", "tenant-default", "to-delete", []string{"read"}, 0)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Owner can delete
	err = svc.Delete(context.TODO(), apiKey.ID, "user-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err = svc.Validate(context.TODO(), "dp_anything")
	if err == nil {
		t.Error("key should be deleted")
	}

	// Different user cannot delete
	apiKey2, _, _ := svc.Create(context.TODO(), "user-2", "tenant-default", "other", []string{"read"}, 0)
	err = svc.Delete(context.TODO(), apiKey2.ID, "user-1")
	if err == nil {
		t.Error("Delete() should fail for non-owner")
	}
}

func TestParseScopes(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{`["read","write"]`, []string{"read", "write"}},
		{`["admin"]`, []string{"admin"}},
		{`invalid`, nil},
		{``, nil},
	}
	for _, tt := range tests {
		got := ParseScopes(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("ParseScopes(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestHasScope(t *testing.T) {
	if !HasScope([]string{"read", "write"}, "read") {
		t.Error("should have read scope")
	}
	if !HasScope([]string{"admin"}, "anything") {
		t.Error("admin scope should grant everything")
	}
	if HasScope([]string{"read"}, "write") {
		t.Error("should not have write scope")
	}
}
