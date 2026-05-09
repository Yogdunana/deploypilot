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
		TenantID:   "tenant-test",
		Tier:       "community",
		UseType:    "non_commercial",
		MaxServers: 3,
		MaxApps:    10,
		MaxUsers:   5,
		IssuedAt:   time.Now().Unix(),
		ExpiresAt:  time.Now().Add(24 * time.Hour).Unix(),
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
		TenantID:   "tenant-test",
		Tier:       "pro",
		UseType:    "commercial",
		MaxServers: 10,
		MaxApps:    50,
		MaxUsers:   20,
		IssuedAt:   time.Now().Add(-1 * time.Hour).Unix(),
		ExpiresAt:  time.Now().Add(365 * 24 * time.Hour).Unix(),
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
		if engine.GetTier() != TierPro {
			t.Errorf("expected tier 'pro', got '%s'", engine.GetTier())
		}
		if engine.GetUseType() != UseTypeCommercial {
			t.Errorf("expected use type 'commercial', got '%s'", engine.GetUseType())
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

	t.Run("community features - all features available", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:   "tenant-community",
			Tier:       "community",
			UseType:    "non_commercial",
			MaxServers: 3,
			MaxApps:    10,
			MaxUsers:   5,
			IssuedAt:   time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		// Community should have ALL features (including enterprise ones)
		if !engine.IsFeatureEnabled(FeatureSSL) {
			t.Error("community license should have ssl feature")
		}
		if !engine.IsFeatureEnabled(FeatureMonitoring) {
			t.Error("community license should have monitoring feature")
		}
		if !engine.IsFeatureEnabled(FeatureBackup) {
			t.Error("community license should have backup feature")
		}
		if !engine.IsFeatureEnabled(FeatureOAuth2) {
			t.Error("community license should have oauth2 feature")
		}
		if !engine.IsFeatureEnabled(FeatureSSO) {
			t.Error("community license should have sso feature")
		}
		if !engine.IsFeatureEnabled(FeatureLDAP) {
			t.Error("community license should have ldap feature")
		}
		if !engine.IsFeatureEnabled(FeatureDashboardTV) {
			t.Error("community license should have dashboard_tv feature")
		}
		if !engine.IsFeatureEnabled(FeatureSLA) {
			t.Error("community license should have sla feature")
		}
	})

	t.Run("team features - excludes dashboard_tv, sso, ldap, multi_tenant, federation", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:   "tenant-team",
			Tier:       "team",
			UseType:    "commercial",
			MaxServers: 10,
			MaxApps:    30,
			MaxUsers:   15,
			IssuedAt:   time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		// Team should have basic features
		if !engine.IsFeatureEnabled(FeatureSSL) {
			t.Error("team license should have ssl feature")
		}
		if !engine.IsFeatureEnabled(FeatureOAuth2) {
			t.Error("team license should have oauth2 feature")
		}
		if !engine.IsFeatureEnabled(FeatureCustomBranding) {
			t.Error("team license should have custom_branding feature")
		}
		if !engine.IsFeatureEnabled(FeaturePrioritySupport) {
			t.Error("team license should have priority_support feature")
		}

		// Team should NOT have excluded features
		if engine.IsFeatureEnabled(FeatureDashboardTV) {
			t.Error("team license should not have dashboard_tv feature")
		}
		if engine.IsFeatureEnabled(FeatureSSO) {
			t.Error("team license should not have sso feature")
		}
		if engine.IsFeatureEnabled(FeatureLDAP) {
			t.Error("team license should not have ldap feature")
		}
		if engine.IsFeatureEnabled(FeatureMultiTenant) {
			t.Error("team license should not have multi_tenant feature")
		}
		if engine.IsFeatureEnabled(FeatureFederation) {
			t.Error("team license should not have federation feature")
		}
	})

	t.Run("pro features - excludes only sla", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:   "tenant-pro",
			Tier:       "pro",
			UseType:    "commercial",
			MaxServers: 50,
			MaxApps:    100,
			MaxUsers:   50,
			IssuedAt:   time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		// Pro should have most features
		if !engine.IsFeatureEnabled(FeatureSSL) {
			t.Error("pro license should have ssl feature")
		}
		if !engine.IsFeatureEnabled(FeatureOAuth2) {
			t.Error("pro license should have oauth2 feature")
		}
		if !engine.IsFeatureEnabled(FeatureCluster) {
			t.Error("pro license should have cluster feature")
		}
		if !engine.IsFeatureEnabled(FeatureSSO) {
			t.Error("pro license should have sso feature")
		}
		if !engine.IsFeatureEnabled(FeatureDashboardTV) {
			t.Error("pro license should have dashboard_tv feature")
		}

		// Pro should NOT have SLA
		if engine.IsFeatureEnabled(FeatureSLA) {
			t.Error("pro license should not have sla feature")
		}
	})

	t.Run("enterprise features - all features", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:   "tenant-enterprise",
			Tier:       "enterprise",
			UseType:    "commercial",
			MaxServers: 100,
			MaxApps:    500,
			MaxUsers:   100,
			IssuedAt:   time.Now().Unix(),
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
		if !engine.IsFeatureEnabled(FeatureSLA) {
			t.Error("enterprise license should have sla feature")
		}
		if !engine.IsFeatureEnabled(FeatureDashboardTV) {
			t.Error("enterprise license should have dashboard_tv feature")
		}
	})

	t.Run("addon features work", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:   "tenant-addon",
			Tier:       "team",
			UseType:    "commercial",
			MaxServers: 10,
			MaxApps:    30,
			MaxUsers:   15,
			IssuedAt:   time.Now().Unix(),
			Addons: []Addon{
				{
					Key:         "feature:sso",
					Amount:      0,
					PurchasedAt: time.Now().Unix(),
					ExpiresAt:   time.Now().Add(30 * 24 * time.Hour).Unix(),
				},
			},
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		// SSO should be available via addon
		if !engine.IsFeatureEnabled(FeatureSSO) {
			t.Error("team license with sso addon should have sso feature")
		}
		// Other excluded features should still be unavailable
		if engine.IsFeatureEnabled(FeatureLDAP) {
			t.Error("team license without ldap addon should not have ldap feature")
		}
	})

	t.Run("require feature callback", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:   "tenant-cb",
			Tier:       "team",
			UseType:    "non_commercial",
			MaxServers: 10,
			MaxApps:    30,
			MaxUsers:   15,
			IssuedAt:   time.Now().Unix(),
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
			t.Fatal("expected error for requiring SSO with team license (no addon)")
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
		TenantID:   "tenant-limits",
		Tier:       "pro",
		UseType:    "commercial",
		MaxServers: 5,
		MaxApps:    20,
		MaxUsers:   10,
		IssuedAt:   time.Now().Unix(),
	}
	key, err := GenerateLicenseKey(priv, data)
	if err != nil {
		t.Fatalf("failed to generate license key: %v", err)
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

	t.Run("addon resources are added", func(t *testing.T) {
		engine2 := NewEngine(pub, 7)
		data2 := LicenseData{
			TenantID:   "tenant-addon-limits",
			Tier:       "team",
			UseType:    "commercial",
			MaxServers: 10,
			MaxApps:    30,
			MaxUsers:   15,
			IssuedAt:   time.Now().Unix(),
			Addons: []Addon{
				{
					Key:         "resource:servers",
					Amount:      5,
					PurchasedAt: time.Now().Unix(),
					ExpiresAt:   time.Now().Add(30 * 24 * time.Hour).Unix(),
				},
				{
					Key:         "resource:apps",
					Amount:      20,
					PurchasedAt: time.Now().Unix(),
					ExpiresAt:   time.Now().Add(30 * 24 * time.Hour).Unix(),
				},
			},
		}
		key2, err := GenerateLicenseKey(priv, data2)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine2.LoadLicense(key2); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		s, a, u := engine2.GetLimits()
		if s != 15 || a != 50 || u != 15 {
			t.Errorf("expected limits (15, 50, 15), got (%d, %d, %d)", s, a, u)
		}
	})
}

func TestAddonPausedByTier(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	engine := NewEngine(pub, 7)
	data := LicenseData{
		TenantID:   "tenant-paused",
		Tier:       "team",
		UseType:    "commercial",
		MaxServers: 10,
		MaxApps:    30,
		MaxUsers:   15,
		IssuedAt:   time.Now().Unix(),
		Addons: []Addon{
			{
				Key:         "feature:sso",
				Amount:      0,
				PurchasedAt: time.Now().Add(-60 * 24 * time.Hour).Unix(),
				ExpiresAt:   time.Now().Add(30 * 24 * time.Hour).Unix(),
				PausedAt:    time.Now().Add(-10 * 24 * time.Hour).Unix(),
				PausedDays:  10,
			},
			{
				Key:         "resource:servers",
				Amount:      5,
				PurchasedAt: time.Now().Unix(),
				ExpiresAt:   time.Now().Add(30 * 24 * time.Hour).Unix(),
				PausedAt:    time.Now().Unix(),
				PausedDays:  0,
			},
		},
	}
	key, err := GenerateLicenseKey(priv, data)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	if err := engine.LoadLicense(key); err != nil {
		t.Fatalf("failed to load license: %v", err)
	}

	// Paused addon feature should NOT be available
	if engine.IsFeatureEnabled(FeatureSSO) {
		t.Error("paused addon should not provide sso feature")
	}

	// Paused addon resource should NOT be counted
	s, _, _ := engine.GetLimits()
	if s != 10 {
		t.Errorf("paused resource addon should not count, expected 10, got %d", s)
	}
}

func TestAddonExpired(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	engine := NewEngine(pub, 7)
	data := LicenseData{
		TenantID:   "tenant-expired-addon",
		Tier:       "team",
		UseType:    "commercial",
		MaxServers: 10,
		MaxApps:    30,
		MaxUsers:   15,
		IssuedAt:   time.Now().Unix(),
		Addons: []Addon{
			{
				Key:         "feature:sso",
				Amount:      0,
				PurchasedAt: time.Now().Add(-60 * 24 * time.Hour).Unix(),
				ExpiresAt:   time.Now().Add(-1 * 24 * time.Hour).Unix(), // expired yesterday
			},
			{
				Key:         "resource:servers",
				Amount:      5,
				PurchasedAt: time.Now().Add(-60 * 24 * time.Hour).Unix(),
				ExpiresAt:   time.Now().Add(-1 * 24 * time.Hour).Unix(), // expired yesterday
			},
		},
	}
	key, err := GenerateLicenseKey(priv, data)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	if err := engine.LoadLicense(key); err != nil {
		t.Fatalf("failed to load license: %v", err)
	}

	// Expired addon feature should NOT be available
	if engine.IsFeatureEnabled(FeatureSSO) {
		t.Error("expired addon should not provide sso feature")
	}

	// Expired addon resource should NOT be counted
	s, _, _ := engine.GetLimits()
	if s != 10 {
		t.Errorf("expired resource addon should not count, expected 10, got %d", s)
	}
}

func TestUseType(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("commercial license", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:   "tenant-commercial",
			Tier:       "pro",
			UseType:    "commercial",
			MaxServers: 50,
			MaxApps:    100,
			MaxUsers:   50,
			IssuedAt:   time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		if engine.GetUseType() != UseTypeCommercial {
			t.Errorf("expected use type 'commercial', got '%s'", engine.GetUseType())
		}
		if !engine.IsCommercial() {
			t.Error("expected IsCommercial() to return true")
		}
	})

	t.Run("non-commercial license", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		data := LicenseData{
			TenantID:   "tenant-noncommercial",
			Tier:       "community",
			UseType:    "non_commercial",
			MaxServers: 3,
			MaxApps:    10,
			MaxUsers:   5,
			IssuedAt:   time.Now().Unix(),
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if err := engine.LoadLicense(key); err != nil {
			t.Fatalf("failed to load license: %v", err)
		}

		if engine.GetUseType() != UseTypeNonCommercial {
			t.Errorf("expected use type 'non_commercial', got '%s'", engine.GetUseType())
		}
		if engine.IsCommercial() {
			t.Error("expected IsCommercial() to return false")
		}
	})

	t.Run("no license defaults to non-commercial", func(t *testing.T) {
		engine := NewEngine(pub, 7)
		if engine.GetUseType() != UseTypeNonCommercial {
			t.Errorf("expected default use type 'non_commercial', got '%s'", engine.GetUseType())
		}
		if engine.IsCommercial() {
			t.Error("expected IsCommercial() to return false for no license")
		}
	})
}

func TestTierResolution(t *testing.T) {
	// Test resolveTierFeatures directly
	t.Run("community tier has all features", func(t *testing.T) {
		features := resolveTierFeatures("community")
		if len(features) != len(AllFeatures) {
			t.Errorf("expected %d features, got %d", len(AllFeatures), len(features))
		}
		for _, f := range AllFeatures {
			if !features[f] {
				t.Errorf("community tier should have feature %s", f)
			}
		}
	})

	t.Run("team tier excludes correct features", func(t *testing.T) {
		features := resolveTierFeatures("team")
		for f := range TeamExcludedFeatures {
			if features[f] {
				t.Errorf("team tier should not have feature %s", f)
			}
		}
		// Should have features not in excluded list
		if !features[FeatureSSL] {
			t.Error("team tier should have ssl feature")
		}
		if !features[FeatureCustomBranding] {
			t.Error("team tier should have custom_branding feature")
		}
	})

	t.Run("pro tier excludes only sla", func(t *testing.T) {
		features := resolveTierFeatures("pro")
		if features[FeatureSLA] {
			t.Error("pro tier should not have sla feature")
		}
		// Should have everything else
		if !features[FeatureSSO] {
			t.Error("pro tier should have sso feature")
		}
		if !features[FeatureDashboardTV] {
			t.Error("pro tier should have dashboard_tv feature")
		}
	})

	t.Run("enterprise tier has all features", func(t *testing.T) {
		features := resolveTierFeatures("enterprise")
		if len(features) != len(AllFeatures) {
			t.Errorf("expected %d features, got %d", len(AllFeatures), len(features))
		}
		for _, f := range AllFeatures {
			if !features[f] {
				t.Errorf("enterprise tier should have feature %s", f)
			}
		}
	})

	t.Run("unknown tier defaults to community features", func(t *testing.T) {
		features := resolveTierFeatures("unknown")
		if len(features) != len(AllFeatures) {
			t.Errorf("expected %d features for unknown tier, got %d", len(AllFeatures), len(features))
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
		TenantID:   "tenant-expired",
		Tier:       "pro",
		UseType:    "commercial",
		MaxServers: 10,
		MaxApps:    50,
		MaxUsers:   20,
		IssuedAt:   time.Now().Add(-30 * 24 * time.Hour).Unix(),
		ExpiresAt:  time.Now().Add(-10 * 24 * time.Hour).Unix(),
	}

	key, err := GenerateLicenseKey(priv, data)
	if err != nil {
		t.Fatalf("failed to generate license key: %v", err)
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

	t.Run("get tier without license", func(t *testing.T) {
		if tier := engine.GetTier(); tier != TierCommunity {
			t.Errorf("expected TierCommunity, got '%s'", tier)
		}
	})

	t.Run("get limits without license", func(t *testing.T) {
		s, a, u := engine.GetLimits()
		if s != 3 || a != 10 || u != 5 {
			t.Errorf("expected community defaults (3, 10, 5), got (%d, %d, %d)", s, a, u)
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
			TenantID:   "tenant-grace",
			Tier:       "pro",
			UseType:    "commercial",
			MaxServers: 10,
			MaxApps:    50,
			MaxUsers:   20,
			IssuedAt:   time.Now().Add(-30 * 24 * time.Hour).Unix(),
			ExpiresAt:  time.Now().Add(-3 * 24 * time.Hour).Unix(),
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
			TenantID:   "tenant-grace-expired",
			Tier:       "pro",
			UseType:    "commercial",
			MaxServers: 10,
			MaxApps:    50,
			MaxUsers:   20,
			IssuedAt:   time.Now().Add(-30 * 24 * time.Hour).Unix(),
			ExpiresAt:  time.Now().Add(-10 * 24 * time.Hour).Unix(),
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
			TenantID:   "tenant-no-grace",
			Tier:       "pro",
			UseType:    "commercial",
			MaxServers: 10,
			MaxApps:    50,
			MaxUsers:   20,
			IssuedAt:   time.Now().Add(-30 * 24 * time.Hour).Unix(),
			ExpiresAt:  time.Now().Add(-1 * 24 * time.Hour).Unix(),
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
			TenantID:   "tenant-perpetual",
			Tier:       "enterprise",
			UseType:    "commercial",
			MaxServers: 100,
			MaxApps:    500,
			MaxUsers:   100,
			IssuedAt:   time.Now().Add(-1 * 24 * time.Hour).Unix(),
			ExpiresAt:  0, // never expires
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
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("auto-set issued at", func(t *testing.T) {
		data := LicenseData{
			TenantID: "tenant-auto",
			Tier:     "community",
			UseType:  "non_commercial",
			IssuedAt: 0, // should be auto-set
		}
		key, err := GenerateLicenseKey(priv, data)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}

		// Parse and verify issued_at was set
		payload, _, err := parseLicenseKey(key)
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
			TenantID: "tenant-format",
			Tier:     "pro",
			UseType:  "commercial",
			IssuedAt: time.Now().Unix(),
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
		TenantID:   "tenant-info",
		Tier:       "enterprise",
		UseType:    "commercial",
		MaxServers: 100,
		MaxApps:    500,
		MaxUsers:   100,
		IssuedAt:   time.Now().Unix(),
	}
	key, err := GenerateLicenseKey(priv, data)
	if err != nil {
		t.Fatalf("failed to generate license key: %v", err)
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
	if info.Tier != TierEnterprise {
		t.Errorf("expected tier 'enterprise', got '%s'", info.Tier)
	}
	if info.UseType != UseTypeCommercial {
		t.Errorf("expected use type 'commercial', got '%s'", info.UseType)
	}

	// Verify it's a copy (modifying original should not affect the copy)
	engine.LoadLicense(key) // reload
	info.Features[FeatureSSL] = false
	if !engine.IsFeatureEnabled(FeatureSSL) {
		t.Error("modifying GetInfo copy should not affect engine state")
	}
}
