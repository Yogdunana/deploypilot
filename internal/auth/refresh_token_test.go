package auth

import (
	"strconv"
	"testing"
	"time"
)

func TestGenerateRefreshTokenID(t *testing.T) {
	for i := 0; i < 10; i++ {
		tokenID, err := GenerateRefreshTokenID()
		if err != nil {
			t.Fatalf("GenerateRefreshTokenID failed: %v", err)
		}
		if len(tokenID) != 64 {
			t.Errorf("expected tokenID length 64, got %d", len(tokenID))
		}
		for _, c := range tokenID {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				t.Errorf("invalid character in tokenID: %c", c)
			}
		}
	}
}

func TestGenerateRefreshTokenID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		tokenID, err := GenerateRefreshTokenID()
		if err != nil {
			t.Fatalf("GenerateRefreshTokenID failed: %v", err)
		}
		if seen[tokenID] {
			t.Errorf("duplicate tokenID generated: %s", tokenID)
		}
		seen[tokenID] = true
	}
}

func TestMemoryRefreshTokenStore_Store(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := RefreshTokenEntry{
		TokenID:    "token-1",
		UserID:     "user-1",
		Role:       "admin",
		DeviceInfo: "test-device",
		IPAddress:  "192.168.1.1",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	err := store.Store(entry)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	retrieved, err := store.Retrieve("token-1")
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected retrieved entry to not be nil")
	}
	if retrieved.UserID != "user-1" {
		t.Errorf("expected UserID=user-1, got %s", retrieved.UserID)
	}
	if retrieved.Role != "admin" {
		t.Errorf("expected Role=admin, got %s", retrieved.Role)
	}
	if retrieved.DeviceInfo != "test-device" {
		t.Errorf("expected DeviceInfo=test-device, got %s", retrieved.DeviceInfo)
	}
	if retrieved.IPAddress != "192.168.1.1" {
		t.Errorf("expected IPAddress=192.168.1.1, got %s", retrieved.IPAddress)
	}
}

func TestMemoryRefreshTokenStore_Retrieve_NotFound(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	retrieved, err := store.Retrieve("nonexistent-token")
	if err != nil {
		t.Fatalf("Retrieve should not fail for non-existent token: %v", err)
	}
	if retrieved != nil {
		t.Errorf("expected nil for non-existent token, got %v", retrieved)
	}
}

func TestMemoryRefreshTokenStore_Retrieve_Expired(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := RefreshTokenEntry{
		TokenID:    "token-expired",
		UserID:     "user-1",
		Role:       "admin",
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
	}

	err := store.Store(entry)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	retrieved, err := store.Retrieve("token-expired")
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("expected nil for expired token, got %v", retrieved)
	}
}

func TestMemoryRefreshTokenStore_Revoke(t *testing.T) {
	store := NewMemoryRefreshTokenStore()
	entry := RefreshTokenEntry{
		TokenID:   "token-to-revoke",
		UserID:    "user-1",
		Role:      "admin",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := store.Store(entry)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	err = store.Revoke("token-to-revoke")
	if err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	retrieved, err := store.Retrieve("token-to-revoke")
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("expected nil after revoke, got %v", retrieved)
	}
}

func TestMemoryRefreshTokenStore_RevokeAllForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	for i := 0; i < 5; i++ {
		entry := RefreshTokenEntry{
			TokenID:   "token-" + strconv.Itoa(i),
			UserID:    "user-1",
			Role:      "admin",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := store.Store(entry)
		if err != nil {
			t.Fatalf("Store failed for token-%d: %v", i, err)
		}
	}

	for i := 0; i < 3; i++ {
		entry := RefreshTokenEntry{
			TokenID:   "other-token-" + strconv.Itoa(i),
			UserID:    "user-2",
			Role:      "user",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := store.Store(entry)
		if err != nil {
			t.Fatalf("Store failed for other-token-%d: %v", i, err)
		}
	}

	err := store.RevokeAllForUser("user-1")
	if err != nil {
		t.Fatalf("RevokeAllForUser failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		retrieved, err := store.Retrieve("token-" + strconv.Itoa(i))
		if err != nil {
			t.Fatalf("Retrieve failed: %v", err)
		}
		if retrieved != nil {
			t.Errorf("expected nil for token-%d after revoke all", i)
		}
	}

	for i := 0; i < 3; i++ {
		retrieved, err := store.Retrieve("other-token-" + strconv.Itoa(i))
		if err != nil {
			t.Fatalf("Retrieve failed: %v", err)
		}
		if retrieved == nil {
			t.Errorf("expected non-nil for other-token-%d (different user)", i)
		}
	}
}

func TestMemoryRefreshTokenStore_Count(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	for i := 0; i < 3; i++ {
		entry := RefreshTokenEntry{
			TokenID:   "token-" + strconv.Itoa(i),
			UserID:    "user-1",
			Role:      "admin",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := store.Store(entry)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	count, err := store.Count("user-1")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}

	count, err = store.Count("user-nonexistent")
	if err != nil {
		t.Fatalf("Count failed for non-existent user: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count=0 for non-existent user, got %d", count)
	}
}

func TestMemoryRefreshTokenStore_ListForUser(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	for i := 0; i < 3; i++ {
		entry := RefreshTokenEntry{
			TokenID:   "token-" + strconv.Itoa(i),
			UserID:    "user-1",
			Role:      "admin",
			CreatedAt: time.Now().Add(time.Duration(i) * time.Hour),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		err := store.Store(entry)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	entries, err := store.ListForUser("user-1")
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	entries, err = store.ListForUser("user-nonexistent")
	if err != nil {
		t.Fatalf("ListForUser failed for non-existent user: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for non-existent user, got %d", len(entries))
	}
}

func TestMemoryRefreshTokenStore_ListForUser_ExcludesExpired(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	entry1 := RefreshTokenEntry{
		TokenID:   "token-valid",
		UserID:    "user-1",
		Role:      "admin",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := store.Store(entry1)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	entry2 := RefreshTokenEntry{
		TokenID:   "token-expired",
		UserID:    "user-1",
		Role:      "admin",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	err = store.Store(entry2)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	entries, err := store.ListForUser("user-1")
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 valid entry, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].TokenID != "token-valid" {
		t.Errorf("expected token-valid, got %s", entries[0].TokenID)
	}
}

func TestMemoryRefreshTokenStore_Cleanup(t *testing.T) {
	store := NewMemoryRefreshTokenStore()

	for i := 0; i < 5; i++ {
		expiresAt := time.Now().Add(time.Duration(i-3) * time.Hour)
		entry := RefreshTokenEntry{
			TokenID:   "token-" + strconv.Itoa(i),
			UserID:    "user-1",
			Role:      "admin",
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
			ExpiresAt: expiresAt,
		}
		err := store.Store(entry)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	countBefore, _ := store.Count("user-1")
	store.cleanup()
	countAfter, _ := store.Count("user-1")

	if countBefore != 5 {
		t.Errorf("expected 5 before cleanup, got %d", countBefore)
	}
	if countAfter != 1 {
		t.Errorf("expected 1 after cleanup (only token-4 is valid), got %d", countAfter)
	}
}

func TestParseRefreshEntry(t *testing.T) {
	testCases := []struct {
		name     string
		tokenID  string
		data     string
		expected *RefreshTokenEntry
	}{
		{
			name:    "valid entry",
			tokenID: "token-1",
			data:    "user-1|admin|device1|192.168.1.1|1704067200|1704153600",
			expected: &RefreshTokenEntry{
				TokenID:    "token-1",
				UserID:     "user-1",
				Role:       "admin",
				DeviceInfo: "device1",
				IPAddress:  "192.168.1.1",
				CreatedAt:  time.Unix(1704067200, 0),
				ExpiresAt:  time.Unix(1704153600, 0),
			},
		},
		{
			name:    "empty optional fields",
			tokenID: "token-2",
			data:    "user-2|viewer|||1704067200|1704153600",
			expected: &RefreshTokenEntry{
				TokenID:    "token-2",
				UserID:     "user-2",
				Role:       "viewer",
				DeviceInfo: "",
				IPAddress:  "",
				CreatedAt:  time.Unix(1704067200, 0),
				ExpiresAt:  time.Unix(1704153600, 0),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseRefreshEntry(tc.tokenID, tc.data)
			if err != nil {
				t.Fatalf("parseRefreshEntry failed: %v", err)
			}
			if result.UserID != tc.expected.UserID {
				t.Errorf("expected UserID=%s, got %s", tc.expected.UserID, result.UserID)
			}
			if result.Role != tc.expected.Role {
				t.Errorf("expected Role=%s, got %s", tc.expected.Role, result.Role)
			}
			if result.DeviceInfo != tc.expected.DeviceInfo {
				t.Errorf("expected DeviceInfo=%s, got %s", tc.expected.DeviceInfo, result.DeviceInfo)
			}
			if result.IPAddress != tc.expected.IPAddress {
				t.Errorf("expected IPAddress=%s, got %s", tc.expected.IPAddress, result.IPAddress)
			}
			if !result.CreatedAt.Equal(tc.expected.CreatedAt) {
				t.Errorf("expected CreatedAt=%v, got %v", tc.expected.CreatedAt, result.CreatedAt)
			}
			if !result.ExpiresAt.Equal(tc.expected.ExpiresAt) {
				t.Errorf("expected ExpiresAt=%v, got %v", tc.expected.ExpiresAt, result.ExpiresAt)
			}
		})
	}
}

func TestParseRefreshEntry_Invalid(t *testing.T) {
	_, err := parseRefreshEntry("token-1", "invalid")
	if err == nil {
		t.Error("expected error for invalid entry")
	}
}

func TestSplitString(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		sep      rune
		n        int
		expected []string
	}{
		{
			name:     "simple split",
			s:        "a|b|c",
			sep:      '|',
			n:        3,
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "split with limit",
			s:        "a|b|c|d",
			sep:      '|',
			n:        3,
			expected: []string{"a", "b", "c|d"},
		},
		{
			name:     "empty string",
			s:        "",
			sep:      '|',
			n:        2,
			expected: []string{""},
		},
		{
			name:     "no separator",
			s:        "abc",
			sep:      '|',
			n:        2,
			expected: []string{"abc"},
		},
		{
			name:     "leading separator",
			s:        "|a|b",
			sep:      '|',
			n:        3,
			expected: []string{"", "a", "b"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitString(tc.s, tc.sep, tc.n)
			if len(result) != len(tc.expected) {
				t.Errorf("expected %d parts, got %d", len(tc.expected), len(result))
				return
			}
			for i, expected := range tc.expected {
				if result[i] != expected {
					t.Errorf("part %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}