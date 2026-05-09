package model

import "time"

// OutboundWebhook represents an outbound webhook configuration.
type OutboundWebhook struct {
	ID             string     `gorm:"primaryKey" json:"id"`
	TenantID       string     `gorm:"index" json:"tenant_id,omitempty"`
	Name           string     `gorm:"not null;size:200" json:"name"`
	URL            string     `gorm:"not null;size:500" json:"url"`
	Secret         string     `gorm:"size:500" json:"-"`
	Format         string     `gorm:"size:20;default:json" json:"format"`
	EventTypes     string     `gorm:"type:text" json:"event_types"`
	SeverityFilter string     `gorm:"type:text" json:"severity_filter"`
	AppFilter      string     `gorm:"type:text" json:"app_filter"`
	ServerFilter   string     `gorm:"type:text" json:"server_filter"`
	Enabled        bool       `gorm:"default:true" json:"enabled"`
	MaxRetries     int        `gorm:"default:5" json:"max_retries"`
	Timeout        int        `gorm:"default:10" json:"timeout"`
	Description    string     `json:"description,omitempty"`
	LastDeliveryAt *time.Time `json:"last_delivery_at,omitempty"`
	LastStatus     string     `gorm:"size:20" json:"last_status"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (OutboundWebhook) TableName() string { return "outbound_webhooks" }

// WebhookDelivery records each delivery attempt.
type WebhookDelivery struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	WebhookID     string    `gorm:"index" json:"webhook_id"`
	TenantID      string    `gorm:"index" json:"tenant_id,omitempty"`
	EventID       string    `json:"event_id"`
	EventType     string    `gorm:"size:20;index" json:"event_type"`
	StatusCode    int       `json:"status_code"`
	LatencyMs     int       `json:"latency_ms"`
	Attempt       int       `json:"attempt"`
	Success       bool      `json:"success"`
	ErrorResponse string    `gorm:"type:text" json:"error_response,omitempty"`
	RequestBody   string    `gorm:"type:text" json:"request_body,omitempty"`
	CreatedAt     time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (WebhookDelivery) TableName() string { return "webhook_deliveries" }
