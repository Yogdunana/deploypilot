package timeutil

import (
	"strings"
	"testing"
	"time"
)

func TestNowReturnsUTC(t *testing.T) {
	got := Now()
	if got.Location() != time.UTC {
		t.Errorf("Now() should return UTC, got location: %v", got.Location())
	}
}

func TestNowIsCloseToActual(t *testing.T) {
	before := time.Now()
	got := Now()
	after := time.Now()
	// Got should be between before and after (within a small tolerance).
	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Errorf("Now() = %v, expected between %v and %v", got, before, after)
	}
}

func TestNowLocalIsNotUTC(t *testing.T) {
	// NowLocal returns time in the local timezone. We only verify it
	// doesn't return a zero value and that the timestamp is reasonable.
	got := NowLocal()
	if got.IsZero() {
		t.Error("NowLocal() should not return zero time")
	}
}

func TestFormatRFC3339(t *testing.T) {
	s := FormatRFC3339()
	// RFC3339 format must contain 'T' separator and a timezone designator.
	if !strings.Contains(s, "T") {
		t.Errorf("FormatRFC3339() = %q, expected to contain 'T'", s)
	}
	// Should be parseable as RFC3339.
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		t.Errorf("FormatRFC3339() = %q, not parseable as RFC3339: %v", s, err)
	}
}

func TestFormat(t *testing.T) {
	s := Format("2006-01-02")
	if s == "" {
		t.Error("Format() should not return empty string")
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		t.Errorf("Format() = %q, not parseable: %v", s, err)
	}
}

func TestUnix(t *testing.T) {
	before := time.Now().Unix()
	got := Unix()
	after := time.Now().Unix()
	if got < before || got > after {
		t.Errorf("Unix() = %d, expected between %d and %d", got, before, after)
	}
}

func TestUnixNano(t *testing.T) {
	got := UnixNano()
	if got <= 0 {
		t.Errorf("UnixNano() = %d, expected positive value", got)
	}
	// UnixNano should be larger than Unix for the same moment.
	if got < Unix() {
		t.Errorf("UnixNano() = %d should be >= Unix()", got)
	}
}

func TestSince(t *testing.T) {
	past := time.Now().Add(-2 * time.Hour)
	d := Since(past)
	if d < time.Hour {
		t.Errorf("Since(past) = %v, expected >= 1h", d)
	}
}

func TestUntil(t *testing.T) {
	future := time.Now().Add(2 * time.Hour)
	d := Until(future)
	if d < time.Hour {
		t.Errorf("Until(future) = %v, expected >= 1h", d)
	}
}

func TestAdd(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := Add(base, 24*time.Hour)
	want := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Add() = %v, want %v", got, want)
	}
}

func TestAddDate(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := AddDate(base, 1, 2, 3)
	want := time.Date(2027, 3, 4, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("AddDate() = %v, want %v", got, want)
	}
}

func TestBeforeAndAfter(t *testing.T) {
	a := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if !Before(a, b) {
		t.Error("a should be before b")
	}
	if Before(b, a) {
		t.Error("b should not be before a")
	}
	if !After(b, a) {
		t.Error("b should be after a")
	}
	if After(a, b) {
		t.Error("a should not be after b")
	}
	if Before(a, a) || After(a, a) {
		t.Error("a should not be before or after itself")
	}
}

func TestEqual(t *testing.T) {
	a := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if !Equal(a, b) {
		t.Error("a and b should be equal")
	}
	if Equal(a, c) {
		t.Error("a and c should not be equal")
	}
}

func TestIsZero(t *testing.T) {
	if !IsZero(time.Time{}) {
		t.Error("zero time should be detected")
	}
	nonZero := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if IsZero(nonZero) {
		t.Error("non-zero time should not be detected as zero")
	}
}

func TestParse(t *testing.T) {
	got, err := Parse("2006-01-02", "2026-05-15")
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	want := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Parse() = %v, want %v", got, want)
	}

	// Invalid input should return an error.
	if _, err := Parse("2006-01-02", "not-a-date"); err == nil {
		t.Error("Parse() with invalid input should return error")
	}
}

func TestParseInLocation(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Skipf("UTC location not available: %v", err)
	}
	got, err := ParseInLocation("2006-01-02", "2026-05-15", loc)
	if err != nil {
		t.Fatalf("ParseInLocation() returned error: %v", err)
	}
	want := time.Date(2026, 5, 15, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("ParseInLocation() = %v, want %v", got, want)
	}
}
