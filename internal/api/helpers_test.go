package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// --- splitAndTrim (additional edge cases beyond what api_test.go covers) ---

// TestSplitAndTrim_WhitespaceOnlyEntriesDropped verifies that entries
// made up only of whitespace are treated as empty and dropped, which
// is the most common source of "empty allowed-IPs entry" bugs that later
// cause the IP-matcher to behave unexpectedly.
func TestSplitAndTrim_WhitespaceOnlyEntriesDropped(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a,  ,  ,b", []string{"a", "b"}}, // multiple whitespace-only fields
		{"  ,\t, \n ,x", []string{"x"}},    // mixed whitespace (\t, \n, space)
		{"  ,  ,  ", nil},                 // all-whitespace
		{", \t , \n", nil},                // whitespace between commas
	}
	for _, c := range cases {
		got := splitAndTrim(c.in)
		if len(got) != len(c.want) {
			t.Errorf("splitAndTrim(%q) length=%d, want %d (got=%v)", c.in, len(got), len(c.want), got)
			continue
		}
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Errorf("splitAndTrim(%q)[%d]=%q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

// TestSplitAndTrim_DoesNotReturnNilForEmpty verifies that the helper
// always returns a non-nil slice (callers may rely on `for _, v := range
// splitAndTrim(...)` to never be a no-op on a nil receiver).
func TestSplitAndTrim_DoesNotReturnNilForEmpty(t *testing.T) {
	got := splitAndTrim("")
	if got == nil {
		t.Error("splitAndTrim should never return nil (must return empty slice)")
	}
}

// --- parsePaginationParams ---

// TestParsePaginationParams_Defaults verifies the no-query-string path
// returns the documented defaults (page=1, pageSize=20).
func TestParsePaginationParams_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/list", nil)

	page, pageSize := parsePaginationParams(c)
	if page != 1 {
		t.Errorf("page=%d, want 1", page)
	}
	if pageSize != 20 {
		t.Errorf("pageSize=%d, want 20", pageSize)
	}
}

// TestParsePaginationParams_CustomValues verifies explicit query params
// are parsed.
func TestParsePaginationParams_CustomValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/list?page=3&page_size=50", nil)

	page, pageSize := parsePaginationParams(c)
	if page != 3 {
		t.Errorf("page=%d, want 3", page)
	}
	if pageSize != 50 {
		t.Errorf("pageSize=%d, want 50", pageSize)
	}
}

// TestParsePaginationParams_NegativeOrZeroPageClampsToOne ensures the
// page parameter is clamped to a positive value. Both 0 and negative
// inputs would otherwise produce a non-positive SQL OFFSET, which
// silently returns the wrong slice (or panics in older databases).
func TestParsePaginationParams_NegativeOrZeroPageClampsToOne(t *testing.T) {
	for _, q := range []string{"page=-5", "page=0", "page=-1"} {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/list?"+q, nil)

		page, _ := parsePaginationParams(c)
		if page != 1 {
			t.Errorf("query %q: page=%d, want 1 (clamped)", q, page)
		}
	}
}

// TestParsePaginationParams_PageSizeBounds verifies the page_size guard:
// values <= 0 or > 100 fall back to the default 20, so a malicious
// caller cannot request a giant page that would DoS the database.
func TestParsePaginationParams_PageSizeBounds(t *testing.T) {
	cases := []struct {
		query string
		want  int
	}{
		{"page_size=0", 20},
		{"page_size=-1", 20},
		{"page_size=101", 20},
		{"page_size=1000", 20},
		{"page_size=1", 1},     // minimum accepted value
		{"page_size=100", 100}, // boundary, accepted
		{"page_size=20", 20},   // default
	}
	for _, c := range cases {
		gin.SetMode(gin.TestMode)
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest("GET", "/list?"+c.query, nil)

		_, pageSize := parsePaginationParams(ctx)
		if pageSize != c.want {
			t.Errorf("query %q: pageSize=%d, want %d", c.query, pageSize, c.want)
		}
	}
}

// TestParsePaginationParams_NonNumericFallsBackToDefault ensures that a
// non-numeric page/page_size falls back to the default rather than
// producing a 0 (which would silently break the SQL offset).
func TestParsePaginationParams_NonNumericFallsBackToDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/list?page=abc&page_size=xyz", nil)

	page, pageSize := parsePaginationParams(c)
	if page != 1 {
		t.Errorf("page=%d, want 1 (default for non-numeric)", page)
	}
	if pageSize != 20 {
		t.Errorf("pageSize=%d, want 20 (default for non-numeric)", pageSize)
	}
}

// TestParsePaginationParams_FloatFallsBackToDefault ensures that
// non-integer numeric strings (e.g. "1.5") also fall back to defaults
// rather than being silently truncated to 1.
func TestParsePaginationParams_FloatFallsBackToDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/list?page=1.5&page_size=10.7", nil)

	page, pageSize := parsePaginationParams(c)
	if page != 1 {
		t.Errorf("page=%d, want 1 (default for non-integer numeric)", page)
	}
	if pageSize != 20 {
		t.Errorf("pageSize=%d, want 20 (default for non-integer numeric)", pageSize)
	}
}
