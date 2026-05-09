package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// GetDegradationStatusHandler returns the current degradation status.
func GetDegradationStatusHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := b.GetDegradationStatus(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// ListDegradationAuditsHandler returns recent degradation audit entries (admin only).
func ListDegradationAuditsHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Limit int `form:"limit"`
		}
		_ = c.ShouldBindQuery(&input)

		result, err := b.ListDegradationAudits(c.Request.Context(), input.Limit)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// ExportDegradationSummaryHandler returns a summary of all data for export.
func ExportDegradationSummaryHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := b.ExportDegradationSummary(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}
