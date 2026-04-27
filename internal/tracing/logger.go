package tracing

import (
	"context"
	"log/slog"
)

// TraceHandler is an slog.Handler wrapper that injects trace_id into log records.
type TraceHandler struct {
	handler slog.Handler
}

// NewTraceHandler wraps an existing slog.Handler to automatically inject
// trace_id from the context into every log record.
func NewTraceHandler(handler slog.Handler) slog.Handler {
	return &TraceHandler{handler: handler}
}

// Enabled returns true if the underlying handler is enabled for the given level.
func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle injects trace_id from the context into the log record before
// passing it to the underlying handler.
func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		r.AddAttrs(slog.String("trace_id", traceID))
	}
	return h.handler.Handle(ctx, r)
}

// WithAttrs returns a new TraceHandler with the given attributes appended.
func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{handler: h.handler.WithAttrs(attrs)}
}

// WithGroup returns a new TraceHandler with the given group name.
func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{handler: h.handler.WithGroup(name)}
}
