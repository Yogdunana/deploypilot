package timeutil

import (
	"context"
	"testing"
	"time"
)

func TestNow_ReturnsUTC(t *testing.T) {
	now := Now()
	if now.Location() != time.UTC {
		t.Errorf("Now() = %v in location %v, want UTC", now, now.Location())
	}
	// Should be close to time.Now()
	delta := time.Since(now)
	if delta > 5*time.Second {
		t.Errorf("Now() = %v too far in the past (delta=%v)", now, delta)
	}
}

func TestNowLocal_LocalTimezone(t *testing.T) {
	now := NowLocal()
	if now.Location() == time.UTC {
		t.Logf("NowLocal() returned UTC location (machine may be in UTC); acceptable")
	}
	delta := time.Since(now)
	if delta > 5*time.Second {
		t.Errorf("NowLocal() = %v too far in the past (delta=%v)", now, delta)
	}
}

func TestFormatRFC3339(t *testing.T) {
	s := FormatRFC3339()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("FormatRFC3339() = %q failed to parse as RFC3339: %v", s, err)
	}
	if parsed.Location() != time.UTC {
		t.Errorf("FormatRFC3339() = %q parsed location = %v, want UTC", s, parsed.Location())
	}
}

func TestFormat_CustomLayout(t *testing.T) {
	s := Format("2006-01-02")
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("Format(\"2006-01-02\") = %q failed to parse: %v", s, err)
	}
	// Should produce today's date
	now := Now()
	if parsed.Year() != now.Year() || int(parsed.Month()) != int(now.Month()) || parsed.Day() != now.Day() {
		t.Errorf("Format(\"2006-01-02\") = %q, want today's date %v", s, now.Format("2006-01-02"))
	}
}

func TestUnix(t *testing.T) {
	got := Unix()
	now := time.Now().Unix()
	if got < now-5 || got > now+5 {
		t.Errorf("Unix() = %d, want close to %d", got, now)
	}
}

func TestUnixNano(t *testing.T) {
	got := UnixNano()
	now := time.Now().UnixNano()
	diff := now - got
	if diff < 0 {
		diff = -diff
	}
	if diff > 5*int64(time.Second) {
		t.Errorf("UnixNano() = %d, want close to %d (diff=%dns)", got, now, diff)
	}
}

func TestSince(t *testing.T) {
	past := Now().Add(-10 * time.Second)
	d := Since(past)
	if d < 8*time.Second || d > 15*time.Second {
		t.Errorf("Since(%v) = %v, want ~10s", past, d)
	}
}

func TestUntil(t *testing.T) {
	future := Now().Add(10 * time.Second)
	d := Until(future)
	if d < 5*time.Second || d > 15*time.Second {
		t.Errorf("Until(%v) = %v, want ~10s", future, d)
	}
}

func TestAdd(t *testing.T) {
	base := Now()
	got := Add(base, 5*time.Second)
	want := base.Add(5 * time.Second)
	if !got.Equal(want) {
		t.Errorf("Add(%v, 5s) = %v, want %v", base, got, want)
	}
}

func TestAddDate(t *testing.T) {
	base := time.Date(2025, time.January, 15, 10, 0, 0, 0, time.UTC)
	got := AddDate(base, 1, 2, 5)
	want := time.Date(2026, time.March, 20, 10, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("AddDate(%v, 1, 2, 5) = %v, want %v", base, got, want)
	}
}

func TestBeforeAfter(t *testing.T) {
	a := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	if !Before(a, b) {
		t.Errorf("Before(a, b) = false, want true")
	}
	if Before(b, a) {
		t.Errorf("Before(b, a) = true, want false")
	}
	if !After(b, a) {
		t.Errorf("After(b, a) = false, want true")
	}
	if After(a, b) {
		t.Errorf("After(a, b) = true, want false")
	}
}

func TestEqual(t *testing.T) {
	a := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC)

	if !Equal(a, b) {
		t.Errorf("Equal(a, b) = false, want true")
	}
	if Equal(a, c) {
		t.Errorf("Equal(a, c) = true, want false")
	}
}

func TestIsZero(t *testing.T) {
	if !IsZero(time.Time{}) {
		t.Errorf("IsZero(zero) = false, want true")
	}
	if IsZero(Now()) {
		t.Errorf("IsZero(now) = true, want false")
	}
}

func TestParse(t *testing.T) {
	got, err := Parse("2006-01-02", "2025-06-15")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if got.Year() != 2025 || got.Month() != time.June || got.Day() != 15 {
		t.Errorf("Parse() = %v, want 2025-06-15", got)
	}

	_, err = Parse("2006-01-02", "not-a-date")
	if err == nil {
		t.Errorf("Parse(invalid) succeeded, want error")
	}
}

func TestParseInLocation(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback: create fixed offset location
		loc = time.FixedZone("EST", -5*60*60)
	}
	got, err := ParseInLocation("2006-01-02 15:04", "2025-06-15 10:30", loc)
	if err != nil {
		t.Fatalf("ParseInLocation failed: %v", err)
	}
	if got.Location() != loc {
		t.Errorf("ParseInLocation location = %v, want %v", got.Location(), loc)
	}
	if got.Hour() != 10 || got.Minute() != 30 {
		t.Errorf("ParseInLocation = %v, want 10:30", got)
	}
}

func TestContext_DoesNotPanic(t *testing.T) {
	// Sanity check: Now works under context-like semantics (no ctx arg but used in request-scoped code)
	_ = context.Background()
	_ = Now()
}
