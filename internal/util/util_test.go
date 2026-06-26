package util

import (
	"net/http"
	"testing"
	"time"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "'simple'"},
		{"", "''"},
		{"hello world", "'hello world'"},
		{"file.txt", "'file.txt'"},
		{"path/to/file", "'path/to/file'"},
		{"with'quote", `'with'"'"'quote'`},
		{"with\"double", "'with\"double'"},
		{"$var", "'$var'"},
		{"cmd && echo", "'cmd && echo'"},
	}

	for _, tt := range tests {
		result := ShellQuote(tt.input)
		if result != tt.expected {
			t.Errorf("ShellQuote(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestShellQuote_ControlCharacters(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello\nworld", "'hello world'"},
		{"hello\r\nworld", "'hello  world'"},
		{"hello\tworld", "'hello world'"},
		{"hello\x00world", "'hello world'"},
	}

	for _, tt := range tests {
		result := ShellQuote(tt.input)
		if result != tt.expected {
			t.Errorf("ShellQuote(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDefaultClient(t *testing.T) {
	if DefaultClient == nil {
		t.Fatal("DefaultClient should not be nil")
	}

	if DefaultClient.Timeout != 30*time.Second {
		t.Errorf("DefaultClient.Timeout = %v, want 30s", DefaultClient.Timeout)
	}

	transport, ok := DefaultClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("DefaultClient.Transport should be *http.Transport")
	}

	if transport.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 10", transport.MaxIdleConnsPerHost)
	}

	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", transport.IdleConnTimeout)
	}
}