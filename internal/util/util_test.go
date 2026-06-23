package util

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestShellQuoteSimpleString(t *testing.T) {
	got := ShellQuote("hello")
	if got != "'hello'" {
		t.Errorf("ShellQuote(hello) = %q, want %q", got, "'hello'")
	}
}

func TestShellQuoteEmptyString(t *testing.T) {
	got := ShellQuote("")
	if got != "''" {
		t.Errorf("ShellQuote(empty) = %q, want %q", got, "''")
	}
}

func TestShellQuoteNeutralizesSingleQuote(t *testing.T) {
	// The classic shell escape attack: try to break out of single-quotes.
	// Input: '; rm -rf /; echo '
	// Quoted output should not contain a naked single-quote outside the
	// escape sequence that re-enters a single-quoted context.
	got := ShellQuote("'; rm -rf /; echo '")
	// The output should still be inside a single-quoted context from the
	// shell's perspective. We can verify that the result, when split on
	// spaces, contains no unquoted shell metacharacters that would be
	// interpreted. The function uses the standard trick: ' -> '"'"'
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("ShellQuote result must be wrapped in single quotes, got %q", got)
	}
	// The escape sequence must appear in the output.
	if !strings.Contains(got, `'"'"'`) {
		t.Errorf("ShellQuote should use '\"'\"' escape for embedded single quotes, got %q", got)
	}
}

func TestShellQuoteStripsControlChars(t *testing.T) {
	// A newline could break out of single-quoted context in some shells.
	// The function replaces control characters with spaces, which keeps the
	// argument safely contained within single quotes.
	got := ShellQuote("hello\nworld")
	if strings.Contains(got, "\n") {
		t.Errorf("ShellQuote should strip newlines, got %q", got)
	}
}

func TestShellQuoteStripsCarriageReturn(t *testing.T) {
	got := ShellQuote("hello\rworld")
	if strings.Contains(got, "\r") {
		t.Errorf("ShellQuote should strip carriage returns, got %q", got)
	}
}

func TestShellQuoteStripsNullByte(t *testing.T) {
	got := ShellQuote("hello\x00world")
	if strings.Contains(got, "\x00") {
		t.Errorf("ShellQuote should strip null bytes, got %q", got)
	}
}

func TestShellQuotePreservesSpecialChars(t *testing.T) {
	// Non-control special characters should NOT be removed; they are safe
	// inside single-quoted context. We just verify they survive.
	in := "a$b`c\"d\\e"
	got := ShellQuote(in)
	if !strings.Contains(got, "$") || !strings.Contains(got, "`") {
		t.Errorf("ShellQuote should preserve $ and backticks inside quotes, got %q", got)
	}
}

func TestShellQuotePreservesUnicode(t *testing.T) {
	// Unicode is not a control character and should not be replaced.
	in := "héllo-世界"
	got := ShellQuote(in)
	if !strings.Contains(got, "é") || !strings.Contains(got, "世") {
		t.Errorf("ShellQuote should preserve unicode characters, got %q", got)
	}
}

func TestShellQuoteMultipleSingleQuotes(t *testing.T) {
	got := ShellQuote("a'b'c")
	// Every ' should be replaced with the escape sequence.
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("output must be wrapped in single quotes, got %q", got)
	}
	// Count the escape sequences — should be 2.
	if got := strings.Count(got, `'"'"'`); got != 2 {
		t.Errorf("expected 2 escape sequences for 2 single quotes, got %d in %q", got, got)
	}
}

func TestDefaultClientHasTimeout(t *testing.T) {
	if DefaultClient == nil {
		t.Fatal("DefaultClient should not be nil")
	}
	if DefaultClient.Timeout == 0 {
		t.Error("DefaultClient should have a non-zero timeout")
	}
	if DefaultClient.Timeout < time.Second {
		t.Errorf("DefaultClient timeout too short: %v", DefaultClient.Timeout)
	}
}

func TestDefaultClientHasTransport(t *testing.T) {
	if DefaultClient.Transport == nil {
		t.Fatal("DefaultClient.Transport should not be nil")
	}
	tr, ok := DefaultClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("DefaultClient.Transport is not *http.Transport, got %T", DefaultClient.Transport)
	}
	if tr.MaxIdleConns <= 0 {
		t.Errorf("MaxIdleConns should be positive, got %d", tr.MaxIdleConns)
	}
	if tr.MaxIdleConnsPerHost <= 0 {
		t.Errorf("MaxIdleConnsPerHost should be positive, got %d", tr.MaxIdleConnsPerHost)
	}
	if tr.IdleConnTimeout <= 0 {
		t.Errorf("IdleConnTimeout should be positive, got %v", tr.IdleConnTimeout)
	}
}
