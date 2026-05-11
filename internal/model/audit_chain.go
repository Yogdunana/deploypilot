package model

import "time"

// AuditHash stores the hash chain entry for an audit log record.
// Each entry links to the previous record, forming a tamper-evident chain.
type AuditHash struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AuditID      string    `gorm:"uniqueIndex;not null;size:36" json:"audit_id"`
	Hash         string    `gorm:"size:128;not null" json:"hash"`
	PreviousHash string    `gorm:"size:128" json:"previous_hash"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AuditHash) TableName() string { return "audit_hashes" }
