package service

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Yogdunana/deploypilot/internal/license"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// ---------- License Management ----------

// ActivateLicense activates a license key or creates a community license.
// If licenseKey is provided, it validates the signature, loads into engine, and saves to DB.
// If useType is "non_commercial" and agreeTerms is true, it creates a community license.
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
func (b *Bridge) GetLicenseStatus(ctx context.Context) (interface{}, error) {
	// Try engine first
	if b.LicenseEngine != nil {
		info := b.LicenseEngine.GetInfo()
		if info != nil {
			// Check validation
			validationErr := b.LicenseEngine.Validate()

			features := make([]string, 0, len(info.Features))
			for f := range info.Features {
				features = append(features, string(f))
			}

			maxServers, maxApps, maxUsers := b.LicenseEngine.GetLimits()

			status := "active"
			if validationErr != nil {
				status = "expired"
			}

			return map[string]interface{}{
				"status":    status,
				"tier":      string(info.Tier),
				"use_type":  string(info.UseType),
				"features":  features,
				"addons":    info.Addons,
				"limits": map[string]int{
					"max_servers": maxServers,
					"max_apps":    maxApps,
					"max_users":   maxUsers,
				},
				"valid_from":  info.ValidFrom,
				"valid_to":    info.ValidTo,
				"tenant_id":   info.Data.TenantID,
				"issuer_role": info.Data.IssuerRole,
			}, nil
		}
	}

	// Fall back to DB
	var lic model.License
	if err := b.DB.Where("status = ?", model.LicenseStatusActive).First(&lic).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return map[string]interface{}{
				"status":   "none",
				"tier":     "community",
				"use_type": "non_commercial",
				"features": []string{},
				"addons":   []interface{}{},
				"limits": map[string]int{
					"max_servers": 3,
					"max_apps":    10,
					"max_users":   5,
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to query license: %w", err)
	}

	var features []string
	if lic.Features != "" {
		if err := json.Unmarshal([]byte(lic.Features), &features); err != nil {
			slog.Warn("failed to parse license features", "error", err)
		}
	}

	var addons []license.Addon
	if lic.Addons != "" {
		if err := json.Unmarshal([]byte(lic.Addons), &addons); err != nil {
			slog.Warn("failed to parse license addons", "error", err)
		}
	}

	return map[string]interface{}{
		"status":     lic.Status,
		"tier":       lic.Tier,
		"use_type":   lic.UseType,
		"features":   features,
		"addons":     addons,
		"expires_at": lic.ExpiresAt,
		"limits": map[string]int{
			"max_servers": lic.MaxServers,
			"max_apps":    lic.MaxApps,
			"max_users":   lic.MaxUsers,
		},
	}, nil
}

// IssueLicense creates a new license key (developer/distributor only).
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

// LoadLicenseFromDB loads the active license from the database into the engine.
func (b *Bridge) LoadLicenseFromDB(ctx context.Context) error {
	if b.LicenseEngine == nil {
		return nil
	}

	var lic model.License
	if err := b.DB.Where("status = ?", model.LicenseStatusActive).First(&lic).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			slog.Info("no active license found in database")
			return nil
		}
		return fmt.Errorf("failed to query active license: %w", err)
	}

	if lic.LicenseKey == "" {
		slog.Info("active license has no key, skipping engine load")
		return nil
	}

	if err := b.LicenseEngine.LoadLicense(lic.LicenseKey); err != nil {
		slog.Warn("failed to load license from DB into engine", "error", err)
		return nil // non-fatal: license might have been signed with a different key
	}

	slog.Info("license loaded from database", "tier", lic.Tier, "use_type", lic.UseType)
	return nil
}

// InitLicenseEngine initializes the license engine with the given public key.
func (b *Bridge) InitLicenseEngine(publicKey ed25519.PublicKey, graceDays int) {
	b.LicenseEngine = license.NewEngine(publicKey, graceDays)
}

// SetLicensePrivateKey sets the private key for license issuance (developer mode).
func (b *Bridge) SetLicensePrivateKey(privKey ed25519.PrivateKey) {
	b.LicensePrivKey = privKey
}

// ResolveTierFeaturesForTier resolves features for a given tier.
// This is a service-layer helper since the license package's resolveTierFeatures is unexported.
func ResolveTierFeaturesForTier(tier string) map[license.Feature]bool {
	features := make(map[license.Feature]bool, len(license.AllFeatures))

	switch license.Tier(tier) {
	case license.TierCommunity, license.TierEnterprise:
		for _, f := range license.AllFeatures {
			features[f] = true
		}
	case license.TierTeam:
		for _, f := range license.AllFeatures {
			if !license.TeamExcludedFeatures[f] {
				features[f] = true
			}
		}
	case license.TierPro:
		for _, f := range license.AllFeatures {
			if !license.ProExcludedFeatures[f] {
				features[f] = true
			}
		}
	default:
		for _, f := range license.AllFeatures {
			features[f] = true
		}
	}
	return features
}

// LoadPublicKeyFromFileOrBase64 loads an Ed25519 public key from a file or base64 string.
func LoadPublicKeyFromFileOrBase64(file string, base64Str string) (ed25519.PublicKey, error) {
	if base64Str != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(base64Str)
		if err != nil {
			return nil, fmt.Errorf("failed to decode public key base64: %w", err)
		}
		if len(keyBytes) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid public key size: got %d, want %d", len(keyBytes), ed25519.PublicKeySize)
		}
		return ed25519.PublicKey(keyBytes), nil
	}

	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key file: %w", err)
		}

		// Try PEM format first
		block, _ := pem.Decode(data)
		if block != nil && len(block.Bytes) == ed25519.PublicKeySize {
			return ed25519.PublicKey(block.Bytes), nil
		}

		// Try raw base64
		keyBytes, err := base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return nil, fmt.Errorf("failed to decode public key file content: %w", err)
		}
		if len(keyBytes) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid public key size from file: got %d, want %d", len(keyBytes), ed25519.PublicKeySize)
		}
		return ed25519.PublicKey(keyBytes), nil
	}

	return nil, fmt.Errorf("no public key configured (set license.public_key_file or license.public_key_base64)")
}
