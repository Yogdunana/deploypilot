package audit

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// ComplianceReport represents a summary of compliance status for a tenant.
type ComplianceReport struct {
	TenantID          string            `json:"tenant_id"`
	PeriodStart       time.Time         `json:"period_start"`
	PeriodEnd         time.Time         `json:"period_end"`
	TotalRecords      int64             `json:"total_records"`
	RecordsByType     map[string]int64  `json:"records_by_type"`
	UniqueUsers       int64             `json:"unique_users"`
	ChainVerified     bool              `json:"chain_verified"`
	ChainIssues       int               `json:"chain_issues"`
	RetentionDays     int               `json:"retention_days"`
	RecordsExpiring   int64             `json:"records_expiring"`
	GeneratedAt       time.Time         `json:"generated_at"`
}

// ExportUserData collects all user-related data for GDPR data export.
// It returns a map containing audit logs, API keys, sessions, and other user data.
func ExportUserData(db *gorm.DB, userID string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["user_id"] = userID
	result["exported_at"] = time.Now().UTC().Format(time.RFC3339)

	// Collect audit logs
	var auditLogs []model.AuditLog
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Limit(10000).Find(&auditLogs).Error; err != nil {
		return nil, fmt.Errorf("failed to query user audit logs: %w", err)
	}
	if auditLogs == nil {
		auditLogs = []model.AuditLog{}
	}
	result["audit_logs"] = auditLogs
	result["audit_log_count"] = len(auditLogs)

	// Collect API keys (anonymized - only metadata, not the actual key)
	var apiKeys []model.APIKey
	if err := db.Where("user_id = ?", userID).Find(&apiKeys).Error; err != nil {
		// API keys table may not have user_id column; skip if error
		slog.Debug("failed to query user API keys", "error", err)
	} else {
		// Anonymize key values
		type safeAPIKey struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			LastUsedAt  string `json:"last_used_at,omitempty"`
			ExpiresAt   string `json:"expires_at,omitempty"`
			CreatedAt   string `json:"created_at"`
		}
		safeKeys := make([]safeAPIKey, 0, len(apiKeys))
		for _, k := range apiKeys {
			sk := safeAPIKey{
				ID:        k.ID,
				Name:      k.Name,
				CreatedAt: k.CreatedAt.Format(time.RFC3339),
			}
			if k.LastUsedAt != nil {
				sk.LastUsedAt = k.LastUsedAt.Format(time.RFC3339)
			}
			if k.ExpiresAt != nil {
				sk.ExpiresAt = k.ExpiresAt.Format(time.RFC3339)
			}
			safeKeys = append(safeKeys, sk)
		}
		result["api_keys"] = safeKeys
	}

	return result, nil
}

// DeleteUserData performs GDPR right-to-be-forgotten by anonymizing user data.
// Audit logs are preserved but anonymized (username, IP, user agent are cleared).
func DeleteUserData(db *gorm.DB, userID string) error {
	// Anonymize audit logs: keep the record but remove personal identifiers
	result := db.Model(&model.AuditLog{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"username":  "[GDPR-DELETED]",
			"ip_address": "[GDPR-DELETED]",
			"user_agent": "[GDPR-DELETED]",
			"user_id":    "",
		})
	if result.Error != nil {
		return fmt.Errorf("failed to anonymize audit logs: %w", result.Error)
	}

	slog.Info("GDPR data deletion completed",
		"user_id", userID,
		"anonymized_audit_logs", result.RowsAffected,
	)

	return nil
}

// DataRetentionPolicy deletes audit records older than the configured retention period.
// It only deletes archived records to maintain append-only protection for active records.
func DataRetentionPolicy(db *gorm.DB, cfg *config.AuditConfig) error {
	retentionDays := cfg.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 90 // default retention
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// Only delete archived records beyond retention period
	result := db.Where("created_at < ? AND archived = ?", cutoff, true).
		Delete(&model.AuditLog{})
	if result.Error != nil {
		return fmt.Errorf("failed to apply data retention policy: %w", result.Error)
	}

	// Also clean up associated hash entries for deleted audit logs
	var deletedAuditIDs []string
	if err := db.Model(&model.AuditLog{}).
		Select("id").
		Where("created_at < ?", cutoff).
		Find(&deletedAuditIDs).Error; err == nil && len(deletedAuditIDs) > 0 {
		db.Where("audit_id IN ?", deletedAuditIDs).Delete(&model.AuditHash{})
	}

	slog.Info("data retention policy applied",
		"deleted_records", result.RowsAffected,
		"retention_days", retentionDays,
		"cutoff", cutoff.Format(time.RFC3339),
	)

	return nil
}

// GenerateComplianceReport generates a compliance summary report for a tenant.
func GenerateComplianceReport(db *gorm.DB, tenantID string, startTime, endTime time.Time) (ComplianceReport, error) {
	report := ComplianceReport{
		TenantID:    tenantID,
		PeriodStart: startTime,
		PeriodEnd:   endTime,
		GeneratedAt: time.Now().UTC(),
		RecordsByType: map[string]int64{
			"auth":      0,
			"operation": 0,
			"security":  0,
			"system":    0,
			"other":     0,
		},
	}

	query := db.Model(&model.AuditLog{})
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	// Total records
	if err := query.Count(&report.TotalRecords).Error; err != nil {
		return report, fmt.Errorf("failed to count total records: %w", err)
	}

	// Records by type
	type typeCount struct {
		LogType string
		Count   int64
	}
	var typeCounts []typeCount
	typeQuery := db.Model(&model.AuditLog{})
	if tenantID != "" {
		typeQuery = typeQuery.Where("tenant_id = ?", tenantID)
	}
	if !startTime.IsZero() {
		typeQuery = typeQuery.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		typeQuery = typeQuery.Where("created_at <= ?", endTime)
	}
	if err := typeQuery.Select("log_type, COUNT(*) as count").
		Group("log_type").
		Find(&typeCounts).Error; err != nil {
		return report, fmt.Errorf("failed to count records by type: %w", err)
	}
	for _, tc := range typeCounts {
		if _, ok := report.RecordsByType[tc.LogType]; ok {
			report.RecordsByType[tc.LogType] = tc.Count
		} else {
			report.RecordsByType["other"] += tc.Count
		}
	}

	// Unique users
	userQuery := db.Model(&model.AuditLog{}).Where("user_id != ''")
	if tenantID != "" {
		userQuery = userQuery.Where("tenant_id = ?", tenantID)
	}
	if !startTime.IsZero() {
		userQuery = userQuery.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		userQuery = userQuery.Where("created_at <= ?", endTime)
	}
	if err := userQuery.Distinct("user_id").Count(&report.UniqueUsers).Error; err != nil {
		// Non-critical, continue
		slog.Debug("failed to count unique users", "error", err)
	}

	// Chain verification
	chain := NewAuditChain(db, []byte("compliance-check"))
	results, err := chain.VerifyChain()
	if err != nil {
		slog.Debug("failed to verify chain for compliance report", "error", err)
		report.ChainVerified = false
	} else {
		report.ChainVerified = true
		for _, r := range results {
			if !r.Valid {
				report.ChainIssues++
			}
		}
	}

	// Records expiring (within 7 days of retention)
	expiringCutoff := time.Now().AddDate(0, 0, 7)
	expiringQuery := db.Model(&model.AuditLog{}).Where("created_at < ?", expiringCutoff)
	if tenantID != "" {
		expiringQuery = expiringQuery.Where("tenant_id = ?", tenantID)
	}
	if err := expiringQuery.Count(&report.RecordsExpiring).Error; err != nil {
		slog.Debug("failed to count expiring records", "error", err)
	}

	return report, nil
}

// MarshalComplianceReportJSON returns the compliance report as JSON bytes.
func MarshalComplianceReportJSON(report ComplianceReport) ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal compliance report: %w", err)
	}
	return data, nil
}
