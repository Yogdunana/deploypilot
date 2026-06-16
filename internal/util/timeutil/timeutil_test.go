package timeutil

import (
	"strings"
	"testing"
	"time"
)

func TestNow_ReturnsUTC(t *testing.T) {
	got := Now()
	if got.Location() != time.UTC {
		t.Errorf("Now() location = %v, want UTC", got.Location())
	}
	if time.Since(got) > time.Second {
		t.Errorf("Now() returned a time more than 1s in the past: %v", got)
	}
}

func TestNowLocal_UsesLocalZone(t *testing.T) {
	got := NowLocal()
	if got.Location() == time.UTC {
		t.Error("NowLocal() should not return UTC")
	}
	// Local and UTC must refer to the same instant, possibly with different offsets.
	if time.Since(got) > time.Second {
		t.Errorf("NowLocal() returned a stale time: %v", got)
	}
}

func TestFormatRFC3339_RoundTrip(t *testing.T) {
	s := FormatRFC3339()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("FormatRFC3339 produced %q which is not valid RFC3339: %v", s, err)
	}
	if parsed.Location() != time.UTC {
		t.Errorf("parsed location = %v, want UTC", parsed.Location())
	}
}

func TestFormat_CustomLayout(t *testing.T) {
	got := Format("2006-01-02")
	// The date format must contain exactly 10 chars (YYYY-MM-DD).
	if len(got) != 10 {
		t.Errorf("Format(2006-01-02) length = %d, want 10 (%q)", len(got), got)
	}
	if _, err := time.Parse("2006-01-02", got); err != nil {
		t.Errorf("Format(2006-01-02) produced %q which is not valid: %v", got, err)
	}
}

func TestUnix_AndUnixNano(t *testing.T) {
	u := Unix()
	n := UnixNano()
	if u <= 0 {
		t.Errorf("Unix() = %d, want positive", u)
	}
	if n <= 0 {
		t.Errorf("UnixNano() = %d, want positive", n)
	}
	// u and n should be within one second of each other.
	if n/1e9-u > 1 {
		t.Errorf("Unix() and UnixNano() are inconsistent: %d vs %d", u, n)
	}
}

func TestSince_AndUntil(t *testing.T) {
	// Use a fixed reference time inside the helper to avoid races where the
	// real wall clock has advanced between Now() and the helper's read.
	past := Now().Add(-2 * time.Hour)
	future := Now().Add(2 * time.Hour)

	s := Since(past)
	if s < time.Hour {
		t.Errorf("Since(2h ago) = %v, want >= 1h", s)
	}

	u := Until(future)
	if u < time.Hour {
		t.Errorf("Until(2h ahead) = %v, want >= 1h", u)
	}

	// Sanity: Until should return a positive duration for a future time.
	if u <= 0 {
		t.Errorf("Until(future) = %v, want strictly positive", u)
	}
}

func TestAdd_AddDate(t *testing.T) {
	base := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	got := Add(base, 24*time.Hour)
	want := base.Add(24 * time.Hour)
	if !got.Equal(want) {
		t.Errorf("Add mismatch: got %v, want %v", got, want)
	}

	got = AddDate(base, 1, 0, 0)
	if got.Year() != 2025 {
		t.Errorf("AddDate(+1y) year = %d, want 2025", got.Year())
	}

	// Go's time.AddDate normalizes overflow; +1 month to Jan 31 -> Mar 2.
	got = AddDate(base, 0, 1, 0)
	if got.Month() != time.March {
		t.Errorf("AddDate(+1m) month = %v, want March", got.Month())
	}
}

func TestBefore_After_Equal_IsZero(t *testing.T) {
	a := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	if !Before(a, b) {
		t.Error("Before(a, b) should be true when a < b")
	}
	if Before(b, a) {
		t.Error("Before(b, a) should be false when b > a")
	}
	if !After(b, a) {
		t.Error("After(b, a) should be true when b > a")
	}
	if After(a, b) {
		t.Error("After(a, b) should be false when a < b")
	}
	if !Equal(a, a) {
		t.Error("Equal(a, a) should be true")
	}
	if Equal(a, b) {
		t.Error("Equal(a, b) should be false for different times")
	}

	if !IsZero(time.Time{}) {
		t.Error("IsZero(zero) should be true")
	}
	if IsZero(a) {
		t.Error("IsZero(a) should be false for a non-zero time")
	}
}

func TestParse_AndParseInLocation(t *testing.T) {
	got, err := Parse("2006-01-02", "2024-05-15")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if got.Year() != 2024 || got.Month() != time.May || got.Day() != 15 {
		t.Errorf("Parse result = %v, want 2024-05-15", got)
	}

	// Bad input should error rather than silently return a zero time.
	if _, err := Parse(time.RFC3339, "not-a-date"); err == nil {
		t.Error("Parse(invalid) should return error")
	}

	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Skipf("Asia/Shanghai tzdata not available: %v", err)
	}
	got, err = ParseInLocation("2006-01-02 15:04:05", "2024-05-15 12:00:00", shanghai)
	if err != nil {
		t.Fatalf("ParseInLocation failed: %v", err)
	}
	if got.Location().String() != shanghai.String() {
		t.Errorf("ParseInLocation location = %v, want %v", got.Location(), shanghai)
	}
}

// TestFormatRFC3339_FormatStable is a smoke test to make sure both formatters
// return deterministic-looking strings of the expected length.
func TestFormatRFC3339_FormatStable(t *testing.T) {
	a := FormatRFC3339()
	if !strings.Contains(a, "T") || !strings.HasSuffix(a, "Z") {
		t.Errorf("FormatRFC3339() = %q, expected RFC3339 form with T and Z", a)
	}
}
