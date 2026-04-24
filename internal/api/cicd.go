package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// TriggerCIBuild triggers a CI/CD build.
func TriggerCIBuild(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Provider string `json:"provider" binding:"required"`
			Repo     string `json:"repo" binding:"required"`
			Branch   string `json:"branch" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		result, err := bridge.TriggerCIBuild(c.Request.Context(), input.Provider, input.Repo, input.Branch)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// GetCIBuildStatus gets the status of a CI/CD build.
func GetCIBuildStatus(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Query("provider")
		if provider == "" {
			respondError(c, http.StatusBadRequest, "provider query parameter is required")
			return
		}
		runID := c.Param("runID")
		if runID == "" {
			respondError(c, http.StatusBadRequest, "runID is required")
			return
		}

		result, err := bridge.GetCIBuildStatus(c.Request.Context(), provider, runID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}
