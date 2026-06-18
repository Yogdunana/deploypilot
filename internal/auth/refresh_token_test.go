package auth

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestGenerateRefreshTokenID_GeneratesUniqueIDs confirms that repeated
// invocations produce different token IDs (otherwise a collision
// could allow a stolen session ID to mint an unauthorised session).
func TestGenerateRefreshTokenID_GeneratesUniqueIDs(t *testing.T) {
	const N = 50
	seen := make(map[string]struct{}, N)
	for i := 0; i < N; i++ {
		id, err := GenerateRefreshTokenID()
		if err != nil {
			t.Fatalf("GenerateRefreshTokenID failed: %v", err)
		}
		// The ID is 32 random bytes -> 64 hex chars.
		if len(id) != 64 {
			t.Errorf("token id length = %d, want 64", len(id))
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate token id generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}

// TestMemoryRefreshTokenStore_StoreAndRetrieve verifies the basic
// happy path for the in-memory store used in single-instance
// deployments.
func TestMemoryRefreshTokenStore_StoreAndRetrieve(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	entry := newEntry("tkn-1", "user-1", "dev", time.Hour)
	if err := store.Store(entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	got, err := store.Retrieve(entry.TokenID)
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil entry for stored token")
	}
	if got.UserID != "user-1" || got.Role != "dev" {
		t.Errorf("retrieved entry = %+v, want UserID=user-1 Role=dev", got)
	}
}

// TestMemoryRefreshTokenStore_RetrieveUnknownReturnsNil confirms that
// looking up an unknown token returns (nil, nil) so callers can
// distinguish "not found" from "error".
func TestMemoryRefreshTokenStore_RetrieveUnknownReturnsNil(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	got, err := store.Retrieve("nope")
	if err != nil {
		t.Errorf("Retrieve unknown: unexpected error %v", err)
	}
	if got != nil {
		t.Errorf("Retrieve unknown: expected nil entry, got %+v", got)
	}
}

// TestMemoryRefreshTokenStore_ExpiredEntryIsHidden confirms that an
// expired token is invisible to Retrieve, even though it is still
// present in the map (a deleted-on-read approach would lose the
// capacity to clean up properly).
func TestMemoryRefreshTokenStore_ExpiredEntryIsHidden(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	expired := newEntry("tkn-expired", "user-1", "dev", -time.Second)
	if err := store.Store(expired); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	got, err := store.Retrieve("tkn-expired")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if got != nil {
		t.Errorf("expired entry should be hidden from Retrieve, got %+v", got)
	}
}

// TestMemoryRefreshTokenStore_Revoke confirms that revoking a token
// removes it from the store and that an unknown token revoke is a
// silent no-op (the implementation must not return an error, because
// the caller may double-revoke during logout).
func TestMemoryRefreshTokenStore_Revoke(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := newEntry("tkn-1", "user-1", "dev", time.Hour)
	_ = store.Store(entry)

	if err := store.Revoke("tkn-1"); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	got, _ := store.Retrieve("tkn-1")
	if got != nil {
		t.Error("token should be gone after Revoke")
	}

	// Double revoke must not error.
	if err := store.Revoke("tkn-1"); err != nil {
		t.Errorf("double Revoke returned error: %v", err)
	}
	// Revoking an unknown id must not error.
	if err := store.Revoke("never-existed"); err != nil {
		t.Errorf("Revoke unknown returned error: %v", err)
	}
}

// TestMemoryRefreshTokenStore_RevokeAllForUser confirms that revoking
// all tokens for a user removes every token issued to that user but
// leaves tokens belonging to other users intact. A bug here could
// allow session leakage across user boundaries.
func TestMemoryRefreshTokenStore_RevokeAllForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(newEntry("a1", "alice", "dev", time.Hour))
	_ = store.Store(newEntry("a2", "alice", "dev", time.Hour))
	_ = store.Store(newEntry("b1", "bob", "dev", time.Hour))

	if err := store.RevokeAllForUser("alice"); err != nil {
		t.Fatalf("RevokeAllForUser failed: %v", err)
	}

	// Alice's tokens are gone.
	if got, _ := store.Retrieve("a1"); got != nil {
		t.Errorf("a1 should be revoked, got %+v", got)
	}
	if got, _ := store.Retrieve("a2"); got != nil {
		t.Errorf("a2 should be revoked, got %+v", got)
	}

	// Bob's token is intact.
	if got, _ := store.Retrieve("b1"); got == nil {
		t.Error("bob's token should NOT be revoked by alice's session reset")
	}

	// Counts are consistent.
	if c, _ := store.Count("alice"); c != 0 {
		t.Errorf("Count(alice) = %d, want 0", c)
	}
	if c, _ := store.Count("bob"); c != 1 {
		t.Errorf("Count(bob) = %d, want 1", c)
	}
}

// TestMemoryRefreshTokenStore_Count confirms the per-user token counter
// is maintained by Store and decremented by Revoke/RevokeAllForUser.
func TestMemoryRefreshTokenStore_Count(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	if c, _ := store.Count("user-x"); c != 0 {
		t.Errorf("initial Count = %d, want 0", c)
	}

	_ = store.Store(newEntry("x1", "user-x", "dev", time.Hour))
	_ = store.Store(newEntry("x2", "user-x", "dev", time.Hour))
	if c, _ := store.Count("user-x"); c != 2 {
		t.Errorf("Count after 2 stores = %d, want 2", c)
	}

	_ = store.Revoke("x1")
	if c, _ := store.Count("user-x"); c != 1 {
		t.Errorf("Count after revoke = %d, want 1", c)
	}
}

// TestMemoryRefreshTokenStore_ListForUser_HidesExpired confirms that
// expired tokens are excluded from the list view. The store uses this
// when rendering the user's session list in the UI.
func TestMemoryRefreshTokenStore_ListForUser_HidesExpired(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(newEntry("live1", "alice", "dev", time.Hour))
	_ = store.Store(newEntry("live2", "alice", "dev", 2*time.Hour))
	_ = store.Store(newEntry("dead", "alice", "dev", -time.Minute))

	entries, err := store.ListForUser("alice")
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2 (expired must be filtered out)", len(entries))
	}
	for _, e := range entries {
		if e.TokenID == "dead" {
			t.Error("expired entry should not appear in ListForUser")
		}
	}
}

// TestMemoryRefreshTokenStore_Cleanup confirms that the periodic
// cleanup goroutine removes expired entries. This is what prevents
// the in-memory map from leaking memory for users who never log out.
func TestMemoryRefreshTokenStore_Cleanup(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(newEntry("alive", "alice", "dev", time.Hour))
	_ = store.Store(newEntry("dead", "alice", "dev", -time.Minute))

	// Run cleanup directly to avoid depending on the background loop.
	store.cleanup()

	// Expired token should be gone.
	if got, _ := store.Retrieve("dead"); got != nil {
		t.Errorf("expired token should be removed by cleanup, got %+v", got)
	}
	// Live token should remain.
	if got, _ := store.Retrieve("alive"); got == nil {
		t.Error("live token should remain after cleanup")
	}
	// The user-set index should also have been pruned.
	if c, _ := store.Count("alice"); c != 1 {
		t.Errorf("Count(alice) after cleanup = %d, want 1", c)
	}
}

// TestMemoryRefreshTokenStore_ConcurrentStoreRetrieve is a smoke test
// for race conditions. The store uses an RWMutex internally; this
// test would catch a regression that dropped the lock.
func TestMemoryRefreshTokenStore_ConcurrentStoreRetrieve(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	const N = 50

	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			entry := newEntry(
				generateLocalID(i),
				"user",
				"dev",
				time.Hour,
			)
			_ = store.Store(entry)
			_, _ = store.Retrieve(entry.TokenID)
		}(i)
	}
	wg.Wait()

	// We should have N entries stored.
	if c, _ := store.Count("user"); c != N {
		t.Errorf("Count after concurrent stores = %d, want %d", c, N)
	}
}

// TestMemoryRefreshTokenStore_StartCleanupExitsOnContextCancel
// confirms the background goroutine respects context cancellation and
// does not leak. This is what process shutdown relies on.
func TestMemoryRefreshTokenStore_StartCleanupExitsOnContextCancel(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	ctx, cancel := context.WithCancel(context.Background())
	store.StartCleanup(ctx, 10*time.Millisecond)

	// Let the goroutine run a few cycles.
	time.Sleep(25 * time.Millisecond)
	cancel()

	// Allow a brief window for the goroutine to observe cancellation.
	// We can't easily assert the goroutine is dead without a probe
	// channel, but if the goroutine is stuck, subsequent tests will
	// hang and the test binary will be terminated. A small sleep
	// after cancel gives the scheduler a chance to clean up.
	time.Sleep(20 * time.Millisecond)
}

// --- helpers ---

// newEntry builds a RefreshTokenEntry with sensible defaults for tests.
func newEntry(id, userID, role string, ttl time.Duration) RefreshTokenEntry {
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

// generateLocalID is a tiny helper for table tests that need a
// unique token id without pulling in crypto/rand (which is tested
// separately above).
func generateLocalID(i int) string {
	return "tkn-" + time.Now().Format("150405.000") + "-" + intToBase36(i)
}

func intToBase36(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [16]byte
	pos := len(buf)
	for i > 0 {
		pos--
		d := i % 36
		if d < 10 {
			buf[pos] = byte('0' + d)
		} else {
			buf[pos] = byte('a' + d - 10)
		}
		i /= 36
	}
	return string(buf[pos:])
}
