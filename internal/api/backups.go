package api

import (
	"net/http"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/backup"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// ListBackups lists backups for an app.
// @Summary      List backups
// @Description  Retrieve backup records for an application
// @Tags         Backups
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Success      200 {object} map[string]interface{} "status, data (array of backup.Record)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id}/backups [get]
func ListBackups(backupSvc *backup.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.Param("id")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		records, err := backupSvc.ListByApp(appID, limit)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		if records == nil {
			records = []backup.Record{}
		}
		respondSuccess(c, records)
	}
}

// DeleteBackup deletes a backup record and its file.
// @Summary      Delete a backup
// @Description  Delete a backup record by ID
// @Tags         Backups
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Application ID"
// @Param        backupId path string true "Backup ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.backup_id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /apps/{id}/backups/{backupId} [delete]
func DeleteBackup(backupSvc *backup.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		backupID := c.Param("backupId")
		if err := backupSvc.DeleteRecord(backupID); err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}
		respondSuccess(c, gin.H{"message": "backup deleted", "backup_id": backupID})
	}
}

// ListAuditLogs lists audit logs with pagination and filtering.
// @Summary      List audit logs
// @Description  Retrieve audit logs with pagination and optional filtering by user, action, or resource type
// @Tags         Audit
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page number" default(1)
// @Param        page_size query int false "Page size (1-100)" default(20)
// @Param        user_id query string false "Filter by user ID"
// @Param        action query string false "Filter by action type"
// @Param        resource_type query string false "Filter by resource type"
// @Success      200 {object} map[string]interface{} "status, data.logs, data.total, data.page, data.page_size"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /audit-logs [get]
func ListAuditLogs(auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		filter := service.AuditFilter{
			Page:     page,
			PageSize: pageSize,
		}
		if userID := c.Query("user_id"); userID != "" {
			uid, _ := strconv.ParseUint(userID, 10, 64)
			filter.UserID = uint(uid)
		}
		if action := c.Query("action"); action != "" {
			filter.Action = action
		}
		if resourceType := c.Query("resource_type"); resourceType != "" {
			filter.ResourceType = resourceType
		}

		logs, total, err := auditSvc.List(c.Request.Context(), filter)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{
			"logs":      logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
	}
}
