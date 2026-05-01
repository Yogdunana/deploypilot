package api

import (
	"time"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// AlertSSE handles SSE connections for alert events.
// GET /api/v1/sse/alerts
func AlertSSE(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set SSE headers
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		ctx := c.Request.Context()

		if bridge.TypedBus == nil {
			c.SSEvent("error", gin.H{"message": "typed event bus not available"})
			c.Writer.Flush()
			return
		}

		ch := bridge.TypedBus.Subscribe(ctx, "alert:all")

		// Send heartbeat every 15s
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		// Send initial connection event
		c.SSEvent("connected", gin.H{"topic": "alert:all", "timestamp": time.Now().Format(time.RFC3339)})
		c.Writer.Flush()

		for {
			select {
			case <-ctx.Done():
				return
			case event := <-ch:
				c.SSEvent("alert", event)
				c.Writer.Flush()
			case <-heartbeat.C:
				c.SSEvent("heartbeat", gin.H{"timestamp": time.Now().Format(time.RFC3339)})
				c.Writer.Flush()
			}
		}
	}
}
