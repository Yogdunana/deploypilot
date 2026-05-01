package model

import "time"

// AlertSilence represents a silence period during which alerts are suppressed.
type AlertSilence struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	TenantID    string    `gorm:"index" json:"tenant_id"`
	Name        string    `gorm:"size:200" json:"name"`
	Reason      string    `gorm:"type:text" json:"reason"`
	Matchers    string    `gorm:"type:text" json:"matchers"` // JSON: {"severity":["warning"],"server_id":"xxx"}
	StartsAt    time.Time `gorm:"index" json:"starts_at"`
	EndsAt      time.Time `gorm:"index" json:"ends_at"`
	CreatedBy   string    `gorm:"size:100" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func (AlertSilence) TableName() string { return "alert_silences" }

// AlertEscalation represents an escalation policy for unresolved alerts.
type AlertEscalation struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	TenantID       string    `gorm:"index" json:"tenant_id"`
	Name           string    `gorm:"size:200" json:"name"`
	RuleIDs        string    `gorm:"type:text" json:"rule_ids"`          // JSON array of rule IDs, empty = all rules
	Steps          string    `gorm:"type:text" json:"steps"`              // JSON: [{"after_minutes":30,"severity":"critical","channels":["telegram","sms"]}]
	RepeatInterval int       `gorm:"default:60" json:"repeat_interval"` // minutes between repeated escalations
	Enabled        bool      `gorm:"default:true" json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (AlertEscalation) TableName() string { return "alert_escalations" }

// AlertGroup represents an alert group for deduplication.
type AlertGroup struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	TenantID      string    `gorm:"index" json:"tenant_id"`
	GroupKey      string    `gorm:"size:200;index" json:"group_key"` // e.g. "server:xxx" or "app:yyy"
	RuleID        string    `gorm:"index" json:"rule_id"`
	Severity      string    `gorm:"size:20" json:"severity"`
	AlertCount    int       `json:"alert_count"`
	FirstAlertAt  time.Time `json:"first_alert_at"`
	LastAlertAt   time.Time `json:"last_alert_at"`
	Status        string    `gorm:"size:20;default:firing" json:"status"` // firing, resolved
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (AlertGroup) TableName() string { return "alert_groups" }
