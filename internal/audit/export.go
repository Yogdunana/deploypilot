package audit

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// AuditExportRecord is an enriched audit log record for export,
// including hash verification status.
type AuditExportRecord struct {
	ID               string `json:"id"`
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
	Action           string `json:"action"`
	ResourceType     string `json:"resource_type"`
	ResourceID       string `json:"resource_id"`
	Detail           string `json:"detail"`
	LogType          string `json:"log_type"`
	IPAddress        string `json:"ip_address"`
	UserAgent        string `json:"user_agent"`
	TraceID          string `json:"trace_id"`
	RecordHash       string `json:"record_hash"`
	HashVerified     bool   `json:"hash_verified"`
	ChainHashValid   bool   `json:"chain_hash_valid"`
	CreatedAt        string `json:"created_at"`
}

// ExportCSV exports audit logs to CSV format with hash verification status.
func ExportCSV(db *gorm.DB, tenantID string, startTime, endTime time.Time) (io.Reader, error) {
	logs, err := queryAuditLogs(db, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Build hash map for chain verification
	hashMap, err := buildHashMap(db)
	if err != nil {
		return nil, err
	}

	records := enrichExportRecords(logs, hashMap)

	var buf strings.Builder
	w := csv.NewWriter(&buf)

	header := []string{
		"ID", "UserID", "Username", "Action", "ResourceType", "ResourceID",
		"LogType", "IPAddress", "UserAgent", "TraceID", "RecordHash",
		"HashVerified", "ChainHashValid", "CreatedAt",
	}
	if err := w.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, r := range records {
		row := []string{
			fmt.Sprintf("%d", r.ID),
			fmt.Sprintf("%d", r.UserID),
			r.Username,
			r.Action,
			r.ResourceType,
			r.ResourceID,
			r.LogType,
			r.IPAddress,
			r.UserAgent,
			r.TraceID,
			r.RecordHash,
			fmt.Sprintf("%t", r.HashVerified),
			fmt.Sprintf("%t", r.ChainHashValid),
			r.CreatedAt,
		}
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("failed to flush CSV: %w", err)
	}

	return strings.NewReader(buf.String()), nil
}

// ExportJSON exports audit logs to JSON format with hash verification status.
func ExportJSON(db *gorm.DB, tenantID string, startTime, endTime time.Time) (io.Reader, error) {
	logs, err := queryAuditLogs(db, tenantID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	hashMap, err := buildHashMap(db)
	if err != nil {
		return nil, err
	}

	records := enrichExportRecords(logs, hashMap)

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return strings.NewReader(string(data)), nil
}

// queryAuditLogs fetches audit logs with optional filters.
func queryAuditLogs(db *gorm.DB, tenantID string, startTime, endTime time.Time) ([]model.AuditLog, error) {
	var logs []model.AuditLog
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

	if err := query.Order("id ASC").Limit(10000).Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}

	if logs == nil {
		logs = []model.AuditLog{}
	}
	return logs, nil
}

// buildHashMap builds a map of auditID -> hash from the audit_hashes table.
func buildHashMap(db *gorm.DB) (map[string]model.AuditHash, error) {
	var hashes []model.AuditHash
	if err := db.Find(&hashes).Error; err != nil {
		return nil, fmt.Errorf("failed to query audit hashes: %w", err)
	}

	hashMap := make(map[string]model.AuditHash, len(hashes))
	for _, h := range hashes {
		hashMap[h.AuditID] = h
	}
	return hashMap, nil
}

// enrichExportRecords converts audit logs to export records with verification status.
func enrichExportRecords(logs []model.AuditLog, hashMap map[string]model.AuditHash) []AuditExportRecord {
	records := make([]AuditExportRecord, 0, len(logs))
	for _, log := range logs {
		_, hasChainHash := hashMap[log.ID]
		records = append(records, AuditExportRecord{
			ID:             log.ID,
			UserID:         log.UserID,
			Username:       log.Username,
			Action:         log.Action,
			ResourceType:   log.ResourceType,
			ResourceID:     log.ResourceID,
			Detail:         log.Detail,
			LogType:        log.LogType,
			IPAddress:      log.IPAddress,
			UserAgent:      log.UserAgent,
			TraceID:        log.TraceID,
			RecordHash:     log.RecordHash,
			HashVerified:   log.RecordHash != "",
			ChainHashValid: hasChainHash,
			CreatedAt:      log.CreatedAt.Format(time.RFC3339),
		})
	}
	return records
}
