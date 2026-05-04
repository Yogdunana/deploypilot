package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/license"
	"github.com/Yogdunana/deploypilot/internal/model"
)

func (b *Bridge) ActivateLicense(ctx context.Context, licenseKey string, useType string, agreeTerms bool) (interface{}, error) {
	if licenseKey != "" {
		return b.activateWithKey(ctx, licenseKey)
	}
	if useType == "non_commercial" && agreeTerms {
		return b.activateCommunity(ctx)
	}
	return nil, fmt.Errorf("must provide either a license_key or agree to non-commercial terms")
}

// activateWithKey validates and activates a license key.
func (b *Bridge) activateWithKey(ctx context.Context, licenseKey string) (interface{}, error) {
	if b.LicenseEngine == nil {
		return nil, fmt.Errorf("license engine not initialized")
	}

	// Validate and load the license key into the engine
	if err := b.LicenseEngine.LoadLicense(licenseKey); err != nil {
		return nil, fmt.Errorf("failed to load license: %w", err)
	}

	// Validate the license
	if err := b.LicenseEngine.Validate(); err != nil {
		return nil, fmt.Errorf("license validation failed: %w", err)
	}

	info := b.LicenseEngine.GetInfo()
	if info == nil {
		return nil, fmt.Errorf("license info not available after loading")
	}

	// Deactivate any existing active license
	if err := b.deactivateExistingLicense(ctx); err != nil {
		slog.Warn("failed to deactivate existing license", "error", err)
	}

	// Serialize addons to JSON
	addonsJSON, err := json.Marshal(info.Data.Addons)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize addons: %w", err)
	}

	// Build features list
	features := make([]string, 0, len(info.Features))
	for f := range info.Features {
		features = append(features, string(f))
	}
	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize features: %w", err)
	}

	// Build limits JSON
	limitsJSON, err := json.Marshal(map[string]int{
		"max_servers": info.Data.MaxServers,
		"max_apps":    info.Data.MaxApps,
		"max_users":   info.Data.MaxUsers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize limits: %w", err)
	}

	now := time.Now()
	graceDays := 0
	if !info.ValidTo.IsZero() {
		graceDays = int(info.ValidTo.Sub(now).Hours() / 24)
		if graceDays < 0 {
			graceDays = 0
		}
	}

	lic := model.License{
		ID:          fmt.Sprintf("lic-%d", now.UnixNano()),
		TenantID:    info.Data.TenantID,
		LicenseKey:  licenseKey,
		Tier:        model.Tier(info.Data.Tier),
		UseType:     model.UseType(info.Data.UseType),
		Status:      model.LicenseStatusActive,
		Features:    string(featuresJSON),
		Limits:      string(limitsJSON),
		MaxServers:  info.Data.MaxServers,
		MaxApps:     info.Data.MaxApps,
		MaxUsers:    info.Data.MaxUsers,
		IssuerRole:  info.Data.IssuerRole,
		IssuedTo:    info.Data.IssuedTo,
		MaxIssued:   info.Data.MaxIssued,
		IssuedCount: info.Data.IssuedCount,
		Addons:      string(addonsJSON),
		GraceDays:   graceDays,
		MachineID:   info.Data.MachineID,
		ActivatedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if info.Data.ExpiresAt > 0 {
		exp := time.Unix(info.Data.ExpiresAt, 0)
		lic.ExpiresAt = &exp
	}

	if err := b.DB.Create(&lic).Error; err != nil {
		return nil, fmt.Errorf("failed to save license to database: %w", err)
	}

	slog.Info("license activated", "id", lic.ID, "tier", lic.Tier, "use_type", lic.UseType)

	// Convert trial period to licensed status
	_ = b.ConvertTrial(ctx)

	return map[string]interface{}{
		"id":         lic.ID,
		"tier":       lic.Tier,
		"use_type":   lic.UseType,
		"status":     lic.Status,
		"expires_at": lic.ExpiresAt,
		"features":   features,
		"limits": map[string]int{
			"max_servers": lic.MaxServers,
			"max_apps":    lic.MaxApps,
			"max_users":   lic.MaxUsers,
		},
	}, nil
}

// activateCommunity creates a community (non-commercial) license.
func (b *Bridge) activateCommunity(ctx context.Context) (interface{}, error) {
	// Deactivate any existing active license
	if err := b.deactivateExistingLicense(ctx); err != nil {
		slog.Warn("failed to deactivate existing license", "error", err)
	}

	features := make([]string, 0, len(license.AllFeatures))
	for _, f := range license.AllFeatures {
		features = append(features, string(f))
	}
	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize features: %w", err)
	}

	limits := license.TierLimits[license.TierCommunity]
	limitsJSON, err := json.Marshal(map[string]int{
		"max_servers": limits.MaxServers,
		"max_apps":    limits.MaxApps,
		"max_users":   limits.MaxUsers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize limits: %w", err)
	}

	now := time.Now()
	lic := model.License{
		ID:          fmt.Sprintf("lic-%d", now.UnixNano()),
		TenantID:    "default",
		Tier:        model.TierCommunity,
		UseType:     model.UseTypeNonCommercial,
		Status:      model.LicenseStatusActive,
		Features:    string(featuresJSON),
		Limits:      string(limitsJSON),
		MaxServers:  limits.MaxServers,
		MaxApps:     limits.MaxApps,
		MaxUsers:    limits.MaxUsers,
		IssuerRole:  "system",
		GraceDays:   7,
		ActivatedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := b.DB.Create(&lic).Error; err != nil {
		return nil, fmt.Errorf("failed to save community license: %w", err)
	}

	// Load into engine if available
	if b.LicenseEngine != nil && b.LicensePrivKey != nil {
		communityKey, err := license.GenerateLicenseKey(
			b.LicensePrivKey,
			license.LicenseData{
				TenantID:   "default",
				UseType:    string(license.UseTypeNonCommercial),
				Tier:       string(license.TierCommunity),
				IssuerRole: "system",
				MaxServers: limits.MaxServers,
				MaxApps:    limits.MaxApps,
				MaxUsers:   limits.MaxUsers,
				IssuedAt:   now.Unix(),
			},
		)
		if err != nil {
			slog.Warn("failed to generate community license key for engine", "error", err)
		} else {
			if loadErr := b.LicenseEngine.LoadLicense(communityKey); loadErr != nil {
				slog.Warn("failed to load community license into engine", "error", loadErr)
			} else {
				// Update the license key in DB
				b.DB.Model(&lic).Update("license_key", communityKey)
			}
		}
	}

	slog.Info("community license activated", "id", lic.ID)

	return map[string]interface{}{
		"id":       lic.ID,
		"tier":     lic.Tier,
		"use_type": lic.UseType,
		"status":   lic.Status,
		"features": features,
		"limits": map[string]int{
			"max_servers": lic.MaxServers,
			"max_apps":    lic.MaxApps,
			"max_users":   lic.MaxUsers,
		},
	}, nil
}

// deactivateExistingLicense marks all active licenses as expired.
func (b *Bridge) deactivateExistingLicense(ctx context.Context) error {
	return b.DB.Model(&model.License{}).
		Where("status = ?", model.LicenseStatusActive).
		Update("status", model.LicenseStatusExpired).Error
}

// DeactivateLicense deactivates the current license.
func (b *Bridge) DeactivateLicense(ctx context.Context) (interface{}, error) {
	result := b.DB.Model(&model.License{}).
		Where("status = ?", model.LicenseStatusActive).
		Update("status", model.LicenseStatusExpired)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to deactivate license: %w", result.Error)
	}

	// Clear engine
	if b.LicenseEngine != nil {
		// The engine does not have an unload method, so we note this.
		// The license will be re-evaluated on next startup.
		slog.Info("license deactivated (engine will be cleared on restart)")
	}

	return map[string]interface{}{
		"message":       "license deactivated",
		"rows_affected": result.RowsAffected,
	}, nil
}

// GetLicenseStatus returns current license info from the engine or DB.
