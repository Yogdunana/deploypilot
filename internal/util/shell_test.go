package util

import (
	"strings"
	"testing"
)

func TestShellQuote_SimpleString(t *testing.T) {
	result := ShellQuote("hello")
	expected := "'hello'"
	if result != expected {
		t.Errorf("ShellQuote(%q) = %q, want %q", "hello", result, expected)
	}
}

func TestShellQuote_EmptyString(t *testing.T) {
	result := ShellQuote("")
	expected := "''"
	if result != expected {
		t.Errorf("ShellQuote(%q) = %q, want %q", "", result, expected)
	}
}

func TestShellQuote_SingleQuote(t *testing.T) {
	result := ShellQuote("it's")
	expected := "'it'\"'\"'s'"
	if result != expected {
		t.Errorf("ShellQuote(%q) = %q, want %q", "it's", result, expected)
	}
}

func TestShellQuote_MultipleSingleQuotes(t *testing.T) {
	result := ShellQuote("a'b'c")
	expected := "'a'\"'\"'b'\"'\"'c'"
	if result != expected {
		t.Errorf("ShellQuote(%q) = %q, want %q", "a'b'c", result, expected)
	}
}

func TestShellQuote_Spaces(t *testing.T) {
	result := ShellQuote("hello world")
	expected := "'hello world'"
	if result != expected {
		t.Errorf("ShellQuote(%q) = %q, want %q", "hello world", result, expected)
	}
}

func TestShellQuote_SpecialChars(t *testing.T) {
	specialChars := []string{
		"hello$var",
		"hello`cmd`",
		"hello; rm -rf /",
		"hello && echo pwned",
		"hello || echo pwned",
		"hello > /etc/passwd",
		"hello < /etc/passwd",
		"hello (world)",
		"hello {world}",
		"hello [world]",
		"hello\\world",
		"hello!world",
		"hello~world",
		"hello|world",
		"hello&world",
	}
	for _, s := range specialChars {
		result := ShellQuote(s)
		if !strings.HasPrefix(result, "'") || !strings.HasSuffix(result, "'") {
			t.Errorf("ShellQuote(%q) = %q, should be wrapped in single quotes", s, result)
		}
	}
}

func TestShellQuote_NewlineReplaced(t *testing.T) {
	result := ShellQuote("hello\nworld")
	if strings.Contains(result, "\n") {
		t.Errorf("ShellQuote should replace newlines, got: %q", result)
	}
	if !strings.Contains(result, "hello") || !strings.Contains(result, "world") {
		t.Errorf("ShellQuote should preserve content (with newline replaced), got: %q", result)
	}
}

func TestShellQuote_TabReplaced(t *testing.T) {
	result := ShellQuote("hello\tworld")
	if strings.ContainsRune(result, '\t') {
		t.Errorf("ShellQuote should replace tabs, got: %q", result)
	}
}

func TestShellQuote_ControlCharsReplaced(t *testing.T) {
	controlChars := []rune{
		'\x00', '\x01', '\x02', '\x03', '\x04', '\x05', '\x06', '\x07',
		'\x08', '\x0b', '\x0c', '\x0e', '\x0f',
		'\x10', '\x11', '\x12', '\x13', '\x14', '\x15', '\x16', '\x17',
		'\x18', '\x19', '\x1a', '\x1b', '\x1c', '\x1d', '\x1e', '\x1f',
		'\x7f',
	}
	for _, c := range controlChars {
		s := "hello" + string(c) + "world"
		result := ShellQuote(s)
		if strings.ContainsRune(result, c) {
			t.Errorf("ShellQuote should replace control char %U, got: %q", c, result)
		}
	}
}

func TestShellQuote_Unicode(t *testing.T) {
	result := ShellQuote("你好世界")
	expected := "'你好世界'"
	if result != expected {
		t.Errorf("ShellQuote(%q) = %q, want %q", "你好世界", result, expected)
	}
}

func TestShellQuote_AllQuotes(t *testing.T) {
	result := ShellQuote(`'")`)
	if !strings.HasPrefix(result, "'") || !strings.HasSuffix(result, "'") {
		t.Errorf("ShellQuote should wrap in single quotes, got: %q", result)
	}
}

func TestShellQuote_PathTraversal(t *testing.T) {
	result := ShellQuote("../../../etc/passwd")
	expected := "'../../../etc/passwd'"
	if result != expected {
		t.Errorf("ShellQuote(%q) = %q, want %q", "../../../etc/passwd", result, expected)
	}
}
