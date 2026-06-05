package auth

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGenerateRefreshTokenID(t *testing.T) {
	id1, err := GenerateRefreshTokenID()
	if err != nil {
		t.Fatalf("GenerateRefreshTokenID() error = %v", err)
	}
	if len(id1) != 64 { // 32 bytes -> 64 hex chars
		t.Errorf("GenerateRefreshTokenID() len = %d, want 64", len(id1))
	}

	id2, err := GenerateRefreshTokenID()
	if err != nil {
		t.Fatalf("GenerateRefreshTokenID() error = %v", err)
	}
	if id1 == id2 {
		t.Error("GenerateRefreshTokenID() should generate unique IDs")
	}
}

func TestMemoryRefreshTokenStore_StoreAndRetrieve(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := RefreshTokenEntry{
		TokenID:   "token-1",
		UserID:    "user-1",
		Role:      "dev",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := store.Store(entry); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	retrieved, err := store.Retrieve("token-1")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("Retrieve() returned nil")
	}
	if retrieved.TokenID != entry.TokenID {
		t.Errorf("TokenID = %q, want %q", retrieved.TokenID, entry.TokenID)
	}
	if retrieved.UserID != entry.UserID {
		t.Errorf("UserID = %q, want %q", retrieved.UserID, entry.UserID)
	}
}

func TestMemoryRefreshTokenStore_RetrieveExpired(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := RefreshTokenEntry{
		TokenID:   "expired-token",
		UserID:    "user-1",
		Role:      "dev",
		CreatedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-24 * time.Hour), // expired 24 hours ago
	}

	if err := store.Store(entry); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	retrieved, err := store.Retrieve("expired-token")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if retrieved != nil {
		t.Error("Retrieve() should return nil for expired token")
	}
}

func TestMemoryRefreshTokenStore_RetrieveNonexistent(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	retrieved, err := store.Retrieve("nonexistent")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if retrieved != nil {
		t.Error("Retrieve() should return nil for nonexistent token")
	}
}

func TestMemoryRefreshTokenStore_Revoke(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := RefreshTokenEntry{
		TokenID:   "revoke-token",
		UserID:    "user-1",
		Role:      "dev",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := store.Store(entry); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	if err := store.Revoke("revoke-token"); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	retrieved, _ := store.Retrieve("revoke-token")
	if retrieved != nil {
		t.Error("Retrieve() should return nil after Revoke()")
	}
}

func TestMemoryRefreshTokenStore_RevokeAllForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	// Store multiple tokens for user-1
	for i := 0; i < 3; i++ {
		entry := RefreshTokenEntry{
			TokenID:   "user1-token-" + string(rune('a'+i)),
			UserID:    "user-1",
			Role:      "dev",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		if err := store.Store(entry); err != nil {
			t.Fatalf("Store() error = %v", err)
		}
	}

	// Store a token for user-2
	entry2 := RefreshTokenEntry{
		TokenID:   "user2-token",
		UserID:    "user-2",
		Role:      "dev",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := store.Store(entry2); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	if err := store.RevokeAllForUser("user-1"); err != nil {
		t.Fatalf("RevokeAllForUser() error = %v", err)
	}

	// user-1's tokens should be gone
	for _, id := range []string{"user1-token-a", "user1-token-b", "user1-token-c"} {
		retrieved, _ := store.Retrieve(id)
		if retrieved != nil {
			t.Errorf("token %s should be revoked", id)
		}
	}

	// user-2's token should remain
	retrieved, _ := store.Retrieve("user2-token")
	if retrieved == nil {
		t.Error("user-2's token should remain after RevokeAllForUser(user-1)")
	}
}

func TestMemoryRefreshTokenStore_Count(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	count, err := store.Count("user-1")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}

	// Add tokens
	for i := 0; i < 3; i++ {
		entry := RefreshTokenEntry{
			TokenID:   "user1-token-" + string(rune('a'+i)),
			UserID:    "user-1",
			Role:      "dev",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		store.Store(entry)
	}

	count, err = store.Count("user-1")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

func TestMemoryRefreshTokenStore_ListForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	// Add tokens
	for i := 0; i < 3; i++ {
		entry := RefreshTokenEntry{
			TokenID:   "user1-token-" + string(rune('a'+i)),
			UserID:    "user-1",
			Role:      "dev",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		store.Store(entry)
	}

	entries, err := store.ListForUser("user-1")
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("ListForUser() returned %d entries, want 3", len(entries))
	}

	// Expired tokens should be filtered out
	expiredEntry := RefreshTokenEntry{
		TokenID:   "expired-token",
		UserID:    "user-1",
		Role:      "dev",
		CreatedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	store.Store(expiredEntry)

	entries, _ = store.ListForUser("user-1")
	if len(entries) != 3 {
		t.Errorf("ListForUser() should filter expired tokens, got %d", len(entries))
	}
}

func TestMemoryRefreshTokenStore_ListForUserEmpty(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	entries, err := store.ListForUser("nonexistent")
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if entries != nil {
		t.Errorf("ListForUser() for nonexistent user = %v, want nil", entries)
	}
}

func TestMemoryRefreshTokenStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				entry := RefreshTokenEntry{
					TokenID:   "user-token",
					UserID:    "user-1",
					Role:      "dev",
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				store.Store(entry)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				store.Retrieve("user-token")
				store.Count("user-1")
			}
		}()
	}

	// Concurrent revoke
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			store.Revoke("user-token")
		}
	}()

	wg.Wait()
	// If we get here without deadlock or panic, test passes
}

func TestMemoryRefreshTokenStore_Cleanup(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	// Add expired tokens
	for i := 0; i < 5; i++ {
		entry := RefreshTokenEntry{
			TokenID:   "expired-token-" + string(rune('a'+i)),
			UserID:    "user-1",
			Role:      "dev",
			CreatedAt: time.Now().Add(-48 * time.Hour),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		store.Store(entry)
	}

	// Add valid token
	validEntry := RefreshTokenEntry{
		TokenID:   "valid-token",
		UserID:    "user-1",
		Role:      "dev",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	store.Store(validEntry)

	store.cleanup()

	count, _ := store.Count("user-1")
	if count != 1 {
		t.Errorf("after cleanup, Count() = %d, want 1 (only valid token)", count)
	}

	retrieved, _ := store.Retrieve("valid-token")
	if retrieved == nil {
		t.Error("valid token should remain after cleanup")
	}
}

func TestStartCleanup(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	ctx, cancel := context.WithCancel(context.Background())

	// Add expired tokens
	for i := 0; i < 3; i++ {
		entry := RefreshTokenEntry{
			TokenID:   "expired-" + string(rune('a'+i)),
			UserID:    "user-1",
			Role:      "dev",
			CreatedAt: time.Now().Add(-48 * time.Hour),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		store.Store(entry)
	}

	store.StartCleanup(ctx, 10*time.Millisecond)

	// Wait for cleanup to run
	time.Sleep(50 * time.Millisecond)
	cancel()

	count, _ := store.Count("user-1")
	if count != 0 {
		t.Errorf("after cleanup goroutine, Count() = %d, want 0", count)
	}
}

func TestSplitString(t *testing.T) {
	tests := []struct {
		input    string
		sep      rune
		n        int
		expected []string
	}{
		{"a|b|c", '|', 4, []string{"a", "b", "c"}},
		{"a|b|c", '|', 2, []string{"a", "b|c"}},
		{"a:b:c:d", ':', 3, []string{"a", "b", "c:d"}},
		{"no-separator", '-', 2, []string{"no", "separator"}},
		{"", '|', 2, []string{""}},
	}

	for _, tt := range tests {
		result := splitString(tt.input, tt.sep, tt.n)
		if len(result) != len(tt.expected) {
			t.Errorf("splitString(%q, %q, %d) returned %v, want %v",
				tt.input, string(tt.sep), tt.n, result, tt.expected)
			continue
		}
		for i, s := range result {
			if s != tt.expected[i] {
				t.Errorf("splitString(%q, %q, %d)[%d] = %q, want %q",
					tt.input, string(tt.sep), tt.n, i, s, tt.expected[i])
			}
		}
	}
}

func TestSplitN(t *testing.T) {
	// splitN is just an alias for splitString
	result := splitN("a|b|c", '|', 3)
	if len(result) != 3 {
		t.Errorf("splitN() len = %d, want 3", len(result))
	}
}

func TestParseUnix(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1609459200", 1609459200},
		{"0", 0},
		{"invalid", 0},
		{"-123", -123},
	}

	for _, tt := range tests {
		result := parseUnix(tt.input)
		if result != tt.expected {
			t.Errorf("parseUnix(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestParseRefreshEntry(t *testing.T) {
	data := "user-1|dev|Chrome|Win10|1609459200|1609545600"
	entry, err := parseRefreshEntry("token-123", data)
	if err != nil {
		t.Fatalf("parseRefreshEntry() error = %v", err)
	}

	if entry.TokenID != "token-123" {
		t.Errorf("TokenID = %q, want %q", entry.TokenID, "token-123")
	}
	if entry.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", entry.UserID, "user-1")
	}
	if entry.Role != "dev" {
		t.Errorf("Role = %q, want %q", entry.Role, "dev")
	}
	if entry.DeviceInfo != "Chrome" {
		t.Errorf("DeviceInfo = %q, want %q", entry.DeviceInfo, "Chrome")
	}
	if entry.IPAddress != "Win10" {
		t.Errorf("IPAddress = %q, want %q", entry.IPAddress, "Win10")
	}
}

func TestParseRefreshEntry_InvalidData(t *testing.T) {
	_, err := parseRefreshEntry("token", "invalid")
	if err == nil {
		t.Error("parseRefreshEntry() should return error for invalid data")
	}

	_, err = parseRefreshEntry("token", "a|b") // too few parts
	if err == nil {
		t.Error("parseRefreshEntry() should return error for insufficient parts")
	}
}

func TestParseRefreshEntry_WithRFC3339Timestamps(t *testing.T) {
	data := "user-1|dev|Chrome|Win10|2021-01-01T00:00:00Z|2021-01-02T00:00:00Z"
	entry, err := parseRefreshEntry("token-123", data)
	if err != nil {
		t.Fatalf("parseRefreshEntry() error = %v", err)
	}

	if entry.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", entry.UserID, "user-1")
	}
}

func TestRefreshTokenEntry_EmptyOptionalFields(t *testing.T) {
	data := "user-1|dev||" + "|1609459200|1609545600"
	entry, err := parseRefreshEntry("token-123", data)
	if err != nil {
		t.Fatalf("parseRefreshEntry() error = %v", err)
	}

	if entry.DeviceInfo != "" {
		t.Errorf("DeviceInfo = %q, want empty", entry.DeviceInfo)
	}
	if entry.IPAddress != "" {
		t.Errorf("IPAddress = %q, want empty", entry.IPAddress)
	}
}
