package mcp

import (
	"container/list"
	"sync"
	"time"
)

// ContextEntry represents a single operation in the session context.
type ContextEntry struct {
	Tool     string    `json:"tool"`
	Args     string    `json:"args,omitempty"`
	Result   string    `json:"result,omitempty"`
	Success  bool      `json:"success"`
	Error    string    `json:"error,omitempty"`
	Duration string    `json:"duration,omitempty"`
	Time     time.Time `json:"time"`
}

// SessionContext manages per-session operation history.
type SessionContext struct {
	mu          sync.RWMutex
	entries     *list.List
	maxEntries  int
	maxMemory   int64 // bytes
	currentSize int64
	lastAccess  time.Time
	sessionID   string
}

// NewSessionContext creates a new session context.
func NewSessionContext(sessionID string) *SessionContext {
	return &SessionContext{
		entries:    list.New(),
		maxEntries: 50,
		maxMemory:  10 * 1024 * 1024, // 10MB
		lastAccess: time.Now(),
		sessionID:  sessionID,
	}
}

// entrySize computes the approximate byte size of a ContextEntry.
func entrySize(e ContextEntry) int64 {
	return int64(len(e.Tool) + len(e.Args) + len(e.Result) + len(e.Error) + len(e.Duration))
}

// AddEntry adds an operation to the context history.
func (sc *SessionContext) AddEntry(entry ContextEntry) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.lastAccess = time.Now()

	// Evict oldest if at capacity
	for sc.entries.Len() >= sc.maxEntries {
		if front := sc.entries.Front(); front != nil {
			if e, ok := front.Value.(ContextEntry); ok {
				sc.currentSize -= entrySize(e)
			}
			sc.entries.Remove(front)
		}
	}

	es := entrySize(entry)

	// Evict for memory too
	for sc.currentSize+es > sc.maxMemory && sc.entries.Len() > 0 {
		if front := sc.entries.Front(); front != nil {
			if e, ok := front.Value.(ContextEntry); ok {
				sc.currentSize -= entrySize(e)
			}
			sc.entries.Remove(front)
		}
	}

	sc.entries.PushBack(entry)
	sc.currentSize += es
}

// GetEntries returns all entries (most recent last).
func (sc *SessionContext) GetEntries() []ContextEntry {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	entries := make([]ContextEntry, 0, sc.entries.Len())
	for e := sc.entries.Front(); e != nil; e = e.Next() {
		if entry, ok := e.Value.(ContextEntry); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

// GetSummary returns a summary of the session.
func (sc *SessionContext) GetSummary() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return map[string]interface{}{
		"session_id":   sc.sessionID,
		"entries":      sc.entries.Len(),
		"memory_usage": sc.currentSize,
		"last_access":  sc.lastAccess,
		"max_entries":  sc.maxEntries,
	}
}

// Clear removes all entries.
func (sc *SessionContext) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.entries.Init()
	sc.currentSize = 0
}

// IsExpired checks if the session has been inactive for too long.
func (sc *SessionContext) IsExpired(timeout time.Duration) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return time.Since(sc.lastAccess) > timeout
}
