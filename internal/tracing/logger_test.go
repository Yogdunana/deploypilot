package tracing

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// mockHandler collects records for inspection.
type mockHandler struct {
	records *[]mockRecord
}

type mockRecord struct {
	msg   string
	attrs map[string]string
}

func newMockHandler() *mockHandler {
	return &mockHandler{records: &[]mockRecord{}}
}

func (h *mockHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *mockHandler) Handle(_ context.Context, r slog.Record) error {
	msg := r.Message
	rec := mockRecord{msg: msg, attrs: make(map[string]string)}
	r.Attrs(func(a slog.Attr) bool {
		rec.attrs[a.Key] = a.Value.String()
		return true
	})
	*h.records = append(*h.records, rec)
	return nil
}

func (h *mockHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &mockHandler{records: h.records}
}

func (h *mockHandler) WithGroup(name string) slog.Handler {
	return &mockHandler{records: h.records}
}

func TestTraceHandler_InjectsTraceID(t *testing.T) {
	mock := newMockHandler()
	handler := NewTraceHandler(mock)

	traceID := "abc-123-def"
	ctx := ContextWithTraceID(context.Background(), traceID)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	if err := handler.Handle(ctx, r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	recs := *mock.records
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	if recs[0].attrs["trace_id"] != traceID {
		t.Errorf("trace_id = %q, want %q", recs[0].attrs["trace_id"], traceID)
	}
}

func TestTraceHandler_NoTraceID(t *testing.T) {
	mock := newMockHandler()
	handler := NewTraceHandler(mock)

	ctx := context.Background()

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	if err := handler.Handle(ctx, r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	recs := *mock.records
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	if _, ok := recs[0].attrs["trace_id"]; ok {
		t.Error("trace_id should not be present when context has no trace ID")
	}
}

func TestTraceHandler_WithAttrs(t *testing.T) {
	mock := newMockHandler()
	handler := NewTraceHandler(mock)

	attrs := []slog.Attr{slog.String("service", "deploypilot")}
	wrapped := handler.WithAttrs(attrs)

	traceID := "trace-456"
	ctx := ContextWithTraceID(context.Background(), traceID)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	if err := wrapped.Handle(ctx, r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	recs := *mock.records
	if len(recs) != 1 {
		t.Fatalf("expected 1 record from Handle, got %d", len(recs))
	}
	if recs[0].attrs["trace_id"] != traceID {
		t.Errorf("trace_id = %q, want %q", recs[0].attrs["trace_id"], traceID)
	}
	if recs[0].attrs["service"] != "deploypilot" {
		t.Errorf("service attr = %q, want %q", recs[0].attrs["service"], "deploypilot")
	}
}

func TestTraceHandler_WithGroup(t *testing.T) {
	mock := newMockHandler()
	handler := NewTraceHandler(mock)

	wrapped := handler.WithGroup("request")

	traceID := "trace-789"
	ctx := ContextWithTraceID(context.Background(), traceID)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	if err := wrapped.Handle(ctx, r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	recs := *mock.records
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	if recs[0].attrs["trace_id"] != traceID {
		t.Errorf("trace_id = %q, want %q", recs[0].attrs["trace_id"], traceID)
	}
}

func TestTraceHandler_Enabled(t *testing.T) {
	mock := newMockHandler()
	handler := NewTraceHandler(mock)

	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Enabled() should return true")
	}
}
