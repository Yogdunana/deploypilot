package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrors "github.com/Yogdunana/deploypilot/pkg/errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestRespondAppError_UsesMappedStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/err", func(c *gin.Context) {
		RespondAppError(c, appErrors.ErrAppNotFound)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/err", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["code"] != "E011" {
		t.Errorf("code = %v, want E011", body["code"])
	}
}

func TestRespondAppError_Nil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/err", func(c *gin.Context) {
		RespondAppError(c, nil)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/err", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestParsePaginationParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		query        string
		wantPage     int
		wantPageSize int
	}{
		{"", 1, 20},
		{"?page=3&page_size=50", 3, 50},
		{"?page=abc&page_size=xyz", 1, 20},
		{"?page=0&page_size=0", 1, 20},
		{"?page=-2&page_size=200", 1, 20},
		{"?page=2&page_size=100", 2, 100},
	}

	for _, tc := range tests {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/items"+tc.query, nil)
		page, pageSize := parsePaginationParams(c)
		if page != tc.wantPage || pageSize != tc.wantPageSize {
			t.Errorf("query %q => (%d, %d), want (%d, %d)", tc.query, page, pageSize, tc.wantPage, tc.wantPageSize)
		}
	}
}

func TestIsRecordNotFound(t *testing.T) {
	if !isRecordNotFound(gorm.ErrRecordNotFound) {
		t.Fatal("expected gorm.ErrRecordNotFound to match")
	}
	if isRecordNotFound(nil) {
		t.Fatal("nil should not be not-found")
	}
}
