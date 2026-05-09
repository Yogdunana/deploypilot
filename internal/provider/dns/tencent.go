package dns

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TencentProvider implements DNSProvider for Tencent Cloud DNSPod.
type TencentProvider struct {
	secretID   string
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

// NewTencentProvider creates a new Tencent Cloud DNSPod provider.
func NewTencentProvider(secretID, secretKey string) *TencentProvider {
	return &TencentProvider{
		secretID:  secretID,
		secretKey: secretKey,
		baseURL:   "https://dnspod.tencentcloudapi.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL allows overriding the API base URL (for testing).
func (t *TencentProvider) SetBaseURL(u string) {
	t.baseURL = u
}

// Name returns the provider name.
func (t *TencentProvider) Name() string { return "tencent" }

// tencentAPIResponse represents a Tencent Cloud API response.
type tencentAPIResponse struct {
	Response struct {
		RecordId   int64 `json:"RecordId"`
		RecordInfo struct {
			RecordId int64  `json:"RecordId"`
			Name     string `json:"Name"`
			Type     string `json:"Type"`
			Value    string `json:"Value"`
			TTL      int    `json:"TTL"`
			Status   string `json:"Status"`
		} `json:"RecordInfo"`
		RecordList []struct {
			RecordId int64  `json:"RecordId"`
			Name     string `json:"Name"`
			Type     string `json:"Type"`
			Value    string `json:"Value"`
			TTL      int    `json:"TTL"`
		} `json:"RecordList"`
		RecordCountInfo struct {
			TotalCount int `json:"TotalCount"`
		} `json:"RecordCountInfo"`
		Error struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
		RequestId string `json:"RequestId"`
	} `json:"Response"`
}

// CreateRecord creates a DNS record via the Tencent Cloud DNSPod API.
func (t *TencentProvider) CreateRecord(ctx context.Context, req *DNSRecord) error {
	payload := map[string]interface{}{
		"Domain": req.Domain,
		"SubDomain": req.Name,
		"RecordType": req.Type,
		"Value":     req.Value,
		"RecordLine": "默认",
		"TTL":       req.TTL,
	}

	resp, err := t.doRequest(ctx, "CreateRecord", payload)
	if err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tencent API error %d: %s", resp.StatusCode, string(body))
	}

	var result tencentAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Response.Error.Code != "" {
		return fmt.Errorf("tencent error: %s - %s", result.Response.Error.Code, result.Response.Error.Message)
	}

	return nil
}

// UpdateRecord updates a DNS record via the Tencent Cloud DNSPod API.
func (t *TencentProvider) UpdateRecord(ctx context.Context, req *DNSRecord) error {
	// Find record ID first
	recordID, err := t.findRecordID(ctx, req.Domain, req.Type, req.Name)
	if err != nil {
		return fmt.Errorf("failed to find record: %w", err)
	}

	payload := map[string]interface{}{
		"Domain":     req.Domain,
		"RecordId":   recordID,
		"SubDomain":  req.Name,
		"RecordType": req.Type,
		"Value":      req.Value,
		"RecordLine": "默认",
		"TTL":        req.TTL,
	}

	resp, err := t.doRequest(ctx, "ModifyRecord", payload)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tencent API error %d: %s", resp.StatusCode, string(body))
	}

	var result tencentAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Response.Error.Code != "" {
		return fmt.Errorf("tencent error: %s - %s", result.Response.Error.Code, result.Response.Error.Message)
	}

	return nil
}

// DeleteRecord deletes a DNS record via the Tencent Cloud DNSPod API.
func (t *TencentProvider) DeleteRecord(ctx context.Context, domain, recordType, name string) error {
	recordID, err := t.findRecordID(ctx, domain, recordType, name)
	if err != nil {
		return fmt.Errorf("failed to find record: %w", err)
	}

	payload := map[string]interface{}{
		"Domain":   domain,
		"RecordId": recordID,
	}

	resp, err := t.doRequest(ctx, "DeleteRecord", payload)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tencent API error %d: %s", resp.StatusCode, string(body))
	}

	var result tencentAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Response.Error.Code != "" {
		return fmt.Errorf("tencent error: %s - %s", result.Response.Error.Code, result.Response.Error.Message)
	}

	return nil
}

// GetRecord retrieves a DNS record via the Tencent Cloud DNSPod API.
func (t *TencentProvider) GetRecord(ctx context.Context, domain, recordType, name string) (*DNSRecord, error) {
	recordID, err := t.findRecordID(ctx, domain, recordType, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find record: %w", err)
	}

	payload := map[string]interface{}{
		"Domain":   domain,
		"RecordId": recordID,
	}

	resp, err := t.doRequest(ctx, "DescribeRecord", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tencent API error %d: %s", resp.StatusCode, string(body))
	}

	var result tencentAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Response.Error.Code != "" {
		return nil, fmt.Errorf("tencent error: %s - %s", result.Response.Error.Code, result.Response.Error.Message)
	}

	return &DNSRecord{
		Domain: domain,
		Type:   result.Response.RecordInfo.Type,
		Name:   result.Response.RecordInfo.Name,
		Value:  result.Response.RecordInfo.Value,
		TTL:    result.Response.RecordInfo.TTL,
	}, nil
}

// ListRecords lists all DNS records for a domain via the Tencent Cloud DNSPod API.
func (t *TencentProvider) ListRecords(ctx context.Context, domain string) ([]*DNSRecord, error) {
	payload := map[string]interface{}{
		"Domain": domain,
		"Limit":  100,
	}

	resp, err := t.doRequest(ctx, "DescribeRecordList", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tencent API error %d: %s", resp.StatusCode, string(body))
	}

	var result tencentAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Response.Error.Code != "" {
		return nil, fmt.Errorf("tencent error: %s - %s", result.Response.Error.Code, result.Response.Error.Message)
	}

	records := make([]*DNSRecord, 0, len(result.Response.RecordList))
	for _, r := range result.Response.RecordList {
		records = append(records, &DNSRecord{
			Domain: domain,
			Type:   r.Type,
			Name:   r.Name,
			Value:  r.Value,
			TTL:    r.TTL,
		})
	}

	return records, nil
}

// findRecordID finds a record ID by domain, type, and subdomain.
func (t *TencentProvider) findRecordID(ctx context.Context, domain, recordType, name string) (int64, error) {
	payload := map[string]interface{}{
		"Domain":     domain,
		"Subdomain":  name,
		"RecordType": recordType,
		"Limit":      100,
	}

	resp, err := t.doRequest(ctx, "DescribeRecordList", payload)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("tencent API error %d: %s", resp.StatusCode, string(body))
	}

	var result tencentAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	for _, r := range result.Response.RecordList {
		if strings.EqualFold(r.Name, name) && strings.EqualFold(r.Type, recordType) {
			return r.RecordId, nil
		}
	}

	return 0, fmt.Errorf("record not found: %s %s.%s", recordType, name, domain)
}

// doRequest executes a TC3-HMAC-SHA256 signed request to the Tencent Cloud API.
func (t *TencentProvider) doRequest(ctx context.Context, action string, payload map[string]interface{}) (*http.Response, error) {
	now := time.Now().UTC()
	date := now.Format("2006-01-02")
	timestamp := fmt.Sprintf("%d", now.Unix())

	// Build the request body
	bodyPayload := map[string]interface{}{
		"Action":   action,
		"Version":  "2021-03-23",
		"Region":   "",
		"Timestamp": timestamp,
	}
	for k, v := range payload {
		bodyPayload[k] = v
	}

	bodyBytes, err := json.Marshal(bodyPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build canonical request
	host := strings.TrimPrefix(t.baseURL, "https://")
	httpMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	contentType := "application/json; charset=utf-8"
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		contentType, host, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"

	hashedPayload := sha256Hex(bodyBytes)
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpMethod, canonicalURI, canonicalQueryString,
		canonicalHeaders, signedHeaders, hashedPayload)

	// Build string to sign
	algorithm := "TC3-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, "dnspod")
	stringToSign := fmt.Sprintf("%s\n%s\n%s",
		algorithm, timestamp, sha256Hex([]byte(canonicalRequest)))

	// Calculate signature
	secretDate := hmacSHA256([]byte("TC3"+t.secretKey), date)
	secretService := hmacSHA256(secretDate, "dnspod")
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))

	// Build authorization header
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, t.secretKey, credentialScope, signedHeaders, signature)

	req, err := http.NewRequestWithContext(ctx, httpMethod, t.baseURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", "2021-03-23")
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("X-TC-Region", "")
	req.Header.Set("Authorization", authorization)

	return t.httpClient.Do(req)
}

// sha256Hex returns the hex-encoded SHA-256 hash of data.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// hmacSHA256 returns the HMAC-SHA256 result of key and data.
func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
