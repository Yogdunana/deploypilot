package model

import "time"

// DegradationAction represents the type of degraded action.
type DegradationAction string

const (
	DegradationActionFeatureGated DegradationAction = "feature_gated"
	DegradationActionReadOnly     DegradationAction = "read_only_blocked"
	DegradationActionTierDowngrade DegradationAction = "tier_downgrade"
	DegradationActionExport       DegradationAction = "data_export"
)

// DegradationAudit represents an audit log entry for a degradation event.
type DegradationAudit struct {
	ID        string            `gorm:"primaryKey" json:"id"`
	Action    DegradationAction `gorm:"size:30;not null;index" json:"action"`
	Feature   string            `gorm:"size:100;not null;index" json:"feature"` // feature key or endpoint
	Reason    string            `gorm:"size:200;not null" json:"reason"`
	TenantID  string            `gorm:"size:100;index" json:"tenant_id"`
	UserID    string            `gorm:"size:100" json:"user_id"`
	IPAddress string            `gorm:"size:45" json:"ip_address"`
	CreatedAt time.Time         `gorm:"autoCreateTime;index" json:"created_at"`
}

func (DegradationAudit) TableName() string { return "degradation_audits" }
