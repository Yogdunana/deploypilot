package model

import "time"

// TrialStatus represents the status of a trial period.
type TrialStatus string

const (
	TrialActive    TrialStatus = "active"
	TrialExpired   TrialStatus = "expired"
	TrialExtended  TrialStatus = "extended"
	TrialConverted TrialStatus = "converted" // user activated a license
)

// TrialPeriod represents the trial period state for an instance.
type TrialPeriod struct {
	ID             string      `gorm:"primaryKey" json:"id"`
	MachineID      string      `gorm:"uniqueIndex;not null;size:128" json:"machine_id"`
	Status         TrialStatus `gorm:"size:20;not null;default:active" json:"status"`
	StartedAt      time.Time   `gorm:"not null" json:"started_at"`
	ExpiresAt      time.Time   `gorm:"not null" json:"expires_at"`
	ExtendedDays   int         `gorm:"not null;default:0" json:"extended_days"`
	OriginalDays   int         `gorm:"not null;default:30" json:"original_days"`
	LastCheckedAt  time.Time   `json:"last_checked_at"`
	ConvertedAt    *time.Time  `json:"converted_at,omitempty"`
	// UsageStats stores JSON with deployment counts, server counts, etc.
	UsageStats string `gorm:"type:text" json:"usage_stats"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (TrialPeriod) TableName() string { return "trial_periods" }
