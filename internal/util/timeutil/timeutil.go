// Package timeutil provides timezone-aware time utilities.
// All functions return UTC time by default to ensure consistency across servers.
package timeutil

import (
	"time"
)

// Now returns the current time in UTC.
// Use this instead of time.Now() to ensure timezone consistency.
func Now() time.Time {
	return time.Now().UTC()
}

// NowLocal returns the current time in the local timezone.
// Only use this for display purposes, never for storage or comparison.
func NowLocal() time.Time {
	return time.Now()
}

// FormatRFC3339 returns the current UTC time formatted as RFC3339.
func FormatRFC3339() string {
	return Now().Format(time.RFC3339)
}

// Format returns the current UTC time with the given format.
func Format(layout string) string {
	return Now().Format(layout)
}

// Unix returns the current UTC time as Unix timestamp.
func Unix() int64 {
	return Now().Unix()
}

// UnixNano returns the current UTC time as Unix timestamp in nanoseconds.
func UnixNano() int64 {
	return Now().UnixNano()
}

// Since returns the time elapsed since t.
func Since(t time.Time) time.Duration {
	return Now().Sub(t)
}

// Until returns the duration until t.
func Until(t time.Time) time.Duration {
	return t.Sub(Now())
}

// Add returns the time t+d.
func Add(t time.Time, d time.Duration) time.Time {
	return t.Add(d)
}

// AddDate returns the time corresponding to adding the given number of years, months, and days to t.
func AddDate(t time.Time, years, months, days int) time.Time {
	return t.AddDate(years, months, days)
}

// Before reports whether the time instant t is before u.
func Before(t, u time.Time) bool {
	return t.Before(u)
}

// After reports whether the time instant t is after u.
func After(t, u time.Time) bool {
	return t.After(u)
}

// Equal reports whether t and u represent the same time instant.
func Equal(t, u time.Time) bool {
	return t.Equal(u)
}

// IsZero reports whether t represents the zero time instant.
func IsZero(t time.Time) bool {
	return t.IsZero()
}

// Parse parses a formatted string and returns the time value it represents.
func Parse(layout, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

// ParseInLocation is like Parse but differs in two ways.
func ParseInLocation(layout, value string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(layout, value, loc)
}
