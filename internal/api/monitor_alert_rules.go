package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateAlertRule creates a new alert rule.
// POST /api/v1/monitor/alert-rules
func CreateAlertRule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rule model.AlertRuleRecord
		if err := c.ShouldBindJSON(&rule); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		svc := service.NewAlertRuleService(db)
		if err := svc.CreateAlertRule(&rule); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error", err.Error())
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": rule})
	}
}

// UpdateAlertRule updates an existing alert rule.
// PUT /api/v1/monitor/alert-rules/:id
func UpdateAlertRule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var rule model.AlertRuleRecord
		if err := c.ShouldBindJSON(&rule); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}
		rule.ID = id

		svc := service.NewAlertRuleService(db)
		if err := svc.UpdateAlertRule(&rule); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": rule})
	}
}

// DeleteAlertRule deletes an alert rule.
// DELETE /api/v1/monitor/alert-rules/:id
func DeleteAlertRule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		svc := service.NewAlertRuleService(db)
		if err := svc.DeleteAlertRule(id); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "alert rule deleted"})
	}
}

// GetAlertRule retrieves a single alert rule.
// GET /api/v1/monitor/alert-rules/:id
func GetAlertRule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		svc := service.NewAlertRuleService(db)
		rule, err := svc.GetAlertRule(id)
		if err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": rule})
	}
}
