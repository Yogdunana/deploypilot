package auth

import (
	"context"
	"testing"
	"time"
)

func TestGenerateTicket_ReturnsNonEmpty(t *testing.T) {
	store := NewWSTicketStore()
	ticket, err := store.GenerateTicket("user-1", "admin", 30*time.Second)
	if err != nil {
		t.Fatalf("GenerateTicket returned error: %v", err)
	}
	if ticket == "" {
		t.Fatal("GenerateTicket returned empty ticket")
	}
}

func TestValidateTicket_ReturnsCorrectUserIDAndRole(t *testing.T) {
	store := NewWSTicketStore()
	ticket, err := store.GenerateTicket("user-42", "dev", 30*time.Second)
	if err != nil {
		t.Fatalf("GenerateTicket returned error: %v", err)
	}

	userID, role, err := store.ValidateTicket(ticket)
	if err != nil {
		t.Fatalf("ValidateTicket returned error: %v", err)
	}
	if userID != "user-42" {
		t.Errorf("expected userID user-42, got %s", userID)
	}
	if role != "dev" {
		t.Errorf("expected role dev, got %s", role)
	}
}

func TestValidateTicket_SecondUseFails(t *testing.T) {
	store := NewWSTicketStore()
	ticket, err := store.GenerateTicket("user-1", "viewer", 30*time.Second)
	if err != nil {
		t.Fatalf("GenerateTicket returned error: %v", err)
	}

	// First use should succeed
	_, _, err = store.ValidateTicket(ticket)
	if err != nil {
		t.Fatalf("first ValidateTicket should succeed, got error: %v", err)
	}

	// Second use should fail (ticket consumed)
	_, _, err = store.ValidateTicket(ticket)
	if err == nil {
		t.Fatal("second ValidateTicket should fail, but it succeeded")
	}
}

func TestValidateTicket_ExpiredTicketFails(t *testing.T) {
	store := NewWSTicketStore()
	// Generate a ticket that expires immediately (negative duration)
	ticket, err := store.GenerateTicket("user-1", "viewer", -1*time.Second)
	if err != nil {
		t.Fatalf("GenerateTicket returned error: %v", err)
	}

	_, _, err = store.ValidateTicket(ticket)
	if err == nil {
		t.Fatal("ValidateTicket should fail for expired ticket, but it succeeded")
	}
}

func TestValidateTicket_UnknownTicketFails(t *testing.T) {
	store := NewWSTicketStore()
	_, _, err := store.ValidateTicket("nonexistent-ticket-id")
	if err == nil {
		t.Fatal("ValidateTicket should fail for unknown ticket, but it succeeded")
	}
}

func TestStartCleanup_RemovesExpiredTickets(t *testing.T) {
	store := NewWSTicketStore()

	// Generate a ticket that is already expired
	_, err := store.GenerateTicket("user-expired", "viewer", -1*time.Second)
	if err != nil {
		t.Fatalf("GenerateTicket returned error: %v", err)
	}

	// Generate a valid ticket
	validTicket, err := store.GenerateTicket("user-valid", "viewer", 30*time.Second)
	if err != nil {
		t.Fatalf("GenerateTicket returned error: %v", err)
	}

	// Run cleanup
	store.cleanup()

	// Expired ticket should be gone
	_, _, err = store.ValidateTicket("user-expired") // this won't match because we don't have the ticket ID
	// Instead, check that the store has exactly 1 ticket remaining
	store.mu.RLock()
	count := len(store.tickets)
	store.mu.RUnlock()
	if count != 1 {
		t.Errorf("expected 1 ticket after cleanup, got %d", count)
	}

	// Valid ticket should still work
	_, _, err = store.ValidateTicket(validTicket)
	if err != nil {
		t.Fatalf("valid ticket should still work after cleanup, got error: %v", err)
	}
}

func TestStartCleanup_StopsOnContextCancel(t *testing.T) {
	store := NewWSTicketStore()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		store.StartCleanup(ctx, 50*time.Millisecond)
		close(done)
	}()

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for goroutine to stop (should be quick)
	select {
	case <-done:
		// Good, goroutine stopped
	case <-time.After(2 * time.Second):
		t.Fatal("StartCleanup did not stop after context cancellation")
	}
}
