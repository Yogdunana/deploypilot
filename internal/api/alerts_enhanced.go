package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateAlertSilence creates a new alert silence period.
// POST /api/v1/alerts/silences
func CreateAlertSilence(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var silence model.AlertSilence
		if err := c.ShouldBindJSON(&silence); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		if silence.Name == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}
		if silence.EndsAt.Before(time.Now()) {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		silence.ID = generateID("silence")
		if err := db.Create(&silence).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		respondSuccess(c, silence)
	}
}

// ListAlertSilences returns active and upcoming silence periods.
// GET /api/v1/alerts/silences
func ListAlertSilences(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var silences []model.AlertSilence
		if err := db.Where("ends_at >= ?", time.Now()).Order("starts_at ASC").Find(&silences).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, silences)
	}
}

// DeleteAlertSilence deletes a silence period.
// DELETE /api/v1/alerts/silences/:id
func DeleteAlertSilence(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := db.Delete(&model.AlertSilence{}, "id = ?", id).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{"deleted": true})
	}
}

// CreateAlertEscalation creates a new escalation policy.
// POST /api/v1/alerts/escalations
func CreateAlertEscalation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var esc model.AlertEscalation
		if err := c.ShouldBindJSON(&esc); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		if esc.Name == "" || esc.Steps == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		esc.ID = generateID("esc")
		if err := db.Create(&esc).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		respondSuccess(c, esc)
	}
}

// ListAlertEscalations returns all escalation policies.
// GET /api/v1/alerts/escalations
func ListAlertEscalations(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var escalations []model.AlertEscalation
		if err := db.Order("created_at DESC").Find(&escalations).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, escalations)
	}
}

// DeleteAlertEscalation deletes an escalation policy.
// DELETE /api/v1/alerts/escalations/:id
func DeleteAlertEscalation(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := db.Delete(&model.AlertEscalation{}, "id = ?", id).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{"deleted": true})
	}
}

// ListAlertGroups returns active alert groups.
// GET /api/v1/alerts/groups
func ListAlertGroups(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var groups []model.AlertGroup
		if err := db.Where("status = ?", "firing").Order("last_alert_at DESC").Find(&groups).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, groups)
	}
}

// generateID creates a unique ID with a prefix.
func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
