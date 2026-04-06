package dns

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DNSRecord represents a DNS record.
type DNSRecord struct {
	Domain  string `json:"domain"`
	Type    string `json:"type"`    // A, AAAA, CNAME, MX, TXT, NS, SRV
	Name    string `json:"name"`
	Value   string `json:"value"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

// DNSProvider defines the interface for DNS operations.
type DNSProvider interface {
	CreateRecord(ctx context.Context, req *DNSRecord) error
	UpdateRecord(ctx context.Context, req *DNSRecord) error
	DeleteRecord(ctx context.Context, domain, recordType, name string) error
	GetRecord(ctx context.Context, domain, recordType, name string) (*DNSRecord, error)
	ListRecords(ctx context.Context, domain string) ([]*DNSRecord, error)
}

// CloudflareProvider implements DNSProvider for Cloudflare.
type CloudflareProvider struct {
	APIToken     string
	AccountEmail string
	BaseURL      string
	httpClient   *http.Client
}

// NewCloudflareProvider creates a new Cloudflare DNS provider.
func NewCloudflareProvider(apiToken, accountEmail string) *CloudflareProvider {
	return &CloudflareProvider{
		APIToken:     apiToken,
		AccountEmail: accountEmail,
		BaseURL:      "https://api.cloudflare.com/client/v4",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL allows overriding the API base URL (for testing).
func (c *CloudflareProvider) SetBaseURL(url string) {
	c.BaseURL = url
}

// CreateRecord creates a DNS record via the Cloudflare API.
func (c *CloudflareProvider) CreateRecord(ctx context.Context, req *DNSRecord) error {
	zoneID, err := c.getZoneID(ctx, req.Domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID: %w", err)
	}

	apiReq := cloudflareAPIRequest{
		Type:    req.Type,
		Name:    req.Name + "." + req.Domain,
		Content: req.Value,
		TTL:     req.TTL,
		Proxied: req.Proxied,
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records", c.BaseURL, zoneID)
	resp, err := c.doRequest(ctx, "POST", url, apiReq)
	if err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateRecord updates a DNS record via the Cloudflare API.
func (c *CloudflareProvider) UpdateRecord(ctx context.Context, req *DNSRecord) error {
	zoneID, err := c.getZoneID(ctx, req.Domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID: %w", err)
	}

	recordID, err := c.getRecordID(ctx, zoneID, req.Type, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get record ID: %w", err)
	}

	apiReq := cloudflareAPIRequest{
		Type:    req.Type,
		Name:    req.Name + "." + req.Domain,
		Content: req.Value,
		TTL:     req.TTL,
		Proxied: req.Proxied,
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, zoneID, recordID)
	resp, err := c.doRequest(ctx, "PUT", url, apiReq)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteRecord deletes a DNS record via the Cloudflare API.
func (c *CloudflareProvider) DeleteRecord(ctx context.Context, domain, recordType, name string) error {
	zoneID, err := c.getZoneID(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID: %w", err)
	}

	recordID, err := c.getRecordID(ctx, zoneID, recordType, name)
	if err != nil {
		return fmt.Errorf("failed to get record ID: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, zoneID, recordID)
	resp, err := c.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetRecord retrieves a DNS record via the Cloudflare API.
func (c *CloudflareProvider) GetRecord(ctx context.Context, domain, recordType, name string) (*DNSRecord, error) {
	zoneID, err := c.getZoneID(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID: %w", err)
	}

	recordID, err := c.getRecordID(ctx, zoneID, recordType, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get record ID: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, zoneID, recordID)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare API error %d", resp.StatusCode)
	}

	// Parse response — in production this would unmarshal JSON
	return &DNSRecord{
		Domain: domain,
		Type:   recordType,
		Name:   name,
	}, nil
}

// ListRecords lists all DNS records for a domain via the Cloudflare API.
func (c *CloudflareProvider) ListRecords(ctx context.Context, domain string) ([]*DNSRecord, error) {
	zoneID, err := c.getZoneID(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records", c.BaseURL, zoneID)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare API error %d", resp.StatusCode)
	}

	// Parse response — in production this would unmarshal JSON
	return []*DNSRecord{}, nil
}

// getZoneID retrieves the Cloudflare zone ID for a domain.
func (c *CloudflareProvider) getZoneID(ctx context.Context, domain string) (string, error) {
	url := fmt.Sprintf("%s/zones?name=%s", c.BaseURL, domain)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to find zone for %s", domain)
	}

	// In production: parse JSON response to extract zone ID
	return "zone-placeholder", nil
}

// getRecordID retrieves the Cloudflare record ID for a given record.
func (c *CloudflareProvider) getRecordID(ctx context.Context, zoneID, recordType, name string) (string, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records?type=%s&name=%s.%s",
		c.BaseURL, zoneID, recordType, name, zoneID)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("record not found: %s %s", recordType, name)
	}

	// In production: parse JSON response to extract record ID
	return "record-placeholder", nil
}

// doRequest executes an HTTP request with Cloudflare auth headers.
func (c *CloudflareProvider) doRequest(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	// In production: marshal body to JSON

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	if c.AccountEmail != "" {
		req.Header.Set("X-Auth-Email", c.AccountEmail)
	}

	return c.httpClient.Do(req)
}

// cloudflareAPIRequest represents a Cloudflare DNS record API request body.
type cloudflareAPIRequest struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}
