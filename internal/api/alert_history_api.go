package api

import (
	"net/http"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListAlertHistory returns paginated alert history.
// GET /api/v1/alerts/history?page=1&page_size=20&status=firing&severity=critical&rule_id=xxx
func ListAlertHistory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, pageSize := parsePaginationParams(c)

		status := c.Query("status")
		severity := c.Query("severity")
		ruleID := c.Query("rule_id")

		query := db.Model(&model.AlertHistory{})
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if severity != "" {
			query = query.Where("severity = ?", severity)
		}
		if ruleID != "" {
			query = query.Where("rule_id = ?", ruleID)
		}

		var total int64
		query.Count(&total)

		var alerts []model.AlertHistory
		offset := (page - 1) * pageSize
		if err := query.Order("fired_at DESC").Offset(offset).Limit(pageSize).Find(&alerts).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		respondSuccess(c, gin.H{
			"data":        alerts,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		})
	}
}

// GetAlertHistory returns a single alert with related events.
// GET /api/v1/alerts/history/:id
func GetAlertHistory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var alert model.AlertHistory
		if err := db.Where("id = ?", id).First(&alert).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}

		var related []model.AlertHistory
		db.Where("rule_id = ? AND id != ? AND fired_at BETWEEN ? AND ?",
			alert.RuleID, id,
			alert.FiredAt.Add(-1*time.Hour),
			alert.FiredAt.Add(1*time.Hour),
		).Order("fired_at ASC").Limit(10).Find(&related)

		respondSuccess(c, gin.H{
			"alert":   alert,
			"related": related,
		})
	}
}

// GetAlertStats returns summary statistics for alerts.
// GET /api/v1/alerts/stats?period=7d
func GetAlertStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		period := c.DefaultQuery("period", "7d")
		dur := parseMonitorDuration(period, 7*24*time.Hour)
		since := time.Now().Add(-dur)

		type Stats struct {
			Total    int64 `json:"total"`
			Firing   int64 `json:"firing"`
			Resolved int64 `json:"resolved"`
			Critical int64 `json:"critical"`
			Warning  int64 `json:"warning"`
			Info     int64 `json:"info"`
		}

		var stats Stats
		db.Model(&model.AlertHistory{}).Where("fired_at >= ?", since).Count(&stats.Total)
		db.Model(&model.AlertHistory{}).Where("fired_at >= ? AND status = ?", since, "firing").Count(&stats.Firing)
		db.Model(&model.AlertHistory{}).Where("fired_at >= ? AND status = ?", since, "resolved").Count(&stats.Resolved)
		db.Model(&model.AlertHistory{}).Where("fired_at >= ? AND severity = ?", since, "critical").Count(&stats.Critical)
		db.Model(&model.AlertHistory{}).Where("fired_at >= ? AND severity = ?", since, "warning").Count(&stats.Warning)
		db.Model(&model.AlertHistory{}).Where("fired_at >= ? AND severity = ?", since, "info").Count(&stats.Info)

		respondSuccess(c, gin.H{
			"stats":  stats,
			"period": period,
			"since":  since.Format(time.RFC3339),
		})
	}
}
