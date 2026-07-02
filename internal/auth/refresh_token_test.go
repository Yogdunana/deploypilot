package auth

import (
	"context"
	"testing"
	"time"
)

func TestGenerateRefreshTokenID(t *testing.T) {
	id, err := GenerateRefreshTokenID()
	if err != nil {
		t.Fatalf("GenerateRefreshTokenID failed: %v", err)
	}
	if len(id) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected 64-char token ID, got %d chars", len(id))
	}

	// Should generate unique IDs
	id2, err := GenerateRefreshTokenID()
	if err != nil {
		t.Fatalf("second GenerateRefreshTokenID failed: %v", err)
	}
	if id == id2 {
		t.Error("expected unique token IDs")
	}
}

// --- MemoryRefreshTokenStore tests ---

func TestMemoryRefreshTokenStore_StoreAndRetrieve(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	now := time.Now()
	entry := RefreshTokenEntry{
		TokenID:    "tid-1",
		UserID:     "user-1",
		Role:       "admin",
		DeviceInfo: "chrome",
		IPAddress:  "10.0.0.1",
		CreatedAt:  now,
		ExpiresAt:  now.Add(24 * time.Hour),
	}

	if err := store.Store(entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	retrieved, err := store.Retrieve("tid-1")
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected non-nil entry")
	}
	if retrieved.TokenID != "tid-1" {
		t.Errorf("expected TokenID=tid-1, got %s", retrieved.TokenID)
	}
	if retrieved.UserID != "user-1" {
		t.Errorf("expected UserID=user-1, got %s", retrieved.UserID)
	}
	if retrieved.Role != "admin" {
		t.Errorf("expected Role=admin, got %s", retrieved.Role)
	}
}

func TestMemoryRefreshTokenStore_RetrieveExpired(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := RefreshTokenEntry{
		TokenID:   "tid-expired",
		UserID:    "user-1",
		Role:      "viewer",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // already expired
	}
	store.Store(entry)

	retrieved, err := store.Retrieve("tid-expired")
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil for expired token")
	}
}

func TestMemoryRefreshTokenStore_RetrieveNonExistent(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	retrieved, err := store.Retrieve("nonexistent")
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil for non-existent token")
	}
}

func TestMemoryRefreshTokenStore_Revoke(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := RefreshTokenEntry{
		TokenID:   "tid-revoke",
		UserID:    "user-1",
		Role:      "dev",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	store.Store(entry)

	if err := store.Revoke("tid-revoke"); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	retrieved, _ := store.Retrieve("tid-revoke")
	if retrieved != nil {
		t.Error("expected nil after revocation")
	}
}

func TestMemoryRefreshTokenStore_RevokeNonExistent(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	err := store.Revoke("nonexistent")
	if err != nil {
		t.Fatalf("Revoke of non-existent token should not error, got: %v", err)
	}
}

func TestMemoryRefreshTokenStore_RevokeAllForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	now := time.Now()

	for i := 0; i < 3; i++ {
		entry := RefreshTokenEntry{
			TokenID:   "tid-" + string(rune('a'+i)),
			UserID:    "user-1",
			Role:      "dev",
			CreatedAt: now,
			ExpiresAt: now.Add(24 * time.Hour),
		}
		store.Store(entry)
	}

	// Add a token for a different user
	store.Store(RefreshTokenEntry{
		TokenID:   "tid-other",
		UserID:    "user-2",
		Role:      "viewer",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	})

	if err := store.RevokeAllForUser("user-1"); err != nil {
		t.Fatalf("RevokeAllForUser failed: %v", err)
	}

	count, _ := store.Count("user-1")
	if count != 0 {
		t.Errorf("expected 0 tokens for user-1, got %d", count)
	}

	// user-2 should still have their token
	count, _ = store.Count("user-2")
	if count != 1 {
		t.Errorf("expected 1 token for user-2, got %d", count)
	}
}

func TestMemoryRefreshTokenStore_Count(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	now := time.Now()

	count, err := store.Count("user-x")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	for i := 0; i < 5; i++ {
		store.Store(RefreshTokenEntry{
			TokenID:   "cnt-" + string(rune('0'+i)),
			UserID:    "user-x",
			Role:      "dev",
			CreatedAt: now,
			ExpiresAt: now.Add(24 * time.Hour),
		})
	}

	count, _ = store.Count("user-x")
	if count != 5 {
		t.Errorf("expected 5, got %d", count)
	}
}

func TestMemoryRefreshTokenStore_ListForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	now := time.Now()

	// No tokens yet
	entries, err := store.ListForUser("user-1")
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}
	if entries != nil {
		t.Error("expected nil for user with no tokens")
	}

	store.Store(RefreshTokenEntry{
		TokenID:   "l1",
		UserID:    "user-1",
		Role:      "admin",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	})
	store.Store(RefreshTokenEntry{
		TokenID:   "l2",
		UserID:    "user-1",
		Role:      "dev",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	})
	// Expired token should be filtered out
	store.Store(RefreshTokenEntry{
		TokenID:   "l3-expired",
		UserID:    "user-1",
		Role:      "viewer",
		CreatedAt: now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
	})

	entries, _ = store.ListForUser("user-1")
	if len(entries) != 2 {
		t.Errorf("expected 2 active entries, got %d", len(entries))
	}
}

func TestMemoryRefreshTokenStore_Cleanup(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	now := time.Now()

	store.Store(RefreshTokenEntry{
		TokenID:   "keep",
		UserID:    "user-1",
		Role:      "admin",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	})
	store.Store(RefreshTokenEntry{
		TokenID:   "expire",
		UserID:    "user-1",
		Role:      "dev",
		CreatedAt: now.Add(-2 * time.Hour),
		ExpiresAt: now.Add(-1 * time.Hour),
	})

	store.cleanup()

	count, _ := store.Count("user-1")
	if count != 1 {
		t.Errorf("expected 1 token after cleanup, got %d", count)
	}

	retrieved, _ := store.Retrieve("keep")
	if retrieved == nil {
		t.Error("expected 'keep' token to survive cleanup")
	}
}

func TestMemoryRefreshTokenStore_StartCleanup(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store.StartCleanup(ctx, 50*time.Millisecond)

	// Add an expired token
	store.Store(RefreshTokenEntry{
		TokenID:   "to-clean",
		UserID:    "user-1",
		Role:      "dev",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})

	// Wait for cleanup to run
	time.Sleep(150 * time.Millisecond)

	count, _ := store.Count("user-1")
	if count != 0 {
		t.Errorf("expected 0 tokens after background cleanup, got %d", count)
	}
}

// --- parseRefreshEntry tests ---

func TestParseRefreshEntry_Valid(t *testing.T) {
	now := time.Now()
	data := "user-1|admin|chrome|10.0.0.1|" + now.Format(time.RFC3339) + "|" + now.Add(24*time.Hour).Format(time.RFC3339)
	entry, err := parseRefreshEntry("tid-1", data)
	if err != nil {
		t.Fatalf("parseRefreshEntry failed: %v", err)
	}
	if entry.UserID != "user-1" {
		t.Errorf("expected UserID=user-1, got %s", entry.UserID)
	}
	if entry.Role != "admin" {
		t.Errorf("expected Role=admin, got %s", entry.Role)
	}
	if entry.DeviceInfo != "chrome" {
		t.Errorf("expected DeviceInfo=chrome, got %s", entry.DeviceInfo)
	}
	if entry.IPAddress != "10.0.0.1" {
		t.Errorf("expected IPAddress=10.0.0.1, got %s", entry.IPAddress)
	}
}

func TestParseRefreshEntry_InvalidTooFewParts(t *testing.T) {
	_, err := parseRefreshEntry("tid-1", "user-1|admin")
	if err == nil {
		t.Error("expected error for data with too few parts")
	}
}

func TestParseRefreshEntry_UnixTimestamp(t *testing.T) {
	ts := time.Now().Unix()
	expTs := time.Now().Add(24 * time.Hour).Unix()
	data := "user-1|admin|||" + itoa64(ts) + "|" + itoa64(expTs)
	entry, err := parseRefreshEntry("tid-1", data)
	if err != nil {
		t.Fatalf("parseRefreshEntry with unix timestamps failed: %v", err)
	}
	if entry.UserID != "user-1" {
		t.Errorf("expected UserID=user-1, got %s", entry.UserID)
	}
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
