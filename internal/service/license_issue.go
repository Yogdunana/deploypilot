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

func (b *Bridge) IssueLicense(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if b.LicensePrivKey == nil {
		return nil, fmt.Errorf("license private key not configured (developer mode required)")
	}

	tenantID, _ := params["tenant_id"].(string)
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	tierStr, _ := params["tier"].(string)
	if tierStr == "" {
		tierStr = string(license.TierCommunity)
	}

	useTypeStr, _ := params["use_type"].(string)
	if useTypeStr == "" {
		useTypeStr = string(license.UseTypeNonCommercial)
	}

	issuerRole, _ := params["issuer_role"].(string)
	if issuerRole == "" {
		issuerRole = "developer"
	}

	issuedTo, _ := params["issued_to"].(string)
	maxIssued := toInt(params["max_issued"])

	// Parse expiration
	var expiresAt int64
	if expStr, _ := params["expires_at"].(string); expStr != "" {
		t, err := time.Parse(time.RFC3339, expStr)
		if err != nil {
			return nil, fmt.Errorf("invalid expires_at format (use RFC3339): %w", err)
		}
		expiresAt = t.Unix()
	}

	// Parse addon keys
	var addons []license.Addon
	if addonKeys, ok := params["addons"].([]interface{}); ok {
		for _, ak := range addonKeys {
			if keyStr, ok := ak.(string); ok {
				addons = append(addons, license.Addon{
					Key:         keyStr,
					Amount:      0,
					PurchasedAt: time.Now().Unix(),
				})
			}
		}
	}

	// Get tier limits
	limits, ok := license.TierLimits[license.Tier(tierStr)]
	if !ok {
		limits = license.TierLimits[license.TierCommunity]
	}

	// Override limits if specified
	maxServers := limits.MaxServers
	maxApps := limits.MaxApps
	maxUsers := limits.MaxUsers
	if v := toInt(params["max_servers"]); v > 0 {
		maxServers = v
	}
	if v := toInt(params["max_apps"]); v > 0 {
		maxApps = v
	}
	if v := toInt(params["max_users"]); v > 0 {
		maxUsers = v
	}

	data := license.LicenseData{
		TenantID:   tenantID,
		UseType:    useTypeStr,
		Tier:       tierStr,
		IssuerRole: issuerRole,
		IssuedTo:   issuedTo,
		MaxIssued:  maxIssued,
		Addons:     addons,
		MaxServers: maxServers,
		MaxApps:    maxApps,
		MaxUsers:   maxUsers,
		IssuedAt:   time.Now().Unix(),
		ExpiresAt:  expiresAt,
	}

	licenseKey, err := license.GenerateLicenseKey(b.LicensePrivKey, data)
	if err != nil {
		return nil, fmt.Errorf("failed to generate license key: %w", err)
	}

	// Save to DB
	features := ResolveTierFeaturesForTier(tierStr)
	featureList := make([]string, 0, len(features))
	for f := range features {
		featureList = append(featureList, string(f))
	}
	featuresJSON, err := json.Marshal(featureList)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize features: %w", err)
	}

	addonsJSON, err := json.Marshal(addons)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize addons: %w", err)
	}

	limitsJSON, err := json.Marshal(map[string]int{
		"max_servers": maxServers,
		"max_apps":    maxApps,
		"max_users":   maxUsers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize limits: %w", err)
	}

	now := time.Now()
	lic := model.License{
		ID:         fmt.Sprintf("lic-%d", now.UnixNano()),
		TenantID:   tenantID,
		LicenseKey: licenseKey,
		Tier:       model.Tier(tierStr),
		UseType:    model.UseType(useTypeStr),
		Status:     model.LicenseStatusActive,
		Features:   string(featuresJSON),
		Limits:     string(limitsJSON),
		MaxServers: maxServers,
		MaxApps:    maxApps,
		MaxUsers:   maxUsers,
		IssuerRole: issuerRole,
		IssuedTo:   issuedTo,
		MaxIssued:  maxIssued,
		Addons:     string(addonsJSON),
		GraceDays:  7,
		CreatedBy:  "developer",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if expiresAt > 0 {
		exp := time.Unix(expiresAt, 0)
		lic.ExpiresAt = &exp
	}

	if err := b.DB.Create(&lic).Error; err != nil {
		return nil, fmt.Errorf("failed to save issued license: %w", err)
	}

	slog.Info("license issued", "id", lic.ID, "tier", lic.Tier, "tenant", tenantID)

	return map[string]interface{}{
		"id":          lic.ID,
		"license_key": licenseKey,
		"tier":        lic.Tier,
		"use_type":    lic.UseType,
		"tenant_id":   tenantID,
		"expires_at":  lic.ExpiresAt,
	}, nil
}

// ListIssuedLicenses lists all issued licenses (developer only).
func (b *Bridge) ListIssuedLicenses(ctx context.Context) (interface{}, error) {
	var licenses []model.License
	if err := b.DB.Order("created_at DESC").Find(&licenses).Error; err != nil {
		return nil, fmt.Errorf("failed to list licenses: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(licenses))
	for _, lic := range licenses {
		item := map[string]interface{}{
			"id":           lic.ID,
			"tenant_id":    lic.TenantID,
			"tier":         lic.Tier,
			"use_type":     lic.UseType,
			"status":       lic.Status,
			"max_servers":  lic.MaxServers,
			"max_apps":     lic.MaxApps,
			"max_users":    lic.MaxUsers,
			"issuer_role":  lic.IssuerRole,
			"issued_to":    lic.IssuedTo,
			"max_issued":   lic.MaxIssued,
			"issued_count": lic.IssuedCount,
			"expires_at":   lic.ExpiresAt,
			"created_at":   lic.CreatedAt,
		}
		if lic.RevokedReason != "" {
			item["revoked_reason"] = lic.RevokedReason
		}
		result = append(result, item)
	}

	return result, nil
}

// RevokeLicense revokes a license (developer/distributor only).
func (b *Bridge) RevokeLicense(ctx context.Context, licenseID string, reason string) (interface{}, error) {
	if licenseID == "" {
		return nil, fmt.Errorf("license_id is required")
	}
	if reason == "" {
		return nil, fmt.Errorf("reason is required")
	}

	result := b.DB.Model(&model.License{}).
		Where("id = ? AND status = ?", licenseID, model.LicenseStatusActive).
		Updates(map[string]interface{}{
			"status":         model.LicenseStatusRevoked,
			"revoked_reason": reason,
			"revoked_at":     time.Now(),
		})
	if result.Error != nil {
		return nil, fmt.Errorf("failed to revoke license: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("active license not found with id: %s", licenseID)
	}

	slog.Info("license revoked", "id", licenseID, "reason", reason)

	return map[string]interface{}{
		"message":       "license revoked",
		"license_id":    licenseID,
		"rows_affected": result.RowsAffected,
	}, nil
}

// PurchaseAddon purchases an addon for the current license.
