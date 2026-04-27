package tracing

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// mockHandler collects records for inspection.
type mockHandler struct {
	records []mockRecord
}

type mockRecord struct {
	msg    string
	attrs  map[string]string
	groups []string
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
	h.records = append(h.records, rec)
	return nil
}

func (h *mockHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := &mockHandler{records: h.records}
	for _, a := range attrs {
		clone.records = append(clone.records, mockRecord{attrs: map[string]string{a.Key: a.Value.String()}})
	}
	return clone
}

func (h *mockHandler) WithGroup(name string) slog.Handler {
	return &mockHandler{records: h.records}
}

func TestTraceHandler_InjectsTraceID(t *testing.T) {
	mock := &mockHandler{}
	handler := NewTraceHandler(mock)

	traceID := "abc-123-def"
	ctx := ContextWithTraceID(context.Background(), traceID)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message")
	if err := handler.Handle(ctx, r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	if len(mock.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mock.records))
	}
	if mock.records[0].attrs["trace_id"] != traceID {
		t.Errorf("trace_id = %q, want %q", mock.records[0].attrs["trace_id"], traceID)
	}
}

func TestTraceHandler_NoTraceID(t *testing.T) {
	mock := &mockHandler{}
	handler := NewTraceHandler(mock)

	ctx := context.Background()

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message")
	if err := handler.Handle(ctx, r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	if len(mock.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mock.records))
	}
	if _, ok := mock.records[0].attrs["trace_id"]; ok {
		t.Error("trace_id should not be present when context has no trace ID")
	}
}

func TestTraceHandler_WithAttrs(t *testing.T) {
	mock := &mockHandler{}
	handler := NewTraceHandler(mock)

	attrs := []slog.Attr{slog.String("service", "deploypilot")}
	wrapped := handler.WithAttrs(attrs)

	traceID := "trace-456"
	ctx := ContextWithTraceID(context.Background(), traceID)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test")
	if err := wrapped.Handle(ctx, r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	if len(mock.records) != 2 {
		t.Fatalf("expected 2 records (1 from WithAttrs, 1 from Handle), got %d", len(mock.records))
	}
}

func TestTraceHandler_WithGroup(t *testing.T) {
	mock := &mockHandler{}
	handler := NewTraceHandler(mock)

	wrapped := handler.WithGroup("request")

	traceID := "trace-789"
	ctx := ContextWithTraceID(context.Background(), traceID)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test")
	if err := wrapped.Handle(ctx, r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	if len(mock.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mock.records))
	}
	if mock.records[0].attrs["trace_id"] != traceID {
		t.Errorf("trace_id = %q, want %q", mock.records[0].attrs["trace_id"], traceID)
	}
}

func TestTraceHandler_Enabled(t *testing.T) {
	mock := &mockHandler{}
	handler := NewTraceHandler(mock)

	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Enabled() should return true")
	}
}
