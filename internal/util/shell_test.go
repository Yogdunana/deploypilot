package util

import (
	"strings"
	"testing"
)

// TestShellQuote_SafePassthrough verifies that ordinary alphanumeric
// strings are wrapped in single quotes and that no metacharacter
// expansion happens. This is the most common usage pattern across the
// codebase (paths, image references, container names, etc.).
func TestShellQuote_SafePassthrough(t *testing.T) {
	cases := []string{
		"hello",
		"image:tag",
		"my-container_v2",
		"/var/lib/data",
		"redis-7.0.1",
	}
	for _, in := range cases {
		got := ShellQuote(in)
		if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
			t.Errorf("ShellQuote(%q) = %q, want wrapped in single quotes", in, got)
		}
		// No bare (unescaped) single quotes inside the output.
		if hasBareSingleQuote(got) {
			t.Errorf("ShellQuote(%q) = %q contains a bare single quote", in, got)
		}
	}
}

// TestShellQuote_NeutralizesInjection verifies that shell metacharacters
// that would normally break out of an argument or chain commands are
// rendered inert. This is the central reason this helper exists.
func TestShellQuote_NeutralizesInjection(t *testing.T) {
	dangerous := []string{
		`evil"; rm -rf / #`, // quote-break + command chain
		"$(whoami)",         // command substitution
		"`id`",              // backtick substitution
		"a && b && c",       // command chaining
		"a | b",             // pipe
		"a > /etc/passwd",   // redirection
		"plain'single",      // embedded single quote
		"$HOME",             // variable expansion
	}
	for _, in := range dangerous {
		got := ShellQuote(in)
		if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
			t.Errorf("ShellQuote(%q) = %q, want wrapped in single quotes", in, got)
		}
		// The quoted form must remain a single shell token: it must
		// not contain a bare (un-escaped) single quote, otherwise the
		// caller would have a quoting hole.
		if hasBareSingleQuote(got) {
			t.Errorf("ShellQuote(%q) = %q contains a bare single quote", in, got)
		}
	}
}

// TestShellQuote_EmbeddedSingleQuote verifies that the canonical escape
// sequence is used for inner single quotes. The expected output for
// "it's" is 'it'"'"'s'.
func TestShellQuote_EmbeddedSingleQuote(t *testing.T) {
	got := ShellQuote("it's")
	want := `'it'"'"'s'`
	if got != want {
		t.Errorf("ShellQuote(\"it's\") = %q, want %q", got, want)
	}
}

// TestShellQuote_StripsControlChars documents the deliberate behavior
// of replacing control characters with spaces. This prevents newline
// injection attacks that would otherwise escape a single-quoted argument.
func TestShellQuote_StripsControlChars(t *testing.T) {
	in := "before\nafter\tstill\rmore"
	got := ShellQuote(in)
	if strings.ContainsRune(got, '\n') || strings.ContainsRune(got, '\r') || strings.ContainsRune(got, '\t') {
		t.Errorf("ShellQuote should strip control characters, got %q", got)
	}
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("ShellQuote should still wrap in single quotes, got %q", got)
	}
}

// TestShellQuote_EmptyString is a regression guard: an empty input must
// produce an empty single-quoted string, not an unquoted argument.
func TestShellQuote_EmptyString(t *testing.T) {
	got := ShellQuote("")
	if got != "''" {
		t.Errorf("ShellQuote(\"\") = %q, want %q", got, "''")
	}
}

// hasBareSingleQuoteInInner reports whether the substring between the
// outer single-quote delimiters contains a single quote that is not
// part of the canonical 5-char escape produced by ShellQuote. The
// escape occupies exactly 5 bytes: ' (close), " (open), ' (literal),
// " (close), ' (reopen).
func hasBareSingleQuoteInInner(inner string) bool {
	i := 0
	for i < len(inner) {
		if inner[i] != '\'' {
			i++
			continue
		}
		// The escape '\''"'"' occupies exactly 5 bytes.
		if i+5 <= len(inner) && inner[i+1] == '"' && inner[i+2] == '\'' && inner[i+3] == '"' && inner[i+4] == '\'' {
			i += 5
			continue
		}
		return true
	}
	return false
}

// hasBareSingleQuote inspects the full output of ShellQuote (which must
// begin and end with a single quote) and reports whether any inner
// single quote is not part of the canonical 4-char escape. The wrapping
// opening and closing quotes themselves are not flagged.
func hasBareSingleQuote(s string) bool {
	if len(s) < 2 || s[0] != '\'' || s[len(s)-1] != '\'' {
		return true // malformed output: not properly wrapped
	}
	return hasBareSingleQuoteInInner(s[1 : len(s)-1])
}
