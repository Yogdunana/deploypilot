package timeutil

import (
	"testing"
	"time"
)

func TestNow_ReturnsUTC(t *testing.T) {
	result := Now()
	if result.Location() != time.UTC {
		t.Errorf("Now() returned time in %v, want UTC", result.Location())
	}
}

func TestNowLocal_ReturnsLocal(t *testing.T) {
	result := NowLocal()
	if result.Location() != time.Local {
		t.Errorf("NowLocal() returned time in %v, want Local", result.Location())
	}
}

func TestFormatRFC3339_ReturnsValidFormat(t *testing.T) {
	result := FormatRFC3339()
	_, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("FormatRFC3339() returned invalid RFC3339 string: %q, error: %v", result, err)
	}
}

func TestFormat_AppliesLayout(t *testing.T) {
	layout := "2006-01-02"
	result := Format(layout)
	_, err := time.Parse(layout, result)
	if err != nil {
		t.Errorf("Format(%q) returned invalid string: %q, error: %v", layout, result, err)
	}
}

func TestUnix_ReturnsPositive(t *testing.T) {
	result := Unix()
	if result <= 0 {
		t.Errorf("Unix() returned %d, want positive value", result)
	}
}

func TestUnixNano_ReturnsPositive(t *testing.T) {
	result := UnixNano()
	if result <= 0 {
		t.Errorf("UnixNano() returned %d, want positive value", result)
	}
}

func TestSince_ReturnsPositive(t *testing.T) {
	past := Now().Add(-1 * time.Second)
	result := Since(past)
	if result <= 0 {
		t.Errorf("Since() returned %v, want positive duration", result)
	}
}

func TestUntil_ReturnsPositive(t *testing.T) {
	future := Now().Add(1 * time.Second)
	result := Until(future)
	if result <= 0 {
		t.Errorf("Until() returned %v, want positive duration", result)
	}
}

func TestAdd_ReturnsExpectedTime(t *testing.T) {
	base := Now()
	duration := 10 * time.Minute
	result := Add(base, duration)
	expected := base.Add(duration)
	if !result.Equal(expected) {
		t.Errorf("Add() returned %v, want %v", result, expected)
	}
}

func TestAddDate_ReturnsExpectedTime(t *testing.T) {
	base := Now()
	result := AddDate(base, 1, 2, 3)
	expected := base.AddDate(1, 2, 3)
	if !result.Equal(expected) {
		t.Errorf("AddDate() returned %v, want %v", result, expected)
	}
}

func TestBefore_ReturnsTrue(t *testing.T) {
	earlier := Now().Add(-1 * time.Hour)
	later := Now()
	if !Before(earlier, later) {
		t.Error("Before() should return true when earlier time is before later time")
	}
	if Before(later, earlier) {
		t.Error("Before() should return false when later time is not before earlier time")
	}
}

func TestAfter_ReturnsTrue(t *testing.T) {
	earlier := Now().Add(-1 * time.Hour)
	later := Now()
	if !After(later, earlier) {
		t.Error("After() should return true when later time is after earlier time")
	}
	if After(earlier, later) {
		t.Error("After() should return false when earlier time is not after later time")
	}
}

func TestEqual_ReturnsTrue(t *testing.T) {
	now := Now()
	same := now
	if !Equal(now, same) {
		t.Error("Equal() should return true for same time")
	}
	different := now.Add(1 * time.Second)
	if Equal(now, different) {
		t.Error("Equal() should return false for different times")
	}
}

func TestIsZero_ReturnsTrue(t *testing.T) {
	zero := time.Time{}
	if !IsZero(zero) {
		t.Error("IsZero() should return true for zero time")
	}
	nonZero := Now()
	if IsZero(nonZero) {
		t.Error("IsZero() should return false for non-zero time")
	}
}

func TestParse_ReturnsExpected(t *testing.T) {
	layout := "2006-01-02"
	value := "2024-01-15"
	result, err := Parse(layout, value)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if result.Year() != 2024 || result.Month() != time.January || result.Day() != 15 {
		t.Errorf("Parse() returned %v, want 2024-01-15", result)
	}
}