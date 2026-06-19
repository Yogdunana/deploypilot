package util

import (
	"strings"
	"testing"
)

// TestShellQuote_PlainString verifies that ordinary strings are wrapped in
// single quotes and pass through unchanged.
func TestShellQuote_PlainString(t *testing.T) {
	got := ShellQuote("hello")
	if got != "'hello'" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

// TestShellQuote_EmptyString verifies that an empty string still produces a
// valid (empty-content) single-quoted argument.
func TestShellQuote_EmptyString(t *testing.T) {
	got := ShellQuote("")
	if got != "''" {
		t.Errorf("expected empty single quotes, got %q", got)
	}
}

// TestShellQuote_SingleQuote is the canonical injection test: an attacker
// supplied value containing a single quote MUST NOT terminate the surrounding
// shell quoting. The output must remain a single shell token.
func TestShellQuote_SingleQuote(t *testing.T) {
	input := `'; rm -rf /; '`
	got := ShellQuote(input)
	// The output must remain inside the opening single quote.
	if !strings.HasPrefix(got, "'") {
		t.Fatalf("output should start with single quote, got %q", got)
	}
	// And close it.
	if !strings.HasSuffix(got, "'") {
		t.Fatalf("output should end with single quote, got %q", got)
	}
	// After un-escaping (the documented '"\''" trick), the value must be
	// the original input. We verify that the inner embedded pattern
	// contains the documented close-quote-escape.
	if !strings.Contains(got, `'"'"'`) {
		t.Errorf("expected the single quote to be escaped via %q, got %q", `'"'"'`, got)
	}
}

// TestShellQuote_CommandSubstitution guards against $() / backticks payload
// that could trigger command substitution when used inside double-quoted
// shell contexts. After ShellQuote, the output should still be safe to
// embed in a single-quoted string.
func TestShellQuote_CommandSubstitution(t *testing.T) {
	input := "$(whoami) `id`"
	got := ShellQuote(input)
	if got != "'$(whoami) `id`'" {
		t.Errorf("expected raw wrapping for single quotes, got %q", got)
	}
	// The wrapped output, when literally pasted into a shell single-quoted
	// argument, must NOT contain unescaped $( or backticks that would be
	// expanded. Single-quoted shell strings disable both, but we ensure the
	// wrapping is consistent.
	if strings.Count(got, "'") < 2 {
		t.Errorf("expected at least outer single quotes, got %q", got)
	}
}

// TestShellQuote_NewlineIsStripped ensures that a payload containing a
// newline (which can be used to break out of single-quoted shell arguments)
// has the newline replaced with a space. The original code explicitly maps
// control characters to ' ' for this reason.
func TestShellQuote_NewlineIsStripped(t *testing.T) {
	input := "abc\ndef"
	got := ShellQuote(input)
	if strings.Contains(got, "\n") {
		t.Errorf("output should not contain a newline, got %q", got)
	}
	if !strings.Contains(got, "abc def") {
		t.Errorf("newline should be replaced with space, got %q", got)
	}
}

// TestShellQuote_AllControlCharsAreSanitized ensures every control character
// is sanitized; this is the security-critical behavior.
func TestShellQuote_AllControlCharsAreSanitized(t *testing.T) {
	// Build a string with several control characters.
	input := "a\x00b\x07c\x1Bd"
	got := ShellQuote(input)
	for _, r := range []string{"\x00", "\x07", "\x1B"} {
		if strings.Contains(got, r) {
			t.Errorf("control char %q should be removed, got %q", r, got)
		}
	}
}

// TestShellQuote_PreservesSpaces verifies that spaces inside a value remain
// intact and are NOT word-split by the shell because they sit inside
// single quotes.
func TestShellQuote_PreservesSpaces(t *testing.T) {
	got := ShellQuote("hello world")
	if got != "'hello world'" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

// TestShellQuote_PreservesDoubleQuotes verifies that double quotes inside
// the value are kept literally (single-quoted shell strings do not
// interpret double quotes).
func TestShellQuote_PreservesDoubleQuotes(t *testing.T) {
	got := ShellQuote(`say "hi"`)
	if got != `'say "hi"'` {
		t.Errorf("expected literal double quotes to be preserved, got %q", got)
	}
}

// TestShellQuote_PreservesBackslash verifies that backslashes are kept
// literally inside single quotes (no escape interpretation).
func TestShellQuote_PreservesBackslash(t *testing.T) {
	got := ShellQuote(`a\b\c`)
	if got != `'a\b\c'` {
		t.Errorf("expected backslashes to be preserved, got %q", got)
	}
}

// TestShellQuote_DoesNotIntroduceNewlines is a defensive check: the
// output of ShellQuote must itself be free of newlines so it can be safely
// embedded in commands that span lines.
func TestShellQuote_DoesNotIntroduceNewlines(t *testing.T) {
	inputs := []string{
		"normal",
		"with\nnewline",
		"with\rcarriage",
		"with\ttab",
		"'",
		`complex ' " $ \` + "`" + ` $(rm -rf /)`,
	}
	for _, in := range inputs {
		got := ShellQuote(in)
		if strings.ContainsAny(got, "\n\r") {
			t.Errorf("ShellQuote(%q) introduced a newline/CR: %q", in, got)
		}
	}
}

// TestShellQuote_MultipleQuotesHandled verifies that consecutive single
// quotes (e.g. an intentionally empty argument or a value that is itself a
// single quote) are escaped in the canonical POSIX-safe way: a backslash
// escape around the inner quote, surrounded by single quotes.
func TestShellQuote_MultipleQuotesHandled(t *testing.T) {
	input := "''"
	got := ShellQuote(input)
	// '' becomes: '"'"''"'"' (close, escape, reopen, escape, close)
	if !strings.Contains(got, `'"'"'`) {
		t.Errorf("expected POSIX single-quote escaping, got %q", got)
	}
}
