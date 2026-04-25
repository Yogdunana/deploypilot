package ssl

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSSLProvider(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(filepath.Join(dir, "certs"))
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.certDir == "" {
		t.Error("expected certDir to be set")
	}
	if p.accountKey == nil {
		t.Error("expected accountKey to be set")
	}

	// Verify directory was created
	info, err := os.Stat(p.certDir)
	if err != nil {
		t.Fatalf("cert directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected certDir to be a directory")
	}
}

func TestRequestCertificate(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	cert, err := p.RequestCertificate(context.TODO(), "example.com", "admin@example.com")
	if err != nil {
		t.Fatalf("RequestCertificate failed: %v", err)
	}
	if cert.Domain != "example.com" {
		t.Errorf("expected domain example.com, got %s", cert.Domain)
	}
	if len(cert.CertPEM) == 0 {
		t.Error("expected non-empty cert PEM")
	}
	if len(cert.KeyPEM) == 0 {
		t.Error("expected non-empty key PEM")
	}
	if cert.IssuedAt.IsZero() {
		t.Error("expected non-zero IssuedAt")
	}
	if cert.ExpiresAt.IsZero() {
		t.Error("expected non-zero ExpiresAt")
	}

	// Verify files were created
	certPath := filepath.Join(dir, "example.com.crt")
	keyPath := filepath.Join(dir, "example.com.key")
	if _, err := os.Stat(certPath); err != nil {
		t.Errorf("cert file not created: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("key file not created: %v", err)
	}
}

func TestGetCertificate(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Request a cert first
	_, err = p.RequestCertificate(context.TODO(), "test.example.com", "admin@example.com")
	if err != nil {
		t.Fatalf("RequestCertificate failed: %v", err)
	}

	// Get the cert
	cert, err := p.GetCertificate("test.example.com")
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if cert.Domain != "test.example.com" {
		t.Errorf("expected domain test.example.com, got %s", cert.Domain)
	}
	if cert.CertPEM == nil {
		t.Error("expected non-nil cert PEM")
	}
	if cert.KeyPEM == nil {
		t.Error("expected non-nil key PEM")
	}
}

func TestGetCertificate_NotFound(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	_, err = p.GetCertificate("nonexistent.example.com")
	if err == nil {
		t.Fatal("expected error for nonexistent certificate")
	}
}

func TestDeleteCertificate(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Request a cert first
	_, err = p.RequestCertificate(context.TODO(), "delete.example.com", "admin@example.com")
	if err != nil {
		t.Fatalf("RequestCertificate failed: %v", err)
	}

	// Delete it
	err = p.DeleteCertificate("delete.example.com")
	if err != nil {
		t.Fatalf("DeleteCertificate failed: %v", err)
	}

	// Verify files are gone
	certPath := filepath.Join(dir, "delete.example.com.crt")
	if _, err := os.Stat(certPath); !os.IsNotExist(err) {
		t.Error("expected cert file to be deleted")
	}
}

func TestCheckExpiry(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Request a cert (valid for 90 days)
	_, err = p.RequestCertificate(context.TODO(), "expiry.example.com", "admin@example.com")
	if err != nil {
		t.Fatalf("RequestCertificate failed: %v", err)
	}

	remaining, expiringSoon := p.CheckExpiry("expiry.example.com")
	if expiringSoon {
		t.Error("expected certificate not to be expiring soon (90 days)")
	}
	if remaining < 80*24*time.Hour {
		t.Errorf("expected remaining time > 80 days, got %v", remaining)
	}
}

func TestCheckExpiry_NotFound(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	remaining, expiringSoon := p.CheckExpiry("nonexistent.example.com")
	if !expiringSoon {
		t.Error("expected nonexistent cert to be flagged as expiring")
	}
	if remaining != 0 {
		t.Errorf("expected 0 remaining for nonexistent cert, got %v", remaining)
	}
}

func TestRenewCertificate(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Request initial cert
	_, err = p.RequestCertificate(context.TODO(), "renew.example.com", "admin@example.com")
	if err != nil {
		t.Fatalf("RequestCertificate failed: %v", err)
	}

	// Renew
	cert, err := p.RenewCertificate(context.TODO(), "renew.example.com")
	if err != nil {
		t.Fatalf("RenewCertificate failed: %v", err)
	}
	if cert.Domain != "renew.example.com" {
		t.Errorf("expected domain renew.example.com, got %s", cert.Domain)
	}
	if cert.CertPEM == nil {
		t.Error("expected non-nil cert PEM after renewal")
	}
}

func TestGetTLSConfig(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	_, err = p.RequestCertificate(context.TODO(), "tls.example.com", "admin@example.com")
	if err != nil {
		t.Fatalf("RequestCertificate failed: %v", err)
	}

	tlsCfg, err := p.GetTLSConfig("tls.example.com")
	if err != nil {
		t.Fatalf("GetTLSConfig failed: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil tls.Config")
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(tlsCfg.Certificates))
	}
}
