package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// DeploySSE handles SSE connections for deploy progress.
// GET /api/v1/sse/deploy/:app_id
func DeploySSE(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.Param("app_id")

		// Set SSE headers
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no") // disable nginx buffering

		ctx := c.Request.Context()
		ch := bridge.EventBus.Subscribe(ctx, appID)

		// Send heartbeat every 15s
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		// Send initial connection event
		c.SSEvent("connected", gin.H{"app_id": appID, "timestamp": time.Now().Format(time.RFC3339)})
		c.Writer.Flush()

		for {
			select {
			case <-ctx.Done():
				return
			case event := <-ch:
				c.SSEvent("deploy", event)
				c.Writer.Flush()
				if event.Step == "done" {
					return // close connection when deploy finishes
				}
			case <-heartbeat.C:
				c.SSEvent("heartbeat", gin.H{"timestamp": time.Now().Format(time.RFC3339)})
				c.Writer.Flush()
			}
		}
	}
}

// DeployAsyncHandler starts an async deploy and returns the task ID and SSE URL.
// POST /api/v1/apps/:id/deploy?async=true
func DeployAsyncHandler(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cfg mcp.DeployConfig
		if err := c.ShouldBindJSON(&cfg); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		appID := c.Param("id")

		taskID, err := bridge.DeployAsync(c.Request.Context(), cfg, appID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusAccepted, gin.H{
			"status":  "accepted",
			"task_id": taskID,
			"message": "deploy started",
			"sse_url": fmt.Sprintf("/api/v1/sse/deploy/%s", appID),
		})
	}
}
