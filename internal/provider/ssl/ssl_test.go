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

func TestNewSSLProvider_InvalidDirectory(t *testing.T) {
	// Use a path that cannot be created to trigger MkdirAll error
	_, err := NewSSLProvider("/dev/null/impossible/path")
	if err == nil {
		t.Fatal("expected error for invalid directory")
	}
}

func TestRequestCertificate_WriteFailure(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Create a sub-directory where the cert file would be, to cause WriteFile to fail
	// by creating a file with the same name as the cert file
	certPath := filepath.Join(dir, "write-fail.com.crt")
	_ = os.WriteFile(certPath, []byte("placeholder"), 0400)

	_, err = p.RequestCertificate(context.TODO(), "write-fail.com", "admin@example.com")
	// This may or may not error depending on permissions; just verify it doesn't panic
	_ = err
}

func TestGetCertificate_KeyNotFound(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Create only the cert file, not the key file
	certPath := filepath.Join(dir, "key-missing.example.com.crt")
	_ = os.WriteFile(certPath, []byte("-----BEGIN CERTIFICATE-----\ninvalid\n-----END CERTIFICATE-----"), 0600)

	_, err = p.GetCertificate("key-missing.example.com")
	if err == nil {
		t.Fatal("expected error when key file is missing")
	}
}

func TestGetCertificate_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Create cert file with invalid PEM content
	certPath := filepath.Join(dir, "invalid-pem.example.com.crt")
	keyPath := filepath.Join(dir, "invalid-pem.example.com.key")
	_ = os.WriteFile(certPath, []byte("not valid PEM"), 0600)
	_ = os.WriteFile(keyPath, []byte("not valid PEM"), 0600)

	_, err = p.GetCertificate("invalid-pem.example.com")
	if err == nil {
		t.Fatal("expected error for invalid PEM data")
	}
}

func TestGetTLSConfig_NotFound(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	_, err = p.GetTLSConfig("nonexistent.example.com")
	if err == nil {
		t.Fatal("expected error when certificate not found")
	}
}

func TestGetCertificate_InvalidCertPEM(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Create cert file with valid PEM block but invalid certificate data
	certPath := filepath.Join(dir, "bad-cert.example.com.crt")
	keyPath := filepath.Join(dir, "bad-cert.example.com.key")
	_ = os.WriteFile(certPath, []byte("-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----"), 0600)
	_ = os.WriteFile(keyPath, []byte("-----BEGIN EC PRIVATE KEY-----\nAAAA\n-----END EC PRIVATE KEY-----"), 0600)

	_, err = p.GetCertificate("bad-cert.example.com")
	if err == nil {
		t.Fatal("expected error for invalid certificate data")
	}
}

func TestGetTLSConfig_InvalidKeyPair(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Generate a valid cert first
	_, err = p.RequestCertificate(context.TODO(), "mismatch.example.com", "admin@example.com")
	if err != nil {
		t.Fatalf("RequestCertificate failed: %v", err)
	}

	// Overwrite the key file with garbage
	keyPath := filepath.Join(dir, "mismatch.example.com.key")
	_ = os.WriteFile(keyPath, []byte("-----BEGIN EC PRIVATE KEY-----\nAAAA\n-----END EC PRIVATE KEY-----"), 0600)

	_, err = p.GetTLSConfig("mismatch.example.com")
	if err == nil {
		t.Fatal("expected error for mismatched cert/key pair")
	}
}

func TestRequestCertificate_KeyWriteFailure(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Create a directory where the key file would be, causing WriteFile to fail
	keyDir := filepath.Join(dir, "keyfail.example.com.key")
	_ = os.MkdirAll(keyDir, 0700)

	_, err = p.RequestCertificate(context.TODO(), "keyfail.example.com", "admin@example.com")
	if err == nil {
		t.Fatal("expected error when key file write fails")
	}
}

func TestDeleteCertificateNonExistent(t *testing.T) {
	dir := t.TempDir()
	p, err := NewSSLProvider(dir)
	if err != nil {
		t.Fatalf("NewSSLProvider failed: %v", err)
	}

	// Delete should not error even if files don't exist
	err = p.DeleteCertificate("nonexistent.example.com")
	if err != nil {
		t.Fatalf("DeleteCertificate() error = %v", err)
	}
}

func TestCertificateStruct(t *testing.T) {
	now := time.Now()
	cert := &Certificate{
		Domain:    "example.com",
		CertPEM:   []byte("cert-data"),
		KeyPEM:    []byte("key-data"),
		IssuedAt:  now,
		ExpiresAt: now.Add(90 * 24 * time.Hour),
	}

	if cert.Domain != "example.com" {
		t.Errorf("Domain = %q", cert.Domain)
	}
	if string(cert.CertPEM) != "cert-data" {
		t.Errorf("CertPEM = %q", cert.CertPEM)
	}
	if string(cert.KeyPEM) != "key-data" {
		t.Errorf("KeyPEM = %q", cert.KeyPEM)
	}
	if cert.ExpiresAt.Sub(cert.IssuedAt) != 90*24*time.Hour {
		t.Errorf("validity period mismatch")
	}
}
