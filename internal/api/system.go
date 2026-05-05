package api

import (
	"log/slog"
	"net/http"
	"runtime"
	"strconv"

	"github.com/Yogdunana/deploypilot/internal/backup"
	"github.com/Yogdunana/deploypilot/internal/confirm"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/Yogdunana/deploypilot/internal/version"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetVersion returns the system version information.
// @Summary      Get system version
// @Description  Retrieve the current system version, Go runtime info, and CPU count
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data.version, data.go, data.goos, data.goarch, data.num_cpu"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Router       /system/version [get]
func GetVersion(c *gin.Context) {
	info := version.Info()
	respondSuccess(c, gin.H{
		"version":    info["version"],
		"build_time": info["build_time"],
		"git_commit": info["git_commit"],
		"go":         runtime.Version(),
		"goos":       runtime.GOOS,
		"goarch":     runtime.GOARCH,
		"num_cpu":    runtime.NumCPU(),
	})
}

// CheckUpdate checks for system updates.
// @Summary      Check for system updates
// @Description  Check if a newer version of Deploypilot is available
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (update check result)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /system/update/check [get]
func CheckUpdate(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := bridge.CheckSystemUpdate(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, result)
	}
}

// DoUpdate performs a system upgrade.
// @Summary      Perform system upgrade
// @Description  Trigger a system upgrade to the latest available version
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (upgrade result)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /system/update/perform [post]
func DoUpdate(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := bridge.PerformSystemUpdate(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, result)
	}
}

// SystemHealth checks the health of system components.
// @Summary      System health check
// @Description  Check the health status of system components including database connectivity
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data.status, data.database.status"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      503 {object} map[string]interface{} "service unavailable"
// @Router       /system/health [get]
func SystemHealth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		health := gin.H{"status": "healthy"}

		// Check database
		sqlDB, err := db.DB()
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "database health check failed", "error", err)
			health["status"] = "unhealthy"
			health["database"] = gin.H{"status": "error", "message": "connection failed"}
		} else if err := sqlDB.Ping(); err != nil {
			slog.ErrorContext(c.Request.Context(), "database health check failed", "error", err)
			health["status"] = "unhealthy"
			health["database"] = gin.H{"status": "error", "message": "connection failed"}
		} else {
			health["database"] = gin.H{"status": "ok"}
		}

		code := http.StatusOK
		if health["status"] != "healthy" {
			code = http.StatusServiceUnavailable
		}
		c.JSON(code, gin.H{"status": "success", "data": health})
	}
}

// GetSandboxConfig returns the current sandbox configuration.
// @Summary      Get sandbox configuration
// @Description  Retrieve the current command sandbox rules and mode
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "sandbox config"
// @Router       /system/sandbox [get]
func GetSandboxConfig(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		sb := bridge.GetSandbox()
		if sb == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"mode": "off", "rules": []interface{}{}}})
			return
		}
		cfg := sb.GetConfig()
		// Don't expose compiled regex to JSON
		type ruleOut struct {
			ID          string `json:"id"`
			Pattern     string `json:"pattern"`
			Description string `json:"description"`
			Enabled     bool   `json:"enabled"`
		}
		rules := make([]ruleOut, len(cfg.Rules))
		for i, r := range cfg.Rules {
			rules[i] = ruleOut{ID: r.ID, Pattern: r.Pattern, Description: r.Description, Enabled: r.Enabled}
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{
			"mode":        cfg.Mode,
			"log_blocked": cfg.LogBlocked,
			"rules":       rules,
		}})
	}
}

// ValidateSandboxCommand tests a command against sandbox rules without executing it.
// @Summary      Validate command against sandbox
// @Description  Check if a command would be allowed or blocked by the sandbox
// @Tags         System
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object{command=string} true "Command to validate"
// @Success      200 {object} map[string]interface{} "validation result"
// @Router       /system/sandbox/validate [post]
func ValidateSandboxCommand(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Command string `json:"command" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "command is required"})
			return
		}
		sb := bridge.GetSandbox()
		if sb == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"allowed": true, "mode": "off"}})
			return
		}
		if err := sb.Validate(req.Command); err != nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{
				"allowed": false,
				"error":   err.Error(),
			}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"allowed": true}})
	}
}

// ListConfirmations lists pending confirmation requests.
// @Summary      List confirmations
// @Description  List all pending confirmation requests
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Filter by status (pending/approved/rejected/expired/executed)"
// @Success      200 {object} map[string]interface{}
// @Router       /system/confirmations [get]
func ListConfirmations(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		store := bridge.GetConfirmationStore()
		if store == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": []interface{}{}})
			return
		}
		statusFilter := c.Query("status")
		requests := store.List(confirm.Status(statusFilter))
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": requests})
	}
}

// ConfirmRequest approves a pending confirmation.
// @Summary      Confirm a request
// @Description  Approve a pending confirmation request by ID
// @Tags         System
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Confirmation ID"
// @Success      200 {object} map[string]interface{}
// @Router       /system/confirmations/{id}/confirm [post]
func ConfirmRequest(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		store := bridge.GetConfirmationStore()
		if store == nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "confirmation system not available"})
			return
		}
		req, err := store.Confirm(id, c.GetString("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": req})
	}
}

// GetBruteForceStatus returns locked accounts and blocked IPs.
// @Summary      Get brute-force protection status
// @Description  List locked accounts and blocked IPs
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /system/bruteforce [get]
func GetBruteForceStatus(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		bf := bridge.BFProtector
		if bf == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"locked_accounts": []interface{}{}, "blocked_ips": []interface{}{}}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{
			"locked_accounts": bf.ListLockedAccounts(),
			"blocked_ips":     bf.ListBlockedIPs(),
		}})
	}
}

// UnlockBruteForceAccount unlocks a locked account.
// @Summary      Unlock a locked account
// @Description  Manually unlock an account that was locked by brute-force protection
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Param        username path string true "Username to unlock"
// @Success      200 {object} map[string]interface{}
// @Router       /system/bruteforce/accounts/{username}/unlock [post]
func UnlockBruteForceAccount(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		bf := bridge.BFProtector
		if bf == nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "brute-force protection not available"})
			return
		}
		if bf.UnlockAccount(username) {
			c.JSON(http.StatusOK, gin.H{"status": "success", "message": "account unlocked"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "account is not locked"})
		}
	}
}

// UnlockBruteForceIP unlocks a blocked IP.
// @Summary      Unlock a blocked IP
// @Description  Manually unlock an IP that was blocked by brute-force protection
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Param        ip path string true "IP address to unlock"
// @Success      200 {object} map[string]interface{}
// @Router       /system/bruteforce/ips/{ip}/unlock [post]
func UnlockBruteForceIP(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.Param("ip")
		bf := bridge.BFProtector
		if bf == nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "brute-force protection not available"})
			return
		}
		if bf.UnlockIP(ip) {
			c.JSON(http.StatusOK, gin.H{"status": "success", "message": "IP unlocked"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "IP is not blocked"})
		}
	}
}

// GetBackupStatus returns the database backup service status.
// @Summary      Get backup status
// @Description  Get database auto-backup service status and configuration
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /system/backup/status [get]
func GetBackupStatus(backupSvc *backup.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if backupSvc == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"enabled": false}})
			return
		}
		status := backupSvc.GetStatus()
		// Include cloud storage status
		cloudStatus := backupSvc.GetCloudStatus()
		status["cloud"] = cloudStatus
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": status})
	}
}

// TriggerBackup manually triggers a database backup.
// @Summary      Trigger database backup
// @Description  Manually trigger a database backup
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /system/backup/trigger [post]
func TriggerBackup(backupSvc *backup.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if backupSvc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "backup service not available"})
			return
		}
		record, err := backupSvc.CreateBackup(c.Request.Context(), "manual")
		if err != nil {
			slog.Error("failed to create backup", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": record})
	}
}

// ListBackupRecords lists database backup records.
// @Summary      List backup records
// @Description  List database backup history
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Number of records" default(20)
// @Success      200 {object} map[string]interface{}
// @Router       /system/backup/records [get]
func ListBackupRecords(backupSvc *backup.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if backupSvc == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": []interface{}{}})
			return
		}
		limit := 20
		if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && l > 0 && l <= 100 {
			limit = l
		}
		records, err := backupSvc.ListRecords(limit)
		if err != nil {
			slog.Error("failed to list backup records", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": records})
	}
}

// DeleteBackupRecord deletes a backup record and its file.
// @Summary      Delete backup record
// @Description  Delete a backup record and its associated file
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Backup record ID"
// @Success      200 {object} map[string]interface{}
// @Router       /system/backup/records/{id} [delete]
func DeleteBackupRecord(backupSvc *backup.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if backupSvc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "backup service not available"})
			return
		}
		id := c.Param("id")
		if err := backupSvc.DeleteRecord(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "backup deleted", "id": id})
	}
}

// RejectRequest rejects a pending confirmation.
// @Summary      Reject a request
// @Description  Reject a pending confirmation request by ID
// @Tags         System
// @Accept       json
// @Produce      json
// @Security     BearerAuth

// DownloadCloudBackup downloads a backup from cloud storage to local.
// @Summary      Download cloud backup
// @Description  Download a backup from cloud storage to local filesystem
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Backup record ID"
// @Success      200 {object} map[string]interface{}
// @Router       /system/backup/cloud/download/{id} [post]
func DownloadCloudBackup(backupSvc *backup.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if backupSvc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "backup service not available"})
			return
		}
		id := c.Param("id")
		localPath, err := backupSvc.DownloadFromCloud(c.Request.Context(), id)
		if err != nil {
			slog.Error("failed to download cloud backup", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "backup downloaded", "local_path": localPath})
	}
}

// ApplyCloudRetention triggers cloud backup retention cleanup.
// @Summary      Apply cloud retention
// @Description  Manually trigger cloud backup retention policy cleanup
// @Tags         System
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /system/backup/cloud/retention [post]
func ApplyCloudRetention(backupSvc *backup.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if backupSvc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "backup service not available"})
			return
		}
		backupSvc.ApplyCloudRetention(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "cloud retention applied"})
	}
}
// @Param        id path string true "Confirmation ID"
// @Success      200 {object} map[string]interface{}
// @Router       /system/confirmations/{id}/reject [post]
func RejectRequest(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		store := bridge.GetConfirmationStore()
		if store == nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "confirmation system not available"})
			return
		}
		req, err := store.Reject(id, c.GetString("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": req})
	}
}

// GetAuditLogsByTraceID returns audit logs for a specific trace ID.
func GetAuditLogsByTraceID(auditSvc *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.Param("trace_id")
		logs, total, err := auditSvc.ListByTraceID(c.Request.Context(), traceID)
		if err != nil {
			slog.Error("failed to get audit logs by trace ID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "internal server error"})
			return
		}
		if logs == nil {
			logs = []model.AuditLog{}
		}
		respondSuccess(c, gin.H{"trace_id": traceID, "total": total, "logs": logs})
	}
}
