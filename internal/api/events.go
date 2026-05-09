package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListEventLogs returns paginated event logs with optional type filter.
// GET /api/v1/events?event_type=deploy&page=1&page_size=20
func ListEventLogs(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventType := c.Query("event_type")
		page, pageSize := parsePaginationParams(c)

		query := db.Model(&model.EventLog{})
		if eventType != "" {
			query = query.Where("event_type = ?", eventType)
		}

		var total int64
		query.Count(&total)

		var logs []model.EventLog
		offset := (page - 1) * pageSize
		if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		respondSuccess(c, gin.H{
			"items":     logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
	}
}

// GetEventStats returns event counts grouped by type.
// GET /api/v1/events/stats
func GetEventStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		type CountResult struct {
			EventType string `json:"event_type"`
			Count     int64  `json:"count"`
		}
		var results []CountResult
		if err := db.Model(&model.EventLog{}).
			Select("event_type, COUNT(*) as count").
			Group("event_type").
			Find(&results).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		// Convert to map
		stats := make(map[string]int64)
		for _, r := range results {
			stats[r.EventType] = r.Count
		}

		respondSuccess(c, gin.H{
			"by_type": stats,
		})
	}
}
