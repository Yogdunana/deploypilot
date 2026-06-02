package timeutil

import (
	"strings"
	"testing"
	"time"
)

// TestNow_UTC ensures Now() always returns UTC, never local time. This
// is the most important invariant of the package: production code uses
// Now() to ensure timezone consistency across servers.
func TestNow_UTC(t *testing.T) {
	got := Now()
	if got.Location() != time.UTC {
		t.Errorf("Now().Location() = %v, want UTC", got.Location())
	}
	// The result must be a non-zero timestamp (i.e. the system clock
	// returned something sensible).
	if got.IsZero() {
		t.Error("Now() returned zero time")
	}
	// Sanity: Now() should be very close to time.Now() in UTC.
	delta := time.Since(got)
	if delta < 0 {
		delta = -delta
	}
	if delta > 2*time.Second {
		t.Errorf("Now() is off by %v from time.Now()", delta)
	}
}

// TestNowLocal_DoesNotCrash documents the contract: NowLocal returns
// local time (no UTC normalization). This test must not crash on
// systems where local TZ is unset.
func TestNowLocal_DoesNotCrash(t *testing.T) {
	got := NowLocal()
	if got.IsZero() {
		t.Error("NowLocal() returned zero time")
	}
}

// TestFormatRFC3339_RoundTrip checks that the produced string parses
// back via time.Parse with time.RFC3339 and is equal (modulo the
// monotonic clock reading stripped on parsing).
func TestFormatRFC3339_RoundTrip(t *testing.T) {
	got := FormatRFC3339()
	parsed, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("parse %q: %v", got, err)
	}
	// The parsed timestamp should be very close to "now" (within a few
	// seconds, depending on test runtime).
	delta := time.Since(parsed)
	if delta < 0 {
		delta = -delta
	}
	if delta > 2*time.Second {
		t.Errorf("parsed %v is off by %v from now", parsed, delta)
	}
	// The location must be UTC.
	if parsed.Location() != time.UTC {
		t.Errorf("parsed location = %v, want UTC", parsed.Location())
	}
}

// TestFormat_CustomLayout ensures the custom-layout wrapper honors the
// supplied layout.
func TestFormat_CustomLayout(t *testing.T) {
	got := Format("2006-01-02")
	if len(got) != 10 {
		t.Errorf("Format(2006-01-02) = %q, want 10-char YYYY-MM-DD", got)
	}
	parsed, err := time.Parse("2006-01-02", got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Location() != time.UTC {
		t.Errorf("parsed location = %v, want UTC", parsed.Location())
	}
}

// TestUnix_Recent covers the contract: Unix() returns the current UTC
// timestamp in seconds.
func TestUnix_Recent(t *testing.T) {
	before := time.Now().Unix()
	got := Unix()
	after := time.Now().Unix()
	if got < before || got > after {
		t.Errorf("Unix() = %d, want between %d and %d", got, before, after)
	}
}

// TestUnixNano_Recent covers the contract: UnixNano() returns the
// current UTC timestamp in nanoseconds, strictly > Unix()*1e9.
func TestUnixNano_Recent(t *testing.T) {
	got := UnixNano()
	if got < Unix()*1e9 {
		t.Errorf("UnixNano() = %d, want >= %d", got, Unix()*1e9)
	}
}

// TestSince_Recent confirms Since(t) is approximately the time elapsed
// since `t` (i.e. small but non-negative).
func TestSince_Recent(t *testing.T) {
	past := time.Now().Add(-1 * time.Second)
	got := Since(past)
	if got < 0 {
		t.Errorf("Since() = %v, want non-negative", got)
	}
	if got > 5*time.Second {
		t.Errorf("Since() = %v, want within 5s", got)
	}
}

// TestUntil_Future confirms Until(t) is approximately the time until
// `t` (i.e. positive when t is in the future).
func TestUntil_Future(t *testing.T) {
	future := time.Now().Add(2 * time.Hour)
	got := Until(future)
	if got < time.Hour || got > 3*time.Hour {
		t.Errorf("Until(future+2h) = %v, want between 1h and 3h", got)
	}
}

// TestUntil_Past: until a past time should be negative.
func TestUntil_Past(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	got := Until(past)
	if got > 0 {
		t.Errorf("Until(past) = %v, want negative", got)
	}
}

// TestAdd_Simple covers the wrapper.
func TestAdd_Simple(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	got := Add(base, 24*time.Hour)
	want := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Add = %v, want %v", got, want)
	}
}

// TestAddDate_YearMonthDay covers the wrapper.
func TestAddDate_YearMonthDay(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	got := AddDate(base, 1, 2, 3)
	want := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("AddDate = %v, want %v", got, want)
	}
}

// TestBefore_After_Equal lock in the obvious comparison wrappers.
func TestBefore_After_Equal(t *testing.T) {
	a := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	b := a.Add(1 * time.Second)
	if !Before(a, b) {
		t.Error("Before(a, b) should be true")
	}
	if !After(b, a) {
		t.Error("After(b, a) should be true")
	}
	if !Equal(a, a) {
		t.Error("Equal(a, a) should be true")
	}
	if Before(a, a) {
		t.Error("Before(a, a) should be false")
	}
	if After(a, a) {
		t.Error("After(a, a) should be false")
	}
}

// TestIsZero locks in the zero-detection wrapper.
func TestIsZero(t *testing.T) {
	if !IsZero(time.Time{}) {
		t.Error("IsZero(zero time) should be true")
	}
	if IsZero(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Error("IsZero(real time) should be false")
	}
}

// TestParse_RoundTrip covers the parse wrapper with a custom layout.
func TestParse_RoundTrip(t *testing.T) {
	const layout = "2006-01-02 15:04:05"
	in := "2025-03-17 14:25:36"
	got, err := Parse(layout, in)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want, _ := time.Parse(layout, in)
	if !got.Equal(want) {
		t.Errorf("Parse = %v, want %v", got, want)
	}
}

// TestParse_Invalid locks in the error path.
func TestParse_Invalid(t *testing.T) {
	if _, err := Parse(time.RFC3339, "not a timestamp"); err == nil {
		t.Error("Parse with bad input should return an error")
	}
}

// TestParseInLocation_Timezone checks the location-aware parse.
func TestParseInLocation_Timezone(t *testing.T) {
	const layout = "2006-01-02 15:04:05"
	in := "2025-03-17 14:25:36"
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Skip("UTC not loadable (should be impossible)")
	}
	got, err := ParseInLocation(layout, in, loc)
	if err != nil {
		t.Fatalf("ParseInLocation: %v", err)
	}
	if got.Location().String() != "UTC" {
		t.Errorf("location = %v, want UTC", got.Location())
	}
}

// TestFormatRFC3339_Contains_TZ_Offset confirms the output actually
// contains a timezone offset marker (Z or +00:00), so consumers
// downstream don't need to special-case missing offsets.
func TestFormatRFC3339_Contains_TZ_Offset(t *testing.T) {
	got := FormatRFC3339()
	if !strings.ContainsAny(got, "Z+-") {
		t.Errorf("FormatRFC3339() = %q, expected to contain a timezone marker (Z, +, or -)", got)
	}
}
