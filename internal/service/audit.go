package service

import (
	"context"
	"encoding/json"
	"net"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// AuditService handles audit log operations.
type AuditService struct {
	db *gorm.DB
}

// NewAuditService creates a new AuditService.
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
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

// Record creates an audit log entry.
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

	return s.db.WithContext(ctx).Create(log).Error
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
