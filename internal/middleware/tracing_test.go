package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter() *gin.Engine {
	r := gin.New()
	r.Use(RequestTracing())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

func TestRequestTracing_AutoGenerate(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	resp.Header.Del("Content-Type")

	traceID := resp.Header.Get(TraceIDHeader)
	if traceID == "" {
		t.Fatal("expected trace ID in response header, got empty string")
	}
}

func TestRequestTracing_ReuseFromHeader(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(TraceIDHeader, "my-custom-trace-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	traceID := resp.Header.Get(TraceIDHeader)
	if traceID != "my-custom-trace-id" {
		t.Errorf("trace ID = %q, want %q", traceID, "my-custom-trace-id")
	}
}

func TestRequestTracing_ResponseHeaderContainsTraceID(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	traceID := resp.Header.Get(TraceIDHeader)
	if traceID == "" {
		t.Fatal("expected X-Request-ID in response headers")
	}
}

func TestRequestTracing_GinContextHasTraceID(t *testing.T) {
	r := gin.New()
	r.Use(RequestTracing())
	r.GET("/test", func(c *gin.Context) {
		val, exists := c.Get(TraceIDContextKey)
		if !exists {
			c.String(http.StatusInternalServerError, "no trace_id in gin context")
			return
		}
		traceID, ok := val.(string)
		if !ok || traceID == "" {
			c.String(http.StatusInternalServerError, "trace_id is not a non-empty string")
			return
		}
		c.String(http.StatusOK, traceID)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if w.Body.String() == "" {
		t.Error("expected non-empty trace ID from gin context")
	}
}

func TestRequestTracing_RequestContextHasTraceID(t *testing.T) {
	r := gin.New()
	r.Use(RequestTracing())
	r.GET("/test", func(c *gin.Context) {
		traceID := c.GetString(TraceIDContextKey)
		if traceID == "" {
			c.String(http.StatusInternalServerError, "no trace_id")
			return
		}
		c.String(http.StatusOK, traceID)
	})

	customID := "ctx-trace-123"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(TraceIDHeader, customID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != customID {
		t.Errorf("body = %q, want %q", w.Body.String(), customID)
	}
}
