package middleware

import (
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/tracing"
	"github.com/gin-gonic/gin"
)

const (
	TraceIDHeader     = "X-Request-ID"
	TraceIDContextKey = "trace_id"
)

// RequestTracing is a Gin middleware that extracts or generates a trace ID
// for each request, stores it in both the Gin context and the request context,
// and logs request completion with the trace ID.
func RequestTracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(TraceIDHeader)
		if traceID == "" {
			traceID = tracing.GenerateTraceID()
		}
		c.Set(TraceIDContextKey, traceID)
		ctx := tracing.ContextWithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Header(TraceIDHeader, traceID)

		start := time.Now()
		c.Next()

		slog.InfoContext(ctx, "request completed",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}
