package service

import (
	"context"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/model"
)

// ===================== Credential Service Tests =====================

func TestCreateCredentialWithExpiry(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.CreateCredentialWithExpiry(context.Background(), "tenant-default", "expiry-cred", "password", "secret-value", 30)
	if err != nil {
		t.Fatalf("CreateCredentialWithExpiry failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["name"] != "expiry-cred" {
		t.Errorf("expected name=expiry-cred, got %v", m["name"])
	}
	if m["rotation_days"] != 30 {
		t.Errorf("expected rotation_days=30, got %v", m["rotation_days"])
	}
	if _, hasExpires := m["expires_at"]; !hasExpires {
		t.Error("expected expires_at field")
	}
}

func TestCreateCredentialWithExpiry_NoExpiry(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.CreateCredentialWithExpiry(context.Background(), "tenant-default", "no-expiry-cred", "token", "value", 0)
	if err != nil {
		t.Fatalf("CreateCredentialWithExpiry failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["rotation_days"] != 0 {
		t.Errorf("expected rotation_days=0, got %v", m["rotation_days"])
	}
	if _, hasExpires := m["expires_at"]; hasExpires {
		t.Error("expected no expires_at field when expiresInDays=0")
	}
}

func TestCreateCredentialWithExpiry_NegativeDays(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.CreateCredentialWithExpiry(context.Background(), "tenant-default", "neg-expiry-cred", "ssh_key", "value", -1)
	if err != nil {
		t.Fatalf("CreateCredentialWithExpiry failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if _, hasExpires := m["expires_at"]; hasExpires {
		t.Error("expected no expires_at field for negative days")
	}
}

func TestCreateCredentialWithExpiry_EncryptionError(t *testing.T) {
	b, _ := newTestBridge(t)
	b.EncryptionKey = nil
	_, err := b.CreateCredentialWithExpiry(context.Background(), "tenant-default", "bad-cred", "password", "value", 30)
	if err == nil {
		t.Fatal("expected error with nil encryption key")
	}
}

func TestRotateCredential(t *testing.T) {
	b, _ := newTestBridge(t)
	createResult, err := b.CreateCredential(context.Background(), "tenant-default", "rotate-cred", "token", "old-value")
	if err != nil {
		t.Fatalf("CreateCredential failed: %v", err)
	}
	id := createResult.(map[string]interface{})["id"].(string)

	result, err := b.RotateCredential(context.Background(), id, "new-value")
	if err != nil {
		t.Fatalf("RotateCredential failed: %v", err)
	}
	if result != "rotate-cred" {
		t.Errorf("expected credential name, got %v", result)
	}
}

func TestRotateCredential_NotFound(t *testing.T) {
	b, _ := newTestBridge(t)
	_, err := b.RotateCredential(context.Background(), "nonexistent-id", "new-value")
	if err == nil {
		t.Fatal("expected error for nonexistent credential")
	}
}

func TestRotateCredential_EncryptionError(t *testing.T) {
	b, _ := newTestBridge(t)
	b.EncryptionKey = nil
	_, err := b.RotateCredential(context.Background(), "some-id", "new-value")
	if err == nil {
		t.Fatal("expected error with nil encryption key")
	}
}

// ===================== Server Service Tests =====================

func TestGetServersByTags(t *testing.T) {
	b, _ := newTestBridge(t)
	b.DB.Create(&model.Server{
		ID:       "srv-tag-1",
		Name:     "server1",
		TenantID: "tenant-default",
		Host:     "10.0.0.1",
		Tags:     `["prod", "web"]`,
	})
	b.DB.Create(&model.Server{
		ID:       "srv-tag-2",
		Name:     "server2",
		TenantID: "tenant-default",
		Host:     "10.0.0.2",
		Tags:     `["prod", "api"]`,
	})
	b.DB.Create(&model.Server{
		ID:       "srv-tag-3",
		Name:     "server3",
		TenantID: "tenant-default",
		Host:     "10.0.0.3",
		Tags:     `["staging", "web"]`,
	})

	result, err := b.GetServersByTags(context.Background(), "tenant-default", []string{"prod"})
	if err != nil {
		t.Fatalf("GetServersByTags failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 servers with prod tag, got %d", len(result))
	}
}

func TestGetServersByTags_CommaSeparated(t *testing.T) {
	b, _ := newTestBridge(t)
	b.DB.Create(&model.Server{
		ID:       "srv-comma-1",
		Name:     "comma-server",
		TenantID: "tenant-default",
		Host:     "10.0.0.1",
		Tags:     "prod, web, api",
	})

	result, err := b.GetServersByTags(context.Background(), "tenant-default", []string{"web"})
	if err != nil {
		t.Fatalf("GetServersByTags failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 server with web tag, got %d", len(result))
	}
}

func TestGetServersByTags_NoMatch(t *testing.T) {
	b, _ := newTestBridge(t)
	b.DB.Create(&model.Server{
		ID:       "srv-no-match",
		Name:     "no-match-server",
		TenantID: "tenant-default",
		Host:     "10.0.0.1",
		Tags:     `["prod"]`,
	})

	result, err := b.GetServersByTags(context.Background(), "tenant-default", []string{"nonexistent"})
	if err != nil {
		t.Fatalf("GetServersByTags failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 servers, got %d", len(result))
	}
}

func TestGetServersByTags_NoTags(t *testing.T) {
	b, _ := newTestBridge(t)
	b.DB.Create(&model.Server{
		ID:       "srv-empty-tags",
		Name:     "empty-tags-server",
		TenantID: "tenant-default",
		Host:     "10.0.0.1",
		Tags:     "",
	})

	result, err := b.GetServersByTags(context.Background(), "tenant-default", []string{"prod"})
	if err != nil {
		t.Fatalf("GetServersByTags failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 servers (no tags), got %d", len(result))
	}
}

func TestGetServersByTags_NilDB(t *testing.T) {
	b := &Bridge{DB: nil}
	_, err := b.GetServersByTags(context.Background(), "tenant-default", []string{"prod"})
	if err == nil {
		t.Fatal("expected error when DB is nil")
	}
}

func TestGetServersByTags_CaseInsensitive(t *testing.T) {
	b, _ := newTestBridge(t)
	b.DB.Create(&model.Server{
		ID:       "srv-case-1",
		Name:     "case-server",
		TenantID: "tenant-default",
		Host:     "10.0.0.1",
		Tags:     `["Production", "Web"]`,
	})

	result, err := b.GetServersByTags(context.Background(), "tenant-default", []string{"production"})
	if err != nil {
		t.Fatalf("GetServersByTags failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 server (case insensitive match), got %d", len(result))
	}
}

// ===================== App Service Tests =====================

func TestGetAppDeploymentHistory_Coverage_Limit(t *testing.T) {
	b, _ := newTestBridge(t)
	id, err := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "limit-history-app", RepoURL: "https://x.com/x"})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}

	b.DB.Create(&model.DeploymentRecord{ID: "dep-l1", AppID: id, Status: "success"})
	b.DB.Create(&model.DeploymentRecord{ID: "dep-l2", AppID: id, Status: "success"})
	b.DB.Create(&model.DeploymentRecord{ID: "dep-l3", AppID: id, Status: "success"})

	records, err := b.GetAppDeploymentHistory(context.Background(), id, 2)
	if err != nil {
		t.Fatalf("GetAppDeploymentHistory failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records (limited), got %d", len(records))
	}
}

func TestGetAppDeploymentHistory_Coverage_DefaultLimit(t *testing.T) {
	b, _ := newTestBridge(t)
	id, err := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "default-limit-app", RepoURL: "https://x.com/x"})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}

	records, err := b.GetAppDeploymentHistory(context.Background(), id, 0)
	if err != nil {
		t.Fatalf("GetAppDeploymentHistory failed: %v", err)
	}
	if records == nil {
		t.Error("expected non-nil records")
	}
}

func TestGetAppDeploymentHistory_Coverage_NilDB(t *testing.T) {
	b := &Bridge{DB: nil}
	_, err := b.GetAppDeploymentHistory(context.Background(), "app-id", 10)
	if err == nil {
		t.Fatal("expected error when DB is nil")
	}
}

func TestGetAppDeploymentHistory_Coverage_InvalidLimit(t *testing.T) {
	b, _ := newTestBridge(t)
	id, err := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "invalid-limit-app", RepoURL: "https://x.com/x"})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}

	records, err := b.GetAppDeploymentHistory(context.Background(), id, 200)
	if err != nil {
		t.Fatalf("GetAppDeploymentHistory failed: %v", err)
	}
	if records != nil && len(records) > 100 {
		t.Errorf("expected max 100 records, got %d", len(records))
	}
}

func TestGetAppDeploymentHistory_Coverage_NoRecords(t *testing.T) {
	b, _ := newTestBridge(t)
	id, err := b.CreateApp(context.Background(), mcp.CreateAppConfig{Name: "no-records-app", RepoURL: "https://x.com/x"})
	if err != nil {
		t.Fatalf("CreateApp failed: %v", err)
	}

	records, err := b.GetAppDeploymentHistory(context.Background(), id, 10)
	if err != nil {
		t.Fatalf("GetAppDeploymentHistory failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

// ===================== SSL Service Tests =====================

func TestRequestSSLCertificate_Coverage(t *testing.T) {
	b, _ := newTestBridge(t)
	result, err := b.RequestSSLCertificate(context.Background(), "test.example.com", "admin@test.com")
	if err != nil {
		t.Fatalf("RequestSSLCertificate failed: %v", err)
	}
	cert, ok := result.(model.SSLCertificate)
	if !ok {
		t.Fatal("expected SSLCertificate")
	}
	if cert.Domain != "test.example.com" {
		t.Errorf("expected domain=test.example.com, got %s", cert.Domain)
	}
	if cert.Status != "pending" {
		t.Errorf("expected status=pending, got %s", cert.Status)
	}
}

func TestRenewSSLCertificate_Coverage(t *testing.T) {
	b, _ := newTestBridge(t)
	b.DB.Create(&model.SSLCertificate{
		Domain:      "renew.example.com",
		Email:       "admin@example.com",
		Status:      "active",
		RetryCount: 0,
	})

	result, err := b.RenewSSLCertificate(context.Background(), "renew.example.com")
	if err != nil {
		t.Fatalf("RenewSSLCertificate failed: %v", err)
	}
	cert, ok := result.(model.SSLCertificate)
	if !ok {
		t.Fatal("expected SSLCertificate")
	}
	if cert.Status != "renewing" {
		t.Errorf("expected status=renewing, got %s", cert.Status)
	}
	if cert.RetryCount != 1 {
		t.Errorf("expected retry_count=1, got %d", cert.RetryCount)
	}
}

func TestDeleteSSLCertificate_Coverage(t *testing.T) {
	b, _ := newTestBridge(t)
	b.DB.Create(&model.SSLCertificate{
		Domain: "delete.example.com",
		Email:  "admin@example.com",
		Status: "active",
	})

	result, err := b.DeleteSSLCertificate(context.Background(), "delete.example.com")
	if err != nil {
		t.Fatalf("DeleteSSLCertificate failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map")
	}
	if m["message"] != "SSL certificate deleted" {
		t.Errorf("expected message, got %v", m["message"])
	}
}

// ===================== System Service Tests =====================

func TestHealContainer_NilHealer(t *testing.T) {
	b, _ := newTestBridge(t)
	b.healer = nil
	_, err := b.HealContainer(context.Background(), "test-container")
	if err == nil {
		t.Fatal("expected error when healer is nil")
	}
}

func TestPerformSystemUpdate_NilUpgradeSvc(t *testing.T) {
	b, _ := newTestBridge(t)
	b.UpgradeSvc = nil
	_, err := b.PerformSystemUpdate(context.Background())
	if err == nil {
		t.Fatal("expected error when UpgradeSvc is nil")
	}
}