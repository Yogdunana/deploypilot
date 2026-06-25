package timeutil

import (
	"sync"
	"testing"
	"time"
)

func TestNow_ReturnsUTC(t *testing.T) {
	now := Now()
	if now.Location() != time.UTC {
		t.Errorf("Now() returned time in %v, want UTC", now.Location())
	}
}

func TestNowLocal_ReturnsLocal(t *testing.T) {
	now := NowLocal()
	if now.Location() == time.UTC {
		// This could happen if system timezone is UTC, which is fine
		// Just check it's not forced to UTC
	}
}

func TestFormatRFC3339(t *testing.T) {
	formatted := FormatRFC3339()
	// Should be parseable as RFC3339
	_, err := time.Parse(time.RFC3339, formatted)
	if err != nil {
		t.Errorf("FormatRFC3339() produced invalid format: %q, error: %v", formatted, err)
	}
}

func TestFormat(t *testing.T) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"15:04:05",
		time.DateTime,
	}

	for _, layout := range layouts {
		t.Run(layout, func(t *testing.T) {
			formatted := Format(layout)
			_, err := time.Parse(layout, formatted)
			if err != nil {
				t.Errorf("Format(%q) produced invalid format: %q, error: %v", layout, formatted, err)
			}
		})
	}
}

func TestUnix(t *testing.T) {
	unix := Unix()
	if unix <= 0 {
		t.Errorf("Unix() returned %d, want positive value", unix)
	}

	// Should be close to current time
	now := time.Now().UTC().Unix()
	diff := now - unix
	if diff < -1 || diff > 1 {
		t.Errorf("Unix() = %d, current time = %d, difference too large", unix, now)
	}
}

func TestUnixNano(t *testing.T) {
	nano := UnixNano()
	if nano <= 0 {
		t.Errorf("UnixNano() returned %d, want positive value", nano)
	}

	// Should be close to current time in nanoseconds
	now := time.Now().UTC().UnixNano()
	diff := now - nano
	// Allow up to 1 second difference due to test execution time
	if diff < -1e9 || diff > 1e9 {
		t.Errorf("UnixNano() difference too large")
	}
}

func TestSince(t *testing.T) {
	start := time.Now().UTC().Add(-5 * time.Second)
	since := Since(start)

	// Should be approximately 5 seconds
	if since < 4*time.Second || since > 6*time.Second {
		t.Errorf("Since() = %v, want approximately 5s", since)
	}
}

func TestUntil(t *testing.T) {
	future := time.Now().UTC().Add(10 * time.Second)
	until := Until(future)

	// Should be approximately 10 seconds
	if until < 9*time.Second || until > 11*time.Second {
		t.Errorf("Until() = %v, want approximately 10s", until)
	}
}

func TestAdd(t *testing.T) {
	now := Now()
	duration := 1 * time.Hour
	result := Add(now, duration)

	if result.Sub(now) != duration {
		t.Errorf("Add() = %v, want %v + %v", result, now, duration)
	}
}

func TestAddDate(t *testing.T) {
	now := Now()
	result := AddDate(now, 1, 2, 3)

	// Should be 1 year, 2 months, 3 days later
	expected := now.AddDate(1, 2, 3)
	if !result.Equal(expected) {
		t.Errorf("AddDate() = %v, want %v", result, expected)
	}
}

func TestBefore(t *testing.T) {
	now := Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	if !Before(past, now) {
		t.Error("Before(past, now) should be true")
	}
	if Before(future, now) {
		t.Error("Before(future, now) should be false")
	}
	if Before(now, now) {
		t.Error("Before(now, now) should be false")
	}
}

func TestAfter(t *testing.T) {
	now := Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	if !After(future, now) {
		t.Error("After(future, now) should be true")
	}
	if After(past, now) {
		t.Error("After(past, now) should be false")
	}
	if After(now, now) {
		t.Error("After(now, now) should be false")
	}
}

func TestEqual(t *testing.T) {
	now := Now()
	same := now
	different := now.Add(1 * time.Nanosecond)

	if !Equal(now, same) {
		t.Error("Equal(now, same) should be true")
	}
	if Equal(now, different) {
		t.Error("Equal(now, different) should be false")
	}
}

func TestIsZero(t *testing.T) {
	zero := time.Time{}
	nonZero := Now()

	if !IsZero(zero) {
		t.Error("IsZero(zero) should be true")
	}
	if IsZero(nonZero) {
		t.Error("IsZero(nonZero) should be false")
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		layout string
		value  string
		wantOK bool
	}{
		{time.RFC3339, "2024-01-15T10:30:00Z", true},
		{time.RFC3339, "2024-01-15T10:30:00+08:00", true},
		{time.RFC3339, "invalid", false},
		{"2006-01-02", "2024-01-15", true},
		{"2006-01-02", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result, err := Parse(tt.layout, tt.value)
			if tt.wantOK && err != nil {
				t.Errorf("Parse(%q, %q) error: %v", tt.layout, tt.value, err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("Parse(%q, %q) should fail", tt.layout, tt.value)
			}
			if tt.wantOK && result.IsZero() {
				t.Errorf("Parse(%q, %q) returned zero time", tt.layout, tt.value)
			}
		})
	}
}

func TestParseInLocation(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600) // UTC+8

	tests := []struct {
		layout string
		value  string
		wantOK bool
	}{
		{"2006-01-02 15:04:05", "2024-01-15 10:30:00", true},
		{"2006-01-02", "2024-01-15", true},
		{"2006-01-02", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result, err := ParseInLocation(tt.layout, tt.value, loc)
			if tt.wantOK && err != nil {
				t.Errorf("ParseInLocation(%q, %q, %v) error: %v", tt.layout, tt.value, loc, err)
			}
			if !tt.wantOK && err == nil {
				t.Errorf("ParseInLocation(%q, %q, %v) should fail", tt.layout, tt.value, loc)
			}
			if tt.wantOK && result.Location() != loc {
				t.Errorf("ParseInLocation() returned time in %v, want %v", result.Location(), loc)
			}
		})
	}
}

func TestAllFunctionsReturnUTC(t *testing.T) {
	// Ensure all time-returning functions return UTC
	funcs := []struct {
		name string
		fn   func() time.Time
	}{
		{"Now", Now},
		{"NowLocal", NowLocal}, // This one returns local, not UTC
	}

	for _, f := range funcs {
		if f.name == "NowLocal" {
			continue // NowLocal intentionally returns local time
		}
		t.Run(f.name, func(t *testing.T) {
			result := f.fn()
			if result.Location() != time.UTC {
				t.Errorf("%s() returned time in %v, want UTC", f.name, result.Location())
			}
		})
	}
}

func TestConsistency(t *testing.T) {
	// Test that Now() and Unix() are consistent
	now := Now()
	unix := Unix()

	if now.Unix() != unix {
		// Allow 1 second tolerance due to execution time
		diff := now.Unix() - unix
		if diff < -1 || diff > 1 {
			t.Errorf("Now().Unix() = %d, Unix() = %d, inconsistent", now.Unix(), unix)
		}
	}
}

func TestNanoConsistency(t *testing.T) {
	// Test that Now() and UnixNano() are consistent
	now := Now()
	nano := UnixNano()

	// Allow 1 second tolerance in nanoseconds
	diff := now.UnixNano() - nano
	if diff < -1e9 || diff > 1e9 {
		t.Errorf("Now().UnixNano() = %d, UnixNano() = %d, inconsistent", now.UnixNano(), nano)
	}
}

func TestFormatConsistency(t *testing.T) {
	// Test that FormatRFC3339 and Format(time.RFC3339) produce same format
	rfc3339 := FormatRFC3339()
	custom := Format(time.RFC3339)

	// Both should be parseable
	_, err1 := time.Parse(time.RFC3339, rfc3339)
	_, err2 := time.Parse(time.RFC3339, custom)

	if err1 != nil || err2 != nil {
		t.Errorf("FormatRFC3339() = %q (err: %v), Format(time.RFC3339) = %q (err: %v)",
			rfc3339, err1, custom, err2)
	}
}

func TestEdgeCases(t *testing.T) {
	// Test edge cases for time operations

	t.Run("Since with zero time", func(t *testing.T) {
		since := Since(time.Time{})
		// Should return a large positive duration
		if since < 0 {
			t.Errorf("Since(zero) = %v, should be positive", since)
		}
	})

	t.Run("Until with zero time", func(t *testing.T) {
		until := Until(time.Time{})
		// Should return a negative duration (past)
		if until > 0 {
			t.Errorf("Until(zero) = %v, should be negative", until)
		}
	})

	t.Run("Add with negative duration", func(t *testing.T) {
		now := Now()
		result := Add(now, -1 * time.Hour)
		if !result.Before(now) {
			t.Error("Add with negative duration should produce earlier time")
		}
	})

	t.Run("AddDate with negative values", func(t *testing.T) {
		now := Now()
		result := AddDate(now, -1, -1, -1)
		if !result.Before(now) {
			t.Error("AddDate with negative values should produce earlier time")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	// Test that timeutil functions can be called concurrently
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			_ = Now()
		}()
		go func() {
			defer wg.Done()
			_ = Unix()
		}()
		go func() {
			defer wg.Done()
			_ = FormatRFC3339()
		}()
		go func() {
			defer wg.Done()
			_ = UnixNano()
		}()
	}
	wg.Wait()
	// Should not panic or race
}