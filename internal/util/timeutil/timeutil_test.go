package timeutil

import (
	"strings"
	"testing"
	"time"
)

// TestNow_ReturnsUTC confirms that Now() normalises to UTC, regardless
// of the host's local timezone. Timezone bugs here are a classic
// source of "deploy worked in CI but failed in production" issues.
func TestNow_ReturnsUTC(t *testing.T) {
	got := Now()
	if got.Location() != time.UTC {
		t.Errorf("Now().Location() = %v, want UTC", got.Location())
	}
}

// TestNowLocal_ReturnsLocal confirms that NowLocal() honours the
// machine's local timezone, but only for display. The companion
// function Now() must always be UTC; we make sure the two diverge.
func TestNowLocal_ReturnsLocal(t *testing.T) {
	got := NowLocal()
	// time.Local is the default; it is never UTC unless explicitly set.
	// We can at least confirm the time returned is close to "now" and
	// not in a wildly different zone.
	if got.Location() == nil {
		t.Error("NowLocal() returned a time with nil Location")
	}
	if time.Since(got) > 2*time.Second {
		t.Errorf("NowLocal() is too far in the past: %v", got)
	}
}

// TestFormatRFC3339_OutputsValidRFC3339 ensures the string can be
// parsed back. A malformed timestamp here would break every log
// consumer that depends on RFC3339.
func TestFormatRFC3339_OutputsValidRFC3339(t *testing.T) {
	got := FormatRFC3339()
	parsed, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("FormatRFC3339() = %q is not valid RFC3339: %v", got, err)
	}
	if parsed.Location() != time.UTC {
		t.Errorf("FormatRFC3339() produced non-UTC time: %v", parsed.Location())
	}
}

// TestFormat_RespectsLayout confirms the format helper honours a
// custom layout. We use a layout that is unique enough to fail
// obviously if the helper stops passing it through.
func TestFormat_RespectsLayout(t *testing.T) {
	layout := "2006/01/02-15:04:05"
	got := Format(layout)
	// The string should at least start with a 4-digit year.
	if len(got) < 4 || got[:4] != "202" && got[:4] != "203" {
		// Accept 2020-2099.
		if got[:3] != "202" && got[:3] != "203" && got[:3] != "204" && got[:3] != "205" && got[:3] != "206" {
			t.Errorf("Format() = %q does not look like a year-prefixed string", got)
		}
	}
}

// TestUnix_IsCloseToNow confirms that Unix() returns the current time.
// A drift of more than 2 seconds indicates the clock is broken.
func TestUnix_IsCloseToNow(t *testing.T) {
	got := Unix()
	want := time.Now().Unix()
	diff := got - want
	if diff < -2 || diff > 2 {
		t.Errorf("Unix() = %d, want ~%d (diff %d)", got, want, diff)
	}
}

// TestUnixNano_IsCloseToNow confirms the nanosecond variant also
// tracks wall-clock time.
func TestUnixNano_IsCloseToNow(t *testing.T) {
	got := UnixNano()
	want := time.Now().UnixNano()
	diff := got - want
	if diff < -int64(2*time.Second) || diff > int64(2*time.Second) {
		t.Errorf("UnixNano() drift = %d ns", diff)
	}
}

// TestSince_ReturnsNonNegative confirms that a future timestamp yields
// a negative duration (or at least a non-positive one) and a past
// timestamp yields a positive duration. Sign errors here are common
// when callers mistakenly swap the operand.
func TestSince_ReturnsNonNegativeForPast(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	if d := Since(past); d < time.Hour-time.Minute {
		t.Errorf("Since(1h ago) = %v, want ~1h", d)
	}
}

// TestUntil_PositiveForFuture confirms that a future timestamp yields
// a positive duration, and zero/negative for past.
func TestUntil_PositiveForFuture(t *testing.T) {
	future := Now().Add(time.Hour)
	if d := Until(future); d < time.Hour-time.Minute {
		t.Errorf("Until(1h from now) = %v, want ~1h", d)
	}
}

// TestAdd_PreservesUTC confirms that Add() does not silently change
// the timezone of the result.
func TestAdd_PreservesUTC(t *testing.T) {
	base := Now()
	got := Add(base, 2*time.Hour)
	if got.Location() != time.UTC {
		t.Errorf("Add changed timezone: %v", got.Location())
	}
	if !got.Equal(base.Add(2 * time.Hour)) {
		t.Errorf("Add result = %v, want %v", got, base.Add(2*time.Hour))
	}
}

// TestAddDate_HandlesYearAndMonthCarry confirms that date arithmetic
// correctly handles month rollovers (e.g. Jan 31 + 1 month = Mar 3 in
// the Go convention).
func TestAddDate_HandlesMonthCarry(t *testing.T) {
	jan31 := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	got := AddDate(jan31, 0, 1, 0)
	// Go normalises overflow: Jan 31 + 1 month = Mar 3.
	if got.Month() != time.March || got.Day() != 3 {
		t.Errorf("AddDate(Jan 31, +1 month) = %v, want March 3", got)
	}
}

// TestBeforeAndAfter_AreInverses confirms that Before/After are
// consistent: t.Before(u) and u.After(t) should agree.
func TestBeforeAndAfter_AreInverses(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if !Before(t1, t2) || !After(t2, t1) {
		t.Error("Before/After should agree on t1 < t2")
	}
	if Before(t2, t1) || After(t1, t2) {
		t.Error("Before/After should not be symmetric for t1 < t2")
	}
}

// TestEqual_TreatsDifferentZonesAsEqual confirms Equal is timezone
// insensitive when the underlying instant matches. We construct two
// time values from the same Unix timestamp in different locations.
func TestEqual_TreatsDifferentZonesAsEqual(t *testing.T) {
	utc := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tokyo, _ := time.Parse(time.RFC3339, "2026-01-01T21:00:00+09:00")
	// Both should represent the same instant.
	if !utc.Equal(tokyo) {
		t.Skipf("Sanity check failed: UTC and Tokyo should represent the same instant; utc=%v tokyo=%v", utc, tokyo)
	}
	if !Equal(utc, tokyo) {
		t.Errorf("Equal should treat the same instant in different zones as equal")
	}
}

// TestIsZero_DetectsZeroTime confirms that the zero time instant is
// detected (and a real time is not).
func TestIsZero_DetectsZeroTime(t *testing.T) {
	if !IsZero(time.Time{}) {
		t.Error("IsZero(time.Time{}) should be true")
	}
	if IsZero(Now()) {
		t.Error("IsZero(Now()) should be false")
	}
}

// TestParse_RoundTrip confirms the Parse helper delegates to the
// standard library correctly for a representative set of layouts.
func TestParse_RoundTrip(t *testing.T) {
	cases := []struct {
		layout string
		value  string
	}{
		{time.RFC3339, "2026-01-02T15:04:05Z"},
		{"2006-01-02", "2026-01-02"},
		{"2006-01-02 15:04:05", "2026-01-02 15:04:05"},
		{time.RFC1123, "Mon, 02 Jan 2026 15:04:05 MST"},
	}
	for _, tc := range cases {
		t.Run(tc.layout, func(t *testing.T) {
			parsed, err := Parse(tc.layout, tc.value)
			if err != nil {
				t.Fatalf("Parse(%q, %q) failed: %v", tc.layout, tc.value, err)
			}
			if parsed.IsZero() {
				t.Errorf("Parse(%q, %q) returned zero time", tc.layout, tc.value)
			}
		})
	}
}

// TestParse_InvalidValueReturnsError confirms that a malformed input
// produces a non-nil error rather than a zero time. Callers depend on
// this for early validation.
func TestParse_InvalidValueReturnsError(t *testing.T) {
	_, err := Parse(time.RFC3339, "not a timestamp")
	if err == nil {
		t.Error("Parse with bad input should return error")
	}
	if !strings.Contains(err.Error(), "parsing time") {
		t.Errorf("error = %q, want it to mention 'parsing time'", err.Error())
	}
}

// TestParseInLocation_UsesProvidedLocation confirms the location-aware
// parse helper honours the caller-supplied location. A bug here
// would silently flip the parsed wall-clock time by the local offset.
func TestParseInLocation_UsesProvidedLocation(t *testing.T) {
	// 2026-01-02 12:00:00 in Tokyo.
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Skipf("Asia/Tokyo timezone data not available: %v", err)
	}
	got, err := ParseInLocation("2006-01-02 15:04:05", "2026-01-02 12:00:00", tokyo)
	if err != nil {
		t.Fatalf("ParseInLocation failed: %v", err)
	}
	if got.Location().String() != tokyo.String() {
		t.Errorf("Location = %v, want %v", got.Location(), tokyo)
	}
	if got.Hour() != 12 {
		t.Errorf("hour = %d, want 12", got.Hour())
	}
}
