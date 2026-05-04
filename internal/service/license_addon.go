package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/license"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// PurchaseAddon purchases an addon for the current license.
func (b *Bridge) PurchaseAddon(ctx context.Context, addonKey string, amount int, durationDays int) (interface{}, error) {
	if addonKey == "" {
		return nil, fmt.Errorf("addon_key is required")
	}

	// Find active license
	var lic model.License
	if err := b.DB.Where("status = ?", model.LicenseStatusActive).First(&lic).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no active license found")
		}
		return nil, fmt.Errorf("failed to find active license: %w", err)
	}

	// Parse existing addons
	var addons []license.Addon
	if lic.Addons != "" {
		if err := json.Unmarshal([]byte(lic.Addons), &addons); err != nil {
			return nil, fmt.Errorf("failed to parse existing addons: %w", err)
		}
	}

	// Create new addon
	now := time.Now()
	newAddon := license.Addon{
		Key:         addonKey,
		Amount:      amount,
		PurchasedAt: now.Unix(),
	}
	if durationDays > 0 {
		newAddon.ExpiresAt = now.Add(time.Duration(durationDays) * 24 * time.Hour).Unix()
	}

	addons = append(addons, newAddon)

	// Serialize and update
	addonsJSON, err := json.Marshal(addons)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize addons: %w", err)
	}

	if err := b.DB.Model(&lic).Update("addons", string(addonsJSON)).Error; err != nil {
		return nil, fmt.Errorf("failed to update license addons: %w", err)
	}

	// Reload into engine if available
	if b.LicenseEngine != nil && lic.LicenseKey != "" {
		if err := b.LicenseEngine.LoadLicense(lic.LicenseKey); err != nil {
			slog.Warn("failed to reload license into engine after addon purchase", "error", err)
		}
	}

	slog.Info("addon purchased", "license_id", lic.ID, "addon_key", addonKey, "amount", amount)

	return map[string]interface{}{
		"message":    "addon purchased",
		"license_id": lic.ID,
		"addon":      newAddon,
	}, nil
}
