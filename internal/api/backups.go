package api

import (
	"net/http"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListBackups lists backups for an app (currently returns empty since backups are in-memory).
func ListBackups(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Backups are currently tracked in-memory in the bridge.
		// Return an empty list as the backup system is not persisted to DB yet.
		respondSuccess(c, []model.DeploymentRecord{})
	}
}

// DeleteBackup deletes a backup (placeholder).
func DeleteBackup(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		backupID := c.Param("backupId")
		// Backups are in-memory; this is a placeholder for future persistence.
		respondSuccess(c, gin.H{"message": "backup deleted", "backup_id": backupID})
	}
}

// ListAuditLogs lists audit logs with pagination and filtering.
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
			respondError(c, http.StatusInternalServerError, err.Error())
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
