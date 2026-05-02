package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// QueryMonitorHistory returns check results for a monitor within a time range.
// GET /api/v1/monitors/:id/history?start=...&end=...&interval=5m&limit=1000
func (m *MonitorAPI) QueryMonitorHistory(c *gin.Context) {
	id := c.Param("id")
	startStr := c.DefaultQuery("start", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.DefaultQuery("end", time.Now().Format(time.RFC3339))
	limit := 1000
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "1000")); err == nil && l > 0 && l <= 10000 {
		limit = l
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid start time format, use RFC3339"})
		return
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid end time format, use RFC3339"})
		return
	}

	results, err := m.monSvc.QueryMonitorHistory(c.Request.Context(), id, start, end, limit)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	interval := c.DefaultQuery("interval", "5m")
	aggregated := aggregateByInterval(results, interval)

	respondSuccess(c, gin.H{
		"raw":        results,
		"aggregated": aggregated,
		"interval":   interval,
		"total":      len(results),
		"start":      startStr,
		"end":        endStr,
	})
}

// QueryMonitorOverview returns aggregated stats for all monitors.
// GET /api/v1/monitors/overview?period=24h
func (m *MonitorAPI) QueryMonitorOverview(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")
	dur := parseMonitorDuration(period, 24*time.Hour)
	since := time.Now().Add(-dur)

	monitors, err := m.monSvc.ListMonitors(c.Request.Context(), c.GetString("tenant_id"))
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	type MonitorOverview struct {
		MonitorID   string  `json:"monitor_id"`
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		Status      string  `json:"status"`
		Uptime      float64 `json:"uptime"`
		AvgLatency  float64 `json:"avg_latency"`
		TotalChecks int     `json:"total_checks"`
		UpChecks    int     `json:"up_checks"`
	}

	overviews := make([]MonitorOverview, len(monitors))
	for i, mon := range monitors {
		overviews[i] = MonitorOverview{
			MonitorID:   mon.ID,
			Name:        mon.Name,
			Type:        mon.Type,
			Status:      mon.Status,
			Uptime:      mon.Uptime,
			AvgLatency:  mon.AvgLatency,
			TotalChecks: mon.TotalChecks,
			UpChecks:    mon.UpChecks,
		}
	}

	respondSuccess(c, gin.H{
		"monitors": overviews,
		"total":    len(overviews),
		"period":   period,
		"since":    since.Format(time.RFC3339),
	})
}

// parseMonitorDuration parses a duration string like "24h", "7d", "30m".
func parseMonitorDuration(s string, defaultDur time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err == nil {
		return d
	}
	if len(s) > 0 && s[len(s)-1] == 'd' {
		if days, err := strconv.Atoi(s[:len(s)-1]); err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	return defaultDur
}

// aggregateByInterval groups check results by time interval for chart display.
func aggregateByInterval(results []service.MonitorCheckResult, interval string) []map[string]interface{} {
	if len(results) == 0 {
		return nil
	}

	dur, err := time.ParseDuration(interval)
	if err != nil {
		if len(interval) > 0 && interval[len(interval)-1] == 'd' {
			if days, err := strconv.Atoi(interval[:len(interval)-1]); err == nil {
				dur = time.Duration(days) * 24 * time.Hour
			}
		}
	}
	if dur == 0 {
		dur = 5 * time.Minute
	}

	type bucket struct {
		time   time.Time
		count  int
		up     int
		down   int
		latSum float64
	}

	var buckets []bucket
	current := bucket{time: time.Time{}}

	for _, r := range results {
		t, _ := time.Parse(time.RFC3339, r.CreatedAt)
		if current.time.IsZero() || t.Sub(current.time) >= dur {
			if current.count > 0 {
				buckets = append(buckets, current)
			}
			current = bucket{time: t}
		}
		current.count++
		if r.Status == "up" {
			current.up++
		} else {
			current.down++
		}
		current.latSum += r.Latency
	}
	if current.count > 0 {
		buckets = append(buckets, current)
	}

	result := make([]map[string]interface{}, len(buckets))
	for i, b := range buckets {
		avgLat := 0.0
		if b.count > 0 {
			avgLat = b.latSum / float64(b.count)
		}
		uptime := 0.0
		if b.count > 0 {
			uptime = float64(b.up) / float64(b.count) * 100
		}
		result[i] = map[string]interface{}{
			"time":        b.time.Format(time.RFC3339),
			"total":       b.count,
			"up":          b.up,
			"down":        b.down,
			"avg_latency": avgLat,
			"uptime_pct":  uptime,
		}
	}
	return result
}
