package util

import (
	"strings"
	"testing"
)

// TestShellQuote_BasicString verifies that a plain string is wrapped in
// single quotes without modification.
func TestShellQuote_BasicString(t *testing.T) {
	got := ShellQuote("hello world")
	want := "'hello world'"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "hello world", got, want)
	}
}

// TestShellQuote_EmptyString verifies the edge case of an empty input.
func TestShellQuote_EmptyString(t *testing.T) {
	got := ShellQuote("")
	want := "''"
	if got != want {
		t.Errorf("ShellQuote(\"\") = %q, want %q", got, want)
	}
}

// TestShellQuote_StripsControlCharacters verifies that control chars such as
// newlines cannot break out of the single-quoted argument. This is the
// primary defence against shell injection via embedded newlines.
func TestShellQuote_StripsControlCharacters(t *testing.T) {
	// A newline embedded in a single-quoted shell argument would terminate
	// the argument in some shells; ShellQuote must neutralise it.
	inputs := []string{
		"line1\nline2",
		"with\rcarriage",
		"tab\there",
		"inject\n; rm -rf /",
		"a\x00b", // NUL byte
	}
	for _, in := range inputs {
		got := ShellQuote(in)
		if strings.ContainsAny(got, "\n\r\x00") {
			t.Errorf("ShellQuote(%q) = %q still contains control characters", in, got)
		}
		if strings.Contains(got, "\t") {
			t.Errorf("ShellQuote(%q) = %q still contains a tab", in, got)
		}
	}
}

// TestShellQuote_PreservesPrintableCharacters ensures that the only
// characters replaced are the control characters, not the printable ones.
func TestShellQuote_PreservesPrintableCharacters(t *testing.T) {
	in := "user@example.com:password!#$%^&*()_+-={}[]|:;<>?,./"
	got := ShellQuote(in)
	// The output should be exactly in wrapped in single quotes since
	// none of these characters are control characters.
	want := "'" + in + "'"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", in, got, want)
	}
}

// TestShellQuote_SingleQuoteEscaping verifies that an embedded single quote
// cannot terminate the quoted argument. The implementation uses the
// classic shell-safe pattern: ' -> '"'"'.
func TestShellQuote_SingleQuoteEscaping(t *testing.T) {
	got := ShellQuote("it's")
	want := `'it'"'"'s'`
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "it's", got, want)
	}
	// The classic shell escape pattern must be present: a single quote
	// is encoded as `'"'"'` (close, double-quoted single, reopen).
	if !strings.Contains(got, `'"'"'`) {
		t.Errorf("ShellQuote did not use the classic shell escape: %q", got)
	}
	// The output must still be wrapped in single quotes so that the
	// surrounding argument is preserved.
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("ShellQuote output not properly wrapped: %q", got)
	}
}

// TestShellQuote_MultipleSingleQuotes verifies escaping of multiple quotes
// in the same string.
func TestShellQuote_MultipleSingleQuotes(t *testing.T) {
	got := ShellQuote("a'b'c")
	want := `'a'"'"'b'"'"'c'`
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "a'b'c", got, want)
	}
}

// TestShellQuote_DollarSignIsSafeInsideSingleQuotes verifies that a $ in a
// single-quoted shell argument is treated as a literal by the shell, not
// expanded as a variable.
func TestShellQuote_DollarSignIsSafeInsideSingleQuotes(t *testing.T) {
	got := ShellQuote("$HOME")
	want := "'$HOME'"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "$HOME", got, want)
	}
}

// TestShellQuote_BacktickIsSafeInsideSingleQuotes verifies that backticks
// (command substitution) inside a single-quoted argument are not expanded.
func TestShellQuote_BacktickIsSafeInsideSingleQuotes(t *testing.T) {
	got := ShellQuote("`whoami`")
	want := "'`whoami`'"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "`whoami`", got, want)
	}
}

// TestShellQuote_InjectionAttempt confirms that a typical injection payload
// is fully neutralised by single-quote wrapping after control chars are
// stripped.
func TestShellQuote_InjectionAttempt(t *testing.T) {
	// Drop the newline first (this is exactly what the implementation
	// does) and verify the resulting string is still safely contained.
	in := "value\n; cat /etc/passwd; #"
	got := ShellQuote(in)
	if strings.Contains(got, "\n") {
		t.Errorf("ShellQuote left a newline in output: %q", got)
	}
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("ShellQuote output not properly wrapped: %q", got)
	}
}
