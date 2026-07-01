package util

import (
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "''"},
		{"simple string", "hello", "'hello'"},
		{"string with spaces", "hello world", "'hello world'"},
		{"string with single quote", `it's a test`, "'it'\"'\"'s a test'"},
		{"string with multiple single quotes", `don't don't`, "'don'\"'\"'t don'\"'\"'t'"},
		{"string with special chars", "hello; rm -rf /", "'hello; rm -rf /'"},
		{"string with backticks", "echo `date`", "'echo `date`'"},
		{"string with dollar sign", "echo $HOME", "'echo $HOME'"},
		{"string with newline", "line1\nline2", "'line1 line2'"},
		{"string with tab", "hello\tworld", "'hello world'"},
		{"string with null byte", "hello\x00world", "'hello world'"},
		{"string with control chars", "hello\r\n\tworld", "'hello   world'"},
		{"string with quotes and spaces", `'single' "double"`, "''\"'\"'single'\"'\"' \"double\"'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShellQuote(tt.input)
			if result != tt.expected {
				t.Errorf("ShellQuote(%q) = %q, want %q", tt.input, result, tt.expected)
			}
			if !strings.HasPrefix(result, "'") || !strings.HasSuffix(result, "'") {
				t.Errorf("ShellQuote(%q) should be wrapped in single quotes, got %q", tt.input, result)
			}
		})
	}
}