package mcp

import (
	"sync"
	"time"
)

// ContextManager manages multiple session contexts.
type ContextManager struct {
	mu       sync.RWMutex
	sessions map[string]*SessionContext
	timeout  time.Duration
}

// NewContextManager creates a new context manager.
func NewContextManager() *ContextManager {
	cm := &ContextManager{
		sessions: make(map[string]*SessionContext),
		timeout:  30 * time.Minute,
	}
	go cm.cleanupLoop()
	return cm
}

// GetOrCreateSession gets an existing session or creates a new one.
func (cm *ContextManager) GetOrCreateSession(sessionID string) *SessionContext {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if session, ok := cm.sessions[sessionID]; ok {
		return session
	}

	session := NewSessionContext(sessionID)
	cm.sessions[sessionID] = session
	return session
}

// GetSession gets an existing session (nil if not found).
func (cm *ContextManager) GetSession(sessionID string) *SessionContext {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.sessions[sessionID]
}

// cleanupLoop periodically removes expired sessions.
func (cm *ContextManager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cm.mu.Lock()
		for id, session := range cm.sessions {
			if session.IsExpired(cm.timeout) {
				session.Clear()
				delete(cm.sessions, id)
			}
		}
		cm.mu.Unlock()
	}
}

// contextManager is the global session context manager.
var contextManager = NewContextManager()
