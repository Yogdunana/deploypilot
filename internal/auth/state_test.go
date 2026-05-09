package auth

import (
	"testing"
	"time"
)

func TestMemoryStateStore_GenerateAndValidate(t *testing.T) {
	store := NewMemoryStateStore()

	if store.Validate("nonexistent") {
		t.Error("nonexistent state should not validate")
	}

	store.Generate("state-1", 5*time.Minute)
	if !store.Validate("state-1") {
		t.Error("valid state should validate")
	}
	// State should be consumed after validation
	if store.Validate("state-1") {
		t.Error("state should be one-time use")
	}
}

func TestMemoryStateStore_Expired(t *testing.T) {
	store := NewMemoryStateStore()
	store.Generate("state-expired", 1*time.Nanosecond)
	time.Sleep(10 * time.Millisecond)

	if store.Validate("state-expired") {
		t.Error("expired state should not validate")
	}
}
