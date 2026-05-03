package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// GetTrialStatusHandler returns the current trial period status.
func GetTrialStatusHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := b.GetTrialStatus(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// ExtendTrialHandler extends a trial period (admin only).
func ExtendTrialHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			MachineID string `json:"machine_id" binding:"required"`
			Days      int    `json:"days" binding:"required,min=1,max=365"`
			Reason    string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "invalid request body")
			return
		}

		result, err := b.ExtendTrial(c.Request.Context(), input.MachineID, input.Days, input.Reason)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}

// ListTrialPeriodsHandler returns all trial periods (admin only).
func ListTrialPeriodsHandler(b *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := b.ListTrialPeriods(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, result)
	}
}
