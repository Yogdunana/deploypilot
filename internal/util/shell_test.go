package util

import (
	"strings"
	"testing"
)

func TestShellQuote_BasicCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "''"},
		{"simple word", "hello", "'hello'"},
		{"word with spaces", "hello world", "'hello world'"},
		{"double quotes only", `hello"world`, `'hello"world'`},
		{"no special chars", `abc123`, `'abc123'`},
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

func TestShellQuote_SingleQuotes(t *testing.T) {
	result := ShellQuote("'")
	expected := "'" + strings.Replace("'", "'", "'\"'\"'", -1) + "'"
	if result != expected {
		t.Errorf("ShellQuote('') = %q, want %q", result, expected)
	}

	result2 := ShellQuote("'foo'")
	expected2 := "'" + strings.Replace("'foo'", "'", "'\"'\"'", -1) + "'"
	if result2 != expected2 {
		t.Errorf("ShellQuote('foo') = %q, want %q", result2, expected2)
	}
}

func TestShellQuote_ControlCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"newline", "hello\nworld"},
		{"carriage return", "hello\rworld"},
		{"tab", "hello\tworld"},
		{"null byte", "hello\x00world"},
		{"bell", "hello\x07world"},
		{"escape", "hello\x1bworld"},
		{"mixed controls", "line1\nline2\ttab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShellQuote(tt.input)
			if len(result) == 0 {
				t.Error("ShellQuote should return non-empty result")
			}
			if result[0] != '\'' || result[len(result)-1] != '\'' {
				t.Error("ShellQuote result should be wrapped in single quotes")
			}
		})
	}
}

func TestShellQuote_ShellInjection(t *testing.T) {
	injectionAttempts := []string{
		`; rm -rf /`,
		`$(rm -rf /)`,
		`|| rm -rf /`,
		`&& rm -rf /`,
		`> /etc/passwd`,
		`< /etc/shadow`,
		`| cat /etc/passwd`,
		"`rm -rf /`",
	}

	for _, attempt := range injectionAttempts {
		t.Run("injection: "+attempt, func(t *testing.T) {
			result := ShellQuote(attempt)
			if result[0] != '\'' || result[len(result)-1] != '\'' {
				t.Error("ShellQuote should wrap potentially dangerous input in single quotes")
			}
		})
	}
}

func TestShellQuote_Unicode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"chinese", "你好世界", "'你好世界'"},
		{"japanese", "こんにちは", "'こんにちは'"},
		{"emoji", "🚀🎉", "'🚀🎉'"},
		{"mixed unicode", "Hello 世界 こんにちは", "'Hello 世界 こんにちは'"},
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