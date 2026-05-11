package service

import (
	"context"
	"crypto/hmac"
	crypto_rand "crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// Log type constants for audit log classification.
const (
	LogTypeAuth      = "auth"      // Authentication events (login, register, 2FA)
	LogTypeOperation = "operation" // Resource operations (create, update, delete, deploy)
	LogTypeSecurity  = "security"  // Security events (password change, 2FA, IP block)
	LogTypeSystem    = "system"    // System events (config change, backup, cleanup)
)

// AuditService handles audit log operations.
type AuditService struct {
	db             *gorm.DB
	hmacKey        []byte
	externalWriter AuditWriter
	onRecord       []func(AuditEntry) // callbacks invoked after successful record
}

// sensitiveFieldPattern matches field names that may contain sensitive data.
var sensitiveFieldPattern = regexp.MustCompile(`(?i)(password|secret|token|api_key|apikey|private_key|credit_card|ssn|authorization|auth)`)

// sanitizeAuditData recursively sanitizes sensitive fields in audit data.
func sanitizeAuditData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		sanitized := make(map[string]interface{}, len(v))
		for key, val := range v {
			if sensitiveFieldPattern.MatchString(key) {
				sanitized[key] = "[REDACTED]"
			} else {
				sanitized[key] = sanitizeAuditData(val)
			}
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, len(v))
		for i, val := range v {
			sanitized[i] = sanitizeAuditData(val)
		}
		return sanitized
	default:
		return v
	}
}

// OnRecord registers a callback to be invoked after each audit record is created.
// This enables real-time notifications without tight coupling to WebSocket or other systems.
func (s *AuditService) OnRecord(fn func(AuditEntry)) {
	s.onRecord = append(s.onRecord, fn)
}

// NewAuditService creates a new AuditService with a randomly generated HMAC key.
// Optionally accepts an external AuditWriter for writing audit entries to external storage.
func NewAuditService(db *gorm.DB, externalWriter ...AuditWriter) *AuditService {
	key := make([]byte, 32)
	if _, err := crypto_rand.Read(key); err != nil {
		key = make([]byte, 32)
	}
	svc := &AuditService{db: db, hmacKey: key}
	if len(externalWriter) > 0 && externalWriter[0] != nil {
		svc.externalWriter = externalWriter[0]
	}
	return svc
}

// SetHMACKey sets the HMAC key (useful for loading a persistent key from config).
func (s *AuditService) SetHMACKey(key []byte) {
	s.hmacKey = key
}

// Close closes the external audit writer if one is configured.
func (s *AuditService) Close() {
	if s.externalWriter != nil {
		if err := s.externalWriter.Close(); err != nil {
			slog.Error("failed to close external audit writer", "error", err)
		}
	}
}

// AuditEntry represents an audit event to record.
type AuditEntry struct {
	UserID       string
	Username     string
	Action       string       // "app.create", "app.deploy", "server.delete", "user.login", etc.
	ResourceType string       // "app", "server", "user", "credential", etc.
	ResourceID   string
	Detail       interface{}  // will be JSON-marshaled
	LogType      string       // "auth", "operation", "security", "system" (auto-detected if empty)
	IPAddress    string
	UserAgent    string
	TraceID      string
}

// AuditFilter defines filtering options for listing audit logs.
type AuditFilter struct {
	UserID       string
	Username     string
	Action       string
	ResourceType string
	LogType      string // filter by log type: auth, operation, security, system
	TraceID      string
	Archived     *bool  // nil = all, true = archived only, false = active only
	StartTime    time.Time
	EndTime      time.Time
	Page         int
	PageSize     int
}

// classifyAction determines the log type based on the action string.
func classifyAction(action string) string {
	switch {
	case strings.HasPrefix(action, "user.") || strings.HasPrefix(action, "2fa.") ||
		strings.HasPrefix(action, "auth."):
		return LogTypeAuth
	case strings.HasPrefix(action, "security.") || strings.HasPrefix(action, "bruteforce.") ||
		strings.HasPrefix(action, "password.") || action == "apikey.update":
		return LogTypeSecurity
	case strings.HasPrefix(action, "system.") || strings.HasPrefix(action, "config.") ||
		strings.HasPrefix(action, "backup.") || strings.HasPrefix(action, "cleanup."):
		return LogTypeSystem
	default:
		return LogTypeOperation
	}
}

// generateUUID creates a random UUID v4 string.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = crypto_rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// computeRecordHash generates an HMAC-SHA256 hash of the audit log fields.
func (s *AuditService) computeRecordHash(log *model.AuditLog) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		log.UserID,
		log.Username,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		log.Detail,
		log.IPAddress,
		log.UserAgent,
	)
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// Record creates an audit log entry with an integrity hash.
// LogType is auto-detected from the action if not explicitly set.
func (s *AuditService) Record(ctx context.Context, entry AuditEntry) error {
	detail := ""
	if entry.Detail != nil {
		// Sanitize sensitive fields before serialization
		sanitized := sanitizeAuditData(entry.Detail)
		b, err := json.Marshal(sanitized)
		if err == nil {
			detail = string(b)
		}
	}

	// Auto-classify log type if not specified
	logType := entry.LogType
	if logType == "" {
		logType = classifyAction(entry.Action)
	}

	log := &model.AuditLog{
		ID:           generateUUID(),
		UserID:       entry.UserID,
		Username:     entry.Username,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		Detail:       detail,
		LogType:      logType,
		IPAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
		TraceID:      entry.TraceID,
	}

	// Compute HMAC-SHA256 hash for tamper-evident integrity
	log.RecordHash = s.computeRecordHash(log)

	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		return err
	}

	// Invoke onRecord callbacks (non-blocking)
	if len(s.onRecord) > 0 {
		for _, fn := range s.onRecord {
			go fn(entry)
		}
	}

	// Write to external storage (non-blocking, best-effort)
	if s.externalWriter != nil {
		go func() {
			if err := s.externalWriter.Write(entry); err != nil {
				slog.Error("failed to write audit log to external storage", "error", err)
			}
		}()
	}

	return nil
}

// VerifyRecord checks whether an audit log record has been tampered with.
func (s *AuditService) VerifyRecord(log model.AuditLog) error {
	if log.RecordHash == "" {
		return fmt.Errorf("audit log record %s has no integrity hash (pre-integrity record)", log.ID)
	}
	expected := s.computeRecordHash(&log)
	if !hmac.Equal([]byte(expected), []byte(log.RecordHash)) {
		return fmt.Errorf("audit log record %s integrity check failed: hash mismatch (possible tampering)", log.ID)
	}
	return nil
}

// VerifyRecords checks a batch of audit log records for integrity.
func (s *AuditService) VerifyRecords(logs []model.AuditLog) []string {
	var failed []string
	for _, log := range logs {
		if err := s.VerifyRecord(log); err != nil {
			failed = append(failed, log.ID)
		}
	}
	return failed
}

// List returns audit logs with pagination and filtering.
func (s *AuditService) List(ctx context.Context, filter AuditFilter) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	query := s.db.WithContext(ctx).Model(&model.AuditLog{})

	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.LogType != "" {
		query = query.Where("log_type = ?", filter.LogType)
	}
	if filter.TraceID != "" {
		query = query.Where("trace_id = ?", filter.TraceID)
	}
	if filter.Archived != nil {
		query = query.Where("archived = ?", *filter.Archived)
	}
	if filter.Username != "" {
		query = query.Where("username LIKE ?", "%"+filter.Username+"%")
	}
	if !filter.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	if logs == nil {
		logs = []model.AuditLog{}
	}

	return logs, total, nil
}

// ListByTraceID returns audit logs filtered by trace ID.
func (s *AuditService) ListByTraceID(ctx context.Context, traceID string) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	query := s.db.WithContext(ctx).Model(&model.AuditLog{}).Where("trace_id = ?", traceID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	if logs == nil {
		logs = []model.AuditLog{}
	}

	return logs, total, nil
}

// Export returns audit logs as a byte slice in the specified format (csv or json).
func (s *AuditService) Export(ctx context.Context, filter AuditFilter, format string) ([]byte, error) {
	// Override pagination to get all results for export
	filter.Page = 1
	filter.PageSize = 10000 // max export limit

	logs, _, err := s.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case "csv":
		return s.exportCSV(logs)
	case "json":
		return s.exportJSON(logs)
	default:
		return s.exportJSON(logs)
	}
}

func (s *AuditService) exportCSV(logs []model.AuditLog) ([]byte, error) {
	var buf strings.Builder
	w := csv.NewWriter(&buf)

	// Header
	_ = w.Write([]string{"ID", "UserID", "Username", "Action", "ResourceType", "ResourceID", "LogType", "IPAddress", "UserAgent", "TraceID", "Detail", "CreatedAt"})

	for _, log := range logs {
		_ = w.Write([]string{
			log.ID,
			log.UserID,
			log.Username,
			log.Action,
			log.ResourceType,
			log.ResourceID,
			log.LogType,
			log.IPAddress,
			log.UserAgent,
			log.TraceID,
			log.Detail,
			log.CreatedAt.Format(time.RFC3339),
		})
	}

	w.Flush()
	return []byte(buf.String()), nil
}

func (s *AuditService) exportJSON(logs []model.AuditLog) ([]byte, error) {
	return json.MarshalIndent(logs, "", "  ")
}

// Archive soft-deletes (marks as archived) audit logs older than the specified number of days.
// Archived logs are preserved but excluded from default queries.
// Returns the number of archived records.
func (s *AuditService) Archive(ctx context.Context, olderThanDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	now := time.Now()
	result := s.db.WithContext(ctx).
		Model(&model.AuditLog{}).
		Where("created_at < ? AND archived = ?", cutoff, false).
		Updates(map[string]interface{}{
			"archived":    true,
			"archived_at": now,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	slog.Info("audit log archive completed",
		"archived", result.RowsAffected,
		"older_than_days", olderThanDays,
	)
	return result.RowsAffected, nil
}

// Stats returns audit log statistics grouped by log type.
func (s *AuditService) Stats(ctx context.Context) (map[string]int64, error) {
	type countResult struct {
		LogType string
		Count   int64
	}

	var results []countResult
	if err := s.db.WithContext(ctx).
		Model(&model.AuditLog{}).
		Select("log_type, COUNT(*) as count").
		Where("archived = ?", false).
		Group("log_type").
		Find(&results).Error; err != nil {
		return nil, err
	}

	stats := map[string]int64{
		LogTypeAuth:      0,
		LogTypeOperation: 0,
		LogTypeSecurity:  0,
		LogTypeSystem:    0,
	}
	for _, r := range results {
		stats[r.LogType] = r.Count
	}
	return stats, nil
}

// Cleanup deletes archived audit logs older than the specified retention period.
// Only archived records are deleted (append-only protection for active records).
// Returns the number of deleted records.
func (s *AuditService) Cleanup(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := s.db.WithContext(ctx).
		Where("created_at < ? AND archived = ?", cutoff, true).
		Delete(&model.AuditLog{})
	if result.Error != nil {
		return 0, result.Error
	}
	slog.Info("audit log cleanup completed",
		"deleted", result.RowsAffected,
		"retention_days", retentionDays,
		"cutoff", cutoff.Format(time.RFC3339),
	)
	return result.RowsAffected, nil
}

// ClientIP extracts the client IP from request headers.
func ClientIP(remoteAddr, xForwardedFor string) string {
	if xForwardedFor != "" {
		parts := strings.SplitN(xForwardedFor, ",", 2)
		ip := strings.TrimSpace(parts[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
