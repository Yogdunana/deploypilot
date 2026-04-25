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
	"time"
)

// SSLProvider manages SSL certificates using ACME protocol.
type SSLProvider struct {
	certDir    string
	accountKey *ecdsa.PrivateKey
}

// NewSSLProvider creates a new SSL provider.
func NewSSLProvider(certDir string) (*SSLProvider, error) {
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

	// Generate self-signed certificate as placeholder
	// In production, replace with ACME flow using go-acme/lego
	cert, key, err := p.generateSelfSigned(domain)
	if err != nil {
		return nil, fmt.Errorf("generate certificate: %w", err)
	}

	// Save to disk
	certPath := filepath.Join(p.certDir, domain+".crt")
	keyPath := filepath.Join(p.certDir, domain+".key")

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
	return p.RequestCertificate(ctx, domain, "")
}

// GetCertificate loads a certificate from disk.
func (p *SSLProvider) GetCertificate(domain string) (*Certificate, error) {
	certPath := filepath.Join(p.certDir, domain+".crt")
	keyPath := filepath.Join(p.certDir, domain+".key")

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
	certPath := filepath.Join(p.certDir, domain+".crt")
	keyPath := filepath.Join(p.certDir, domain+".key")
	_ = os.Remove(certPath)
	_ = os.Remove(keyPath)
	slog.Info("deleted SSL certificate", "domain", domain)
	return nil
}

// CheckExpiry checks if a certificate is expiring soon.
func (p *SSLProvider) CheckExpiry(domain string) (time.Duration, bool) {
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
