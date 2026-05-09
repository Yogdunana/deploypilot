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

	"github.com/Yogdunana/deploypilot/internal/license"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// ---------- License Management ----------

// ActivateLicense activates a license key or creates a community license.
// If licenseKey is provided, it validates the signature, loads into engine, and saves to DB.
// If useType is "non_commercial" and agreeTerms is true, it creates a community license.
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

