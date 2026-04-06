package deployer

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ========== LogStream ==========

func TestLogStreamWriteAndRead(t *testing.T) {
	stream := NewLogStream(10)
	defer stream.Close()

	entry := LogEntry{
		Timestamp: time.Now(),
		Container: "my-app",
		Stream:    "stdout",
		Message:   "hello world",
	}

	ok := stream.Write(entry)
	if !ok {
		t.Error("Write should succeed")
	}

	read := <-stream.Entries()
	if read.Message != "hello world" {
		t.Errorf("Message = %q", read.Message)
	}
	if read.Container != "my-app" {
		t.Errorf("Container = %q", read.Container)
	}
}

func TestLogStreamClose(t *testing.T) {
	stream := NewLogStream(10)
	stream.Close()

	_, ok := <-stream.Entries()
	if ok {
		t.Error("Entries channel should be closed")
	}
}

func TestLogStreamWriteAfterClose(t *testing.T) {
	stream := NewLogStream(10)
	stream.Close()

	ok := stream.Write(LogEntry{Message: "test"})
	if ok {
		t.Error("Write after close should return false")
	}
}

func TestLogStreamBuffered(t *testing.T) {
	stream := NewLogStream(100)
	defer stream.Close()

	for i := 0; i < 50; i++ {
		stream.Write(LogEntry{Message: strings.Repeat("x", 10)})
	}

	// Should be able to read all
	count := 0
	for range stream.Entries() {
		count++
		if count >= 50 {
			break
		}
	}
	if count != 50 {
		t.Errorf("Expected 50 entries, got %d", count)
	}
}

// ========== CollectLogs ==========

func TestCollectLogs(t *testing.T) {
	stream := NewLogStream(100)

	go func() {
		stream.Write(LogEntry{Message: "line 1"})
		stream.Write(LogEntry{Message: "line 2"})
		stream.Write(LogEntry{Message: "line 3"})
		stream.Close()
	}()

	entries := CollectLogs(stream, 2*time.Second)
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}
}

func TestCollectLogsTimeout(t *testing.T) {
	stream := NewLogStream(100)

	go func() {
		stream.Write(LogEntry{Message: "line 1"})
		// Don't close — let timeout handle it
	}()

	entries := CollectLogs(stream, 100*time.Millisecond)
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry (timeout), got %d", len(entries))
	}
	stream.Close()
}

// ========== StreamLogsSimulated ==========

func TestStreamLogsSimulated(t *testing.T) {
	stream := NewLogStream(100)

	messages := []string{"Starting server...", "Listening on :8080", "Ready"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	go StreamLogsSimulated(ctx, "my-app", stream, 10*time.Millisecond, messages)

	entries := CollectLogs(stream, 3*time.Second)
	cancel()

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	if entries[0].Message != "Starting server..." {
		t.Errorf("First message = %q", entries[0].Message)
	}
	if entries[2].Message != "Ready" {
		t.Errorf("Last message = %q", entries[2].Message)
	}
}

func TestStreamLogsSimulatedCancelled(t *testing.T) {
	stream := NewLogStream(100)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	StreamLogsSimulated(ctx, "my-app", stream, 10*time.Millisecond,
		[]string{"line1", "line2", "line3", "line4", "line5"})

	stream.Close()
	entries := CollectLogs(stream, 100*time.Millisecond)
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries (cancelled), got %d", len(entries))
	}
}

// ========== GetHistory ==========

func TestGetHistory(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker logs"] = "2026-04-06 12:00:00 Server started\n2026-04-06 12:00:01 Ready on :8080\n2026-04-06 12:00:02 Request received"

	lr := NewLogReader(mock)
	entries, err := lr.GetHistory(context.Background(), "my-app", 100)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Message, "Server started") {
		t.Errorf("First entry = %q", entries[0].Message)
	}
}

func TestGetHistoryEmpty(t *testing.T) {
	mock := newMockExecutor()
	mock.responses["docker logs"] = ""

	lr := NewLogReader(mock)
	entries, err := lr.GetHistory(context.Background(), "my-app", 100)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

// ========== FilterLogs ==========

func TestFilterLogs(t *testing.T) {
	entries := []LogEntry{
		{Message: "error: connection refused"},
		{Message: "info: server started"},
		{Message: "error: timeout"},
		{Message: "info: request received"},
	}

	errors := FilterLogs(entries, func(e LogEntry) bool {
		return strings.Contains(e.Message, "error:")
	})

	if len(errors) != 2 {
		t.Errorf("Expected 2 error entries, got %d", len(errors))
	}
}

// ========== SearchLogs ==========

func TestSearchLogs(t *testing.T) {
	entries := []LogEntry{
		{Message: "Server started on port 8080"},
		{Message: "Database connected"},
		{Message: "Listening on port 8080"},
	}

	results := SearchLogs(entries, "port 8080")
	if len(results) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(results))
	}

	results = SearchLogs(entries, "nonexistent")
	if len(results) != 0 {
		t.Errorf("Expected 0 matches, got %d", len(results))
	}
}

// ========== FormatLogs ==========

func TestFormatLogs(t *testing.T) {
	entries := []LogEntry{
		{Timestamp: time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC), Stream: "stdout", Message: "hello"},
		{Timestamp: time.Date(2026, 4, 6, 12, 0, 1, 0, time.UTC), Stream: "stderr", Message: "error"},
	}

	output := FormatLogs(entries)
	if !strings.Contains(output, "hello") {
		t.Error("FormatLogs should contain 'hello'")
	}
	if !strings.Contains(output, "error") {
		t.Error("FormatLogs should contain 'error'")
	}
	if !strings.Contains(output, "stdout") {
		t.Error("FormatLogs should contain stream type")
	}
}

// ========== LogWriter ==========

func TestLogWriter(t *testing.T) {
	stream := NewLogStream(100)
	defer stream.Close()

	writer := NewLogWriter(stream, "my-app")

	_, err := writer.Write([]byte("line one\nline two\nline three\n"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	entries := CollectLogs(stream, 1*time.Second)
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}
	if entries[0].Message != "line one" {
		t.Errorf("First message = %q", entries[0].Message)
	}
}
