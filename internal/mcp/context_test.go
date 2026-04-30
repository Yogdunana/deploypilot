package mcp

import (
	"strings"
	"testing"
	"time"
)

func TestNewSessionContext(t *testing.T) {
	sc := NewSessionContext("test-session")
	if sc == nil {
		t.Fatal("NewSessionContext returned nil")
	}
	if sc.sessionID != "test-session" {
		t.Errorf("sessionID = %q, want %q", sc.sessionID, "test-session")
	}
	if sc.maxEntries != 50 {
		t.Errorf("maxEntries = %d, want 50", sc.maxEntries)
	}
	if sc.maxMemory != 10*1024*1024 {
		t.Errorf("maxMemory = %d, want %d", sc.maxMemory, 10*1024*1024)
	}
	if sc.entries.Len() != 0 {
		t.Errorf("entries.Len() = %d, want 0", sc.entries.Len())
	}
}

func TestAddEntry(t *testing.T) {
	sc := NewSessionContext("test")

	entry := ContextEntry{
		Tool:     "deploy_app",
		Args:     `{"image":"nginx"}`,
		Result:   "success",
		Duration: "1.2s",
		Time:     time.Now(),
	}
	sc.AddEntry(entry)

	entries := sc.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Tool != "deploy_app" {
		t.Errorf("entries[0].Tool = %q, want %q", entries[0].Tool, "deploy_app")
	}
}

func TestAddEntry_Eviction(t *testing.T) {
	sc := NewSessionContext("test")
	sc.maxEntries = 3

	for i := 0; i < 5; i++ {
		sc.AddEntry(ContextEntry{
			Tool:     "tool",
			Args:     string(rune('a' + i)),
			Duration: "0s",
			Time:     time.Now(),
		})
	}

	entries := sc.GetEntries()
	if len(entries) != 3 {
		t.Errorf("len(entries) = %d, want 3 (evicted oldest)", len(entries))
	}
	// The oldest entries should have been evicted
	if entries[0].Args != string(rune('c')) {
		t.Errorf("first entry Args = %q, want %q (oldest evicted)", entries[0].Args, string(rune('c')))
	}
}

func TestGetEntries(t *testing.T) {
	sc := NewSessionContext("test")

	sc.AddEntry(ContextEntry{Tool: "first", Time: time.Now()})
	sc.AddEntry(ContextEntry{Tool: "second", Time: time.Now()})
	sc.AddEntry(ContextEntry{Tool: "third", Time: time.Now()})

	entries := sc.GetEntries()
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	if entries[0].Tool != "first" {
		t.Errorf("entries[0].Tool = %q, want %q", entries[0].Tool, "first")
	}
	if entries[2].Tool != "third" {
		t.Errorf("entries[2].Tool = %q, want %q", entries[2].Tool, "third")
	}
}

func TestGetSummary(t *testing.T) {
	sc := NewSessionContext("test-session")
	sc.AddEntry(ContextEntry{Tool: "tool1", Time: time.Now()})
	sc.AddEntry(ContextEntry{Tool: "tool2", Time: time.Now()})

	summary := sc.GetSummary()
	if summary["session_id"] != "test-session" {
		t.Errorf("summary[session_id] = %v, want %q", summary["session_id"], "test-session")
	}
	if summary["entries"] != 2 {
		t.Errorf("summary[entries] = %v, want 2", summary["entries"])
	}
	if summary["max_entries"] != 50 {
		t.Errorf("summary[max_entries] = %v, want 50", summary["max_entries"])
	}
}

func TestClear(t *testing.T) {
	sc := NewSessionContext("test")
	sc.AddEntry(ContextEntry{Tool: "tool1", Time: time.Now()})
	sc.AddEntry(ContextEntry{Tool: "tool2", Time: time.Now()})

	sc.Clear()

	entries := sc.GetEntries()
	if len(entries) != 0 {
		t.Errorf("len(entries) after Clear = %d, want 0", len(entries))
	}
	if sc.currentSize != 0 {
		t.Errorf("currentSize after Clear = %d, want 0", sc.currentSize)
	}
}

func TestIsExpired(t *testing.T) {
	sc := NewSessionContext("test")
	// Freshly created, should not be expired with a long timeout
	if sc.IsExpired(30 * time.Minute) {
		t.Error("fresh session should not be expired")
	}

	// Manually set lastAccess to the past
	sc.mu.Lock()
	sc.lastAccess = time.Now().Add(-1 * time.Hour)
	sc.mu.Unlock()

	if !sc.IsExpired(30 * time.Minute) {
		t.Error("session with old lastAccess should be expired")
	}
}

func TestContextManager_GetOrCreateSession(t *testing.T) {
	cm := NewContextManager()

	// Create new session
	s1 := cm.GetOrCreateSession("s1")
	if s1 == nil {
		t.Fatal("GetOrCreateSession returned nil")
	}
	if s1.sessionID != "s1" {
		t.Errorf("sessionID = %q, want %q", s1.sessionID, "s1")
	}

	// Get existing session
	s1Again := cm.GetOrCreateSession("s1")
	if s1 != s1Again {
		t.Error("GetOrCreateSession should return the same session for the same ID")
	}

	// Different session ID
	s2 := cm.GetOrCreateSession("s2")
	if s2 == s1 {
		t.Error("different session IDs should yield different sessions")
	}
}

func TestContextManager_Cleanup(t *testing.T) {
	cm := &ContextManager{
		sessions: make(map[string]*SessionContext),
		timeout:  50 * time.Millisecond,
	}

	// Create a session and make it expire
	s := cm.GetOrCreateSession("expire-me")
	s.mu.Lock()
	s.lastAccess = time.Now().Add(-1 * time.Hour)
	s.mu.Unlock()

	// Create a fresh session
	cm.GetOrCreateSession("keep-me")

	// Run one cleanup cycle manually
	cm.mu.Lock()
	for id, session := range cm.sessions {
		if session.IsExpired(cm.timeout) {
			session.Clear()
			delete(cm.sessions, id)
		}
	}
	cm.mu.Unlock()

	if cm.GetSession("expire-me") != nil {
		t.Error("expired session should have been cleaned up")
	}
	if cm.GetSession("keep-me") == nil {
		t.Error("fresh session should not have been cleaned up")
	}
}

func TestAddEntry_MemoryEviction(t *testing.T) {
	sc := NewSessionContext("test")
	sc.maxMemory = 50 // very small limit

	sc.AddEntry(ContextEntry{
		Tool:     strings.Repeat("a", 30),
		Args:     strings.Repeat("b", 30),
		Duration: "0s",
		Time:     time.Now(),
	})

	// The entry should have been added (first entry)
	entries := sc.GetEntries()
	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}

	// Add a second large entry that would exceed memory
	sc.AddEntry(ContextEntry{
		Tool:     strings.Repeat("c", 30),
		Args:     strings.Repeat("d", 30),
		Duration: "0s",
		Time:     time.Now(),
	})

	entries = sc.GetEntries()
	if len(entries) == 0 {
		t.Error("should have at least one entry after memory eviction")
	}
}
