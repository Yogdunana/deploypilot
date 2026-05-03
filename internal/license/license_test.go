package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseLicenseKey(t *testing.T) {
	// Create a valid key for testing
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	data := LicenseData{
		TenantID:    "tenant-test",
		LicenseType: "community",
		MaxServers:  3,
		MaxApps:     10,
		MaxUsers:    3,
		IssuedAt:    time.Now().Unix(),
		ExpiresAt:   time.Now().Add(24 * time.Hour).Unix(),
	}

	key, err := GenerateLicenseKey(priv, data)
	if err != nil {
		t.Fatalf("failed to generate license key: %v", err)
	}

	t.Run("valid key", func(t *testing.T) {
		payload, signature, err := parseLicenseKey(key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(payload) == 0 {
			t.Error("expected non-empty payload")
		}
		if len(signature) == 0 {
			t.Error("expected non-empty signature")
		}
		// Verify the signature is valid
		if !ed25519.Verify(pub, payload, signature) {
			t.Error("signature verification failed")
		}
	})

	t.Run("missing separator", func(t *testing.T) {
		_, _, err := parseLicenseKey("invalidkeynoseparator")
		if err == nil {
			t.Fatal("expected error for missing separator")
		}
		if !strings.Contains(err.Error(), "missing separator") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("invalid base64 signature", func(t *testing.T) {
		_, _, err := parseLicenseKey("!!!invalid!!!." + base64.StdEncoding.EncodeToString([]byte("{}")))
		if err == nil {
			t.Fatal("expected error for invalid base64")
		}
		if !strings.Contains(err.Error(), "failed to decode signature") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("invalid base64 payload", func(t *testing.T) {
		_, _, err := parseLicenseKey(base64.StdEncoding.EncodeToString(make([]byte, 64)) + ".!!!invalid!!!")
		if err == nil {
			t.Fatal("expected error for invalid base64 payload")
		}
		if !strings.Contains(err.Error(), "failed to decode payload") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestEngineLoadAndValidate(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	engine := NewEngine(pub, 7)

	data := LicenseData{
		TenantID:    "tenant-test",
		LicenseType: "pro",
		Features:    ProFeatures,
		MaxServers:  10,
		MaxApps:     50,
		MaxUsers:    20,
		IssuedAt:    time.Now().Add(-1 * time.Hour).Unix(),
		ExpiresAt:   time.Now().Add(365 * 24 * time.Hour).Unix(),
	}

	key, err := GenerateLicenseKey(priv, data)
	if err != nil {
		t.Fatalf("failed to generate license key: %v", err)
	}

	t.Run("load and validate successfully", func(t *testing.T) {
		err := engine.LoadLicense(key)
		if err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		err = engine.Validate()
		if err != nil {
			t.Fatalf("license validation failed: %v", err)
		}

		if engine.GetLicenseType() != "pro" {
			t.Errorf("expected license type 'pro', got '%s'", engine.GetLicenseType())
		}
	})

	t.Run("reject tampered key", func(t *testing.T) {
		tamperedKey := "AAAA" + key[4:]
		err := engine.LoadLicense(tamperedKey)
		if err == nil {
			t.Fatal("expected error for tampered key")
		}
		if !strings.Contains(err.Error(), "invalid license signature") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("reject key with wrong public key", func(t *testing.T) {
		otherPub, _, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatalf("failed to generate other key pair: %v", err)
		}
		otherEngine := NewEngine(otherPub, 7)
		err = otherEngine.LoadLicense(key)
		if err == nil {
			t.Fatal("expected error for wrong public key")
		}
		if !strings.Contains(err.Error(), "invalid license signature") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("reject malformed key", func(t *testing.T) {
		err := engine.LoadLicense("not-a-valid-key")
		if err == nil {
			t.Fatal("expected error for malformed key")
		}
	})

	t.Run("no public key configured", func(t *testing.T) {
		emptyEngine := NewEngine(nil, 7)
		err := emptyEngine.LoadLicense(key)
		if err == nil {
			t.Fatal("expected error when no public key configured")
		}
		if !strings.Contains(err.Error(), "public key not configured") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestEngineFeatureCheck(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("community features", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:    "tenant-community",
			LicenseType: "community",
			Features:    CommunityFeatures,
			MaxServers:  3,
			MaxApps:     10,
			MaxUsers:    3,
			IssuedAt:    time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		// Community should have basic features
		if !engine.IsFeatureEnabled(FeatureSSL) {
			t.Error("community license should have ssl feature")
		}
		if !engine.IsFeatureEnabled(FeatureMonitoring) {
			t.Error("community license should have monitoring feature")
		}
		if !engine.IsFeatureEnabled(FeatureBackup) {
			t.Error("community license should have backup feature")
		}

		// Community should NOT have pro/enterprise features
		if engine.IsFeatureEnabled(FeatureOAuth2) {
			t.Error("community license should not have oauth2 feature")
		}
		if engine.IsFeatureEnabled(FeatureSSO) {
			t.Error("community license should not have sso feature")
		}
		if engine.IsFeatureEnabled(FeatureLDAP) {
			t.Error("community license should not have ldap feature")
		}
	})

	t.Run("pro features", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:    "tenant-pro",
			LicenseType: "pro",
			Features:    ProFeatures,
			MaxServers:  10,
			MaxApps:     50,
			MaxUsers:    20,
			IssuedAt:    time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		// Pro should have community + pro features
		if !engine.IsFeatureEnabled(FeatureSSL) {
			t.Error("pro license should have ssl feature")
		}
		if !engine.IsFeatureEnabled(FeatureOAuth2) {
			t.Error("pro license should have oauth2 feature")
		}
		if !engine.IsFeatureEnabled(FeatureCluster) {
			t.Error("pro license should have cluster feature")
		}

		// Pro should NOT have enterprise features
		if engine.IsFeatureEnabled(FeatureSSO) {
			t.Error("pro license should not have sso feature")
		}
		if engine.IsFeatureEnabled(FeatureLDAP) {
			t.Error("pro license should not have ldap feature")
		}
	})

	t.Run("enterprise features", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:    "tenant-enterprise",
			LicenseType: "enterprise",
			Features:    EnterpriseFeatures,
			MaxServers:  100,
			MaxApps:     500,
			MaxUsers:    100,
			IssuedAt:    time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		// Enterprise should have all features
		if !engine.IsFeatureEnabled(FeatureSSL) {
			t.Error("enterprise license should have ssl feature")
		}
		if !engine.IsFeatureEnabled(FeatureOAuth2) {
			t.Error("enterprise license should have oauth2 feature")
		}
		if !engine.IsFeatureEnabled(FeatureSSO) {
			t.Error("enterprise license should have sso feature")
		}
		if !engine.IsFeatureEnabled(FeatureLDAP) {
			t.Error("enterprise license should have ldap feature")
		}
		if !engine.IsFeatureEnabled(FeatureMultiTenant) {
			t.Error("enterprise license should have multi_tenant feature")
		}
		if !engine.IsFeatureEnabled(FeatureFederation) {
			t.Error("enterprise license should have federation feature")
		}
	})

	t.Run("default features when empty", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:    "tenant-default",
			LicenseType: "pro",
			Features:    nil, // empty - should use defaults
			MaxServers:  10,
			MaxApps:     50,
			MaxUsers:    20,
			IssuedAt:    time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		// Should get pro defaults
		if !engine.IsFeatureEnabled(FeatureOAuth2) {
			t.Error("pro default features should include oauth2")
		}
	})

	t.Run("require feature callback", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:    "tenant-cb",
			LicenseType: "community",
			Features:    CommunityFeatures,
			MaxServers:  3,
			MaxApps:     10,
			MaxUsers:    3,
			IssuedAt:    time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		var violatedFeature Feature
		engine.OnViolation(func(f Feature) {
			violatedFeature = f
		})

		err = engine.RequireFeature(FeatureSSO)
		if err == nil {
			t.Fatal("expected error for requiring SSO with community license")
		}
		if violatedFeature != FeatureSSO {
			t.Errorf("expected violated feature SSO, got %s", violatedFeature)
		}
	})
}

func TestEngineLimitCheck(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	engine := NewEngine(pub, 7)
	data := LicenseData{
		TenantID:    "tenant-limits",
		LicenseType: "pro",
		Features:    ProFeatures,
		MaxServers:  5,
		MaxApps:     20,
		MaxUsers:    10,
		IssuedAt:    time.Now().Unix(),
	}
	key, err := GenerateLicenseKey(priv, data)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	if err := engine.LoadLicense(key); err != nil {
		t.Fatalf("failed to load license: %v", err)
	}

	t.Run("within limits", func(t *testing.T) {
		if err := engine.CheckLimit("servers", 3); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := engine.CheckLimit("apps", 15); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := engine.CheckLimit("users", 8); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("at limit", func(t *testing.T) {
		if err := engine.CheckLimit("servers", 5); err == nil {
			t.Error("expected error when at server limit")
		}
		if err := engine.CheckLimit("apps", 20); err == nil {
			t.Error("expected error when at app limit")
		}
		if err := engine.CheckLimit("users", 10); err == nil {
			t.Error("expected error when at user limit")
		}
	})

	t.Run("over limit", func(t *testing.T) {
		if err := engine.CheckLimit("servers", 10); err == nil {
			t.Error("expected error when over server limit")
		}
	})

	t.Run("unknown resource", func(t *testing.T) {
		// Unknown resources should not error
		if err := engine.CheckLimit("unknown", 100); err != nil {
			t.Errorf("unexpected error for unknown resource: %v", err)
		}
	})

	t.Run("get limits", func(t *testing.T) {
		s, a, u := engine.GetLimits()
		if s != 5 || a != 20 || u != 10 {
			t.Errorf("expected limits (5, 20, 10), got (%d, %d, %d)", s, a, u)
		}
	})
}

func TestEngineExpiredLicense(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	engine := NewEngine(pub, 7)

	// License that expired 10 days ago
	data := LicenseData{
		TenantID:    "tenant-expired",
		LicenseType: "pro",
		Features:    ProFeatures,
		MaxServers:  10,
		MaxApps:     50,
		MaxUsers:    20,
		IssuedAt:    time.Now().Add(-30 * 24 * time.Hour).Unix(),
		ExpiresAt:   time.Now().Add(-10 * 24 * time.Hour).Unix(),
	}

	key, err := GenerateLicenseKey(priv, data)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	if err := engine.LoadLicense(key); err != nil {
		t.Fatalf("failed to load license: %v", err)
	}

	err = engine.Validate()
	if err == nil {
		t.Fatal("expected error for expired license")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEngineNoLicense(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	engine := NewEngine(pub, 7)

	t.Run("validate without license", func(t *testing.T) {
		err := engine.Validate()
		if err == nil {
			t.Fatal("expected error when no license loaded")
		}
		if !strings.Contains(err.Error(), "no license loaded") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("feature check without license", func(t *testing.T) {
		if engine.IsFeatureEnabled(FeatureSSL) {
			t.Error("expected feature to be disabled when no license loaded")
		}
	})

	t.Run("get license type without license", func(t *testing.T) {
		if lt := engine.GetLicenseType(); lt != "none" {
			t.Errorf("expected 'none', got '%s'", lt)
		}
	})

	t.Run("get limits without license", func(t *testing.T) {
		s, a, u := engine.GetLimits()
		if s != 3 || a != 10 || u != 3 {
			t.Errorf("expected community defaults (3, 10, 3), got (%d, %d, %d)", s, a, u)
		}
	})

	t.Run("check limit without license", func(t *testing.T) {
		if err := engine.CheckLimit("servers", 2); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := engine.CheckLimit("servers", 3); err == nil {
			t.Error("expected error at community default limit")
		}
	})

	t.Run("get info without license", func(t *testing.T) {
		info := engine.GetInfo()
		if info != nil {
			t.Error("expected nil info when no license loaded")
		}
	})
}

func TestEngineGracePeriod(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("within grace period", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		// License expired 3 days ago, grace period is 7 days
		data := LicenseData{
			TenantID:    "tenant-grace",
			LicenseType: "pro",
			Features:    ProFeatures,
			MaxServers:  10,
			MaxApps:     50,
			MaxUsers:    20,
			IssuedAt:    time.Now().Add(-30 * 24 * time.Hour).Unix(),
			ExpiresAt:   time.Now().Add(-3 * 24 * time.Hour).Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		// Should be valid within grace period
		if err := engine.Validate(); err != nil {
			t.Errorf("expected license to be valid within grace period, got error: %v", err)
		}
	})

	t.Run("grace period expired", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		// License expired 10 days ago, grace period is 7 days
		data := LicenseData{
			TenantID:    "tenant-grace-expired",
			LicenseType: "pro",
			Features:    ProFeatures,
			MaxServers:  10,
			MaxApps:     50,
			MaxUsers:    20,
			IssuedAt:    time.Now().Add(-30 * 24 * time.Hour).Unix(),
			ExpiresAt:   time.Now().Add(-10 * 24 * time.Hour).Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		if err := engine.Validate(); err == nil {
			t.Error("expected error when grace period has expired")
		}
	})

	t.Run("zero grace period", func(t *testing.T) {
		engine := NewEngine(pub, 0)
		// License expired 1 day ago, no grace period
		data := LicenseData{
			TenantID:    "tenant-no-grace",
			LicenseType: "pro",
			Features:    ProFeatures,
			MaxServers:  10,
			MaxApps:     50,
			MaxUsers:    20,
			IssuedAt:    time.Now().Add(-30 * 24 * time.Hour).Unix(),
			ExpiresAt:   time.Now().Add(-1 * 24 * time.Hour).Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		if err := engine.Validate(); err == nil {
			t.Error("expected error when expired with zero grace period")
		}
	})

	t.Run("never expires", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:    "tenant-perpetual",
			LicenseType: "enterprise",
			Features:    EnterpriseFeatures,
			MaxServers:  100,
			MaxApps:     500,
			MaxUsers:    100,
			IssuedAt:    time.Now().Add(-1 * 24 * time.Hour).Unix(),
			ExpiresAt:   0, // never expires
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		if err := engine.Validate(); err != nil {
			t.Errorf("perpetual license should always be valid, got error: %v", err)
		}
	})
}

func TestGenerateLicenseKey(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("auto-set issued at", func(t *testing.T) {
		data := LicenseData{
			TenantID:    "tenant-auto",
			LicenseType: "community",
			IssuedAt:    0, // should be auto-set
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}

		// Parse and verify issued_at was set
		_, payload, err := parseLicenseKey(key)
		if err != nil {
			t.Fatalf("failed to parse key: %v", err)
		}
		var parsed LicenseData
		if err := json.Unmarshal(payload, &parsed); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		if parsed.IssuedAt == 0 {
			t.Error("expected IssuedAt to be auto-set")
		}
	})

	t.Run("key format is base64.base64", func(t *testing.T) {
		data := LicenseData{
			TenantID:    "tenant-format",
			LicenseType: "pro",
			IssuedAt:    time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}

		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 {
			t.Fatalf("expected 2 parts separated by dot, got %d", len(parts))
		}
		if _, err := base64.StdEncoding.DecodeString(parts[0]); err != nil {
			t.Errorf("signature part is not valid base64: %v", err)
		}
		if _, err := base64.StdEncoding.DecodeString(parts[1]); err != nil {
			t.Errorf("payload part is not valid base64: %v", err)
		}
	})
}

func TestGetInfo(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	engine := NewEngine(pub, 7)
	data := LicenseData{
		TenantID:    "tenant-info",
		LicenseType: "enterprise",
		Features:    EnterpriseFeatures,
		MaxServers:  100,
		MaxApps:     500,
		MaxUsers:    100,
		IssuedAt:    time.Now().Unix(),
	}
	key, err := GenerateLicenseKey(priv, data)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	if err := engine.LoadLicense(key); err != nil {
		t.Fatalf("failed to load license: %v", err)
	}

	info := engine.GetInfo()
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Data.TenantID != "tenant-info" {
		t.Errorf("expected tenant_id 'tenant-info', got '%s'", info.Data.TenantID)
	}
	if len(info.Features) == 0 {
		t.Error("expected non-empty features map")
	}

	// Verify it's a copy (modifying original should not affect the copy)
	engine.LoadLicense(key) // reload
	info.Features[FeatureSSL] = false
	if !engine.IsFeatureEnabled(FeatureSSL) {
		t.Error("modifying GetInfo copy should not affect engine state")
	}
}
