// Package confirm provides a two-step confirmation system for dangerous operations.
// Instead of executing immediately, dangerous operations create a pending confirmation
// that must be explicitly confirmed before execution.
package confirm

import (
	"fmt"
	"sync"
	"time"
)

// Status represents the state of a confirmation request.
type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
	StatusExpired  Status = "expired"
	StatusExecuted Status = "executed"
)

// Request represents a pending confirmation for a dangerous operation.
type Request struct {
	ID          string            `json:"id"`
	Tool        string            `json:"tool"`
	Description string            `json:"description"`
	Params      map[string]string `json:"params,omitempty"`
	Status      Status            `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   time.Time         `json:"expires_at"`
	ConfirmedAt *time.Time        `json:"confirmed_at,omitempty"`
	ConfirmedBy string            `json:"confirmed_by,omitempty"`
	Result      string            `json:"result,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// Store manages pending confirmation requests.
type Store struct {
	mu          sync.RWMutex
	requests    map[string]*Request
	defaultTTL  time.Duration
}

// NewStore creates a new confirmation store.
func NewStore() *Store {
	return &Store{
		requests:   make(map[string]*Request),
		defaultTTL: 5 * time.Minute,
	}
}

// Create creates a new pending confirmation request.
func (s *Store) Create(tool, description string, params map[string]string) *Request {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	req := &Request{
		ID:          fmt.Sprintf("cfm-%d", now.UnixNano()),
		Tool:        tool,
		Description: description,
		Params:      params,
		Status:      StatusPending,
		CreatedAt:   now,
		ExpiresAt:   now.Add(s.defaultTTL),
	}
	s.requests[req.ID] = req

	// Auto-expire in background
	go func() {
		time.Sleep(s.defaultTTL + time.Second)
		s.mu.Lock()
		defer s.mu.Unlock()
		if r, ok := s.requests[req.ID]; ok && r.Status == StatusPending {
			r.Status = StatusExpired
		}
	}()

	return req
}

// Confirm approves a pending confirmation.
func (s *Store) Confirm(id, confirmedBy string) (*Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req, ok := s.requests[id]
	if !ok {
		return nil, fmt.Errorf("confirmation %s not found", id)
	}
	if req.Status != StatusPending {
		return nil, fmt.Errorf("confirmation %s is %s (expected pending)", id, req.Status)
	}
	if time.Now().After(req.ExpiresAt) {
		req.Status = StatusExpired
		return nil, fmt.Errorf("confirmation %s has expired", id)
	}

	now := time.Now()
	req.Status = StatusApproved
	req.ConfirmedAt = &now
	req.ConfirmedBy = confirmedBy
	return req, nil
}

// Reject rejects a pending confirmation.
func (s *Store) Reject(id, rejectedBy string) (*Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req, ok := s.requests[id]
	if !ok {
		return nil, fmt.Errorf("confirmation %s not found", id)
	}
	if req.Status != StatusPending {
		return nil, fmt.Errorf("confirmation %s is %s (expected pending)", id, req.Status)
	}

	now := time.Now()
	req.Status = StatusRejected
	req.ConfirmedAt = &now
	req.ConfirmedBy = rejectedBy
	return req, nil
}

// Get retrieves a confirmation by ID.
func (s *Store) Get(id string) (*Request, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.requests[id]
	return req, ok
}

// List returns all confirmations, optionally filtered by status.
func (s *Store) List(statusFilter Status) []*Request {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Request
	for _, req := range s.requests {
		if statusFilter == "" || req.Status == statusFilter {
			result = append(result, req)
		}
	}
	return result
}

// SetResult records the execution result of a confirmed operation.
func (s *Store) SetResult(id, result, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req, ok := s.requests[id]; ok {
		if errMsg != "" {
			req.Error = errMsg
		} else {
			req.Result = result
		}
		req.Status = StatusExecuted
	}
}

// Cleanup removes old confirmations (executed, rejected, expired) older than maxAge.
func (s *Store) Cleanup(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	count := 0
	for id, req := range s.requests {
		if req.Status != StatusPending && req.CreatedAt.Before(cutoff) {
			delete(s.requests, id)
			count++
		}
	}
	return count
}
