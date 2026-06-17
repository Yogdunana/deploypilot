package auth

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// ===================== GenerateRefreshTokenID =====================

func TestGenerateRefreshTokenID_Uniqueness(t *testing.T) {
	const n = 1000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id, err := GenerateRefreshTokenID()
		if err != nil {
			t.Fatalf("GenerateRefreshTokenID failed: %v", err)
		}
		// ID is hex-encoded 32 random bytes -> 64 chars
		if len(id) != 64 {
			t.Errorf("expected 64-char id, got %d chars: %s", len(id), id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate token id generated: %s", id)
		}
		seen[id] = struct{}{}
	}
}

// ===================== MemoryRefreshTokenStore =====================

func newTestEntry(userID, tokenID string, expiresAt time.Time) RefreshTokenEntry {
	return RefreshTokenEntry{
		TokenID:    tokenID,
		UserID:     userID,
		Role:       "admin",
		DeviceInfo: "unit-test-device",
		IPAddress:  "127.0.0.1",
		CreatedAt:  time.Now().Add(-time.Minute),
		ExpiresAt:  expiresAt,
	}
}

func TestMemoryStore_StoreAndRetrieve(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := newTestEntry("user-1", "tok-1", time.Now().Add(time.Hour))

	if err := store.Store(entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	got, err := store.Retrieve("tok-1")
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if got == nil {
		t.Fatal("Retrieve returned nil for stored entry")
	}
	if got.UserID != entry.UserID {
		t.Errorf("expected userID %q, got %q", entry.UserID, got.UserID)
	}
	if got.Role != entry.Role {
		t.Errorf("expected role %q, got %q", entry.Role, got.Role)
	}
}

func TestMemoryStore_Retrieve_NotFound(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	got, err := store.Retrieve("missing")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil entry for missing id, got %+v", got)
	}
}

func TestMemoryStore_Retrieve_ExpiredHidesEntry(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := newTestEntry("user-1", "tok-exp", time.Now().Add(-time.Hour))
	if err := store.Store(entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	got, err := store.Retrieve("tok-exp")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for expired token, got %+v", got)
	}
}

func TestMemoryStore_Revoke(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := newTestEntry("user-1", "tok-rev", time.Now().Add(time.Hour))
	if err := store.Store(entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	if err := store.Revoke("tok-rev"); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
	got, _ := store.Retrieve("tok-rev")
	if got != nil {
		t.Errorf("expected nil after revoke, got %+v", got)
	}
	// Revoking a non-existent id should be a no-op, not an error.
	if err := store.Revoke("does-not-exist"); err != nil {
		t.Errorf("Revoke of missing id should be a no-op, got error: %v", err)
	}
}

func TestMemoryStore_RevokeAllForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	exp := time.Now().Add(time.Hour)
	if err := store.Store(newTestEntry("u1", "tok-a", exp)); err != nil {
		t.Fatal(err)
	}
	if err := store.Store(newTestEntry("u1", "tok-b", exp)); err != nil {
		t.Fatal(err)
	}
	if err := store.Store(newTestEntry("u2", "tok-c", exp)); err != nil {
		t.Fatal(err)
	}

	if err := store.RevokeAllForUser("u1"); err != nil {
		t.Fatalf("RevokeAllForUser failed: %v", err)
	}

	for _, tid := range []string{"tok-a", "tok-b"} {
		got, _ := store.Retrieve(tid)
		if got != nil {
			t.Errorf("expected %s to be revoked, got %+v", tid, got)
		}
	}
	// u2's token should be untouched.
	got, _ := store.Retrieve("tok-c")
	if got == nil {
		t.Error("u2's token was incorrectly revoked")
	}
}

func TestMemoryStore_Count(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	exp := time.Now().Add(time.Hour)
	_ = store.Store(newTestEntry("u1", "tok-1", exp))
	_ = store.Store(newTestEntry("u1", "tok-2", exp))
	_ = store.Store(newTestEntry("u2", "tok-3", exp))

	n, err := store.Count("u1")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if n != 2 {
		t.Errorf("expected Count(u1)=2, got %d", n)
	}
	n, _ = store.Count("nonexistent")
	if n != 0 {
		t.Errorf("expected Count(nonexistent)=0, got %d", n)
	}
}

func TestMemoryStore_ListForUser_FiltersExpired(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(newTestEntry("u1", "live", time.Now().Add(time.Hour)))
	_ = store.Store(newTestEntry("u1", "dead", time.Now().Add(-time.Minute)))

	entries, err := store.ListForUser("u1")
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 active entry, got %d", len(entries))
	}
	if entries[0].TokenID != "live" {
		t.Errorf("expected 'live' entry, got %s", entries[0].TokenID)
	}
}

func TestMemoryStore_ListForUser_Empty(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entries, err := store.ListForUser("nobody")
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty result, got %v", entries)
	}
}

func TestMemoryStore_Cleanup_RemovesExpired(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(newTestEntry("u1", "live", time.Now().Add(time.Hour)))
	_ = store.Store(newTestEntry("u1", "dead1", time.Now().Add(-time.Hour)))
	_ = store.Store(newTestEntry("u1", "dead2", time.Now().Add(-time.Minute)))

	store.cleanup()

	if got, _ := store.Retrieve("dead1"); got != nil {
		t.Error("dead1 should have been cleaned up")
	}
	if got, _ := store.Retrieve("dead2"); got != nil {
		t.Error("dead2 should have been cleaned up")
	}
	if got, _ := store.Retrieve("live"); got == nil {
		t.Error("live token should still be present")
	}
}

func TestMemoryStore_StartCleanup_StopsOnContextCancel(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	ctx, cancel := context.WithCancel(context.Background())
	store.StartCleanup(ctx, 10*time.Millisecond)
	cancel()
	// Give the goroutine time to exit. Use a short wait and the absence
	// of a panic to verify it terminated cleanly.
	time.Sleep(50 * time.Millisecond)
}

func TestMemoryStore_ConcurrentSafety(t *testing.T) {
	// Verify Store + Retrieve + Revoke are safe under concurrent use.
	// Run with -race to detect any data races.
	store := NewMemoryRefreshTokenStore()
	exp := time.Now().Add(time.Hour)

	var wg sync.WaitGroup
	const writers, readers = 20, 50
	wg.Add(writers + readers)

	for i := 0; i < writers; i++ {
		go func(i int) {
			defer wg.Done()
			id := "tok-" + strconv.Itoa(i)
			_ = store.Store(newTestEntry("u1", id, exp))
		}(i)
	}
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			_, _ = store.Count("u1")
			_, _ = store.ListForUser("u1")
		}()
	}
	wg.Wait()

	n, err := store.Count("u1")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if n != writers {
		t.Errorf("expected %d tokens stored, got %d", writers, n)
	}
}

// ===================== parseRefreshEntry (parser round-trip) =====================

func TestParseRefreshEntry_ValidData(t *testing.T) {
	created := time.Now().Add(-time.Minute).UTC().Truncate(time.Second)
	expires := created.Add(time.Hour)
	data := strings.Join([]string{
		"user-1",
		"admin",
		"device-info",
		"10.0.0.1",
		strconv.FormatInt(created.Unix(), 10),
		strconv.FormatInt(expires.Unix(), 10),
	}, "|")

	entry, err := parseRefreshEntry("tok-x", data)
	if err != nil {
		t.Fatalf("parseRefreshEntry failed: %v", err)
	}
	if entry.TokenID != "tok-x" {
		t.Errorf("expected TokenID=tok-x, got %s", entry.TokenID)
	}
	if entry.UserID != "user-1" {
		t.Errorf("expected UserID=user-1, got %s", entry.UserID)
	}
	if entry.Role != "admin" {
		t.Errorf("expected Role=admin, got %s", entry.Role)
	}
	if entry.DeviceInfo != "device-info" {
		t.Errorf("expected DeviceInfo=device-info, got %s", entry.DeviceInfo)
	}
	if entry.IPAddress != "10.0.0.1" {
		t.Errorf("expected IPAddress=10.0.0.1, got %s", entry.IPAddress)
	}
	if entry.ExpiresAt.Unix() != expires.Unix() {
		t.Errorf("expected ExpiresAt=%v, got %v", expires.Unix(), entry.ExpiresAt.Unix())
	}
}

func TestParseRefreshEntry_PreservesMoreThan5Separators(t *testing.T) {
	// splitString is limited to n=6 parts, so an IP address containing '|'
	// must be preserved verbatim inside the IPAddress field.
	data := "user|admin|device|10.0.0.1|123|456"
	entry, err := parseRefreshEntry("tid", data)
	if err != nil {
		t.Fatalf("parseRefreshEntry failed: %v", err)
	}
	if entry.UserID != "user" {
		t.Errorf("expected UserID=user, got %q", entry.UserID)
	}
	// The trailing portion after the 5th separator becomes the IPAddress
	// field because of how splitString caps the number of parts.
	if !strings.HasPrefix(entry.IPAddress, "10.0.0.1") {
		t.Errorf("expected IPAddress to start with 10.0.0.1, got %q", entry.IPAddress)
	}
}

func TestParseRefreshEntry_TooShort(t *testing.T) {
	if _, err := parseRefreshEntry("tid", "only|two|parts"); err == nil {
		t.Error("expected error for malformed data with too few parts")
	}
}

// ===================== Redis store via miniredis =====================

func newTestRedisStore(t *testing.T) (*RedisRefreshTokenStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewRedisRefreshTokenStore(rdb), mr
}

func TestRedisStore_StoreSkipsExpiredEntry(t *testing.T) {
	store, _ := newTestRedisStore(t)
	past := newTestEntry("u1", "tok-past", time.Now().Add(-time.Hour))
	if err := store.Store(past); err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	// The store should not write anything for an already-expired token.
	got, err := store.Retrieve("tok-past")
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for expired entry, got %+v", got)
	}
}

func TestRedisStore_StoreAndRetrieve(t *testing.T) {
	store, _ := newTestRedisStore(t)
	entry := newTestEntry("u1", "tok-1", time.Now().Add(time.Hour))
	if err := store.Store(entry); err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	got, err := store.Retrieve("tok-1")
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected entry, got nil")
	}
	if got.UserID != "u1" {
		t.Errorf("expected userID=u1, got %s", got.UserID)
	}
}

func TestRedisStore_RetrieveMissing(t *testing.T) {
	store, _ := newTestRedisStore(t)
	got, err := store.Retrieve("does-not-exist")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing id, got %+v", got)
	}
}

func TestRedisStore_Revoke(t *testing.T) {
	store, _ := newTestRedisStore(t)
	exp := time.Now().Add(time.Hour)
	_ = store.Store(newTestEntry("u1", "tok-a", exp))
	if err := store.Revoke("tok-a"); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
	got, _ := store.Retrieve("tok-a")
	if got != nil {
		t.Errorf("expected nil after revoke, got %+v", got)
	}
	// Revoking unknown id should not error.
	if err := store.Revoke("not-here"); err != nil {
		t.Errorf("Revoke of missing id should be a no-op, got error: %v", err)
	}
}

func TestRedisStore_RevokeCorruptData(t *testing.T) {
	// If the stored value is corrupt, Revoke should bubble up the error
	// rather than silently succeeding.
	store, mr := newTestRedisStore(t)
	mr.Set("refresh:tok-bad", "garbage")
	err := store.Revoke("tok-bad")
	if err == nil {
		t.Error("expected error when stored data is corrupt")
	}
}

func TestRedisStore_RevokeAllForUser(t *testing.T) {
	store, _ := newTestRedisStore(t)
	exp := time.Now().Add(time.Hour)
	_ = store.Store(newTestEntry("u1", "tok-a", exp))
	_ = store.Store(newTestEntry("u1", "tok-b", exp))
	_ = store.Store(newTestEntry("u2", "tok-c", exp))

	if err := store.RevokeAllForUser("u1"); err != nil {
		t.Fatalf("RevokeAllForUser failed: %v", err)
	}
	for _, tid := range []string{"tok-a", "tok-b"} {
		got, _ := store.Retrieve(tid)
		if got != nil {
			t.Errorf("expected %s revoked, got %+v", tid, got)
		}
	}
	got, _ := store.Retrieve("tok-c")
	if got == nil {
		t.Error("u2's token was incorrectly revoked")
	}
}

func TestRedisStore_Count(t *testing.T) {
	store, _ := newTestRedisStore(t)
	exp := time.Now().Add(time.Hour)
	_ = store.Store(newTestEntry("u1", "tok-1", exp))
	_ = store.Store(newTestEntry("u1", "tok-2", exp))

	n, err := store.Count("u1")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if n != 2 {
		t.Errorf("expected Count=2, got %d", n)
	}
}

func TestRedisStore_ListForUser_SkipsExpiredAndMissing(t *testing.T) {
	store, mr := newTestRedisStore(t)
	exp := time.Now().Add(time.Hour)
	_ = store.Store(newTestEntry("u1", "live", exp))
	// Add a stale member to the set without a corresponding key
	// (simulates an entry that expired and was garbage-collected).
	mr.SAdd("user_refresh:u1", "ghost")
	// Add an entry whose key is corrupt; should be skipped, not returned.
	mr.Set("refresh:corrupt", "garbage")
	mr.SAdd("user_refresh:u1", "corrupt")

	entries, err := store.ListForUser("u1")
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 valid entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].TokenID != "live" {
		t.Errorf("expected 'live' entry, got %s", entries[0].TokenID)
	}
}

func TestRedisStore_ListForUser_Empty(t *testing.T) {
	store, _ := newTestRedisStore(t)
	entries, err := store.ListForUser("nobody")
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty result, got %v", entries)
	}
}
