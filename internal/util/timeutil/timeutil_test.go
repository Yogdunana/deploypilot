package timeutil

import (
	"testing"
	"time"
)

func TestNow_ReturnsUTC(t *testing.T) {
	result := Now()
	if result.Location() != time.UTC {
		t.Errorf("Now() should return UTC time, got %v", result.Location())
	}
}

func TestNowLocal_ReturnsLocal(t *testing.T) {
	result := NowLocal()
	expected := time.Now().Location()
	if result.Location() != expected {
		t.Errorf("NowLocal() should return local time, got %v", result.Location())
	}
}

func TestFormatRFC3339_Format(t *testing.T) {
	result := FormatRFC3339()
	if len(result) != 20 {
		t.Errorf("FormatRFC3339() should return 20-character string, got %d: %q", len(result), result)
	}
	_, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("FormatRFC3339() should return valid RFC3339 format: %v", err)
	}
}

func TestFormat_CustomLayout(t *testing.T) {
	layout := "2006-01-02"
	result := Format(layout)
	_, err := time.Parse(layout, result)
	if err != nil {
		t.Errorf("Format(%q) should return valid formatted time: %v", layout, err)
	}
}

func TestUnix_ReturnsTimestamp(t *testing.T) {
	result := Unix()
	now := time.Now().UTC().Unix()
	if result < now-1 || result > now+1 {
		t.Errorf("Unix() should return current timestamp, got %d, expected around %d", result, now)
	}
}

func TestUnixNano_ReturnsTimestamp(t *testing.T) {
	result := UnixNano()
	now := time.Now().UTC().UnixNano()
	if result < now-1e9 || result > now+1e9 {
		t.Errorf("UnixNano() should return current nanosecond timestamp")
	}
}

func TestSince_CalculatesDuration(t *testing.T) {
	past := time.Now().UTC().Add(-5 * time.Second)
	result := Since(past)
	if result < 4*time.Second || result > 6*time.Second {
		t.Errorf("Since() should return approximately 5 seconds, got %v", result)
	}
}

func TestUntil_CalculatesDuration(t *testing.T) {
	future := time.Now().UTC().Add(5 * time.Second)
	result := Until(future)
	if result < 4*time.Second || result > 6*time.Second {
		t.Errorf("Until() should return approximately 5 seconds, got %v", result)
	}
}

func TestAdd(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	result := Add(base, 1*time.Hour)
	expected := time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Add() = %v, want %v", result, expected)
	}
}

func TestAddDate(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	result := AddDate(base, 1, 2, 3)
	expected := time.Date(2025, 3, 4, 12, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("AddDate() = %v, want %v", result, expected)
	}
}

func TestBefore(t *testing.T) {
	earlier := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	later := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	if !Before(earlier, later) {
		t.Error("Before() should return true when t is before u")
	}
	if Before(later, earlier) {
		t.Error("Before() should return false when t is after u")
	}
}

func TestAfter(t *testing.T) {
	earlier := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	later := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	if !After(later, earlier) {
		t.Error("After() should return true when t is after u")
	}
	if After(earlier, later) {
		t.Error("After() should return false when t is before u")
	}
}

func TestEqual(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	if !Equal(t1, t2) {
		t.Error("Equal() should return true for same time")
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
	layout := time.RFC3339
	value := "2024-01-01T12:00:00Z"
	result, err := Parse(layout, value)
	if err != nil {
		t.Errorf("Parse() returned error: %v", err)
	}
	if result.Year() != 2024 || result.Month() != 1 || result.Day() != 1 {
		t.Errorf("Parse() = %v, want 2024-01-01", result)
	}
}

func TestParseInLocation(t *testing.T) {
	layout := "2006-01-02 15:04:05"
	value := "2024-01-01 12:00:00"
	loc := time.FixedZone("UTC+8", 8*60*60)
	result, err := ParseInLocation(layout, value, loc)
	if err != nil {
		t.Errorf("ParseInLocation() returned error: %v", err)
	}
	if result.Location() != loc {
		t.Errorf("ParseInLocation() location = %v, want %v", result.Location(), loc)
	}
}