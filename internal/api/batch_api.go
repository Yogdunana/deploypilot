package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// BatchDeployHandler handles batch deployment requests.
func BatchDeployHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Apps          []map[string]interface{} `json:"apps" binding:"required"`
			Strategy      string                    `json:"strategy"`
			MaxConcurrent int                       `json:"max_concurrent"`
			BatchSize     int                       `json:"batch_size"`
			ServerIDs     []string                  `json:"server_ids"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "invalid request body")
			return
		}
		if len(input.Apps) == 0 {
			respondErrori18n(c, http.StatusBadRequest, "apps list cannot be empty")
			return
		}
		if len(input.Apps) > 100 {
			respondErrori18n(c, http.StatusBadRequest, "batch size cannot exceed 100")
			return
		}

		config := mcp.BatchDeployConfig{
			Apps:          input.Apps,
			Strategy:      mcp.DeployStrategy(input.Strategy),
			MaxConcurrent: input.MaxConcurrent,
			BatchSize:     input.BatchSize,
			ServerIDs:     input.ServerIDs,
		}

		// Apply defaults
		if config.Strategy == "" {
			config.Strategy = mcp.StrategySequential
		}
		if config.MaxConcurrent <= 0 {
			config.MaxConcurrent = 5
		}
		if config.BatchSize <= 0 {
			config.BatchSize = 3
		}

		result, err := b.BatchDeployWithConfig(c.Request.Context(), config)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		respondSuccess(c, result)
	}
}
