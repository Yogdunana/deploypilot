package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Feature represents a feature flag.
type Feature string

const (
	FeatureSSL              Feature = "ssl"
	FeatureMonitoring       Feature = "monitoring"
	FeatureAlerting         Feature = "alerting"
	FeatureWebhooks         Feature = "webhooks"
	FeaturePlugins          Feature = "plugins"
	FeatureOAuth2           Feature = "oauth2"
	FeatureGrafana          Feature = "grafana"
	FeatureAuditExport      Feature = "audit_export"
	FeatureAPIKeys          Feature = "api_keys"
	Feature2FA              Feature = "2fa"
	FeatureIPWhitelist      Feature = "ip_whitelist"
	FeatureDeviceBinding    Feature = "device_binding"
	FeatureCodeSigning      Feature = "code_signing"
	FeatureBackup           Feature = "backup"
	FeatureCluster          Feature = "cluster"
	FeatureRegistry         Feature = "registry"
	FeatureBatchOperations  Feature = "batch_operations"
	FeatureToolbox          Feature = "toolbox"
	FeatureSSHKeyManagement Feature = "ssh_key_management"
	FeatureDashboardTV      Feature = "dashboard_tv"
	FeatureCustomBranding   Feature = "custom_branding"
	FeaturePrioritySupport  Feature = "priority_support"
	FeatureSLA              Feature = "sla"
	FeatureSSO              Feature = "sso"
	FeatureLDAP             Feature = "ldap"
	FeatureMultiTenant      Feature = "multi_tenant"
	FeatureFederation       Feature = "federation"
)

// Tier represents a license tier level.
type Tier string

const (
	TierCommunity  Tier = "community"
	TierTeam       Tier = "team"
	TierPro        Tier = "pro"
	TierEnterprise Tier = "enterprise"
)

// UseType represents the license usage type.
type UseType string

const (
	UseTypeNonCommercial UseType = "non_commercial"
	UseTypeCommercial     UseType = "commercial"
)

// AllFeatures is the complete set of all features.
var AllFeatures = []Feature{
	FeatureSSL, FeatureMonitoring, FeatureAlerting, FeatureWebhooks,
	FeaturePlugins, FeatureOAuth2, FeatureGrafana, FeatureAuditExport,
	FeatureAPIKeys, Feature2FA, FeatureIPWhitelist, FeatureDeviceBinding,
	FeatureCodeSigning, FeatureBackup, FeatureCluster, FeatureRegistry,
	FeatureBatchOperations, FeatureToolbox, FeatureSSHKeyManagement,
	FeatureDashboardTV, FeatureCustomBranding, FeaturePrioritySupport,
	FeatureSLA, FeatureSSO, FeatureLDAP, FeatureMultiTenant, FeatureFederation,
}

// TeamExcludedFeatures are features NOT available in Team tier (but available in Community).
var TeamExcludedFeatures = map[Feature]bool{
	FeatureDashboardTV: true,
	FeatureSSO:         true,
	FeatureLDAP:        true,
	FeatureMultiTenant: true,
	FeatureFederation:  true,
}

// ProExcludedFeatures are features NOT available in Pro tier.
var ProExcludedFeatures = map[Feature]bool{
	FeatureSLA: true,
}

// TierLimits defines default resource limits per tier.
var TierLimits = map[Tier]struct {
	MaxServers int
	MaxApps    int
	MaxUsers   int
}{
	TierCommunity:  {3, 10, 5},
	TierTeam:       {10, 30, 15},
	TierPro:        {50, 100, 50},
	TierEnterprise: {0, 0, 0}, // 0 = unlimited
}

// Addon represents a license add-on (feature unlock or resource boost).
type Addon struct {
	Key         string `json:"key"`                    // "feature:dashboard_tv" / "resource:servers:10"
	Amount      int    `json:"amount"`                 // for resource addons: quantity; for features: 0
	PurchasedAt int64  `json:"purchased_at"`
	ExpiresAt   int64  `json:"expires_at"`
	PausedAt    int64  `json:"paused_at,omitempty"`
	PausedDays  int64  `json:"paused_days,omitempty"`
}

// LicenseData is the JSON payload that gets signed.
type LicenseData struct {
	TenantID    string    `json:"tenant_id"`
	UseType     string    `json:"use_type"`                // "non_commercial" / "commercial"
	Tier        string    `json:"tier"`                     // "community" / "team" / "pro" / "enterprise"
	IssuerRole  string    `json:"issuer_role"`              // "developer" / "distributor" / "user"
	IssuedTo    string    `json:"issued_to,omitempty"`      // distributor's tenant_id (when developer issues)
	MaxIssued   int       `json:"max_issued,omitempty"`     // max sub-licenses a distributor can issue
	IssuedCount int       `json:"issued_count,omitempty"`
	Addons      []Addon   `json:"addons,omitempty"`
	MaxServers  int       `json:"max_servers"`
	MaxApps     int       `json:"max_apps"`
	MaxUsers    int       `json:"max_users"`
	IssuedAt    int64     `json:"issued_at"`
	ExpiresAt   int64     `json:"expires_at,omitempty"` // 0 = never expires (community)
	MachineID   string    `json:"machine_id,omitempty"`
}

// LicenseInfo is the parsed and validated license info cached in memory.
type LicenseInfo struct {
	Data      LicenseData
	Signature []byte
	ValidFrom time.Time
	ValidTo   time.Time
	Features  map[Feature]bool
	UseType   UseType
	Tier      Tier
	Addons    []Addon
}

// Engine is the license validation and feature evaluation engine.
// It is safe for concurrent use.
type Engine struct {
	mu          sync.RWMutex
	publicKey   ed25519.PublicKey
	info        *LicenseInfo
	graceDays   int
	onExpired   func() // callback when license expires
	onViolation func(feature Feature)
}

// NewEngine creates a new license engine with the given Ed25519 public key.
func NewEngine(publicKey ed25519.PublicKey, graceDays int) *Engine {
	return &Engine{
		publicKey: publicKey,
		graceDays: graceDays,
	}
}

// LoadLicense validates and loads a license key (base64-encoded signature + JSON payload).
// The format is: base64(signature) + "." + base64(json_payload)
func (e *Engine) LoadLicense(licenseKey string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.publicKey == nil {
		return fmt.Errorf("license public key not configured")
	}

	// Parse license key: signature.payload (both base64)
	payload, signature, err := parseLicenseKey(licenseKey)
	if err != nil {
		return fmt.Errorf("failed to parse license key: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(e.publicKey, payload, signature) {
		return fmt.Errorf("invalid license signature")
	}

	// Decode payload
	var data LicenseData
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("failed to decode license payload: %w", err)
	}

	// Resolve features based on tier
	features := resolveTierFeatures(data.Tier)

	info := &LicenseInfo{
		Data:      data,
		Signature: signature,
		ValidFrom: time.Unix(data.IssuedAt, 0),
		Features:  features,
		UseType:   UseType(data.UseType),
		Tier:      Tier(data.Tier),
		Addons:    data.Addons,
	}

	if data.ExpiresAt > 0 {
		info.ValidTo = time.Unix(data.ExpiresAt, 0)
	}

	e.info = info
	slog.Info("license loaded", "tier", data.Tier, "use_type", data.UseType, "tenant", data.TenantID,
		"expires", info.ValidTo, "features", len(features), "addons", len(data.Addons))
	return nil
}

// Validate checks if the current license is valid (not expired, not revoked).
func (e *Engine) Validate() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.info == nil {
		return fmt.Errorf("no license loaded")
	}

	now := time.Now()

	// Check not-before
	if now.Before(e.info.ValidFrom) {
		return fmt.Errorf("license is not yet valid (valid from %s)", e.info.ValidFrom.Format(time.RFC3339))
	}

	// Check expiration with grace period
	if !e.info.ValidTo.IsZero() && now.After(e.info.ValidTo.Add(time.Duration(e.graceDays)*24*time.Hour)) {
		return fmt.Errorf("license expired on %s (grace period: %d days)", e.info.ValidTo.Format(time.RFC3339), e.graceDays)
	}

	return nil
}

// IsFeatureEnabled checks if a specific feature is available in the current license.
func (e *Engine) IsFeatureEnabled(feature Feature) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.info == nil {
		return false
	}

	// 1. Check if current tier includes this feature
	if e.info.Features[feature] {
		return true
	}

	// 2. Check addons (non-paused, non-expired)
	now := time.Now()
	for _, addon := range e.info.Addons {
		if addon.PausedAt > 0 {
			continue // paused by higher tier
		}
		if addon.ExpiresAt > 0 && now.After(time.Unix(addon.ExpiresAt, 0)) {
			continue // expired
		}
		if addon.Key == "feature:"+string(feature) {
			return true
		}
	}
	return false
}

// RequireFeature checks if a feature is enabled; if not, calls the onViolation callback.
func (e *Engine) RequireFeature(feature Feature) error {
	if e.IsFeatureEnabled(feature) {
		return nil
	}
	if e.onViolation != nil {
		e.onViolation(feature)
	}
	return fmt.Errorf("feature '%s' requires a higher license tier", feature)
}

// GetUseType returns the current license use type.
func (e *Engine) GetUseType() UseType {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.info == nil {
		return UseTypeNonCommercial
	}
	return e.info.UseType
}

// GetTier returns the current license tier.
func (e *Engine) GetTier() Tier {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.info == nil {
		return TierCommunity
	}
	return e.info.Tier
}

// IsCommercial returns true if the license is for commercial use.
func (e *Engine) IsCommercial() bool {
	return e.GetUseType() == UseTypeCommercial
}

// GetLicenseType returns the current license type (backward compatible, returns tier).
func (e *Engine) GetLicenseType() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.info == nil {
		return "none"
	}
	return e.info.Data.Tier
}

// GetLimits returns the current license limits including addon resources.
func (e *Engine) GetLimits() (maxServers, maxApps, maxUsers int) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.info == nil {
		return 3, 10, 5 // community defaults
	}
	maxServers = e.info.Data.MaxServers
	maxApps = e.info.Data.MaxApps
	maxUsers = e.info.Data.MaxUsers

	now := time.Now()
	for _, addon := range e.info.Addons {
		if addon.PausedAt > 0 {
			continue
		}
		if addon.ExpiresAt > 0 && now.After(time.Unix(addon.ExpiresAt, 0)) {
			continue
		}
		switch addon.Key {
		case "resource:servers":
			maxServers += addon.Amount
		case "resource:apps":
			maxApps += addon.Amount
		case "resource:users":
			maxUsers += addon.Amount
		}
	}
	return
}

// CheckLimit checks if a usage count is within the license limit.
func (e *Engine) CheckLimit(resource string, current int) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.info == nil {
		// No license = community defaults
		switch resource {
		case "servers":
			if current >= 3 {
				return fmt.Errorf("server limit reached (3/3), upgrade to Pro for more")
			}
		case "apps":
			if current >= 10 {
				return fmt.Errorf("app limit reached (10/10), upgrade to Pro for more")
			}
		case "users":
			if current >= 5 {
				return fmt.Errorf("user limit reached (5/5), upgrade to Pro for more")
			}
		}
		return nil
	}

	maxServers, maxApps, maxUsers := e.getLimitsUnlocked()

	switch resource {
	case "servers":
		if maxServers > 0 && current >= maxServers {
			return fmt.Errorf("server limit reached (%d/%d)", current, maxServers)
		}
	case "apps":
		if maxApps > 0 && current >= maxApps {
			return fmt.Errorf("app limit reached (%d/%d)", current, maxApps)
		}
	case "users":
		if maxUsers > 0 && current >= maxUsers {
			return fmt.Errorf("user limit reached (%d/%d)", current, maxUsers)
		}
	}
	return nil
}

// getLimitsUnlocked returns limits without locking (caller must hold lock).
func (e *Engine) getLimitsUnlocked() (maxServers, maxApps, maxUsers int) {
	if e.info == nil {
		return 3, 10, 5
	}
	maxServers = e.info.Data.MaxServers
	maxApps = e.info.Data.MaxApps
	maxUsers = e.info.Data.MaxUsers

	now := time.Now()
	for _, addon := range e.info.Addons {
		if addon.PausedAt > 0 {
			continue
		}
		if addon.ExpiresAt > 0 && now.After(time.Unix(addon.ExpiresAt, 0)) {
			continue
		}
		switch addon.Key {
		case "resource:servers":
			maxServers += addon.Amount
		case "resource:apps":
			maxApps += addon.Amount
		case "resource:users":
			maxUsers += addon.Amount
		}
	}
	return
}

// GetInfo returns a copy of the current license info.
func (e *Engine) GetInfo() *LicenseInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.info == nil {
		return nil
	}
	cp := *e.info
	return &cp
}

// OnExpired sets a callback invoked when the license is detected as expired.
func (e *Engine) OnExpired(fn func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onExpired = fn
}

// OnViolation sets a callback invoked when a feature access is denied.
func (e *Engine) OnViolation(fn func(feature Feature)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onViolation = fn
}

// parseLicenseKey parses "base64_sig.base64_payload" format.
func parseLicenseKey(key string) (payload []byte, signature []byte, err error) {
	// Format: base64(signature).base64(json_payload)
	// Find the last dot separator
	lastDot := -1
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			lastDot = i
			break
		}
	}
	if lastDot == -1 {
		return nil, nil, fmt.Errorf("invalid license key format: missing separator")
	}

	sigStr := key[:lastDot]
	payloadStr := key[lastDot+1:]

	// Base64 decode
	signature, err = base64.StdEncoding.DecodeString(sigStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode signature: %w", err)
	}
	payload, err = base64.StdEncoding.DecodeString(payloadStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	return payload, signature, nil
}

// resolveTierFeatures returns the feature set for a given tier.
func resolveTierFeatures(tier string) map[Feature]bool {
	features := make(map[Feature]bool, len(AllFeatures))

	switch Tier(tier) {
	case TierCommunity, TierEnterprise:
		// All features
		for _, f := range AllFeatures {
			features[f] = true
		}
	case TierTeam:
		// All except TeamExcludedFeatures
		for _, f := range AllFeatures {
			if !TeamExcludedFeatures[f] {
				features[f] = true
			}
		}
	case TierPro:
		// All except ProExcludedFeatures
		for _, f := range AllFeatures {
			if !ProExcludedFeatures[f] {
				features[f] = true
			}
		}
	default:
		// Unknown tier = community features
		for _, f := range AllFeatures {
			features[f] = true
		}
	}
	return features
}

// GenerateLicenseKey creates a signed license key string.
// This is used by the license issuer (not by the licensee).
func GenerateLicenseKey(privateKey ed25519.PrivateKey, data LicenseData) (string, error) {
	if data.IssuedAt == 0 {
		data.IssuedAt = time.Now().Unix()
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal license data: %w", err)
	}

	signature := ed25519.Sign(privateKey, payload)

	return base64.StdEncoding.EncodeToString(signature) + "." + base64.StdEncoding.EncodeToString(payload), nil
}
