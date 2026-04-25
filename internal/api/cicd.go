package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// TriggerCIBuild triggers a CI/CD build.
// @Summary      Trigger a CI/CD build
// @Description  Trigger a new CI/CD build for a repository on a specific branch
// @Tags         CI/CD
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{provider=string,repo=string,branch=string} true "CI build trigger request"
// @Success      200 {object} map[string]interface{} "status, data (build result)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /cicd/trigger [post]
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
// @Summary      Get CI/CD build status
// @Description  Retrieve the status of a specific CI/CD build run
// @Tags         CI/CD
// @Produce      json
// @Security     BearerAuth
// @Param        provider query string true "CI provider name"
// @Param        runID path string true "CI build run ID"
// @Success      200 {object} map[string]interface{} "status, data (build status)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /cicd/status/{runID} [get]
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
