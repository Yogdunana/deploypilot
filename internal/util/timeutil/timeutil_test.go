package timeutil

import (
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	result := Now()
	if result.IsZero() {
		t.Error("expected non-zero time")
	}
	if result.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", result.Location())
	}
}

func TestNowLocal(t *testing.T) {
	result := NowLocal()
	if result.IsZero() {
		t.Error("expected non-zero time")
	}
}

func TestFormatRFC3339(t *testing.T) {
	result := FormatRFC3339()
	if result == "" {
		t.Error("expected non-empty string")
	}
	_, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("expected valid RFC3339 format, got %q: %v", result, err)
	}
}

func TestFormat(t *testing.T) {
	result := Format("2006-01-02")
	if result == "" {
		t.Error("expected non-empty string")
	}
	if len(result) != 10 {
		t.Errorf("expected length 10, got %d", len(result))
	}
}

func TestUnix(t *testing.T) {
	result := Unix()
	if result <= 0 {
		t.Errorf("expected positive Unix timestamp, got %d", result)
	}
}

func TestUnixNano(t *testing.T) {
	result := UnixNano()
	if result <= 0 {
		t.Errorf("expected positive UnixNano timestamp, got %d", result)
	}
}

func TestSince(t *testing.T) {
	past := time.Now().Add(-1 * time.Second)
	result := Since(past)
	if result <= 0 {
		t.Errorf("expected positive duration, got %v", result)
	}
}

func TestUntil(t *testing.T) {
	future := time.Now().Add(1 * time.Second)
	result := Until(future)
	if result <= 0 {
		t.Errorf("expected positive duration, got %v", result)
	}
}

func TestAdd(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	result := Add(base, 1*time.Hour)
	expected := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestAddDate(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	result := AddDate(base, 1, 2, 3)
	expected := time.Date(2027, 3, 4, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestBefore(t *testing.T) {
	earlier := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	if !Before(earlier, later) {
		t.Error("expected earlier to be before later")
	}
	if Before(later, earlier) {
		t.Error("expected later to not be before earlier")
	}
}

func TestAfter(t *testing.T) {
	earlier := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	if !After(later, earlier) {
		t.Error("expected later to be after earlier")
	}
	if After(earlier, later) {
		t.Error("expected earlier to not be after later")
	}
}

func TestEqual(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	if !Equal(t1, t2) {
		t.Error("expected equal times to be equal")
	}
	if Equal(t1, t3) {
		t.Error("expected different times to not be equal")
	}
}

func TestIsZero(t *testing.T) {
	zero := time.Time{}
	nonZero := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	if !IsZero(zero) {
		t.Error("expected zero time to be zero")
	}
	if IsZero(nonZero) {
		t.Error("expected non-zero time to not be zero")
	}
}

func TestParse(t *testing.T) {
	result, err := Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsZero() {
		t.Error("expected non-zero time")
	}
}

func TestParseInLocation(t *testing.T) {
	result, err := ParseInLocation(time.RFC3339, "2026-01-01T00:00:00Z", time.UTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", result.Location())
	}
}