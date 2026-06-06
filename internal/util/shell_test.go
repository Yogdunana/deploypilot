package util

import "testing"

// TestShellQuote_Plain confirms that a normal string is wrapped in single
// quotes and returned unchanged in content.
func TestShellQuote_Plain(t *testing.T) {
	got := ShellQuote("hello")
	want := "'hello'"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "hello", got, want)
	}
}

// TestShellQuote_Empty ensures empty input is safely quoted to an empty
// single-quoted string rather than producing invalid shell syntax.
func TestShellQuote_Empty(t *testing.T) {
	got := ShellQuote("")
	want := "''"
	if got != want {
		t.Errorf("ShellQuote(\"\") = %q, want %q", got, want)
	}
}

// TestShellQuote_SingleQuote covers the classic shell-injection vector: an
// embedded single quote must be escaped using the standard '"'"' trick so
// the value cannot terminate the surrounding quoted string.
func TestShellQuote_SingleQuote(t *testing.T) {
	got := ShellQuote("o'reilly")
	want := "'o'\"'\"'reilly'"
	if got != want {
		t.Errorf("ShellQuote(%q) = %q, want %q", "o'reilly", got, want)
	}
}

// TestShellQuote_ControlCharsReplaced exercises the defense against
// newline/control-character injection: a newline inside the input must NOT
// be allowed to break out of the quoted region, so it is replaced with a
// space.
func TestShellQuote_ControlCharsReplaced(t *testing.T) {
	got := ShellQuote("safe\nrm -rf /")
	// newline → space, so command becomes: 'safe rm -rf /'
	want := "'safe rm -rf /'"
	if got != want {
		t.Errorf("ShellQuote with newline = %q, want %q", got, want)
	}
	// Sanity: the dangerous command must not be reconstructable.
	if got == "'\nrm -rf /'" || got == "'safe\nrm -rf /'" {
		t.Errorf("ShellQuote did not sanitize newline, got %q", got)
	}
}
