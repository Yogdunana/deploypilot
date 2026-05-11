package api

import (
	"io"
	"net/http"
	"time"

	"github.com/Yogdunana/deploypilot/internal/audit"
	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuditVerificationAPI holds dependencies for audit verification and compliance endpoints.
type AuditVerificationAPI struct {
	db        *gorm.DB
	auditCfg  *config.AuditConfig
	secretKey []byte
}

// globalAuditVerificationAPI is the package-level AuditVerificationAPI instance.
var globalAuditVerificationAPI *AuditVerificationAPI

// NewAuditVerificationAPI creates a new AuditVerificationAPI.
func NewAuditVerificationAPI(db *gorm.DB, auditCfg *config.AuditConfig, secretKey []byte) *AuditVerificationAPI {
	return &AuditVerificationAPI{
		db:        db,
		auditCfg:  auditCfg,
		secretKey: secretKey,
	}
}

// SetAuditVerificationAPI sets the global audit verification API instance.
func SetAuditVerificationAPI(api *AuditVerificationAPI) {
	globalAuditVerificationAPI = api
}

// VerifyAuditChain verifies the hash chain integrity of all audit records.
// GET /api/v1/audit/verify
func VerifyAuditChain(c *gin.Context) {
	if globalAuditVerificationAPI == nil {
		respondError(c, http.StatusInternalServerError, "audit verification not configured")
		return
	}

	chain := audit.NewAuditChain(globalAuditVerificationAPI.db, globalAuditVerificationAPI.secretKey)
	results, err := chain.VerifyChain()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to verify audit chain")
		return
	}

	validCount := 0
	invalidCount := 0
	var failedIDs []string
	for _, r := range results {
		if r.Valid {
			validCount++
		} else {
			invalidCount++
			failedIDs = append(failedIDs, r.RecordID)
		}
	}

	status := "all records intact"
	if invalidCount > 0 {
		status = "integrity check failed"
	}

	respondSuccess(c, gin.H{
		"total_records": len(results),
		"valid_count":   validCount,
		"invalid_count": invalidCount,
		"failed_ids":    failedIDs,
		"status":        status,
		"results":       results,
	})
}

// ExportAuditLogsV2 exports audit logs in CSV or JSON format with hash verification status.
// GET /api/v1/audit/export?format=csv|json&start=&end=
func ExportAuditLogsV2(c *gin.Context) {
	if globalAuditVerificationAPI == nil {
		respondError(c, http.StatusInternalServerError, "audit export not configured")
		return
	}

	format := c.DefaultQuery("format", "json")
	if format != "csv" && format != "json" {
		format = "json"
	}

	var startTime, endTime time.Time
	if v := c.Query("start"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			startTime = t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			startTime = t
		}
	}
	if v := c.Query("end"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			endTime = t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			endTime = t
		}
	}

	var reader io.Reader
	var err error
	switch format {
	case "csv":
		reader, err = audit.ExportCSV(globalAuditVerificationAPI.db, "", startTime, endTime)
	default:
		reader, err = audit.ExportJSON(globalAuditVerificationAPI.db, "", startTime, endTime)
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to export audit logs")
		return
	}

	contentType := "application/json"
	fileExt := ".json"
	if format == "csv" {
		contentType = "text/csv"
		fileExt = ".csv"
	}

	filename := "audit_export_" + time.Now().Format("20060102_150405") + fileExt
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.DataFromReader(http.StatusOK, -1, contentType, reader, nil)
}

// GDPRExportUserData exports the current user's data for GDPR compliance.
// GET /api/v1/audit/gdpr/export
func GDPRExportUserData(c *gin.Context) {
	if globalAuditVerificationAPI == nil {
		respondError(c, http.StatusInternalServerError, "audit verification not configured")
		return
	}

	userIDVal, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondError(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var userID string
	switch v := userIDVal.(type) {
	case string:
		userID = v
	default:
		respondError(c, http.StatusInternalServerError, "invalid user ID type")
		return
	}

	data, err := audit.ExportUserData(globalAuditVerificationAPI.db, userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to export user data")
		return
	}

	respondSuccess(c, data)
}

// GDPRDeleteUserData requests data deletion for a user (admin only).
// DELETE /api/v1/audit/gdpr/delete?user_id=123
func GDPRDeleteUserData(c *gin.Context) {
	if globalAuditVerificationAPI == nil {
		respondError(c, http.StatusInternalServerError, "audit verification not configured")
		return
	}

	// Check admin role
	roleVal, exists := c.Get("role")
	if !exists {
		respondError(c, http.StatusForbidden, "admin access required")
		return
	}
	role, ok := roleVal.(string)
	if !ok || (role != "owner" && role != "admin") {
		respondError(c, http.StatusForbidden, "admin access required")
		return
	}

	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", "user_id is required")
		return
	}

	if err := audit.DeleteUserData(globalAuditVerificationAPI.db, userIDStr); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to delete user data")
		return
	}

	respondSuccess(c, gin.H{
		"message": "user data anonymized successfully",
		"user_id": userIDStr,
	})
}

// ComplianceReport generates a compliance summary report.
// GET /api/v1/audit/compliance?start=&end=
func ComplianceReport(c *gin.Context) {
	if globalAuditVerificationAPI == nil {
		respondError(c, http.StatusInternalServerError, "audit verification not configured")
		return
	}

	var startTime, endTime time.Time
	if v := c.Query("start"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			startTime = t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			startTime = t
		}
	}
	if v := c.Query("end"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			endTime = t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			endTime = t
		}
	}

	report, err := audit.GenerateComplianceReport(
		globalAuditVerificationAPI.db,
		"", startTime, endTime,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate compliance report")
		return
	}

	report.RetentionDays = 90
	if globalAuditVerificationAPI.auditCfg != nil && globalAuditVerificationAPI.auditCfg.RetentionDays > 0 {
		report.RetentionDays = globalAuditVerificationAPI.auditCfg.RetentionDays
	}

	respondSuccess(c, report)
}
