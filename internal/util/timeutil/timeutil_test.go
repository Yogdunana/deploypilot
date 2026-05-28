package timeutil

import (
	"testing"
	"time"
)

func TestNow_ReturnsUTC(t *testing.T) {
	result := Now()
	if result.Location() != time.UTC {
		t.Errorf("Now() returned time in %v, expected UTC", result.Location())
	}
}

func TestNowLocal_ReturnsLocal(t *testing.T) {
	result := NowLocal()
	if result.Location() == time.UTC {
		t.Log("Warning: Local timezone appears to be UTC")
	}
}

func TestNow_NotZero(t *testing.T) {
	result := Now()
	if result.IsZero() {
		t.Error("Now() should not return zero time")
	}
}

func TestNow_RecentTime(t *testing.T) {
	result := Now()
	now := time.Now().UTC()
	diff := now.Sub(result)

	if diff < 0 {
		diff = -diff
	}

	if diff > 5*time.Second {
		t.Errorf("Now() returned time too far in the past or future: %v", diff)
	}
}

func TestFormatRFC3339(t *testing.T) {
	result := FormatRFC3339()

	if len(result) == 0 {
		t.Error("FormatRFC3339() returned empty string")
	}

	_, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("FormatRFC3339() returned invalid RFC3339 format: %q, error: %v", result, err)
	}
}

func TestFormat(t *testing.T) {
	result := Format(time.RFC1123)

	if len(result) == 0 {
		t.Error("Format() returned empty string")
	}

	_, err := time.Parse(time.RFC1123, result)
	if err != nil {
		t.Errorf("Format() returned invalid format for RFC1123: %q, error: %v", result, err)
	}
}

func TestUnix(t *testing.T) {
	result := Unix()

	if result <= 0 {
		t.Errorf("Unix() returned non-positive timestamp: %d", result)
	}

	now := time.Now().UTC().Unix()
	diff := now - result

	if diff < 0 {
		diff = -diff
	}

	if diff > 5 {
		t.Errorf("Unix() returned timestamp too far in the past or future: %d seconds difference", diff)
	}
}

func TestUnixNano(t *testing.T) {
	result := UnixNano()

	if result <= 0 {
		t.Errorf("UnixNano() returned non-positive timestamp: %d", result)
	}

	now := time.Now().UTC().UnixNano()
	diff := now - result

	if diff < 0 {
		diff = -diff
	}

	if diff > 5*1e9 {
		t.Errorf("UnixNano() returned timestamp too far in the past or future: %d ns difference", diff)
	}
}

func TestSince(t *testing.T) {
	past := time.Now().UTC().Add(-100 * time.Millisecond)
	result := Since(past)

	if result < 90*time.Millisecond || result > 200*time.Millisecond {
		t.Errorf("Since() = %v, expected around 100ms", result)
	}
}

func TestUntil(t *testing.T) {
	future := time.Now().UTC().Add(100 * time.Millisecond)
	result := Until(future)

	if result < 90*time.Millisecond || result > 200*time.Millisecond {
		t.Errorf("Until() = %v, expected around 100ms", result)
	}
}

func TestAdd(t *testing.T) {
	base := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	result := Add(base, 1*time.Hour)

	expected := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Add() = %v, want %v", result, expected)
	}
}

func TestAddDate(t *testing.T) {
	base := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	result := AddDate(base, 1, 2, 3)

	expected := time.Date(2024, 3, 4, 12, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("AddDate() = %v, want %v", result, expected)
	}
}

func TestBefore(t *testing.T) {
	earlier := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	later := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)

	if !Before(earlier, later) {
		t.Error("Before() should return true when earlier < later")
	}

	if Before(later, earlier) {
		t.Error("Before() should return false when later < earlier")
	}
}

func TestAfter(t *testing.T) {
	earlier := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	later := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)

	if !After(later, earlier) {
		t.Error("After() should return true when later > earlier")
	}

	if After(earlier, later) {
		t.Error("After() should return false when earlier > later")
	}
}

func TestEqual(t *testing.T) {
	time1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	time3 := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)

	if !Equal(time1, time2) {
		t.Error("Equal() should return true for same time")
	}

	if Equal(time1, time3) {
		t.Error("Equal() should return false for different times")
	}
}

func TestIsZero(t *testing.T) {
	zero := time.Time{}
	nonZero := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	if !IsZero(zero) {
		t.Error("IsZero() should return true for zero time")
	}

	if IsZero(nonZero) {
		t.Error("IsZero() should return false for non-zero time")
	}
}

func TestParse(t *testing.T) {
	result, err := Parse(time.RFC3339, "2023-01-01T12:00:00Z")
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	expected := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Parse() = %v, want %v", result, expected)
	}
}

func TestParse_Invalid(t *testing.T) {
	_, err := Parse(time.RFC3339, "invalid-time")
	if err == nil {
		t.Error("Parse() should return error for invalid input")
	}
}

func TestParseInLocation(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	result, err := ParseInLocation(time.RFC3339, "2023-01-01T12:00:00Z", loc)
	if err != nil {
		t.Fatalf("ParseInLocation() returned error: %v", err)
	}

	expected := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseInLocation() = %v, want %v", result, expected)
	}
}

func TestNow_Monotonic(t *testing.T) {
	t1 := Now()
	t2 := Now()

	if t2.Before(t1) {
		t.Error("Now() should return monotonically increasing time")
	}
}