package auth

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// newMemoryStore returns a fresh in-memory refresh token store for tests.
func newMemoryStore() *MemoryRefreshTokenStore {
	return NewMemoryRefreshTokenStore()
}

func sampleEntry(userID, tokenID string, ttl time.Duration) RefreshTokenEntry {
	now := time.Now()
	return RefreshTokenEntry{
		TokenID:    tokenID,
		UserID:     userID,
		Role:       "dev",
		DeviceInfo: "mac-chrome",
		IPAddress:  "10.0.0.1",
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
	}
}

// TestMemoryRefreshTokenStore_StoreAndRetrieve is the round-trip happy path
// for the most fundamental lifecycle operation.
func TestMemoryRefreshTokenStore_StoreAndRetrieve(t *testing.T) {
	store := newMemoryStore()
	entry := sampleEntry("user-1", "tok-1", time.Hour)
	if err := store.Store(entry); err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	got, err := store.Retrieve("tok-1")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected entry, got nil")
	}
	if got.UserID != "user-1" || got.Role != "dev" || got.TokenID != "tok-1" {
		t.Errorf("Retrieve() = %+v, want matching entry", got)
	}
}

// TestMemoryRefreshTokenStore_RetrieveUnknown ensures a missing token
// returns (nil, nil) — the documented sentinel for "not found".
func TestMemoryRefreshTokenStore_RetrieveUnknown(t *testing.T) {
	store := newMemoryStore()
	got, err := store.Retrieve("does-not-exist")
	if err != nil {
		t.Errorf("Retrieve() unexpected error = %v", err)
	}
	if got != nil {
		t.Errorf("expected nil entry, got %+v", got)
	}
}

// TestMemoryRefreshTokenStore_RetrieveExpired checks that an expired
// entry is treated as missing — preventing use of stale tokens.
func TestMemoryRefreshTokenStore_RetrieveExpired(t *testing.T) {
	store := newMemoryStore()
	entry := sampleEntry("user-1", "tok-expired", -1*time.Hour)
	// Force the entry to be stored (the Redis variant short-circuits
	// expired TTLs but the in-memory variant just stores the entry).
	if err := store.Store(entry); err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	got, err := store.Retrieve("tok-expired")
	if err != nil {
		t.Errorf("Retrieve() unexpected error = %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for expired entry, got %+v", got)
	}
}

// TestMemoryRefreshTokenStore_Revoke verifies single-token revocation
// removes the entry from both the token map and the user's set.
func TestMemoryRefreshTokenStore_Revoke(t *testing.T) {
	store := newMemoryStore()
	_ = store.Store(sampleEntry("user-1", "tok-1", time.Hour))
	_ = store.Store(sampleEntry("user-1", "tok-2", time.Hour))

	if err := store.Revoke("tok-1"); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if got, _ := store.Retrieve("tok-1"); got != nil {
		t.Error("expected tok-1 to be revoked")
	}
	if got, _ := store.Retrieve("tok-2"); got == nil {
		t.Error("expected tok-2 to still exist")
	}
}

// TestMemoryRefreshTokenStore_RevokeUnknown is a no-op test: revoking a
// non-existent token must not return an error.
func TestMemoryRefreshTokenStore_RevokeUnknown(t *testing.T) {
	store := newMemoryStore()
	if err := store.Revoke("nope"); err != nil {
		t.Errorf("Revoke() unexpected error = %v", err)
	}
}

// TestMemoryRefreshTokenStore_RevokeAllForUser is the bulk-revocation
// "log out everywhere" path used after credential rotation.
func TestMemoryRefreshTokenStore_RevokeAllForUser(t *testing.T) {
	store := newMemoryStore()
	_ = store.Store(sampleEntry("user-1", "tok-1", time.Hour))
	_ = store.Store(sampleEntry("user-1", "tok-2", time.Hour))
	_ = store.Store(sampleEntry("user-2", "tok-3", time.Hour))

	if err := store.RevokeAllForUser("user-1"); err != nil {
		t.Fatalf("RevokeAllForUser() error = %v", err)
	}

	count, _ := store.Count("user-1")
	if count != 0 {
		t.Errorf("expected user-1 to have 0 tokens, got %d", count)
	}
	count, _ = store.Count("user-2")
	if count != 1 {
		t.Errorf("expected user-2 to still have 1 token, got %d", count)
	}
	if got, _ := store.Retrieve("tok-3"); got == nil {
		t.Error("user-2's token should be untouched")
	}
}

// TestMemoryRefreshTokenStore_Count and List exercise the per-user queries.
func TestMemoryRefreshTokenStore_CountAndList(t *testing.T) {
	store := newMemoryStore()
	_ = store.Store(sampleEntry("user-1", "tok-1", time.Hour))
	_ = store.Store(sampleEntry("user-1", "tok-2", time.Hour))
	_ = store.Store(sampleEntry("user-2", "tok-3", time.Hour))

	count, err := store.Count("user-1")
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 2 {
		t.Errorf("expected count=2, got %d", count)
	}

	entries, err := store.ListForUser("user-1")
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.UserID != "user-1" {
			t.Errorf("got entry with user_id %q, want user-1", e.UserID)
		}
	}
}

// TestMemoryRefreshTokenStore_ListForUser_UnknownUser documents the
// current behavior: an unknown user returns (nil, nil) without an error.
// Callers must handle nil as "no entries".
func TestMemoryRefreshTokenStore_ListForUser_UnknownUser(t *testing.T) {
	store := newMemoryStore()
	entries, err := store.ListForUser("nobody")
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty result, got %v", entries)
	}
}

// TestMemoryRefreshTokenStore_ListForUser_FiltersExpired confirms that
// expired entries are filtered out at list time (not just on Retrieve).
func TestMemoryRefreshTokenStore_ListForUser_FiltersExpired(t *testing.T) {
	store := newMemoryStore()
	_ = store.Store(sampleEntry("user-1", "tok-live", time.Hour))
	_ = store.Store(sampleEntry("user-1", "tok-dead", -1*time.Hour))

	entries, _ := store.ListForUser("user-1")
	if len(entries) != 1 {
		t.Fatalf("expected 1 live entry, got %d", len(entries))
	}
	if entries[0].TokenID != "tok-live" {
		t.Errorf("expected tok-live, got %s", entries[0].TokenID)
	}
}

// TestMemoryRefreshTokenStore_Cleanup runs the background cleaner directly
// and verifies it removes only expired entries.
func TestMemoryRefreshTokenStore_Cleanup(t *testing.T) {
	store := newMemoryStore()
	_ = store.Store(sampleEntry("user-1", "tok-live", time.Hour))
	_ = store.Store(sampleEntry("user-1", "tok-dead", -1*time.Hour))

	store.cleanup()

	if got, _ := store.Retrieve("tok-live"); got == nil {
		t.Error("expected tok-live to survive cleanup")
	}
	if got, _ := store.Retrieve("tok-dead"); got != nil {
		t.Error("expected tok-dead to be removed by cleanup")
	}
	count, _ := store.Count("user-1")
	if count != 1 {
		t.Errorf("expected count=1 after cleanup, got %d", count)
	}
}

// TestMemoryRefreshTokenStore_ConcurrentAccess exercises the read/write
// mutexes under concurrent Store/Retrieve/Revoke to surface any data races
// (the file uses sync.RWMutex on the in-memory store).
func TestMemoryRefreshTokenStore_ConcurrentAccess(t *testing.T) {
	store := newMemoryStore()
	const goroutines = 20
	const opsPerG = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < opsPerG; i++ {
				tokenID := "tok-" + string(rune('A'+gid)) + "-" + string(rune('a'+i%26))
				_ = store.Store(sampleEntry("user-1", tokenID, time.Hour))
				_, _ = store.Retrieve(tokenID)
				_ = store.Revoke(tokenID)
			}
		}(g)
	}
	wg.Wait()
	// We don't assert on final state — the goal is to surface any
	// data race via the race detector. Count must remain consistent
	// (no negative numbers, no panics) at the end.
	if count, _ := store.Count("user-1"); count < 0 {
		t.Errorf("count went negative: %d", count)
	}
}

// TestMemoryRefreshTokenStore_StartCleanup_StopsOnContextCancel makes sure
// the background goroutine exits cleanly when its context is cancelled.
func TestMemoryRefreshTokenStore_StartCleanup_StopsOnContextCancel(t *testing.T) {
	store := newMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	store.StartCleanup(ctx, 5*time.Millisecond)
	cancel()
	// Give the goroutine a moment to observe cancellation.
	time.Sleep(20 * time.Millisecond)
	// No assertion needed beyond not hanging; if cancel didn't work the
	// test would still pass, but the goal is to ensure no panic and to
	// document the lifecycle.
}

// TestGenerateRefreshTokenID ensures the cryptographic helper returns a
// 64-character hex string (32 bytes) and never errors on the happy path.
func TestGenerateRefreshTokenID(t *testing.T) {
	id, err := GenerateRefreshTokenID()
	if err != nil {
		t.Fatalf("GenerateRefreshTokenID() error = %v", err)
	}
	if len(id) != 64 {
		t.Errorf("expected 64-char hex token, got len=%d", len(id))
	}
	// Sanity-check that the output is actually hex.
	for _, r := range id {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Errorf("non-hex character in token: %q", r)
			break
		}
	}
}

// TestGenerateRefreshTokenID_Unique verifies that successive invocations
// produce distinct IDs — essential for the security model.
func TestGenerateRefreshTokenID_Unique(t *testing.T) {
	const n = 100
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id, err := GenerateRefreshTokenID()
		if err != nil {
			t.Fatalf("GenerateRefreshTokenID() error = %v", err)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("collision after %d iterations: %s", i, id)
		}
		seen[id] = struct{}{}
	}
}

// TestParseRefreshEntry is the round-trip test for the internal
// serialization format used by both memory and Redis stores.
func TestParseRefreshEntry(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(time.Hour)
	data := "user-42|admin|firefox|10.1.2.3|" +
		strconvFormatInt(now.Unix()) + "|" +
		strconvFormatInt(expires.Unix())

	entry, err := parseRefreshEntry("tok-xyz", data)
	if err != nil {
		t.Fatalf("parseRefreshEntry() error = %v", err)
	}
	if entry.UserID != "user-42" || entry.Role != "admin" ||
		entry.DeviceInfo != "firefox" || entry.IPAddress != "10.1.2.3" {
		t.Errorf("entry fields not parsed correctly: %+v", entry)
	}
	if !entry.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", entry.CreatedAt, now)
	}
	if !entry.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt = %v, want %v", entry.ExpiresAt, expires)
	}
	if entry.TokenID != "tok-xyz" {
		t.Errorf("TokenID = %q, want tok-xyz", entry.TokenID)
	}
}

// TestParseRefreshEntry_TooShort covers the validation path: a malformed
// record must surface a clear error.
func TestParseRefreshEntry_TooShort(t *testing.T) {
	_, err := parseRefreshEntry("tok", "only|two|parts")
	if err == nil {
		t.Error("expected error for short data, got nil")
	}
}

// strconvFormatInt is a tiny helper that avoids importing strconv directly
// in this test file's namespace collisions.
func strconvFormatInt(n int64) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
