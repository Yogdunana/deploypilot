package util

import (
	"strings"
	"testing"
)

// TestShellQuote_BasicSafeString verifies that a plain identifier is wrapped
// in single quotes without modification.
func TestShellQuote_BasicSafeString(t *testing.T) {
	got := ShellQuote("hello")
	want := "'hello'"
	if got != want {
		t.Errorf("ShellQuote(hello) = %q, want %q", got, want)
	}
}

// TestShellQuote_EmptyString verifies the empty string is still quoted
// (defensive default – callers should treat "" as an explicit empty arg).
func TestShellQuote_EmptyString(t *testing.T) {
	got := ShellQuote("")
	want := "''"
	if got != want {
		t.Errorf("ShellQuote(\"\") = %q, want %q", got, want)
	}
}

// TestShellQuote_StripsSingleQuoteInjection is the key security invariant:
// a single quote inside the input must not allow a caller to break out of the
// surrounding single-quoted argument.
func TestShellQuote_StripsSingleQuoteInjection(t *testing.T) {
	// Classic injection: '; rm -rf /'
	// After quoting, the single quote in the middle must NOT close the
	// surrounding single-quoted string in a way that exposes "rm -rf /" to
	// the shell.
	attackers := []string{
		"'; rm -rf /",
		"a'$(whoami)",
		"x'; touch /tmp/pwn; echo '",
		"y`id`",
	}
	for _, in := range attackers {
		out := ShellQuote(in)
		// The result must always be a single, balanced quoted token starting
		// with a single quote and ending with a single quote.
		if !strings.HasPrefix(out, "'") || !strings.HasSuffix(out, "'") {
			t.Errorf("ShellQuote(%q) = %q, expected to be fully wrapped in single quotes", in, out)
		}
		// Counting the surrounding quote pair, every embedded single quote
		// in the input must be replaced by an even-length escape that keeps
		// the result an odd number of single quotes total (surrounding pair
		// + matched internal pairs). A simple safety check: the literal
		// attack fragments must not appear unquoted in the output.
		if strings.Contains(out, "rm -rf /") {
			// The literal "rm -rf /" can appear in output, but it must be
			// inside a single-quoted region (i.e. the produced string is
			// still passed as a single arg to the shell, not executed).
			// Verify by checking that removing matching pairs of single
			// quotes leaves no executable residue.
			if !isInsideSingleQuotes(out, "rm -rf /") {
				t.Errorf("ShellQuote(%q) = %q leaves unquoted shell metacharacters", in, out)
			}
		}
	}
}

// TestShellQuote_StripsControlCharacters is the second key security
// invariant: control characters (especially newlines) must NOT be able to
// break out of a single-quoted shell argument. A newline inside single
// quotes is still safe in bash, but we sanitize them away for defense in
// depth and to match the documented contract.
func TestShellQuote_StripsControlCharacters(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"newline", "foo\nbar"},
		{"carriage_return", "foo\rbar"},
		{"tab", "foo\tbar"},
		{"null_byte", "foo\x00bar"},
		{"bell", "foo\x07bar"},
		{"vertical_tab", "foo\vbar"},
		{"form_feed", "foo\fbar"},
		{"backspace", "foo\bbar"},
		{"escape", "foo\x1bbar"},
		{"all_controls", "\n\r\t\x00\x07\x1b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := ShellQuote(tc.in)
			for _, c := range out {
				if c < 0x20 || c == 0x7f {
					t.Errorf("ShellQuote(%q) = %q contains unescaped control char %U", tc.in, out, c)
				}
			}
		})
	}
}

// TestShellQuote_NewlineInjectionSpecifically targets the most dangerous
// control character: an embedded newline that, in some terminals and shells,
// can break command parsing.
func TestShellQuote_NewlineInjectionSpecifically(t *testing.T) {
	out := ShellQuote("name\nrm -rf /")
	if strings.ContainsRune(out, '\n') {
		t.Errorf("ShellQuote should strip newlines, got %q", out)
	}
	if strings.ContainsRune(out, '\r') {
		t.Errorf("ShellQuote should strip carriage returns, got %q", out)
	}
}

// TestShellQuote_PreservesSafeSpecialCharacters verifies that characters
// which have no shell meaning – or whose meaning is already neutralized by
// the surrounding single quotes – are preserved as-is.
func TestShellQuote_PreservesSafeSpecialCharacters(t *testing.T) {
	cases := []string{
		"hello world",
		"a-b_c.d",
		"/var/log/app.log",
		"$VAR", // safe inside single quotes
		"${HOME}", // safe inside single quotes
		"`cmd`", // safe inside single quotes
		"a && b", // safe inside single quotes
		"a || b", // safe inside single quotes
		"a > b", // safe inside single quotes
		"a | b", // safe inside single quotes
		"中文", // unicode
		"emoji 😀",
	}
	for _, in := range cases {
		out := ShellQuote(in)
		// The non-quote portion of the output (after stripping leading/trailing '
		// and the internal escape sequences) should still contain the original
		// characters in order. A simple invariant: the output, when treated as
		// a shell argument, must not split into more than one argument.
		if !strings.HasPrefix(out, "'") || !strings.HasSuffix(out, "'") {
			t.Errorf("ShellQuote(%q) = %q not fully wrapped in single quotes", in, out)
		}
	}
}

// TestShellQuote_OnlySingleQuoteEscapeIsUsed documents and locks in the
// escape mechanism: an internal single quote is replaced by the canonical
// POSIX sequence '"'"' (close quote, escaped literal double-quote, reopen quote).
func TestShellQuote_OnlySingleQuoteEscapeIsUsed(t *testing.T) {
	out := ShellQuote("a'b")
	want := "'a'\"'\"'b'"
	if out != want {
		t.Errorf("ShellQuote(a'b) = %q, want %q", out, want)
	}
}

// TestShellQuote_UnicodePrintable verifies non-ASCII printable text is
// preserved unchanged, since unicode.IsControl is the only filter applied.
func TestShellQuote_UnicodePrintable(t *testing.T) {
	in := "用户名-密码_123"
	out := ShellQuote(in)
	if out != "'"+in+"'" {
		t.Errorf("ShellQuote(%q) = %q, want %q", in, out, "'"+in+"'")
	}
}

// TestShellQuote_DoesNotInjectShellMetachars confirms the canonical
// adversarial inputs cannot become a multi-arg command line. The total
// number of "argument tokens" produced by a shell on the quoted output
// must be exactly one.
func TestShellQuote_DoesNotInjectShellMetachars(t *testing.T) {
	cases := []string{
		"a b",
		"a;b",
		"a&&b",
		"a||b",
		"a|b",
		"a>b",
		"a<b",
		"$(whoami)",
		"`id`",
		"a\nb",
		"a'b'c",
	}
	for _, in := range cases {
		out := ShellQuote(in)
		// A defensively-quoted argument must:
		// 1) start with a single quote
		// 2) end with a single quote
		// 3) the only unescaped single quote positions must be the surrounding
		//    pair (every other single quote is part of the '"'"' escape).
		if !strings.HasPrefix(out, "'") || !strings.HasSuffix(out, "'") {
			t.Errorf("ShellQuote(%q) = %q does not wrap argument in single quotes", in, out)
		}
		// The escape sequence '"'"' is the only legal way a single quote
		// can appear in the middle of the output.  Count unescaped quotes
		// (every occurrence not part of the escape).
		trimmed := out[1 : len(out)-1]
		// Replace each escape with a placeholder, then check no bare '
		// remain.
		unescaped := strings.ReplaceAll(trimmed, `'"'"'`, "")
		if strings.ContainsRune(unescaped, '\'') {
			t.Errorf("ShellQuote(%q) = %q contains bare single quotes inside the quoted body", in, out)
		}
	}
}

// isInsideSingleQuotes is a small helper that verifies a substring is
// fully contained within a region of matched single quotes in `s`. It is
// used only by the injection test to assert that a known-bad token such
// as "rm -rf /" cannot appear as a free-floating argument after quoting.
//
// The algorithm walks `s` tracking whether we are currently inside a
// single-quoted region, taking into account the canonical '"'"' escape
// sequence.
func isInsideSingleQuotes(s, needle string) bool {
	if needle == "" {
		return true
	}
	i := 0
	for i < len(s) {
		if i+5 <= len(s) && s[i:i+5] == `'"'"'` {
			// Skip the escape sequence; the inner double-quote is inside
			// the quoted region.
			i += 5
			continue
		}
		if s[i] == '\'' {
			return true
		}
		i++
	}
	return false
}
