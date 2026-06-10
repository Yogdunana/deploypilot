package timeutil

import (
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	now := Now()
	expected := time.Now().UTC()

	if now.After(expected.Add(time.Second)) || now.Before(expected.Add(-time.Second)) {
		t.Errorf("Now() returned %v, expected time close to %v", now, expected)
	}

	if now.Location() != time.UTC {
		t.Errorf("Now() returned time in %v, expected UTC", now.Location())
	}
}

func TestNowLocal(t *testing.T) {
	nowLocal := NowLocal()
	expected := time.Now()

	if nowLocal.After(expected.Add(time.Second)) || nowLocal.Before(expected.Add(-time.Second)) {
		t.Errorf("NowLocal() returned %v, expected time close to %v", nowLocal, expected)
	}
}

func TestFormatRFC3339(t *testing.T) {
	formatted := FormatRFC3339()

	parsed, err := time.Parse(time.RFC3339, formatted)
	if err != nil {
		t.Errorf("FormatRFC3339() returned invalid RFC3339 string: %q, error: %v", formatted, err)
	}

	if parsed.Location() != time.UTC {
		t.Errorf("FormatRFC3339() returned time in %v, expected UTC", parsed.Location())
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name   string
		layout string
	}{
		{"RFC3339", time.RFC3339},
		{"RFC822", time.RFC822},
		{"custom", "2006-01-02"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := Format(tt.layout)

			_, err := time.Parse(tt.layout, formatted)
			if err != nil {
				t.Errorf("Format(%q) returned invalid string: %q, error: %v", tt.layout, formatted, err)
			}
		})
	}
}

func TestUnix(t *testing.T) {
	ts := Unix()
	expected := time.Now().UTC().Unix()

	if ts != expected && ts != expected-1 && ts != expected+1 {
		t.Errorf("Unix() returned %d, expected %d", ts, expected)
	}
}

func TestUnixNano(t *testing.T) {
	ts := UnixNano()
	expected := time.Now().UTC().UnixNano()

	diff := ts - expected
	if diff < -1e9 || diff > 1e9 {
		t.Errorf("UnixNano() returned %d, expected %d (diff: %d)", ts, expected, diff)
	}
}

func TestSince(t *testing.T) {
	past := time.Now().UTC().Add(-5 * time.Second)

	d := Since(past)

	if d < 4*time.Second || d > 6*time.Second {
		t.Errorf("Since() returned %v, expected ~5s", d)
	}
}

func TestUntil(t *testing.T) {
	future := time.Now().UTC().Add(5 * time.Second)

	d := Until(future)

	if d < 4*time.Second || d > 6*time.Second {
		t.Errorf("Until() returned %v, expected ~5s", d)
	}
}

func TestAdd(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	expected := time.Date(2024, 1, 1, 12, 0, 30, 0, time.UTC)

	result := Add(base, 30*time.Second)

	if !result.Equal(expected) {
		t.Errorf("Add() returned %v, expected %v", result, expected)
	}
}

func TestAddDate(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	expected := time.Date(2025, 3, 5, 12, 0, 0, 0, time.UTC)

	result := AddDate(base, 1, 2, 4)

	if !result.Equal(expected) {
		t.Errorf("AddDate() returned %v, expected %v", result, expected)
	}
}

func TestBefore(t *testing.T) {
	earlier := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	later := time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)

	if !Before(earlier, later) {
		t.Error("Before() should return true when first time is earlier")
	}
	if Before(later, earlier) {
		t.Error("Before() should return false when first time is later")
	}
}

func TestAfter(t *testing.T) {
	earlier := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	later := time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)

	if After(earlier, later) {
		t.Error("After() should return false when first time is earlier")
	}
	if !After(later, earlier) {
		t.Error("After() should return true when first time is later")
	}
}

func TestEqual(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)

	if !Equal(t1, t2) {
		t.Error("Equal() should return true for identical times")
	}
	if Equal(t1, t3) {
		t.Error("Equal() should return false for different times")
	}
}

func TestIsZero(t *testing.T) {
	zero := time.Time{}
	nonZero := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	if !IsZero(zero) {
		t.Error("IsZero() should return true for zero time")
	}
	if IsZero(nonZero) {
		t.Error("IsZero() should return false for non-zero time")
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		layout   string
		value    string
		expected time.Time
	}{
		{"RFC3339", time.RFC3339, "2024-01-01T12:00:00Z", time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
		{"custom", "2006-01-02", "2024-06-15", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.layout, tt.value)
			if err != nil {
				t.Errorf("Parse(%q, %q) returned error: %v", tt.layout, tt.value, err)
				return
			}
			if !result.Equal(tt.expected) {
				t.Errorf("Parse() returned %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestParseInLocation(t *testing.T) {
	layout := "2006-01-02 15:04:05"
	value := "2024-01-01 12:00:00"
	loc, _ := time.LoadLocation("America/New_York")

	result, err := ParseInLocation(layout, value, loc)
	if err != nil {
		t.Errorf("ParseInLocation() returned error: %v", err)
		return
	}

	if result.Location() != loc {
		t.Errorf("ParseInLocation() returned time in %v, expected %v", result.Location(), loc)
	}
}

func TestParse_Invalid(t *testing.T) {
	_, err := Parse(time.RFC3339, "invalid-time-string")
	if err == nil {
		t.Error("Parse() should return error for invalid input")
	}
}