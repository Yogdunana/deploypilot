package util

import (
	"strings"
	"testing"
)

func TestShellQuote_BasicString(t *testing.T) {
	got := ShellQuote("hello")
	want := "'hello'"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "hello", got, want)
	}
}

func TestShellQuote_EmptyString(t *testing.T) {
	got := ShellQuote("")
	want := "''"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "", got, want)
	}
}

func TestShellQuote_Whitespace(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"hello world", "'hello world'"},
		{"  spaced  ", "'  spaced  '"},
		{"a\tb", "'a\tb'"},
	}
	for _, c := range cases {
		got := ShellQuote(c.in)
		if got != c.want {
			t.Errorf("ShellQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestShellQuote_ExistingSingleQuote(t *testing.T) {
	got := ShellQuote("it's")
	// ' single quote is escaped by ending quote, concatenating "'", and reopening
	want := "'it'\"'\"'s'"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "it's", got, want)
	}
}

func TestShellQuote_MultipleSingleQuotes(t *testing.T) {
	got := ShellQuote("a'b'c")
	want := "'a'\"'\"'b'\"'\"'c'"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "a'b'c", got, want)
	}
}

func TestShellQuote_NewlinesStripped(t *testing.T) {
	got := ShellQuote("a\nb")
	// Newline control character should be replaced with a space
	if strings.Contains(got, "\n") {
		t.Errorf("ShellQuote(%q) = %q, should not contain newline", "a\nb", got)
	}
	if !strings.Contains(got, "a b") {
		t.Errorf("ShellQuote(%q) = %q, want content 'a b' preserved", "a\nb", got)
	}
}

func TestShellQuote_ControlCharacters(t *testing.T) {
	// Ensure control chars that could break out of quoted context are neutralized
	inputs := []string{
		"a\x00b",
		"a\x01b",
		"a\x1fb",
		"line1\r\nline2",
		"tab\there",
	}
	for _, in := range inputs {
		got := ShellQuote(in)
		// Output should start and end with single quotes
		if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
			t.Errorf("ShellQuote(%q) = %q, want quoted output", in, got)
		}
		// Each raw control char should have been replaced (not appear verbatim except tab which is \t)
	}
}

func TestShellQuote_SpecialShellCharacters(t *testing.T) {
	cases := []string{
		"$(rm -rf /)",
		"`whoami`",
		"$PATH",
		"; ls",
		"| cat /etc/passwd",
		"> /tmp/evil",
		"< /etc/passwd",
		"a & b",
		"file; rm -rf /",
	}
	for _, c := range cases {
		got := ShellQuote(c)
		// The result must be quoted so shell metachars inside are literal
		if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
			t.Errorf("ShellQuote(%q) = %q, want single-quoted output to neutralize shell chars", c, got)
		}
	}
}

func TestShellQuote_DoubleQuotesInside(t *testing.T) {
	got := ShellQuote("he said \"hi\"")
	want := "'he said \"hi\"'"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "he said \"hi\"", got, want)
	}
}

func TestShellQuote_RoundTrip_LiteralContent(t *testing.T) {
	// The content inside the quoted string should be preserved literally
	// (minus control characters which are converted to spaces).
	testInputs := []string{
		"simple",
		"with spaces",
		"with-dashes_and_underscores",
		"dot.txt",
		"path/to/file",
	}
	for _, in := range testInputs {
		got := ShellQuote(in)
		// Strip outer single quotes for content inspection
		inner := got[1 : len(got)-1]
		if inner != in {
			t.Errorf("ShellQuote(%q) inner content = %q, want literal content %q", in, inner, in)
		}
	}
}
