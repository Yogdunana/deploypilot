package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
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

// ListAuditLogs lists audit logs (placeholder using deployment records).
func ListAuditLogs(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var records []model.DeploymentRecord
		query := db.Model(&model.DeploymentRecord{}).Order("created_at DESC").Limit(100)
		if err := query.Find(&records).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if records == nil {
			records = []model.DeploymentRecord{}
		}
		respondSuccess(c, records)
	}
}
