package model

import "time"

// GrafanaCustomDashboard represents a custom Grafana dashboard stored in the database.
type GrafanaCustomDashboard struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	TenantID    string    `gorm:"index" json:"tenant_id"`
	Name        string    `gorm:"not null" json:"name"`
	UID         string    `gorm:"uniqueIndex;not null" json:"uid"`
	Description string    `json:"description"`
	JSON        string    `gorm:"type:text;not null" json:"json"` // full Grafana dashboard JSON
	Tags        string    `gorm:"type:text" json:"tags"`          // JSON array string
	IsBuiltIn   bool      `gorm:"default:false" json:"is_built_in"`
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (GrafanaCustomDashboard) TableName() string { return "grafana_custom_dashboards" }

// GrafanaSyncLog records Grafana sync operations for auditing.
type GrafanaSyncLog struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	TenantID  string    `gorm:"index" json:"tenant_id"`
	Action    string    `gorm:"not null;size:50" json:"action"` // sync_dashboard, delete_dashboard, create_datasource, etc.
	Status    string    `gorm:"not null;size:20" json:"status"` // success, failed
	Message   string    `gorm:"type:text" json:"message"`
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (GrafanaSyncLog) TableName() string { return "grafana_sync_logs" }
