package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/provider/dns"
)

// getDNSProvider loads the enabled DNS provider from the database and returns a DNSProvider interface.
func (b *Bridge) getDNSProvider(ctx context.Context) (dns.DNSProvider, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	var provider model.Provider
	err := b.DB.Where("type LIKE ? AND enabled = ?", "dns-%", true).First(&provider).Error
	if err != nil {
		return nil, fmt.Errorf("no enabled DNS provider found: %w", err)
	}
	// Parse config JSON
	var cfg struct {
		APIToken        string `json:"api_token"`
		AccountEmail    string `json:"account_email"`
		AccessKeyID     string `json:"access_key_id"`
		AccessKeySecret string `json:"access_key_secret"`
		SecretID        string `json:"secret_id"`
		SecretKey       string `json:"secret_key"`
	}
	if err := json.Unmarshal([]byte(provider.Config), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse DNS provider config: %w", err)
	}

	switch provider.Type {
	case "dns-cloudflare":
		return dns.NewCloudflareProvider(cfg.APIToken, cfg.AccountEmail), nil
	case "dns-aliyun":
		return dns.NewAliyunProvider(cfg.AccessKeyID, cfg.AccessKeySecret), nil
	case "dns-tencent":
		return dns.NewTencentProvider(cfg.SecretID, cfg.SecretKey), nil
	case "dns-west-dns":
		return dns.NewWestDNSProvider(cfg.APIToken, cfg.AccessKeySecret), nil
	default:
		return nil, fmt.Errorf("unsupported DNS provider type: %s", provider.Type)
	}
}

// ---------- 19. DNSCreateRecord ----------

func (b *Bridge) DNSCreateRecord(ctx context.Context, domain, recordType, name, value string) (interface{}, error) {
	provider, err := b.getDNSProvider(ctx)
	if err != nil {
		slog.Error("DNS provider error", "error", err)
		return map[string]interface{}{
			"status":  "error",
			"domain":  domain,
			"type":    recordType,
			"name":    name,
			"value":   value,
			"message": err.Error(),
		}, nil
	}
	record := &dns.DNSRecord{
		Domain: domain,
		Type:   recordType,
		Name:   name,
		Value:  value,
		TTL:    1,
	}
	if err := provider.CreateRecord(ctx, record); err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		}, nil
	}
	return map[string]interface{}{
		"status": "success",
		"domain": domain,
		"type":   recordType,
		"name":   name,
		"value":  value,
	}, nil
}

// ---------- 20. DNSDeleteRecord ----------

func (b *Bridge) DNSDeleteRecord(ctx context.Context, recordID string) error {
	provider, err := b.getDNSProvider(ctx)
	if err != nil {
		return err
	}
	// recordID format: "domain:type:name"
	parts := strings.SplitN(recordID, ":", 3)
	if len(parts) != 3 {
		return fmt.Errorf("invalid record ID format, expected domain:type:name")
	}
	return provider.DeleteRecord(ctx, parts[0], parts[1], parts[2])
}

// ---------- 21. DNSListRecords ----------

func (b *Bridge) DNSListRecords(ctx context.Context, domain string) (interface{}, error) {
	provider, err := b.getDNSProvider(ctx)
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"domain":  domain,
			"message": err.Error(),
		}, nil
	}
	records, err := provider.ListRecords(ctx, domain)
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		}, nil
	}
	// Convert to response format
	result := make([]map[string]interface{}, 0, len(records))
	for _, r := range records {
		result = append(result, map[string]interface{}{
			"domain":  r.Domain,
			"type":    r.Type,
			"name":    r.Name,
			"value":   r.Value,
			"ttl":     r.TTL,
			"proxied": r.Proxied,
		})
	}
	return map[string]interface{}{
		"status":  "success",
		"domain":  domain,
		"records": result,
	}, nil
}

// ---------- 30. UpdateDNSRecord ----------

func (b *Bridge) UpdateDNSRecord(ctx context.Context, domain, subdomain, recordType, newValue string) (interface{}, error) {
	provider, err := b.getDNSProvider(ctx)
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		}, nil
	}
	record := &dns.DNSRecord{
		Domain: domain,
		Type:   recordType,
		Name:   subdomain,
		Value:  newValue,
		TTL:    1,
	}
	if err := provider.UpdateRecord(ctx, record); err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		}, nil
	}
	return map[string]interface{}{
		"status":    "success",
		"domain":    domain,
		"subdomain": subdomain,
		"type":      recordType,
		"value":     newValue,
	}, nil
}

// ---------- 38. BatchDNS ----------

func (b *Bridge) BatchDNS(ctx context.Context, records []map[string]interface{}) (interface{}, error) {
	results := make([]map[string]interface{}, 0, len(records))
	for i, rec := range records {
		domain := toStringOrDefault(rec["domain"], "")
		subdomain := toStringOrDefault(rec["subdomain"], "")
		recordType := toStringOrDefault(rec["type"], "")
		value := toStringOrDefault(rec["value"], "")

		res, err := b.DNSCreateRecord(ctx, domain, recordType, subdomain, value)
		status := "success"
		if err != nil {
			status = "error"
		} else if m, ok := res.(map[string]interface{}); ok && m["status"] == "error" {
			status = "error"
		}
		results = append(results, map[string]interface{}{
			"index":  i,
			"status": status,
			"record": res,
		})
	}
	return map[string]interface{}{
		"total":   len(records),
		"results": results,
	}, nil
}
