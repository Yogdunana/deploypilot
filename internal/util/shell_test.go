package util

import (
	"strings"
	"testing"
)

func TestShellQuote_NormalString(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello", "'hello'"},
		{"hello world", "'hello world'"},
		{"test123", "'test123'"},
		{"my-file.txt", "'my-file.txt'"},
		{"path/to/file", "'path/to/file'"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ShellQuote(tc.input)
			if result != tc.expected {
				t.Errorf("ShellQuote(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestShellQuote_SpecialCharacters(t *testing.T) {
	testCases := []struct {
		input string
	}{
		{"hello; rm -rf /"},
		{"$(rm -rf /)"},
		{"`rm -rf /`"},
		{"|| rm -rf /"},
		{"&& rm -rf /"},
		{"; ls"},
		{"< /etc/passwd"},
		{"> /dev/null"},
		{"'\"`$\\!@#%^&*()"},
	}

	for _, tc := range testCases {
		t.Run("special_chars", func(t *testing.T) {
			result := ShellQuote(tc.input)
			if !strings.HasPrefix(result, "'") || !strings.HasSuffix(result, "'") {
				t.Errorf("ShellQuote(%q) = %q, expected to be wrapped in single quotes", tc.input, result)
			}
		})
	}
}

func TestShellQuote_SingleQuotes(t *testing.T) {
	input := "it's a test"
	result := ShellQuote(input)

	expected := "'it'\"'\"'s a test'"

	if result != expected {
		t.Errorf("ShellQuote(%q) = %q, want %q", input, result, expected)
	}
}

func TestShellQuote_EmptyString(t *testing.T) {
	result := ShellQuote("")
	if result != "''" {
		t.Errorf("ShellQuote(\"\") = %q, want %q", result, "''")
	}
}

func TestShellQuote_ControlCharacters(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"newline", "hello\nworld"},
		{"tab", "hello\tworld"},
		{"carriage_return", "hello\rworld"},
		{"null", "hello\x00world"},
		{"bell", "hello\x07world"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ShellQuote(tc.input)

			if strings.Contains(result, "\n") {
				t.Error("ShellQuote should replace newline characters")
			}
			if strings.Contains(result, "\t") {
				t.Error("ShellQuote should replace tab characters")
			}
			if strings.Contains(result, "\r") {
				t.Error("ShellQuote should replace carriage return characters")
			}

			if !strings.HasPrefix(result, "'") || !strings.HasSuffix(result, "'") {
				t.Errorf("ShellQuote(%q) = %q, expected to be wrapped in single quotes", tc.input, result)
			}
		})
	}
}

func TestShellQuote_LongString(t *testing.T) {
	longString := strings.Repeat("a", 1000)
	result := ShellQuote(longString)

	if !strings.HasPrefix(result, "'") || !strings.HasSuffix(result, "'") {
		t.Errorf("ShellQuote long string should be wrapped in quotes")
	}

	if len(result) != len(longString)+2 {
		t.Errorf("ShellQuote long string length = %d, want %d", len(result), len(longString)+2)
	}
}

func TestShellQuote_Unicode(t *testing.T) {
	testCases := []string{
		"你好世界",
		"こんにちは",
		"Привет",
		"Γειά σου",
		"Hello 世界 123",
	}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			result := ShellQuote(input)

			if !strings.HasPrefix(result, "'") || !strings.HasSuffix(result, "'") {
				t.Errorf("ShellQuote(%q) = %q, expected to be wrapped in single quotes", input, result)
			}
		})
	}
}

func TestShellQuote_ShellInjectionAttempt(t *testing.T) {
	injectionAttempts := []string{
		"'; rm -rf /; echo '",
		"'; cat /etc/passwd; '",
		"\"; ls; echo \"",
		"$(cat /etc/passwd)",
		"`cat /etc/passwd`",
	}

	for _, attempt := range injectionAttempts {
		t.Run("injection_attempt", func(t *testing.T) {
			result := ShellQuote(attempt)

			if !strings.HasPrefix(result, "'") || !strings.HasSuffix(result, "'") {
				t.Errorf("ShellQuote(%q) = %q, expected to be wrapped in single quotes for injection protection", attempt, result)
			}

			if strings.Contains(attempt, "'") {
				if !strings.Contains(result, "'\"'\"'") {
					t.Errorf("ShellQuote(%q) = %q, expected escaped single quotes", attempt, result)
				}
			}
		})
	}
}