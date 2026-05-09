package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ExportMonitorData exports monitoring check results.
// GET /api/v1/monitors/:id/export?format=csv&start=...&end=...&limit=10000
func ExportMonitorData(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		format := c.DefaultQuery("format", "csv")
		limit := 10000
		if l, err := strconv.Atoi(c.DefaultQuery("limit", "10000")); err == nil && l > 0 && l <= 50000 {
			limit = l
		}

		startStr := c.DefaultQuery("start", time.Now().Add(-7*24*time.Hour).Format(time.RFC3339))
		endStr := c.DefaultQuery("end", time.Now().Format(time.RFC3339))
		start, _ := time.Parse(time.RFC3339, startStr)
		end, _ := time.Parse(time.RFC3339, endStr)

		var results []map[string]interface{}
		db.Table("monitor_check_results").
			Where("monitor_id = ? AND created_at >= ? AND created_at <= ?", id, start, end).
			Order("created_at ASC").
			Limit(limit).
			Find(&results)

		if len(results) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "no data found for the given parameters"})
			return
		}

		filename := fmt.Sprintf("monitor-%s-%s-to-%s", id, start.Format("20060102"), end.Format("20060102"))

		switch format {
		case "json":
			c.Header("Content-Type", "application/json")
			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.json"`, filename))
			_ = json.NewEncoder(c.Writer).Encode(gin.H{
				"monitor_id": id,
				"start":      startStr,
				"end":        endStr,
				"total":      len(results),
				"data":       results,
			})
		default:
			c.Header("Content-Type", "text/csv")
			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
			writer := csv.NewWriter(c.Writer)
			defer writer.Flush()
			_ = writer.Write([]string{"ID", "Monitor ID", "Status", "Status Code", "Latency (ms)", "Message", "Created At"})
			for _, r := range results {
				_ = writer.Write([]string{
					fmt.Sprint(r["id"]),
					fmt.Sprint(r["monitor_id"]),
					fmt.Sprint(r["status"]),
					fmt.Sprint(r["status_code"]),
					fmt.Sprint(r["latency"]),
					fmt.Sprint(r["message"]),
					fmt.Sprint(r["created_at"]),
				})
			}
		}
	}
}

// ExportAlertHistory exports alert history.
// GET /api/v1/alerts/export?format=csv&status=firing&severity=critical
func ExportAlertHistory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		format := c.DefaultQuery("format", "csv")
		limit := 10000
		if l, err := strconv.Atoi(c.DefaultQuery("limit", "10000")); err == nil && l > 0 && l <= 50000 {
			limit = l
		}
		status := c.Query("status")
		severity := c.Query("severity")

		query := db.Table("alert_histories")
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if severity != "" {
			query = query.Where("severity = ?", severity)
		}

		var alerts []map[string]interface{}
		query.Order("fired_at DESC").Limit(limit).Find(&alerts)

		if len(alerts) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "no data found for the given parameters"})
			return
		}

		filename := fmt.Sprintf("alerts-%s", time.Now().Format("20060102"))

		switch format {
		case "json":
			c.Header("Content-Type", "application/json")
			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.json"`, filename))
			_ = json.NewEncoder(c.Writer).Encode(gin.H{"total": len(alerts), "data": alerts})
		default:
			c.Header("Content-Type", "text/csv")
			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
			writer := csv.NewWriter(c.Writer)
			defer writer.Flush()
			_ = writer.Write([]string{"ID", "Rule ID", "Rule Name", "Severity", "Message", "Value", "Threshold", "Status", "Fired At", "Resolved At"})
			for _, a := range alerts {
				_ = writer.Write([]string{
					fmt.Sprint(a["id"]),
					fmt.Sprint(a["rule_id"]),
					fmt.Sprint(a["rule_name"]),
					fmt.Sprint(a["severity"]),
					fmt.Sprint(a["message"]),
					fmt.Sprint(a["value"]),
					fmt.Sprint(a["threshold"]),
					fmt.Sprint(a["status"]),
					fmt.Sprint(a["fired_at"]),
					fmt.Sprint(a["resolved_at"]),
				})
			}
		}
	}
}
