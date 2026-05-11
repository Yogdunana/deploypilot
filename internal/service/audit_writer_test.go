package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileAuditWriter_Write(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	writer, err := NewFileAuditWriter(filePath)
	if err != nil {
		t.Fatalf("NewFileAuditWriter() error = %v", err)
	}
	defer writer.Close()

	entry := AuditEntry{
		UserID:       "user-1",
		Username:     "testuser",
		Action:       "login",
		ResourceType: "auth",
		ResourceID:   "",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent",
	}

	if err := writer.Write(entry); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read audit file: %v", err)
	}

	// Verify JSON Lines format (one JSON object per line)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if result["action"] != "login" {
		t.Errorf("expected action 'login', got %v", result["action"])
	}
	if result["user_id"] != "user-1" {
		t.Errorf("expected user_id 'user-1', got %v", result["user_id"])
	}
	if result["timestamp"] == nil {
		t.Error("expected timestamp to be set")
	}
}

func TestFileAuditWriter_MultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	writer, err := NewFileAuditWriter(filePath)
	if err != nil {
		t.Fatalf("NewFileAuditWriter() error = %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := writer.Write(AuditEntry{Action: "test", UserID: "user-1"}); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	writer.Close()

	data, _ := os.ReadFile(filePath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestFileAuditWriter_Close(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	writer, err := NewFileAuditWriter(filePath)
	if err != nil {
		t.Fatalf("NewFileAuditWriter() error = %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestMultiAuditWriter(t *testing.T) {
	tmpDir := t.TempDir()
	filePath1 := filepath.Join(tmpDir, "audit1.log")
	filePath2 := filepath.Join(tmpDir, "audit2.log")

	w1, _ := NewFileAuditWriter(filePath1)
	w2, _ := NewFileAuditWriter(filePath2)
	multi := NewMultiAuditWriter(w1, w2)

	entry := AuditEntry{Action: "multi-test", UserID: "user-1"}
	if err := multi.Write(entry); err != nil {
		t.Fatalf("MultiAuditWriter.Write() error = %v", err)
	}
	multi.Close()

	for _, path := range []string{filePath1, filePath2} {
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "multi-test") {
			t.Errorf("expected 'multi-test' in %s", path)
		}
	}
}
