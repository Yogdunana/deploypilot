package util

import (
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
		{"string with single quote", "it's a test", "'it'\"'\"'s a test'"},
		{"string with multiple single quotes", "don't don't", "'don'\"'\"'t don'\"'\"'t'"},
		{"string with special chars", "test$var; rm -rf", "'test$var; rm -rf'"},
		{"string with backslash", "path\\to\\file", "'path\\to\\file'"},
		{"string with newlines", "line1\nline2", "'line1 line2'"},
		{"string with tabs", "col1\tcol2", "'col1 col2'"},
		{"string with control chars", "hello\x00world", "'hello world'"},
		{"string with quotes and spaces", "he said 'hello'", "'he said '\"'\"'hello'\"'\"''"},
		{"path with spaces", "/my documents/file.txt", "'/my documents/file.txt'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShellQuote(tt.input)
			if result != tt.expected {
				t.Errorf("ShellQuote(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShellQuote_PreservesValidContent(t *testing.T) {
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-/@$%^&*()"
	result := ShellQuote(validChars)
	
	expected := "'" + validChars + "'"
	if result != expected {
		t.Errorf("ShellQuote() did not preserve valid characters: got %q, want %q", result, expected)
	}
}

func TestShellQuote_RejectsControlCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
	}{
		{"null byte", "test\x00"},
		{"bell", "test\x07"},
		{"backspace", "test\x08"},
		{"carriage return", "test\r"},
		{"vertical tab", "test\x0b"},
		{"form feed", "test\x0c"},
		{"del", "test\x7f"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShellQuote(tt.input)
			if len(result) < 2 {
				t.Error("ShellQuote() returned unexpected result")
			}
			if result[0] != '\'' || result[len(result)-1] != '\'' {
				t.Errorf("ShellQuote() should wrap result in single quotes")
			}
		})
	}
}

func TestShellQuote_ScriptInjection(t *testing.T) {
	injectionAttempts := []string{
		"; rm -rf /",
		"&& rm -rf /",
		"|| rm -rf /",
		"`rm -rf /`",
		"$(rm -rf /)",
		"> /etc/passwd",
		"< /etc/shadow",
	}

	for _, attempt := range injectionAttempts {
		t.Run("injection: "+attempt, func(t *testing.T) {
			result := ShellQuote(attempt)
			if !isSafeShellArgument(result) {
				t.Errorf("ShellQuote(%q) = %q may be vulnerable to injection", attempt, result)
			}
		})
	}
}

func isSafeShellArgument(arg string) bool {
	if len(arg) < 2 || arg[0] != '\'' || arg[len(arg)-1] != '\'' {
		return false
	}
	return true
}