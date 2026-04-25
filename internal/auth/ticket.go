package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ticketEntry holds the metadata for a single WebSocket ticket.
type ticketEntry struct {
	UserID    string
	Role      string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// WSTicketStore manages one-time WebSocket authentication tickets.
// Tickets are generated from a valid JWT and consumed (deleted) on first use.
type WSTicketStore struct {
	mu      sync.RWMutex
	tickets map[string]*ticketEntry
}

// NewWSTicketStore creates a new WSTicketStore.
func NewWSTicketStore() *WSTicketStore {
	return &WSTicketStore{
		tickets: make(map[string]*ticketEntry),
	}
}

// GenerateTicket creates a new one-time ticket for the given user.
// The ticket expires after the specified duration.
func (s *WSTicketStore) GenerateTicket(userID, role string, expire time.Duration) (string, error) {
	id := uuid.New().String()
	now := time.Now()

	s.mu.Lock()
	s.tickets[id] = &ticketEntry{
		UserID:    userID,
		Role:      role,
		CreatedAt: now,
		ExpiresAt: now.Add(expire),
	}
	s.mu.Unlock()

	return id, nil
}

// ValidateTicket checks if the ticket is valid and consumes it (one-time use).
// Returns the userID and role associated with the ticket.
// Returns an error if the ticket is missing, expired, or already consumed.
func (s *WSTicketStore) ValidateTicket(ticket string) (string, string, error) {
	s.mu.Lock()
	entry, ok := s.tickets[ticket]
	if !ok {
		s.mu.Unlock()
		return "", "", errors.New("ticket not found")
	}

	// Always delete (consume) the ticket on validation attempt.
	delete(s.tickets, ticket)
	s.mu.Unlock()

	if time.Now().After(entry.ExpiresAt) {
		return "", "", errors.New("ticket expired")
	}

	return entry.UserID, entry.Role, nil
}

// StartCleanup runs a background goroutine that removes expired tickets
// at the given interval. It stops when the provided context is cancelled.
func (s *WSTicketStore) StartCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup removes all expired tickets from the store.
func (s *WSTicketStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, entry := range s.tickets {
		if now.After(entry.ExpiresAt) {
			delete(s.tickets, id)
		}
	}
}
