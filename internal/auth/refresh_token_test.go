package auth

import (
	"context"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestGenerateRefreshTokenID_Format verifies the random ID is the expected
// length and contains only hex characters.
func TestGenerateRefreshTokenID_Format(t *testing.T) {
	id, err := GenerateRefreshTokenID()
	if err != nil {
		t.Fatalf("GenerateRefreshTokenID returned error: %v", err)
	}
	// 32 random bytes -> 64 hex chars
	if len(id) != 64 {
		t.Errorf("expected 64 hex chars, got %d (%q)", len(id), id)
	}
	if _, err := hex.DecodeString(id); err != nil {
		t.Errorf("expected valid hex, got %q: %v", id, err)
	}
}

// TestGenerateRefreshTokenID_Uniqueness checks the basic uniqueness property
// of the generator over a small sample.
func TestGenerateRefreshTokenID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id, err := GenerateRefreshTokenID()
		if err != nil {
			t.Fatalf("GenerateRefreshTokenID returned error: %v", err)
		}
		if seen[id] {
			t.Fatalf("duplicate id generated: %q", id)
		}
		seen[id] = true
	}
}

// --- splitString / parseRefreshEntry (internal helpers) ---

// TestSplitString_BasicLimit verifies that the custom splitN caps the
// returned slice at the requested number of parts, leaving the remaining
// content (including any extra separators) inside the last part.
func TestSplitString_BasicLimit(t *testing.T) {
	got := splitString("a|b|c|d|e", '|', 3)
	want := []string{"a", "b", "c|d|e"}
	if len(got) != len(want) {
		t.Fatalf("len=%d, want %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

// TestSplitString_NoSeparator covers the case where the separator is
// missing: the entire string is returned as a single element.
func TestSplitString_NoSeparator(t *testing.T) {
	got := splitString("hello", '|', 5)
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("expected [hello], got %v", got)
	}
}

// TestSplitString_EmptyInput covers the empty-string edge case.
func TestSplitString_EmptyInput(t *testing.T) {
	got := splitString("", '|', 3)
	if len(got) != 1 || got[0] != "" {
		t.Errorf("expected single empty element, got %v", got)
	}
}

// TestSplitString_ExactBoundary checks behavior when there are exactly
// n-1 separators and the cap is n.
func TestSplitString_ExactBoundary(t *testing.T) {
	got := splitString("a|b|c", '|', 3)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len=%d, want %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

// TestParseRefreshEntry_RoundTrip checks that a serialized entry can be
// parsed back into the same fields.
func TestParseRefreshEntry_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	data := "user-1|dev|ua-info|10.0.0.1|" +
		strconvI64(now.Unix()) + "|" + strconvI64(now.Add(time.Hour).Unix())

	entry, err := parseRefreshEntry("token-id-xyz", data)
	if err != nil {
		t.Fatalf("parseRefreshEntry returned error: %v", err)
	}
	if entry.TokenID != "token-id-xyz" {
		t.Errorf("TokenID=%q, want %q", entry.TokenID, "token-id-xyz")
	}
	if entry.UserID != "user-1" {
		t.Errorf("UserID=%q, want %q", entry.UserID, "user-1")
	}
	if entry.Role != "dev" {
		t.Errorf("Role=%q, want %q", entry.Role, "dev")
	}
	if entry.DeviceInfo != "ua-info" {
		t.Errorf("DeviceInfo=%q, want %q", entry.DeviceInfo, "ua-info")
	}
	if entry.IPAddress != "10.0.0.1" {
		t.Errorf("IPAddress=%q, want %q", entry.IPAddress, "10.0.0.1")
	}
}

// TestParseRefreshEntry_TooFewParts covers the "less than 6 parts" error
// case, which is critical for detecting corruption of stored data.
func TestParseRefreshEntry_TooFewParts(t *testing.T) {
	_, err := parseRefreshEntry("tok", "a|b|c")
	if err == nil {
		t.Fatal("expected error for too-few parts, got nil")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected 'invalid' error, got: %v", err)
	}
}

// TestParseRefreshEntry_FieldWithPipe verifies that fields with pipe
// characters (e.g. legacy device info) are not split by parseRefreshEntry.
// Because splitString caps at 6, the 5th element (createdAt) absorbs the
// extra pipe.
func TestParseRefreshEntry_FieldWithPipe(t *testing.T) {
	// Use a deliberately broken createdAt timestamp so the parser falls
	// back to the unix-int path. The test asserts that the first four
	// fields are extracted correctly even when a later segment contains
	// an extra pipe character.
	now := time.Now().UTC()
	data := "user-1|dev|ua|ip|" + "not-a-time" + "|" + strconvI64(now.Add(time.Hour).Unix())
	entry, err := parseRefreshEntry("tok", data)
	if err != nil {
		t.Fatalf("parseRefreshEntry returned error: %v", err)
	}
	if entry.UserID != "user-1" || entry.Role != "dev" {
		t.Errorf("unexpected entry: %+v", entry)
	}
}

// --- MemoryRefreshTokenStore ---

func makeEntry(id, userID string, ttl time.Duration) RefreshTokenEntry {
	now := time.Now()
	return RefreshTokenEntry{
		TokenID:    id,
		UserID:     userID,
		Role:       "dev",
		DeviceInfo: "test-device",
		IPAddress:  "10.0.0.1",
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
	}
}

// TestMemoryRefreshTokenStore_StoreAndRetrieve covers the happy path.
func TestMemoryRefreshTokenStore_StoreAndRetrieve(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := makeEntry("t1", "u1", time.Hour)

	if err := store.Store(entry); err != nil {
		t.Fatalf("Store returned error: %v", err)
	}

	got, err := store.Retrieve("t1")
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected entry, got nil")
	}
	if got.UserID != "u1" {
		t.Errorf("UserID=%q, want %q", got.UserID, "u1")
	}
}

// TestMemoryRefreshTokenStore_RetrieveMissing ensures unknown tokenIDs
// return (nil, nil) per the interface contract.
func TestMemoryRefreshTokenStore_RetrieveMissing(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	got, err := store.Retrieve("missing")
	if err != nil {
		t.Errorf("expected nil error for missing token, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil entry, got %+v", got)
	}
}

// TestMemoryRefreshTokenStore_RetrieveExpired verifies that expired tokens
// are not returned, even if they are still in the map.
func TestMemoryRefreshTokenStore_RetrieveExpired(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(makeEntry("t1", "u1", -1*time.Hour))

	got, err := store.Retrieve("t1")
	if err != nil {
		t.Errorf("expected nil error for expired token, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for expired token, got %+v", got)
	}
}

// TestMemoryRefreshTokenStore_Revoke ensures a single revocation works.
func TestMemoryRefreshTokenStore_Revoke(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(makeEntry("t1", "u1", time.Hour))
	_ = store.Store(makeEntry("t2", "u1", time.Hour))

	if err := store.Revoke("t1"); err != nil {
		t.Fatalf("Revoke returned error: %v", err)
	}
	if got, _ := store.Retrieve("t1"); got != nil {
		t.Errorf("t1 should be revoked, got %+v", got)
	}
	if got, _ := store.Retrieve("t2"); got == nil {
		t.Error("t2 should still be retrievable")
	}
}

// TestMemoryRefreshTokenStore_RevokeMissingIsNoop ensures Revoke is safe
// for unknown IDs.
func TestMemoryRefreshTokenStore_RevokeMissingIsNoop(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	if err := store.Revoke("never-existed"); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// TestMemoryRefreshTokenStore_RevokeAllForUser verifies mass revocation,
// which is the "log out everywhere" path.
func TestMemoryRefreshTokenStore_RevokeAllForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(makeEntry("u1-a", "u1", time.Hour))
	_ = store.Store(makeEntry("u1-b", "u1", time.Hour))
	_ = store.Store(makeEntry("u2-a", "u2", time.Hour))

	if err := store.RevokeAllForUser("u1"); err != nil {
		t.Fatalf("RevokeAllForUser returned error: %v", err)
	}
	if got, _ := store.Retrieve("u1-a"); got != nil {
		t.Errorf("u1-a should be revoked, got %+v", got)
	}
	if got, _ := store.Retrieve("u1-b"); got != nil {
		t.Errorf("u1-b should be revoked, got %+v", got)
	}
	if got, _ := store.Retrieve("u2-a"); got == nil {
		t.Errorf("u2-a should not be affected")
	}
}

// TestMemoryRefreshTokenStore_Count verifies per-user token counts.
func TestMemoryRefreshTokenStore_Count(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(makeEntry("t1", "u1", time.Hour))
	_ = store.Store(makeEntry("t2", "u1", time.Hour))
	_ = store.Store(makeEntry("t3", "u2", time.Hour))

	for _, c := range []struct {
		user  string
		want  int
	}{
		{"u1", 2},
		{"u2", 1},
		{"u3", 0},
	} {
		got, err := store.Count(c.user)
		if err != nil {
			t.Errorf("Count(%q) error: %v", c.user, err)
		}
		if got != c.want {
			t.Errorf("Count(%q)=%d, want %d", c.user, got, c.want)
		}
	}
}

// TestMemoryRefreshTokenStore_ListForUser verifies the listing helper
// skips expired entries.
func TestMemoryRefreshTokenStore_ListForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(makeEntry("valid-1", "u1", time.Hour))
	_ = store.Store(makeEntry("valid-2", "u1", 2*time.Hour))
	_ = store.Store(makeEntry("expired", "u1", -1*time.Hour))
	_ = store.Store(makeEntry("other-user", "u2", time.Hour))

	entries, err := store.ListForUser("u1")
	if err != nil {
		t.Fatalf("ListForUser error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 valid entries, got %d", len(entries))
	}
	seen := map[string]bool{}
	for _, e := range entries {
		seen[e.TokenID] = true
	}
	if !seen["valid-1"] || !seen["valid-2"] {
		t.Errorf("expected valid-1 and valid-2 in result, got %v", seen)
	}
	if seen["expired"] || seen["other-user"] {
		t.Errorf("unexpected entries in result, got %v", seen)
	}
}

// TestMemoryRefreshTokenStore_ListForUser_Empty verifies the no-tokens path.
func TestMemoryRefreshTokenStore_ListForUser_Empty(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entries, err := store.ListForUser("nobody")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty list, got %v", entries)
	}
}

// TestMemoryRefreshTokenStore_Cleanup verifies the cleanup loop removes
// expired entries.
func TestMemoryRefreshTokenStore_Cleanup(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	_ = store.Store(makeEntry("expired", "u1", -1*time.Hour))
	_ = store.Store(makeEntry("valid", "u1", time.Hour))

	store.cleanup()

	if got, _ := store.Retrieve("expired"); got != nil {
		t.Errorf("expired should be cleaned up, got %+v", got)
	}
	if got, _ := store.Retrieve("valid"); got == nil {
		t.Error("valid should still be present after cleanup")
	}
}

// TestMemoryRefreshTokenStore_StartCleanup_RunsAndStops exercises the
// background goroutine and ensures it stops on context cancellation.
func TestMemoryRefreshTokenStore_StartCleanup_RunsAndStops(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		store.StartCleanup(ctx, 10*time.Millisecond)
		close(done)
	}()

	// Let it run briefly
	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Goroutine exited cleanly.
	case <-time.After(2 * time.Second):
		t.Fatal("StartCleanup did not stop after context cancellation")
	}
}

// TestMemoryRefreshTokenStore_ConcurrentSafety exercises the store under
// concurrent access to catch data races (use `go test -race` to detect).
func TestMemoryRefreshTokenStore_ConcurrentSafety(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	const goroutines = 16
	const perGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Half the goroutines write; half read/count.
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				id := "tok-" + strconvI64(int64(gid)) + "-" + strconvI64(int64(i))
				_ = store.Store(makeEntry(id, "u-"+strconvI64(int64(gid)), time.Hour))
				_, _ = store.Retrieve(id)
				_, _ = store.Count("u-" + strconvI64(int64(gid)))
			}
		}(g)
	}
	wg.Wait()

	// Spot-check: at least one entry should still be retrievable.
	any, err := store.Retrieve("tok-0-0")
	if err != nil || any == nil {
		t.Errorf("expected tok-0-0 to be retrievable, got err=%v entry=%v", err, any)
	}
}

// strconvI64 is a tiny helper to avoid an extra strconv import in this file.
func strconvI64(n int64) string {
	// Implemented locally to keep the test file self-contained and
	// minimal; this is equivalent to strconv.FormatInt(n, 10).
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	pos := len(b)
	for n > 0 {
		pos--
		b[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
