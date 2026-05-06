package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditWriter defines the interface for writing audit entries to external storage.
type AuditWriter interface {
	Write(entry AuditEntry) error
	Close() error
}

// FileAuditWriter writes audit entries to a file in JSON Lines format.
type FileAuditWriter struct {
	filePath string
	file     *os.File
	mu       sync.Mutex
}

// NewFileAuditWriter creates a new file audit writer. The file is opened in append mode.
func NewFileAuditWriter(filePath string) (*FileAuditWriter, error) {
	// Ensure parent directory exists
	if dir := filepath.Dir(filePath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	return &FileAuditWriter{filePath: filePath, file: f}, nil
}

// Write appends an audit entry as a JSON line to the file.
func (w *FileAuditWriter) Write(entry AuditEntry) error {
	// Add timestamp
	data := map[string]interface{}{
		"timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
		"user_id":      fmt.Sprintf("%d", entry.UserID),
		"username":     entry.Username,
		"action":       entry.Action,
		"resource_type": entry.ResourceType,
		"resource_id":  entry.ResourceID,
		"detail":       sanitizeAuditData(entry.Detail),
		"ip_address":   entry.IPAddress,
		"user_agent":   entry.UserAgent,
	}

	line, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.file.Write(append(line, '\n'))
	return err
}

// Close flushes and closes the underlying file.
func (w *FileAuditWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

// MultiAuditWriter writes audit entries to multiple writers simultaneously.
type MultiAuditWriter struct {
	writers []AuditWriter
}

// NewMultiAuditWriter creates a writer that writes to all provided writers.
func NewMultiAuditWriter(writers ...AuditWriter) *MultiAuditWriter {
	return &MultiAuditWriter{writers: writers}
}

// Write writes the entry to all underlying writers.
func (w *MultiAuditWriter) Write(entry AuditEntry) error {
	for _, writer := range w.writers {
		if err := writer.Write(entry); err != nil {
			return err
		}
	}
	return nil
}

// Close closes all underlying writers.
func (w *MultiAuditWriter) Close() error {
	var firstErr error
	for _, writer := range w.writers {
		if err := writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
