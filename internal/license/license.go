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
	// Pro features
	FeatureCustomBranding  Feature = "custom_branding"
	FeaturePrioritySupport Feature = "priority_support"
	FeatureSLA             Feature = "sla"
	// Enterprise features
	FeatureSSO         Feature = "sso"
	FeatureLDAP        Feature = "ldap"
	FeatureMultiTenant Feature = "multi_tenant"
	FeatureFederation  Feature = "federation"
)

// CommunityFeatures are features available in the free Community edition.
var CommunityFeatures = []Feature{
	FeatureSSL, FeatureMonitoring, FeatureAlerting, FeatureWebhooks,
	FeaturePlugins, FeatureAuditExport, FeatureAPIKeys, Feature2FA,
	FeatureBackup, FeatureToolbox,
}

// ProFeatures includes all Community features plus Pro-only features.
var ProFeatures = []Feature{
	FeatureSSL, FeatureMonitoring, FeatureAlerting, FeatureWebhooks,
	FeaturePlugins, FeatureOAuth2, FeatureGrafana, FeatureAuditExport,
	FeatureAPIKeys, Feature2FA, FeatureIPWhitelist, FeatureDeviceBinding,
	FeatureCodeSigning, FeatureBackup, FeatureCluster, FeatureRegistry,
	FeatureBatchOperations, FeatureToolbox, FeatureSSHKeyManagement,
	FeatureCustomBranding, FeaturePrioritySupport, FeatureSLA,
}

// EnterpriseFeatures includes all Pro features plus Enterprise-only features.
var EnterpriseFeatures = []Feature{
	FeatureSSL, FeatureMonitoring, FeatureAlerting, FeatureWebhooks,
	FeaturePlugins, FeatureOAuth2, FeatureGrafana, FeatureAuditExport,
	FeatureAPIKeys, Feature2FA, FeatureIPWhitelist, FeatureDeviceBinding,
	FeatureCodeSigning, FeatureBackup, FeatureCluster, FeatureRegistry,
	FeatureBatchOperations, FeatureToolbox, FeatureSSHKeyManagement,
	FeatureCustomBranding, FeaturePrioritySupport, FeatureSLA,
	FeatureSSO, FeatureLDAP, FeatureMultiTenant, FeatureFederation,
}

// LicenseData is the JSON payload that gets signed.
type LicenseData struct {
	TenantID    string    `json:"tenant_id"`
	LicenseType string    `json:"license_type"`
	Features    []Feature `json:"features"`
	MaxServers  int       `json:"max_servers"`
	MaxApps     int       `json:"max_apps"`
	MaxUsers    int       `json:"max_users"`
	IssuedAt    int64     `json:"issued_at"`
	ExpiresAt   int64     `json:"expires_at,omitempty"` // 0 = never expires
	MachineID   string    `json:"machine_id,omitempty"`
}

// LicenseInfo is the parsed and validated license info cached in memory.
type LicenseInfo struct {
	Data      LicenseData
	Signature []byte
	ValidFrom time.Time
	ValidTo   time.Time
	Features  map[Feature]bool
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

	// Build feature map
	features := make(map[Feature]bool, len(data.Features))
	for _, f := range data.Features {
		features[f] = true
	}

	// If no features specified, use defaults based on license type
	if len(data.Features) == 0 {
		defaults := getDefaultFeatures(data.LicenseType)
		for _, f := range defaults {
			features[f] = true
		}
	}

	info := &LicenseInfo{
		Data:      data,
		Signature: signature,
		ValidFrom: time.Unix(data.IssuedAt, 0),
		Features:  features,
	}

	if data.ExpiresAt > 0 {
		info.ValidTo = time.Unix(data.ExpiresAt, 0)
	}

	e.info = info
	slog.Info("license loaded", "type", data.LicenseType, "tenant", data.TenantID,
		"expires", info.ValidTo, "features", len(features))
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
	return e.info.Features[feature]
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

// GetLicenseType returns the current license type.
func (e *Engine) GetLicenseType() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.info == nil {
		return "none"
	}
	return e.info.Data.LicenseType
}

// GetLimits returns the current license limits.
func (e *Engine) GetLimits() (maxServers, maxApps, maxUsers int) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.info == nil {
		return 3, 10, 3 // community defaults
	}
	return e.info.Data.MaxServers, e.info.Data.MaxApps, e.info.Data.MaxUsers
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
			if current >= 3 {
				return fmt.Errorf("user limit reached (3/3), upgrade to Pro for more")
			}
		}
		return nil
	}

	switch resource {
	case "servers":
		if current >= e.info.Data.MaxServers {
			return fmt.Errorf("server limit reached (%d/%d)", current, e.info.Data.MaxServers)
		}
	case "apps":
		if current >= e.info.Data.MaxApps {
			return fmt.Errorf("app limit reached (%d/%d)", current, e.info.Data.MaxApps)
		}
	case "users":
		if current >= e.info.Data.MaxUsers {
			return fmt.Errorf("user limit reached (%d/%d)", current, e.info.Data.MaxUsers)
		}
	}
	return nil
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

// getDefaultFeatures returns default features for a license type.
func getDefaultFeatures(licenseType string) []Feature {
	switch licenseType {
	case "pro":
		return ProFeatures
	case "enterprise":
		return EnterpriseFeatures
	default:
		return CommunityFeatures
	}
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
