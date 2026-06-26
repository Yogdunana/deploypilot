package timeutil

import (
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	result := Now()
	if result.Location() != time.UTC {
		t.Errorf("Now() should return UTC time, got %v", result.Location())
	}
	if result.IsZero() {
		t.Error("Now() should not return zero time")
	}
}

func TestNowLocal(t *testing.T) {
	result := NowLocal()
	if result.IsZero() {
		t.Error("NowLocal() should not return zero time")
	}
}

func TestFormatRFC3339(t *testing.T) {
	result := FormatRFC3339()
	if len(result) == 0 {
		t.Error("FormatRFC3339() should not return empty string")
	}
	_, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("FormatRFC3339() returned invalid RFC3339 string: %q", result)
	}
}

func TestFormat(t *testing.T) {
	result := Format("2006-01-02")
	if len(result) != 10 {
		t.Errorf("Format() returned unexpected length: %d", len(result))
	}
}

func TestUnix(t *testing.T) {
	result := Unix()
	if result <= 0 {
		t.Errorf("Unix() should return positive timestamp, got %d", result)
	}
}

func TestUnixNano(t *testing.T) {
	result := UnixNano()
	if result <= 0 {
		t.Errorf("UnixNano() should return positive timestamp, got %d", result)
	}
}

func TestSince(t *testing.T) {
	past := time.Now().UTC().Add(-1 * time.Second)
	result := Since(past)
	if result < 0 {
		t.Errorf("Since() should return positive duration, got %v", result)
	}
}

func TestUntil(t *testing.T) {
	future := time.Now().UTC().Add(1 * time.Second)
	result := Until(future)
	if result < 0 {
		t.Errorf("Until() should return positive duration, got %v", result)
	}
}

func TestAdd(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result := Add(base, 24*time.Hour)
	expected := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Add() = %v, want %v", result, expected)
	}
}

func TestAddDate(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	result := AddDate(base, 1, 2, 3)
	expected := time.Date(2025, 3, 4, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("AddDate() = %v, want %v", result, expected)
	}
}

func TestBefore(t *testing.T) {
	earlier := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	if !Before(earlier, later) {
		t.Error("Before() should return true when earlier < later")
	}
	if Before(later, earlier) {
		t.Error("Before() should return false when later > earlier")
	}
}

func TestAfter(t *testing.T) {
	earlier := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	if !After(later, earlier) {
		t.Error("After() should return true when later > earlier")
	}
	if After(earlier, later) {
		t.Error("After() should return false when earlier < later")
	}
}

func TestEqual(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	if !Equal(t1, t2) {
		t.Error("Equal() should return true for equal times")
	}
	if Equal(t1, t3) {
		t.Error("Equal() should return false for different times")
	}
}

func TestIsZero(t *testing.T) {
	zero := time.Time{}
	nonZero := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	if !IsZero(zero) {
		t.Error("IsZero() should return true for zero time")
	}
	if IsZero(nonZero) {
		t.Error("IsZero() should return false for non-zero time")
	}
}

func TestParse(t *testing.T) {
	result, err := Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}
	if result.IsZero() {
		t.Error("Parse() should return non-zero time")
	}
}

func TestParseInLocation(t *testing.T) {
	result, err := ParseInLocation("2006-01-02", "2024-01-01", time.UTC)
	if err != nil {
		t.Errorf("ParseInLocation() error = %v", err)
	}
	if result.IsZero() {
		t.Error("ParseInLocation() should return non-zero time")
	}
}