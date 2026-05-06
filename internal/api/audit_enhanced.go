package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// ExportAuditLogs exports audit logs in CSV or JSON format.
// GET /api/v1/audit-logs/export?format=csv&log_type=auth&start_date=...&end_date=...
func ExportAuditLogs(auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		format := c.DefaultQuery("format", "json")

		filter := buildAuditFilter(c)
		// Export only active (non-archived) logs by default
		activeOnly := false
		filter.Archived = &activeOnly

		data, err := auditSvc.Export(c.Request.Context(), filter, format)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to export audit logs"})
			return
		}

		contentType := "application/json"
		fileExt := ".json"
		if format == "csv" {
			contentType = "text/csv"
			fileExt = ".csv"
		}

		filename := "audit_logs_" + time.Now().Format("20060102_150405") + fileExt
		c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
		c.Data(http.StatusOK, contentType, data)
	}
}

// ArchiveAuditLogs archives audit logs older than the specified number of days.
// POST /api/v1/audit-logs/archive
func ArchiveAuditLogs(auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			OlderThanDays int `json:"older_than_days" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		count, err := auditSvc.Archive(c.Request.Context(), input.OlderThanDays)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to archive audit logs"})
			return
		}

		respondSuccess(c, gin.H{
			"message":          "audit logs archived",
			"archived_count":   count,
			"older_than_days":  input.OlderThanDays,
		})
	}
}

// GetAuditStats returns audit log statistics grouped by log type.
// GET /api/v1/audit-logs/stats
func GetAuditStats(auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := auditSvc.Stats(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to get audit stats"})
			return
		}

		var total int64
		for _, count := range stats {
			total += count
		}
		stats["total"] = total

		respondSuccess(c, stats)
	}
}

// VerifyAuditLogs checks integrity of audit logs.
// POST /api/v1/audit-logs/verify
func VerifyAuditLogs(auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		activeOnly := false
		logs, _, err := auditSvc.List(c.Request.Context(), service.AuditFilter{
			Archived: &activeOnly,
			Page:     1,
			PageSize: 1000,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to query audit logs"})
			return
		}

		failed := auditSvc.VerifyRecords(logs)
		if len(failed) == 0 {
			respondSuccess(c, gin.H{
				"verified_count": len(logs),
				"failed_count":   0,
				"status":         "all records intact",
			})
		} else {
			respondSuccess(c, gin.H{
				"verified_count": len(logs),
				"failed_count":   len(failed),
				"failed_ids":     failed,
				"status":         "integrity check failed",
			})
		}
	}
}

// buildAuditFilter constructs an AuditFilter from query parameters.
func buildAuditFilter(c *gin.Context) service.AuditFilter {
	filter := service.AuditFilter{}

	if v := c.Query("user_id"); v != "" {
		if uid, err := strconv.ParseUint(v, 10, 64); err == nil {
			filter.UserID = uint(uid)
		}
	}
	filter.Username = c.Query("username")
	filter.Action = c.Query("action")
	filter.ResourceType = c.Query("resource_type")
	filter.LogType = c.Query("log_type")
	filter.TraceID = c.Query("trace_id")

	if v := c.Query("page"); v != "" {
		if page, err := strconv.Atoi(v); err == nil {
			filter.Page = page
		}
	}
	if v := c.Query("page_size"); v != "" {
		if pageSize, err := strconv.Atoi(v); err == nil && pageSize > 0 && pageSize <= 100 {
			filter.PageSize = pageSize
		}
	}

	if v := c.Query("start_date"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.StartTime = t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.StartTime = t
		}
	}
	if v := c.Query("end_date"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.EndTime = t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.EndTime = t
		}
	}

	return filter
}
