package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "failed to close response body: %v\n", cerr)
		}
	}()

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

	recordID, err := c.getRecordID(ctx, zoneID, req.Domain, req.Type, req.Name)
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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "failed to close response body: %v\n", cerr)
		}
	}()

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

	recordID, err := c.getRecordID(ctx, zoneID, domain, recordType, name)
	if err != nil {
		return fmt.Errorf("failed to get record ID: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, zoneID, recordID)
	resp, err := c.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "failed to close response body: %v\n", cerr)
		}
	}()

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

	recordID, err := c.getRecordID(ctx, zoneID, domain, recordType, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get record ID: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.BaseURL, zoneID, recordID)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "failed to close response body: %v\n", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare API error %d", resp.StatusCode)
	}

	var cfResp cloudflareAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, fmt.Errorf("failed to parse record response: %w", err)
	}

	if !cfResp.Success {
		msg := "unknown error"
		if len(cfResp.Errors) > 0 {
			msg = cfResp.Errors[0].Message
		}
		return nil, fmt.Errorf("cloudflare API error: %s", msg)
	}

	var cfRec cloudflareDNSRecord
	if err := json.Unmarshal(cfResp.Result, &cfRec); err != nil {
		return nil, fmt.Errorf("failed to parse record result: %w", err)
	}

	return &DNSRecord{
		Domain:  domain,
		Type:    cfRec.Type,
		Name:    name,
		Value:   cfRec.Content,
		TTL:     cfRec.TTL,
		Proxied: cfRec.Proxied,
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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "failed to close response body: %v\n", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare API error %d", resp.StatusCode)
	}

	var cfResp cloudflareAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, fmt.Errorf("failed to parse records response: %w", err)
	}

	if !cfResp.Success {
		msg := "unknown error"
		if len(cfResp.Errors) > 0 {
			msg = cfResp.Errors[0].Message
		}
		return nil, fmt.Errorf("cloudflare API error: %s", msg)
	}

	var cfRecords []cloudflareDNSRecord
	if err := json.Unmarshal(cfResp.Result, &cfRecords); err != nil {
		return nil, fmt.Errorf("failed to parse records result: %w", err)
	}

	records := make([]*DNSRecord, 0, len(cfRecords))
	for _, rec := range cfRecords {
		records = append(records, &DNSRecord{
			Domain:  domain,
			Type:    rec.Type,
			Name:    strings.TrimSuffix(rec.Name, "."+domain),
			Value:   rec.Content,
			TTL:     rec.TTL,
			Proxied: rec.Proxied,
		})
	}

	return records, nil
}

// getZoneID retrieves the Cloudflare zone ID for a domain.
func (c *CloudflareProvider) getZoneID(ctx context.Context, domain string) (string, error) {
	url := fmt.Sprintf("%s/zones?name=%s", c.BaseURL, domain)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "failed to close response body: %v\n", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to find zone for %s", domain)
	}

	var cfResp cloudflareAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return "", fmt.Errorf("failed to parse zone response: %w", err)
	}

	if !cfResp.Success {
		msg := "unknown error"
		if len(cfResp.Errors) > 0 {
			msg = cfResp.Errors[0].Message
		}
		return "", fmt.Errorf("cloudflare API error: %s", msg)
	}

	var zones []cloudflareZone
	if err := json.Unmarshal(cfResp.Result, &zones); err != nil {
		return "", fmt.Errorf("failed to parse zone result: %w", err)
	}

	for _, zone := range zones {
		if zone.Name == domain {
			return zone.ID, nil
		}
	}

	return "", fmt.Errorf("zone not found for domain %s", domain)
}

// getRecordID retrieves the Cloudflare record ID for a given record.
func (c *CloudflareProvider) getRecordID(ctx context.Context, zoneID, domain, recordType, name string) (string, error) {
	fullName := name + "." + domain
	url := fmt.Sprintf("%s/zones/%s/dns_records?type=%s&name=%s",
		c.BaseURL, zoneID, recordType, fullName)
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "failed to close response body: %v\n", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("record not found: %s %s", recordType, name)
	}

	var cfResp cloudflareAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return "", fmt.Errorf("failed to parse record response: %w", err)
	}

	if !cfResp.Success {
		msg := "unknown error"
		if len(cfResp.Errors) > 0 {
			msg = cfResp.Errors[0].Message
		}
		return "", fmt.Errorf("cloudflare API error: %s", msg)
	}

	var records []cloudflareDNSRecord
	if err := json.Unmarshal(cfResp.Result, &records); err != nil {
		return "", fmt.Errorf("failed to parse record result: %w", err)
	}

	for _, rec := range records {
		if strings.EqualFold(rec.Type, recordType) && rec.Name == fullName {
			return rec.ID, nil
		}
	}

	return "", fmt.Errorf("record not found: %s %s", recordType, name)
}

// doRequest executes an HTTP request with Cloudflare auth headers.
func (c *CloudflareProvider) doRequest(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

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

// cloudflareAPIResponse represents the top-level Cloudflare API response.
type cloudflareAPIResponse struct {
	Success    bool                   `json:"success"`
	Errors     []cloudflareAPIError   `json:"errors"`
	Result     json.RawMessage        `json:"result"`
	ResultInfo *cloudflareResultInfo  `json:"result_info,omitempty"`
}

// cloudflareAPIError represents an error in the Cloudflare API response.
type cloudflareAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// cloudflareResultInfo contains pagination info from the Cloudflare API.
type cloudflareResultInfo struct {
	TotalCount int `json:"total_count"`
}

// cloudflareZone represents a zone in the Cloudflare API response.
type cloudflareZone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// cloudflareDNSRecord represents a DNS record in the Cloudflare API response.
type cloudflareDNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}
