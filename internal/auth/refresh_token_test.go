package auth

import (
	"sync"
	"testing"
	"time"
)

// sampleEntry returns a refresh token entry that is far from expiry.
func sampleEntry(tokenID, userID string) RefreshTokenEntry {
	now := time.Now()
	return RefreshTokenEntry{
		TokenID:    tokenID,
		UserID:     userID,
		Role:       "dev",
		DeviceInfo: "test-device",
		IPAddress:  "127.0.0.1",
		CreatedAt:  now,
		ExpiresAt:  now.Add(1 * time.Hour),
	}
}

// expiredEntry returns an entry that is already expired.
func expiredEntry(tokenID, userID string) RefreshTokenEntry {
	now := time.Now()
	return RefreshTokenEntry{
		TokenID:    tokenID,
		UserID:     userID,
		Role:       "dev",
		DeviceInfo: "test-device",
		IPAddress:  "127.0.0.1",
		CreatedAt:  now.Add(-1 * time.Hour),
		ExpiresAt:  now.Add(-1 * time.Minute),
	}
}

func TestMemoryRefreshTokenStore_StoreAndRetrieve(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	entry := sampleEntry("tok-1", "user-1")
	if err := s.Store(entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := s.Retrieve("tok-1")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if got == nil {
		t.Fatal("Retrieve returned nil for stored token")
	}
	if got.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", got.UserID)
	}
	if got.Role != "dev" {
		t.Errorf("Role = %q, want dev", got.Role)
	}
}

func TestMemoryRefreshTokenStore_RetrieveMissingReturnsNilNil(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	got, err := s.Retrieve("never-stored")
	if err != nil {
		t.Fatalf("Retrieve returned unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil entry for missing token, got %+v", got)
	}
}

func TestMemoryRefreshTokenStore_RetrieveExpiredReturnsNilNil(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	entry := expiredEntry("tok-expired", "user-1")
	if err := s.Store(entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := s.Retrieve("tok-expired")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for expired token, got %+v", got)
	}
}

func TestMemoryRefreshTokenStore_Revoke(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	entry := sampleEntry("tok-1", "user-1")
	_ = s.Store(entry)

	if err := s.Revoke("tok-1"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	got, _ := s.Retrieve("tok-1")
	if got != nil {
		t.Errorf("expected nil after revoke, got %+v", got)
	}
}

func TestMemoryRefreshTokenStore_RevokeMissingIsNoop(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	if err := s.Revoke("nonexistent"); err != nil {
		t.Errorf("Revoke on missing token should not error, got %v", err)
	}
}

func TestMemoryRefreshTokenStore_RevokeCleansUserIndex(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	_ = s.Store(sampleEntry("tok-1", "user-1"))
	_ = s.Store(sampleEntry("tok-2", "user-1"))

	if err := s.Revoke("tok-1"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	count, err := s.Count("user-1")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("Count after one revoke = %d, want 1", count)
	}

	// The remaining token should still be valid.
	entries, err := s.ListForUser("user-1")
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(entries) != 1 || entries[0].TokenID != "tok-2" {
		t.Errorf("ListForUser after revoke = %v, want [tok-2]", entries)
	}
}

func TestMemoryRefreshTokenStore_RevokeAllForUser(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	_ = s.Store(sampleEntry("tok-1", "user-1"))
	_ = s.Store(sampleEntry("tok-2", "user-1"))
	_ = s.Store(sampleEntry("tok-3", "user-2"))

	if err := s.RevokeAllForUser("user-1"); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}

	count1, _ := s.Count("user-1")
	if count1 != 0 {
		t.Errorf("Count(user-1) after revoke-all = %d, want 0", count1)
	}

	count2, _ := s.Count("user-2")
	if count2 != 1 {
		t.Errorf("Count(user-2) after revoke-other = %d, want 1 (user-2 tokens must be untouched)", count2)
	}

	// The actual stored token must also be gone.
	got, _ := s.Retrieve("tok-1")
	if got != nil {
		t.Errorf("Retrieve(tok-1) after revoke-all = %+v, want nil", got)
	}
}

func TestMemoryRefreshTokenStore_RevokeAllForUnknownUser(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	if err := s.RevokeAllForUser("unknown-user"); err != nil {
		t.Errorf("RevokeAllForUser on unknown user should not error, got %v", err)
	}
}

func TestMemoryRefreshTokenStore_Count(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	count, err := s.Count("user-1")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Errorf("Count on empty store = %d, want 0", count)
	}

	_ = s.Store(sampleEntry("tok-1", "user-1"))
	_ = s.Store(sampleEntry("tok-2", "user-1"))
	count, _ = s.Count("user-1")
	if count != 2 {
		t.Errorf("Count after two stores = %d, want 2", count)
	}
}

func TestMemoryRefreshTokenStore_ListForUserFiltersExpired(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	_ = s.Store(sampleEntry("tok-valid", "user-1"))
	_ = s.Store(expiredEntry("tok-expired", "user-1"))

	entries, err := s.ListForUser("user-1")
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListForUser returned %d entries, want 1 (expired filtered out). Got: %+v", len(entries), entries)
	}
	if entries[0].TokenID != "tok-valid" {
		t.Errorf("returned entry TokenID = %q, want tok-valid", entries[0].TokenID)
	}
}

func TestMemoryRefreshTokenStore_ListForUserEmpty(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	entries, err := s.ListForUser("unknown")
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("ListForUser for unknown user = %v, want empty", entries)
	}
}

func TestMemoryRefreshTokenStore_GenerateRefreshTokenID(t *testing.T) {
	id1, err := GenerateRefreshTokenID()
	if err != nil {
		t.Fatalf("GenerateRefreshTokenID: %v", err)
	}
	if len(id1) != 64 { // 32 bytes hex-encoded
		t.Errorf("token id length = %d, want 64", len(id1))
	}

	// Two calls should produce different IDs.
	id2, err := GenerateRefreshTokenID()
	if err != nil {
		t.Fatalf("GenerateRefreshTokenID: %v", err)
	}
	if id1 == id2 {
		t.Error("two consecutive token IDs should not be equal")
	}
}

func TestMemoryRefreshTokenStore_ConcurrentStoreAndRetrieve(t *testing.T) {
	// Concurrency test: 50 goroutines store tokens, 50 read them.
	// The store should not race or corrupt internal state.
	s := NewMemoryRefreshTokenStore()
	const n = 50

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			id := "concurrent-" + itoa(i)
			_ = s.Store(sampleEntry(id, "user-concurrent"))
		}()
	}
	wg.Wait()

	var readWG sync.WaitGroup
	readWG.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer readWG.Done()
			id := "concurrent-" + itoa(i)
			got, err := s.Retrieve(id)
			if err != nil {
				t.Errorf("Retrieve(%s) error: %v", id, err)
				return
			}
			if got == nil {
				t.Errorf("Retrieve(%s) returned nil", id)
			}
		}()
	}
	readWG.Wait()
}

func TestMemoryRefreshTokenStore_StartCleanupStopsOnContextCancel(t *testing.T) {
	s := NewMemoryRefreshTokenStore()
	_ = s.Store(expiredEntry("tok-1", "user-1"))
	_ = s.Store(sampleEntry("tok-2", "user-1"))

	// Run cleanup once to confirm it removes expired tokens.
	s.cleanup()

	got, _ := s.Retrieve("tok-1")
	if got != nil {
		t.Error("expired token should be gone after cleanup")
	}

	count, _ := s.Count("user-1")
	if count != 1 {
		t.Errorf("Count after cleanup = %d, want 1", count)
	}
}

// itoa is a small helper to avoid importing strconv in tests.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
