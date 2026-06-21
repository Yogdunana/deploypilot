package auth

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// makeEntry builds a RefreshTokenEntry for tests. Tests that need
// control over CreatedAt/ExpiresAt use this helper rather than
// duplicating literal initializers.
func makeEntry(id, userID, role string, ttl time.Duration) RefreshTokenEntry {
	now := time.Now()
	return RefreshTokenEntry{
		TokenID:    id,
		UserID:     userID,
		Role:       role,
		DeviceInfo: "test-device",
		IPAddress:  "127.0.0.1",
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
	}
}

// TestGenerateRefreshTokenID_Unique exercises the basic property of
// the token generator: each call must produce a different value. A
// regression to a constant or low-entropy source would be caught here.
func TestGenerateRefreshTokenID_Unique(t *testing.T) {
	a, err := GenerateRefreshTokenID()
	if err != nil {
		t.Fatalf("GenerateRefreshTokenID: %v", err)
	}
	if a == "" {
		t.Fatal("expected non-empty token ID")
	}
	if strings.Contains(a, " ") || strings.Contains(a, "\n") {
		t.Errorf("token ID should not contain whitespace, got %q", a)
	}
	for i := 0; i < 50; i++ {
		b, err := GenerateRefreshTokenID()
		if err != nil {
			t.Fatalf("GenerateRefreshTokenID: %v", err)
		}
		if b == a {
			t.Fatalf("duplicate token ID after 50 iterations: %q", b)
		}
	}
}

// TestMemoryRefreshTokenStore_StoreAndRetrieve covers the basic
// round-trip in the in-memory implementation.
func TestMemoryRefreshTokenStore_StoreAndRetrieve(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := makeEntry("t1", "user-1", "admin", time.Minute)
	if err := store.Store(entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := store.Retrieve("t1")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if got == nil {
		t.Fatal("Retrieve returned nil for stored token")
	}
	if got.UserID != "user-1" || got.Role != "admin" {
		t.Errorf("Retrieve returned %+v, want user-1/admin", got)
	}
}

// TestMemoryRefreshTokenStore_RetrieveMissing documents the
// nil-on-miss contract used by callers to distinguish "not present"
// from "error".
func TestMemoryRefreshTokenStore_RetrieveMissing(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	got, err := store.Retrieve("does-not-exist")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil entry for missing token, got %+v", got)
	}
}

// TestMemoryRefreshTokenStore_RetrieveExpired asserts that an
// expired token is not returned, even though it has not been
// garbage-collected. A regression here would leak session
// privileges after expiry.
func TestMemoryRefreshTokenStore_RetrieveExpired(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := makeEntry("t1", "user-1", "admin", -time.Second) // already expired
	if err := store.Store(entry); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got, err := store.Retrieve("t1")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for expired token, got %+v", got)
	}
}

// TestMemoryRefreshTokenStore_Revoke is the core logout contract:
// after Revoke, Retrieve must return nil, and Count for the owning
// user must drop.
func TestMemoryRefreshTokenStore_Revoke(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(makeEntry("t1", "user-1", "admin", time.Minute))
	_ = store.Store(makeEntry("t2", "user-1", "admin", time.Minute))

	if err := store.Revoke("t1"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if got, _ := store.Retrieve("t1"); got != nil {
		t.Error("Revoke did not remove token from store")
	}
	// t2 is still valid and still owned by user-1.
	count, err := store.Count("user-1")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("Count(user-1) = %d, want 1", count)
	}
}

// TestMemoryRefreshTokenStore_RevokeMissing is a no-op test: revoking
// a non-existent token must not error.
func TestMemoryRefreshTokenStore_RevokeMissing(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	if err := store.Revoke("missing"); err != nil {
		t.Errorf("Revoke(missing) = %v, want nil", err)
	}
}

// TestMemoryRefreshTokenStore_RevokeAllForUser is the "log out
// everywhere" path: every token issued to a user must be removed in
// one call, while other users' tokens are untouched.
func TestMemoryRefreshTokenStore_RevokeAllForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(makeEntry("a1", "alice", "admin", time.Minute))
	_ = store.Store(makeEntry("a2", "alice", "admin", time.Minute))
	_ = store.Store(makeEntry("b1", "bob", "dev", time.Minute))

	if err := store.RevokeAllForUser("alice"); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}
	if c, _ := store.Count("alice"); c != 0 {
		t.Errorf("Count(alice) = %d, want 0", c)
	}
	if c, _ := store.Count("bob"); c != 1 {
		t.Errorf("Count(bob) = %d, want 1 (other user must not be affected)", c)
	}
}

// TestMemoryRefreshTokenStore_ListForUser_OnlyActive covers the
// filter that hides expired tokens. A session list showing
// already-expired entries would mislead admins during a security
// review.
func TestMemoryRefreshTokenStore_ListForUser_OnlyActive(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(makeEntry("live", "user-1", "admin", time.Minute))
	_ = store.Store(makeEntry("dead", "user-1", "admin", -time.Second))

	got, err := store.ListForUser("user-1")
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListForUser returned %d entries, want 1 (expired must be filtered)", len(got))
	}
	if got[0].TokenID != "live" {
		t.Errorf("ListForUser returned %q, want %q", got[0].TokenID, "live")
	}
}

// TestMemoryRefreshTokenStore_ListForUser_Empty documents the
// nil-on-empty contract used elsewhere in the codebase.
func TestMemoryRefreshTokenStore_ListForUser_Empty(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	got, err := store.ListForUser("no-such-user")
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for unknown user, got %+v", got)
	}
}

// TestMemoryRefreshTokenStore_ConcurrentStoreAndRevoke runs the
// store and revoke paths from many goroutines to surface any data
// race that would corrupt the in-memory maps. A failure here would
// mean concurrent logins/logouts could lose or duplicate tokens.
func TestMemoryRefreshTokenStore_ConcurrentStoreAndRevoke(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	const goroutines = 16
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Writers store tokens in parallel.
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				id := makeEntry(string(rune('A'+g))+"-"+string(rune('a'+i%26)), "user", "admin", time.Minute)
				_ = store.Store(id)
			}
		}(g)
	}

	// Concurrent revokes and retrieves run alongside writers.
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				id := string(rune('A'+g)) + "-" + string(rune('a'+i%26))
				_, _ = store.Retrieve(id)
				_ = store.Revoke(id)
			}
		}(g)
	}

	wg.Wait()
}

// TestMemoryRefreshTokenStore_StartCleanup verifies the background
// goroutine that periodically evicts expired tokens. We override
// the ticker with a very short interval so the test finishes quickly
// and remains deterministic.
func TestMemoryRefreshTokenStore_StartCleanup(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(makeEntry("short", "user-1", "admin", -time.Second))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store.StartCleanup(ctx, 10*time.Millisecond)

	// Wait long enough for the cleanup goroutine to run at least once.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if c, _ := store.Count("user-1"); c == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("StartCleanup did not evict expired token within 500ms")
}

// TestParseRefreshEntry is a focused round-trip on the private parser
// that the Redis implementation uses to decode stored values. The
// helper is unexported, but the memory store does not use it; this
// test is therefore skipped to avoid coupling. It is kept in source
// so future readers can re-enable it when a stub Redis is available.
func TestParseRefreshEntry(t *testing.T) {
	t.Skip("parseRefreshEntry is exercised through the Redis implementation; requires a stub")
}
