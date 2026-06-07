package util

import "testing"

func TestShellQuote_BasicString(t *testing.T) {
	got := ShellQuote("hello")
	want := "'hello'"
	if got != want {
		t.Errorf("ShellQuote(hello) = %q, want %q", got, want)
	}
}

func TestShellQuote_EmptyString(t *testing.T) {
	got := ShellQuote("")
	want := "''"
	if got != want {
		t.Errorf("ShellQuote(\"\") = %q, want %q", got, want)
	}
}

func TestShellQuote_StringWithSingleQuote(t *testing.T) {
	// Single quotes inside single-quoted args must be escaped with '\'' pattern.
	got := ShellQuote("can't")
	want := `'can'"'"'t'`
	if got != want {
		t.Errorf("ShellQuote(cant) = %q, want %q", got, want)
	}
}

func TestShellQuote_StringWithMultipleSingleQuotes(t *testing.T) {
	got := ShellQuote("a'b'c")
	want := `'a'"'"'b'"'"'c'`
	if got != want {
		t.Errorf("ShellQuote(a'b'c) = %q, want %q", got, want)
	}
}

func TestShellQuote_StringWithDoubleQuotes(t *testing.T) {
	// Double quotes are safe inside single-quoted args.
	got := ShellQuote(`say "hi"`)
	want := `'say "hi"'`
	if got != want {
		t.Errorf("ShellQuote(say \"hi\") = %q, want %q", got, want)
	}
}

func TestShellQuote_StringWithSpaces(t *testing.T) {
	got := ShellQuote("hello world")
	want := "'hello world'"
	if got != want {
		t.Errorf("ShellQuote(hello world) = %q, want %q", got, want)
	}
}

func TestShellQuote_StringWithSemicolon(t *testing.T) {
	// Shell metacharacters should be inert when quoted.
	got := ShellQuote("a;b")
	want := "'a;b'"
	if got != want {
		t.Errorf("ShellQuote(a;b) = %q, want %q", got, want)
	}
}

func TestShellQuote_StringWithDollarSign(t *testing.T) {
	// Variable expansion must be prevented.
	got := ShellQuote("$HOME")
	want := "'$HOME'"
	if got != want {
		t.Errorf("ShellQuote($HOME) = %q, want %q", got, want)
	}
}

func TestShellQuote_StringWithBacktick(t *testing.T) {
	// Command substitution via backticks must be prevented.
	got := ShellQuote("`whoami`")
	want := "'`whoami`'"
	if got != want {
		t.Errorf("ShellQuote(backticks) = %q, want %q", got, want)
	}
}

func TestShellQuote_StripsNewline(t *testing.T) {
	// Newlines can break out of single-quoted arguments and must be neutralized.
	got := ShellQuote("a\nb")
	want := "'a b'" // newline (a control char) is replaced with a space
	if got != want {
		t.Errorf("ShellQuote(a\\nb) = %q, want %q", got, want)
	}

	// Newline injection attempts must be neutralized — semicolon and the
	// follow-up command are kept verbatim, but the newline that could
	// close the surrounding context becomes a space.
	got2 := ShellQuote("safe\n; rm -rf /")
	want2 := "'safe ; rm -rf /'"
	if got2 != want2 {
		t.Errorf("ShellQuote(newline injection) = %q, want %q (newline should be replaced with space)", got2, want2)
	}
}

func TestShellQuote_StripsAllControlCharacters(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"tab", "a\tb"},
		{"carriage return", "a\rb"},
		{"null byte", "a\x00b"},
		{"escape", "a\x1bb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ShellQuote(tc.in)
			// Each control char must become a space.
			for _, r := range got {
				if r == '\t' || r == '\n' || r == '\r' || r == 0 || r == 0x1b {
					t.Errorf("ShellQuote(%q) leaked control char %q in output %q", tc.in, r, got)
				}
			}
		})
	}
}

func TestShellQuote_PreservesUnicode(t *testing.T) {
	// Non-control unicode should be preserved verbatim.
	got := ShellQuote("héllo 世界")
	want := "'héllo 世界'"
	if got != want {
		t.Errorf("ShellQuote(unicode) = %q, want %q", got, want)
	}
}

func TestShellQuote_InjectionAttemptIsSafe(t *testing.T) {
	// A realistic injection attempt should not survive ShellQuote intact.
	malicious := "; rm -rf / #"
	got := ShellQuote(malicious)
	want := "'; rm -rf / #'"
	if got != want {
		t.Errorf("ShellQuote(malicious) = %q, want %q", got, want)
	}
}
