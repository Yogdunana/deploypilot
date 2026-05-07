package ssl

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CertificateProvider defines the unified interface for SSL certificate providers.
type CertificateProvider interface {
	RequestCertificate(ctx context.Context, domain, email string) (*Certificate, error)
	RenewCertificate(ctx context.Context, domain string) (*Certificate, error)
	GetCertificate(domain string) (*Certificate, error)
	DeleteCertificate(domain string) error
	CheckExpiry(domain string) (time.Duration, bool)
	GetTLSConfig(domain string) (*tls.Config, error)
}

// SSLProvider manages SSL certificates using ACME protocol.
type SSLProvider struct {
	certDir    string
	accountKey *ecdsa.PrivateKey
}

// validDomainRegex matches valid domain names (prevents path traversal)
var validDomainRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,62}(\.[a-zA-Z0-9][a-zA-Z0-9-]{0,62})*$`)

// validateDomain checks if the domain is valid and safe to use as a filename
func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	// Check for path traversal attempts
	if strings.Contains(domain, "..") || strings.Contains(domain, "/") || strings.Contains(domain, "\\") {
		return fmt.Errorf("invalid domain: contains path traversal characters")
	}
	// Check for valid domain format
	if !validDomainRegex.MatchString(domain) {
		return fmt.Errorf("invalid domain format: %s", domain)
	}
	return nil
}

// safeJoin safely joins a directory with a filename, preventing path traversal
func safeJoin(dir, filename string) (string, error) {
	// Clean the path to resolve any .. sequences
	cleanPath := filepath.Clean(filename)
	// Check for path traversal after cleaning
	if strings.HasPrefix(cleanPath, "..") || strings.HasPrefix(cleanPath, "/") || strings.HasPrefix(cleanPath, "\\") {
		return "", fmt.Errorf("path traversal detected: %s", filename)
	}
	// Join and verify the final path is within the directory
	fullPath := filepath.Join(dir, cleanPath)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for directory: %w", err)
	}
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) && absPath != absDir {
		return "", fmt.Errorf("path escapes directory: %s", filename)
	}
	return fullPath, nil
}

// NewSSLProvider creates a new SSL provider.
func NewSSLProvider(certDir string) (CertificateProvider, error) {
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, fmt.Errorf("create cert directory: %w", err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate account key: %w", err)
	}

	return &SSLProvider{
		certDir:    certDir,
		accountKey: key,
	}, nil
}

// Certificate represents an SSL certificate.
type Certificate struct {
	Domain    string
	CertPEM   []byte
	KeyPEM    []byte
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// RequestCertificate requests a new certificate for the given domain.
// In production this would use ACME (e.g., go-acme/lego library).
// For now, it generates a self-signed certificate as a placeholder.
func (p *SSLProvider) RequestCertificate(ctx context.Context, domain, email string) (*Certificate, error) {
	_ = ctx
	_ = email
	slog.Info("requesting SSL certificate", "domain", domain, "email", email)

	// Validate domain to prevent path traversal
	if err := validateDomain(domain); err != nil {
		return nil, fmt.Errorf("invalid domain: %w", err)
	}

	// Generate self-signed certificate as placeholder
	// In production, replace with ACME flow using go-acme/lego
	cert, key, err := p.generateSelfSigned(domain)
	if err != nil {
		return nil, fmt.Errorf("generate certificate: %w", err)
	}

	// Save to disk using safe path joining
	certPath, err := safeJoin(p.certDir, domain+".crt")
	if err != nil {
		return nil, fmt.Errorf("invalid cert path: %w", err)
	}
	keyPath, err := safeJoin(p.certDir, domain+".key")
	if err != nil {
		return nil, fmt.Errorf("invalid key path: %w", err)
	}

	if err := os.WriteFile(certPath, cert, 0600); err != nil {
		return nil, fmt.Errorf("write cert: %w", err)
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		_ = os.Remove(certPath)
		return nil, fmt.Errorf("write key: %w", err)
	}

	now := time.Now()
	return &Certificate{
		Domain:    domain,
		CertPEM:   cert,
		KeyPEM:    key,
		IssuedAt:  now,
		ExpiresAt: now.Add(90 * 24 * time.Hour), // 90 days
	}, nil
}

// RenewCertificate renews an existing certificate.
func (p *SSLProvider) RenewCertificate(ctx context.Context, domain string) (*Certificate, error) {
	slog.Info("renewing SSL certificate", "domain", domain)
	// Validate domain to prevent path traversal
	if err := validateDomain(domain); err != nil {
		return nil, fmt.Errorf("invalid domain: %w", err)
	}
	return p.RequestCertificate(ctx, domain, "")
}

// GetCertificate loads a certificate from disk.
func (p *SSLProvider) GetCertificate(domain string) (*Certificate, error) {
	// Validate domain to prevent path traversal
	if err := validateDomain(domain); err != nil {
		return nil, fmt.Errorf("invalid domain: %w", err)
	}

	certPath, err := safeJoin(p.certDir, domain+".crt")
	if err != nil {
		return nil, fmt.Errorf("invalid cert path: %w", err)
	}
	keyPath, err := safeJoin(p.certDir, domain+".key")
	if err != nil {
		return nil, fmt.Errorf("invalid key path: %w", err)
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("decode cert PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse cert: %w", err)
	}

	return &Certificate{
		Domain:    domain,
		CertPEM:   certPEM,
		KeyPEM:    keyPEM,
		IssuedAt:  cert.NotBefore,
		ExpiresAt: cert.NotAfter,
	}, nil
}

// DeleteCertificate removes certificate files from disk.
func (p *SSLProvider) DeleteCertificate(domain string) error {
	// Validate domain to prevent path traversal
	if err := validateDomain(domain); err != nil {
		return fmt.Errorf("invalid domain: %w", err)
	}

	certPath, err := safeJoin(p.certDir, domain+".crt")
	if err != nil {
		return fmt.Errorf("invalid cert path: %w", err)
	}
	keyPath, err := safeJoin(p.certDir, domain+".key")
	if err != nil {
		return fmt.Errorf("invalid key path: %w", err)
	}

	_ = os.Remove(certPath)
	_ = os.Remove(keyPath)
	slog.Info("deleted SSL certificate", "domain", domain)
	return nil
}

// CheckExpiry checks if a certificate is expiring soon.
func (p *SSLProvider) CheckExpiry(domain string) (time.Duration, bool) {
	// Validate domain to prevent path traversal
	if err := validateDomain(domain); err != nil {
		slog.Warn("invalid domain in CheckExpiry", "domain", domain, "error", err)
		return 0, true
	}
	cert, err := p.GetCertificate(domain)
	if err != nil {
		return 0, true
	}
	remaining := time.Until(cert.ExpiresAt)
	return remaining, remaining < 7*24*time.Hour
}

// generateSelfSigned creates a self-signed certificate for development.
func (p *SSLProvider) generateSelfSigned(domain string) (certPEM, keyPEM []byte, err error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(90 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{domain},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	certBuf := &bytes.Buffer{}
	if err := pem.Encode(certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return nil, nil, err
	}

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}

	keyBuf := &bytes.Buffer{}
	if err := pem.Encode(keyBuf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, nil, err
	}

	return certBuf.Bytes(), keyBuf.Bytes(), nil
}

// GetTLSConfig returns a tls.Config for the given domain.
func (p *SSLProvider) GetTLSConfig(domain string) (*tls.Config, error) {
	// Validate domain to prevent path traversal
	if err := validateDomain(domain); err != nil {
		return nil, fmt.Errorf("invalid domain: %w", err)
	}
	cert, err := p.GetCertificate(domain)
	if err != nil {
		return nil, err
	}

	keyPair, err := tls.X509KeyPair(cert.CertPEM, cert.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{keyPair},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
