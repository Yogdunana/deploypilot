package tracing

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestContextWithTraceID_Roundtrip(t *testing.T) {
	traceID := "test-trace-id-123"
	ctx := context.Background()
	ctx = ContextWithTraceID(ctx, traceID)

	got := TraceIDFromContext(ctx)
	if got != traceID {
		t.Errorf("TraceIDFromContext() = %q, want %q", got, traceID)
	}
}

func TestTraceIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := TraceIDFromContext(ctx)
	if got != "" {
		t.Errorf("TraceIDFromContext() = %q, want empty string", got)
	}
}

func TestGenerateTraceID_ValidUUID(t *testing.T) {
	id := GenerateTraceID()
	if id == "" {
		t.Error("GenerateTraceID() returned empty string")
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Errorf("GenerateTraceID() = %q, not a valid UUID: %v", id, err)
	}
	if parsed.String() != id {
		t.Errorf("UUID roundtrip mismatch: got %q, want %q", parsed.String(), id)
	}
}

func TestGenerateTraceID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateTraceID()
		if ids[id] {
			t.Errorf("GenerateTraceID() produced duplicate: %q", id)
		}
		ids[id] = true
	}
}
