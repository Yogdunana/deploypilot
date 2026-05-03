package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

const (
	// DefaultTrialDays is the number of days for the trial period.
	DefaultTrialDays = 30
)

// GenerateMachineID creates a machine fingerprint from hostname + OS + CPU count.
// This is a simple fingerprint — not cryptographically secure, but sufficient
// to prevent casual trial resets.
func GenerateMachineID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	os := runtime.GOOS
	arch := runtime.GOARCH
	cpu := runtime.NumCPU()

	data := fmt.Sprintf("%s|%s|%s|%d", hostname, os, arch, cpu)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // 32-char hex string
}

// InitTrialPeriod creates or activates the trial period for this machine.
// It should be called once at startup.
func (b *Bridge) InitTrialPeriod(ctx context.Context) error {
	machineID := GenerateMachineID()

	var trial model.TrialPeriod
	result := b.DB.Where("machine_id = ?", machineID).First(&trial)

	if result.Error == gorm.ErrRecordNotFound {
		// First run: create trial
		now := time.Now()
		trial = model.TrialPeriod{
			ID:           fmt.Sprintf("trial-%d", now.UnixNano()),
			MachineID:    machineID,
			Status:       model.TrialActive,
			StartedAt:    now,
			ExpiresAt:    now.AddDate(0, 0, DefaultTrialDays),
			OriginalDays: DefaultTrialDays,
			LastCheckedAt: now,
		}
		if err := b.DB.Create(&trial).Error; err != nil {
			return fmt.Errorf("failed to create trial period: %w", err)
		}
		slog.Info("trial period activated", "machine_id", machineID, "expires", trial.ExpiresAt.Format(time.RFC3339))
		return nil
	}

	if result.Error != nil {
		return fmt.Errorf("failed to check trial period: %w", result.Error)
	}

	// Trial exists: check if expired
	now := time.Now()
	if trial.Status == model.TrialActive && now.After(trial.ExpiresAt) {
		trial.Status = model.TrialExpired
		b.DB.Save(&trial)
		slog.Info("trial period expired", "machine_id", machineID, "expired_at", trial.ExpiresAt.Format(time.RFC3339))
	}

	// Update last checked time
	b.DB.Model(&trial).Update("last_checked_at", now)
	return nil
}

// GetTrialStatus returns the current trial period status.
func (b *Bridge) GetTrialStatus(ctx context.Context) (interface{}, error) {
	machineID := GenerateMachineID()

	var trial model.TrialPeriod
	if err := b.DB.Where("machine_id = ?", machineID).First(&trial).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return map[string]interface{}{
				"status":        "none",
				"machine_id":    machineID,
				"days_remaining": 0,
				"is_active":     false,
				"is_expired":    false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get trial status: %w", err)
	}

	now := time.Now()
	daysRemaining := 0
	if now.Before(trial.ExpiresAt) {
		daysRemaining = int(trial.ExpiresAt.Sub(now).Hours() / 24)
	}

	return map[string]interface{}{
		"id":             trial.ID,
		"status":         trial.Status,
		"machine_id":     trial.MachineID,
		"started_at":     trial.StartedAt,
		"expires_at":     trial.ExpiresAt,
		"days_remaining": daysRemaining,
		"extended_days":  trial.ExtendedDays,
		"original_days":  trial.OriginalDays,
		"is_active":      trial.Status == model.TrialActive,
		"is_expired":     trial.Status == model.TrialExpired,
		"is_converted":   trial.Status == model.TrialConverted,
		"usage_stats":    trial.UsageStats,
	}, nil
}

// ExtendTrial extends the trial period by the given number of days (admin only).
func (b *Bridge) ExtendTrial(ctx context.Context, machineID string, days int, reason string) (interface{}, error) {
	if days <= 0 || days > 365 {
		return nil, fmt.Errorf("extension days must be between 1 and 365")
	}

	var trial model.TrialPeriod
	if err := b.DB.Where("machine_id = ?", machineID).First(&trial).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no trial period found for machine '%s'", machineID)
		}
		return nil, fmt.Errorf("failed to find trial period: %w", err)
	}

	if trial.Status == model.TrialConverted {
		return nil, fmt.Errorf("cannot extend a converted trial (license already activated)")
	}

	newExpiry := trial.ExpiresAt.AddDate(0, 0, days)
	if err := b.DB.Model(&trial).Updates(map[string]interface{}{
		"status":        model.TrialExtended,
		"expires_at":    newExpiry,
		"extended_days": gorm.Expr("extended_days + ?", days),
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to extend trial: %w", err)
	}

	slog.Info("trial period extended", "machine_id", machineID, "days", days, "reason", reason, "new_expiry", newExpiry.Format(time.RFC3339))

	return map[string]interface{}{
		"machine_id":    machineID,
		"status":        model.TrialExtended,
		"expires_at":    newExpiry,
		"extended_days": trial.ExtendedDays + days,
	}, nil
}

// ConvertTrial marks the trial as converted (user activated a license).
// This is called when a license is successfully activated.
func (b *Bridge) ConvertTrial(ctx context.Context) error {
	machineID := GenerateMachineID()

	result := b.DB.Model(&model.TrialPeriod{}).Where("machine_id = ? AND status != ?", machineID, model.TrialConverted).
		Updates(map[string]interface{}{
			"status":       model.TrialConverted,
			"converted_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to convert trial: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		slog.Info("trial period converted to license", "machine_id", machineID)
	}
	return nil
}

// IsTrialExpired checks if the trial period has expired and no license is active.
// Returns true if the trial is expired AND no license is loaded.
func (b *Bridge) IsTrialExpired(ctx context.Context) bool {
	// If a license is active, trial expiry doesn't matter
	if b.LicenseEngine != nil {
		if err := b.LicenseEngine.Validate(); err == nil {
			return false
		}
	}

	machineID := GenerateMachineID()
	var trial model.TrialPeriod
	if err := b.DB.Where("machine_id = ?", machineID).First(&trial).Error; err != nil {
		// No trial record = not expired (community mode)
		return false
	}

	return trial.Status == model.TrialExpired
}

// IsTrialActive checks if the trial is currently active.
func (b *Bridge) IsTrialActive(ctx context.Context) bool {
	// If license is active, trial doesn't apply
	if b.LicenseEngine != nil {
		if err := b.LicenseEngine.Validate(); err == nil {
			return false
		}
	}

	machineID := GenerateMachineID()
	var trial model.TrialPeriod
	if err := b.DB.Where("machine_id = ?", machineID).First(&trial).Error; err != nil {
		return false
	}

	return trial.Status == model.TrialActive
}

// CheckTrialOrLicense verifies that either a valid license or active trial exists.
// Returns an error if both are expired/missing.
func (b *Bridge) CheckTrialOrLicense(ctx context.Context) error {
	// 1. Check license first
	if b.LicenseEngine != nil {
		if err := b.LicenseEngine.Validate(); err == nil {
			return nil
		}
	}

	// 2. Check trial
	machineID := GenerateMachineID()
	var trial model.TrialPeriod
	if err := b.DB.Where("machine_id = ?", machineID).First(&trial).Error; err != nil {
		// No trial, no license = community mode (allowed)
		return nil
	}

	switch trial.Status {
	case model.TrialActive:
		return nil
	case model.TrialConverted:
		return nil
	case model.TrialExpired:
		return fmt.Errorf("trial period expired on %s. Please activate a license to continue using all features", trial.ExpiresAt.Format("2006-01-02"))
	case model.TrialExtended:
		if time.Now().Before(trial.ExpiresAt) {
			return nil
		}
		return fmt.Errorf("extended trial period expired on %s. Please activate a license to continue", trial.ExpiresAt.Format("2006-01-02"))
	}

	return nil
}

// ListTrialPeriods returns all trial periods (admin only).
func (b *Bridge) ListTrialPeriods(ctx context.Context) (interface{}, error) {
	var trials []model.TrialPeriod
	if err := b.DB.Order("created_at DESC").Find(&trials).Error; err != nil {
		return nil, fmt.Errorf("failed to list trial periods: %w", err)
	}

	now := time.Now()
	result := make([]map[string]interface{}, len(trials))
	for i, t := range trials {
		daysRemaining := 0
		if now.Before(t.ExpiresAt) {
			daysRemaining = int(t.ExpiresAt.Sub(now).Hours() / 24)
		}
		result[i] = map[string]interface{}{
			"id":             t.ID,
			"machine_id":     t.MachineID,
			"status":         t.Status,
			"started_at":     t.StartedAt,
			"expires_at":     t.ExpiresAt,
			"days_remaining": daysRemaining,
			"extended_days":  t.ExtendedDays,
			"original_days":  t.OriginalDays,
			"converted_at":   t.ConvertedAt,
			"last_checked_at": t.LastCheckedAt,
		}
	}
	return map[string]interface{}{
		"trials": result,
		"total":  len(result),
	}, nil
}
