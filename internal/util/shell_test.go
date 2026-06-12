package util

import (
	"strings"
	"testing"
	"unicode"
)

func TestShellQuote_Basic(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "''",
		},
		{
			name:     "simple string",
			input:    "hello",
			expected: "'hello'",
		},
		{
			name:     "string with spaces",
			input:    "hello world",
			expected: "'hello world'",
		},
		{
			name:     "string with single quote",
			input:    "it's a test",
			expected: "'it'\"'\"'s a test'",
		},
		{
			name:     "string with multiple single quotes",
			input:    "don't don't",
			expected: "'don'\"'\"'t don'\"'\"'t'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ShellQuote(tc.input)
			if result != tc.expected {
				t.Errorf("ShellQuote(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestShellQuote_ControlCharacters(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "newline",
			input: "hello\nworld",
		},
		{
			name:  "carriage return",
			input: "hello\rworld",
		},
		{
			name:  "tab",
			input: "hello\tworld",
		},
		{
			name:  "null byte",
			input: "hello\x00world",
		},
		{
			name:  "bell",
			input: "hello\x07world",
		},
		{
			name:  "escape",
			input: "hello\x1bworld",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ShellQuote(tc.input)
			for _, r := range result {
				if unicode.IsControl(r) && r != '\t' {
					t.Errorf("ShellQuote(%q) = %q contains control character %q", tc.input, result, r)
				}
			}
			if strings.Contains(result, "\n") {
				t.Errorf("ShellQuote(%q) = %q contains newline", tc.input, result)
			}
		})
	}
}

func TestShellQuote_ShellInjection(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldBlock bool
	}{
		{
			name:        "command injection via newline",
			input:       "test; rm -rf /\n",
			shouldBlock: true,
		},
		{
			name:        "command injection via semicolon",
			input:       "test; rm -rf /",
			shouldBlock: false, // semicolon is allowed in single quotes
		},
		{
			name:        "backtick injection",
			input:       "test`rm -rf /`",
			shouldBlock: false, // backtick is allowed in single quotes
		},
		{
			name:        "$() injection",
			input:       "test$(rm -rf /)",
			shouldBlock: false, // $() is allowed in single quotes
		},
		{
			name:        "pipe injection",
			input:       "test|rm -rf /",
			shouldBlock: false, // pipe is allowed in single quotes
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ShellQuote(tc.input)
			if tc.shouldBlock {
				if strings.Contains(result, "\n") {
					t.Errorf("ShellQuote(%q) = %q should not contain newline", tc.input, result)
				}
			}
		})
	}
}

func TestShellQuote_SpecialCharacters(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "dollar sign",
			input:    "price is $100",
			expected: "'price is $100'",
		},
		{
			name:     "backtick",
			input:    "echo `date`",
			expected: "'echo `date`'",
		},
		{
			name:     "pipe",
			input:    "ls | grep test",
			expected: "'ls | grep test'",
		},
		{
			name:     "ampersand",
			input:    "cmd &",
			expected: "'cmd &'",
		},
		{
			name:     "redirect",
			input:    "cmd > output.txt",
			expected: "'cmd > output.txt'",
		},
		{
			name:     "question mark",
			input:    "what?",
			expected: "'what?'",
		},
		{
			name:     "asterisk",
			input:    "file*.txt",
			expected: "'file*.txt'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ShellQuote(tc.input)
			if result != tc.expected {
				t.Errorf("ShellQuote(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}