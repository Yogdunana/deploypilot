package util

import (
	"strings"
	"testing"
)

// TestShellQuote_PlainString verifies basic safe quoting of plain text.
func TestShellQuote_PlainString(t *testing.T) {
	got := ShellQuote("hello world")
	if got != "'hello world'" {
		t.Errorf("ShellQuote(plain) = %q, want %q", got, "'hello world'")
	}
}

// TestShellQuote_EmptyString verifies that empty input still produces a valid quoted form.
func TestShellQuote_EmptyString(t *testing.T) {
	got := ShellQuote("")
	if got != "''" {
		t.Errorf("ShellQuote(empty) = %q, want %q", got, "''")
	}
}

// TestShellQuote_SingleQuoteEscape ensures embedded single quotes are properly
// escaped so attackers cannot break out of the quoted argument.
func TestShellQuote_SingleQuoteEscape(t *testing.T) {
	got := ShellQuote("o'malley")
	// Pattern must be: 'o'"'"'malley'
	want := "'o'\"'\"'malley'"
	if got != want {
		t.Errorf("ShellQuote(quoted) = %q, want %q", got, want)
	}
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("ShellQuote output is not wrapped in single quotes: %s", got)
	}
}

// TestShellQuote_NewlineReplaced confirms that newlines cannot be used to
// inject additional shell commands. They must be replaced with spaces.
func TestShellQuote_NewlineReplaced(t *testing.T) {
	got := ShellQuote("foo\nrm -rf /")
	if strings.Contains(got, "\n") {
		t.Errorf("ShellQuote did not strip newline: %q", got)
	}
	if !strings.Contains(got, "rm -rf /") {
		t.Errorf("ShellQuote should preserve neutralized payload for inspection, got %q", got)
	}
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("ShellQuote output is not wrapped in single quotes: %s", got)
	}
}

// TestShellQuote_CarriageReturnReplaced confirms carriage returns are neutralized
// to prevent line-feed bypass tricks on some shells.
func TestShellQuote_CarriageReturnReplaced(t *testing.T) {
	got := ShellQuote("foo\rbar")
	if strings.Contains(got, "\r") {
		t.Errorf("ShellQuote did not strip CR: %q", got)
	}
}

// TestShellQuote_TabAndControlChars confirms tabs and other control characters
// are replaced with spaces.
func TestShellQuote_TabAndControlChars(t *testing.T) {
	got := ShellQuote("foo\tbar\x07baz")
	if strings.ContainsAny(got, "\t\x07") {
		t.Errorf("ShellQuote did not strip tab/bell: %q", got)
	}
	// The three literal parts should now be separated by spaces
	if !strings.Contains(got, "foo bar baz") {
		t.Errorf("ShellQuote should join neutralized parts with spaces, got %q", got)
	}
}

// TestShellQuote_InjectionAttempts ensures dangerous payloads remain harmless
// once quoted. The attacker-controlled string must always start and end with
// a single quote, and never contain a raw newline that would terminate a
// command list in a shell.
func TestShellQuote_InjectionAttempts(t *testing.T) {
	cases := []string{
		"foo; echo pwned",
		"foo && echo pwned",
		"foo | cat /etc/passwd",
		"$(echo pwned)",
		"`echo pwned`",
		"foo > /etc/passwd",
		"foo\nrm -rf /\n",
	}
	for _, c := range cases {
		got := ShellQuote(c)
		if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
			t.Errorf("ShellQuote(%q) not wrapped in single quotes: %s", c, got)
		}
		if strings.Contains(got, "\n") {
			t.Errorf("ShellQuote(%q) contains a raw newline: %s", c, got)
		}
	}
}
