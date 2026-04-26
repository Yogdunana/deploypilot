package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// AuditService handles audit log operations.
type AuditService struct {
	db      *gorm.DB
	hmacKey []byte
}

// NewAuditService creates a new AuditService with a randomly generated HMAC key.
func NewAuditService(db *gorm.DB) *AuditService {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		// Fallback: use a zeroed key (should never happen with crypto/rand on modern systems)
		key = make([]byte, 32)
	}
	return &AuditService{db: db, hmacKey: key}
}

// SetHMACKey sets the HMAC key (useful for loading a persistent key from config).
func (s *AuditService) SetHMACKey(key []byte) {
	s.hmacKey = key
}

// AuditEntry represents an audit event to record.
type AuditEntry struct {
	UserID       uint
	Username     string
	Action       string // "app.create", "app.deploy", "server.delete", "user.login", etc.
	ResourceType string // "app", "server", "user", "credential", etc.
	ResourceID   string
	Detail       interface{} // will be JSON-marshaled
	IPAddress    string
	UserAgent    string
}

// AuditFilter defines filtering options for listing audit logs.
type AuditFilter struct {
	UserID       uint
	Action       string
	ResourceType string
	Page         int
	PageSize     int
}

// computeRecordHash generates an HMAC-SHA256 hash of the audit log fields.
func (s *AuditService) computeRecordHash(log *model.AuditLog) string {
	data := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%s",
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
func (s *AuditService) Record(ctx context.Context, entry AuditEntry) error {
	detail := ""
	if entry.Detail != nil {
		b, err := json.Marshal(entry.Detail)
		if err == nil {
			detail = string(b)
		}
	}

	log := &model.AuditLog{
		UserID:       entry.UserID,
		Username:     entry.Username,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		Detail:       detail,
		IPAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
	}

	// Compute HMAC-SHA256 hash for tamper-evident integrity
	log.RecordHash = s.computeRecordHash(log)

	return s.db.WithContext(ctx).Create(log).Error
}

// VerifyRecord checks whether an audit log record has been tampered with
// by recomputing the HMAC-SHA256 hash and comparing it with the stored hash.
// Returns nil if the record is intact, or an error describing the integrity failure.
func (s *AuditService) VerifyRecord(log model.AuditLog) error {
	if log.RecordHash == "" {
		return fmt.Errorf("audit log record %d has no integrity hash (pre-integrity record)", log.ID)
	}
	expected := s.computeRecordHash(&log)
	if !hmac.Equal([]byte(expected), []byte(log.RecordHash)) {
		return fmt.Errorf("audit log record %d integrity check failed: hash mismatch (possible tampering)", log.ID)
	}
	return nil
}

// VerifyRecords checks a batch of audit log records for integrity.
// Returns a slice of record IDs that failed verification, or nil if all are intact.
func (s *AuditService) VerifyRecords(logs []model.AuditLog) []uint {
	var failed []uint
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

	if filter.UserID != 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
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

// ClientIP extracts the client IP from request headers.
func ClientIP(remoteAddr, xForwardedFor string) string {
	if xForwardedFor != "" {
		parts := strings.SplitN(xForwardedFor, ",", 2)
		ip := strings.TrimSpace(parts[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	// Strip port from remoteAddr
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
