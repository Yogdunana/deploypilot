package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
)

// ---------- 47. ListSSLCertificates ----------

func (b *Bridge) ListSSLCertificates(ctx context.Context) (interface{}, error) {
	_ = ctx
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	var certs []model.SSLCertificate
	if err := b.DB.Find(&certs).Error; err != nil {
		return nil, fmt.Errorf("failed to list SSL certificates: %w", err)
	}
	return certs, nil
}

// ---------- 48. RequestSSLCertificate ----------

func (b *Bridge) RequestSSLCertificate(ctx context.Context, domain, email string) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	cert := model.SSLCertificate{
		Domain:    domain,
		Email:     email,
		Provider:  "cloudflare",
		Status:    "pending",
		AutoRenew: true,
	}
	if err := b.DB.Create(&cert).Error; err != nil {
		return nil, fmt.Errorf("failed to create SSL certificate record: %w", err)
	}
	return cert, nil
}

// ---------- 49. RenewSSLCertificate ----------

func (b *Bridge) RenewSSLCertificate(ctx context.Context, domain string) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	var cert model.SSLCertificate
	if err := b.DB.Where("domain = ?", domain).First(&cert).Error; err != nil {
		return nil, fmt.Errorf("SSL certificate not found for domain %s: %w", domain, err)
	}
	now := time.Now()
	cert.Status = "renewing"
	cert.RetryCount++
	cert.LastRenewed = &now
	if err := b.DB.Save(&cert).Error; err != nil {
		return nil, fmt.Errorf("failed to update SSL certificate: %w", err)
	}
	return cert, nil
}

// ---------- 50. DeleteSSLCertificate ----------

func (b *Bridge) DeleteSSLCertificate(ctx context.Context, domain string) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	result := b.DB.Where("domain = ?", domain).Delete(&model.SSLCertificate{})
	if result.Error != nil {
		return nil, fmt.Errorf("failed to delete SSL certificate: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("SSL certificate not found for domain %s", domain)
	}
	return map[string]interface{}{
		"message": "SSL certificate deleted",
		"domain":  domain,
	}, nil
}
