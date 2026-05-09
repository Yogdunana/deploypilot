package deployer

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/util"
)

// LogEntry represents a single log line from a container.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Container string    `json:"container"`
	Stream    string    `json:"stream"` // stdout, stderr
	Message   string    `json:"message"`
}

// LogStream provides real-time log streaming via channels.
type LogStream struct {
	entries  chan LogEntry
	done     chan struct{}
	mu       sync.Mutex
	closed   bool
}

// NewLogStream creates a new LogStream.
func NewLogStream(bufferSize int) *LogStream {
	return &LogStream{
		entries: make(chan LogEntry, bufferSize),
		done:    make(chan struct{}),
	}
}

// Entries returns the channel for reading log entries.
func (ls *LogStream) Entries() <-chan LogEntry {
	return ls.entries
}

// Write writes a log entry to the stream.
func (ls *LogStream) Write(entry LogEntry) bool {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.closed {
		return false
	}
	select {
	case ls.entries <- entry:
		return true
	case <-ls.done:
		return false
	}
}

// Close closes the log stream.
func (ls *LogStream) Close() {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.closed {
		return
	}
	ls.closed = true
	close(ls.done)
	close(ls.entries)
}

// LogReader reads container logs (history + follow).
type LogReader struct {
	executor CommandExecutor
}

// NewLogReader creates a new LogReader.
func NewLogReader(executor CommandExecutor) *LogReader {
	return &LogReader{executor: executor}
}

// GetHistory retrieves historical logs for a container.
func (lr *LogReader) GetHistory(ctx context.Context, containerName string, tail int) ([]LogEntry, error) {
	cmd := fmt.Sprintf("docker logs --tail %d %s 2>&1", tail, util.ShellQuote(containerName))
	output, err := lr.executor.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	return parseLogLines(containerName, output), nil
}

// StreamLogs streams logs in real-time using docker logs --follow.
// It writes entries to the provided LogStream until context is cancelled.
func (lr *LogReader) StreamLogs(ctx context.Context, containerName string, stream *LogStream) error {
	cmd := fmt.Sprintf("docker logs --follow --tail 0 %s 2>&1", util.ShellQuote(containerName))

	// In production, this would use an SSH session with stdin/stdout pipes.
	// For now, we simulate by running the command and parsing output.
	output, err := lr.executor.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to stream logs: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		entry := LogEntry{
			Timestamp: time.Now(),
			Container: containerName,
			Stream:    "stdout",
			Message:   line,
		}
		if !stream.Write(entry) {
			break // stream closed
		}
	}

	return nil
}

// StreamLogsSimulated simulates real-time log streaming for testing.
// It generates log entries at the given interval until context is cancelled.
func StreamLogsSimulated(ctx context.Context, containerName string, stream *LogStream, interval time.Duration, messages []string) {
	for i := 0; i < len(messages); i++ {
		select {
		case <-ctx.Done():
			return
		case <-stream.done:
			return
		default:
		}

		entry := LogEntry{
			Timestamp: time.Now(),
			Container: containerName,
			Stream:    "stdout",
			Message:   messages[i],
		}

		if !stream.Write(entry) {
			return
		}

		if i < len(messages)-1 {
			time.Sleep(interval)
		}
	}
}

// CollectLogs reads all entries from a LogStream until it's closed or timeout.
func CollectLogs(stream *LogStream, timeout time.Duration) []LogEntry {
	var entries []LogEntry
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case entry, ok := <-stream.Entries():
			if !ok {
				return entries
			}
			entries = append(entries, entry)
		case <-timer.C:
			return entries
		}
	}
}

// FilterLogs filters log entries by a predicate function.
func FilterLogs(entries []LogEntry, predicate func(LogEntry) bool) []LogEntry {
	var filtered []LogEntry
	for _, entry := range entries {
		if predicate(entry) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// SearchLogs searches log entries for a substring.
func SearchLogs(entries []LogEntry, query string) []LogEntry {
	return FilterLogs(entries, func(e LogEntry) bool {
		return strings.Contains(e.Message, query)
	})
}

// FormatLogs formats log entries as plain text.
func FormatLogs(entries []LogEntry) string {
	var sb strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&sb, "[%s] [%s] %s\n",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Stream,
			entry.Message)
	}
	return sb.String()
}

// LogWriter implements io.Writer for streaming logs.
type LogWriter struct {
	stream   *LogStream
	container string
}

// NewLogWriter creates a new LogWriter.
func NewLogWriter(stream *LogStream, container string) *LogWriter {
	return &LogWriter{stream: stream, container: container}
}

// Write implements io.Writer.
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	scanner := bufio.NewScanner(strings.NewReader(string(p)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		lw.stream.Write(LogEntry{
			Timestamp: time.Now(),
			Container: lw.container,
			Stream:    "stdout",
			Message:   line,
		})
	}
	return len(p), nil
}

// parseLogLines parses raw docker log output into LogEntry slices.
func parseLogLines(containerName, output string) []LogEntry {
	var entries []LogEntry
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		entries = append(entries, LogEntry{
			Timestamp: time.Now(),
			Container: containerName,
			Stream:    detectStream(line),
			Message:   line,
		})
	}
	return entries
}

// detectStream guesses stdout vs stderr from log content.
func detectStream(line string) string {
	// Docker prefixes stderr lines, but in combined mode we just default to stdout
	return "stdout"
}
