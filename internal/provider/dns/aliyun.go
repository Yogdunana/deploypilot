package dns

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// AliyunProvider implements DNSProvider for Alibaba Cloud DNS (Alidns).
type AliyunProvider struct {
	accessKeyID     string
	accessKeySecret string
	regionID        string // default: cn-hangzhou
	baseURL         string
	httpClient      *http.Client
}

// NewAliyunProvider creates a new Aliyun DNS provider.
func NewAliyunProvider(accessKeyID, accessKeySecret string) *AliyunProvider {
	return &AliyunProvider{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		regionID:        "cn-hangzhou",
		baseURL:         "https://alidns.aliyuncs.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL allows overriding the API base URL (for testing).
func (a *AliyunProvider) SetBaseURL(u string) {
	a.baseURL = u
}

// Name returns the provider name.
func (a *AliyunProvider) Name() string { return "aliyun" }

// aliyunAPIResponse represents a standard Aliyun API response.
type aliyunAPIResponse struct {
	RequestId  string `json:"RequestId"`
	RecordId   string `json:"RecordId"`
	RecordInfo struct {
		DomainName string `json:"DomainName"`
		RecordId   string `json:"RecordId"`
		RR         string `json:"RR"`
		Type       string `json:"Type"`
		Value      string `json:"Value"`
		TTL        int    `json:"TTL"`
		Priority   int    `json:"Priority"`
		Status     string `json:"Status"`
	} `json:"RecordInfo"`
	Records struct {
		Record []struct {
			DomainName string `json:"DomainName"`
			RecordId   string `json:"RecordId"`
			RR         string `json:"RR"`
			Type       string `json:"Type"`
			Value      string `json:"Value"`
			TTL        int    `json:"TTL"`
			Status     string `json:"Status"`
		} `json:"Record"`
	} `json:"Records"`
	TotalCount int `json:"TotalCount"`
	Code       string `json:"Code"`
	Message    string `json:"Message"`
}

// CreateRecord creates a DNS record via the Alibaba Cloud DNS API.
func (a *AliyunProvider) CreateRecord(ctx context.Context, req *DNSRecord) error {
	params := map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": req.Domain,
		"RR":         req.Name,
		"Type":       req.Type,
		"Value":      req.Value,
		"TTL":        fmt.Sprintf("%d", req.TTL),
	}

	resp, err := a.doRequest(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("aliyun API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateRecord updates a DNS record via the Alibaba Cloud DNS API.
func (a *AliyunProvider) UpdateRecord(ctx context.Context, req *DNSRecord) error {
	// First find the record ID
	recordID, err := a.findRecordID(ctx, req.Domain, req.Type, req.Name)
	if err != nil {
		return fmt.Errorf("failed to find record: %w", err)
	}

	params := map[string]string{
		"Action":     "ModifyDomainRecord",
		"RecordId":   recordID,
		"RR":         req.Name,
		"Type":       req.Type,
		"Value":      req.Value,
		"TTL":        fmt.Sprintf("%d", req.TTL),
	}

	resp, err := a.doRequest(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("aliyun API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteRecord deletes a DNS record via the Alibaba Cloud DNS API.
func (a *AliyunProvider) DeleteRecord(ctx context.Context, domain, recordType, name string) error {
	recordID, err := a.findRecordID(ctx, domain, recordType, name)
	if err != nil {
		return fmt.Errorf("failed to find record: %w", err)
	}

	params := map[string]string{
		"Action":   "DeleteDomainRecord",
		"RecordId": recordID,
	}

	resp, err := a.doRequest(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("aliyun API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetRecord retrieves a DNS record via the Alibaba Cloud DNS API.
func (a *AliyunProvider) GetRecord(ctx context.Context, domain, recordType, name string) (*DNSRecord, error) {
	recordID, err := a.findRecordID(ctx, domain, recordType, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find record: %w", err)
	}

	params := map[string]string{
		"Action":   "DescribeDomainRecordInfo",
		"RecordId": recordID,
	}

	resp, err := a.doRequest(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("aliyun API error %d: %s", resp.StatusCode, string(body))
	}

	var result aliyunAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Code != "" {
		return nil, fmt.Errorf("aliyun error: %s - %s", result.Code, result.Message)
	}

	return &DNSRecord{
		Domain: result.RecordInfo.DomainName,
		Type:   result.RecordInfo.Type,
		Name:   result.RecordInfo.RR,
		Value:  result.RecordInfo.Value,
		TTL:    result.RecordInfo.TTL,
	}, nil
}

// ListRecords lists all DNS records for a domain via the Alibaba Cloud DNS API.
func (a *AliyunProvider) ListRecords(ctx context.Context, domain string) ([]*DNSRecord, error) {
	params := map[string]string{
		"Action":     "DescribeDomainRecords",
		"DomainName": domain,
		"PageSize":   "100",
	}

	resp, err := a.doRequest(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("aliyun API error %d: %s", resp.StatusCode, string(body))
	}

	var result aliyunAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Code != "" {
		return nil, fmt.Errorf("aliyun error: %s - %s", result.Code, result.Message)
	}

	records := make([]*DNSRecord, 0, len(result.Records.Record))
	for _, r := range result.Records.Record {
		records = append(records, &DNSRecord{
			Domain: r.DomainName,
			Type:   r.Type,
			Name:   r.RR,
			Value:  r.Value,
			TTL:    r.TTL,
		})
	}

	return records, nil
}

// findRecordID finds a record ID by domain, type, and RR.
func (a *AliyunProvider) findRecordID(ctx context.Context, domain, recordType, name string) (string, error) {
	params := map[string]string{
		"Action":     "DescribeDomainRecords",
		"DomainName": domain,
		"RRKeyWord":  name,
		"Type":       recordType,
		"PageSize":   "100",
	}

	resp, err := a.doRequest(ctx, params)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("aliyun API error %d: %s", resp.StatusCode, string(body))
	}

	var result aliyunAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	for _, r := range result.Records.Record {
		if r.RR == name && r.Type == recordType {
			return r.RecordId, nil
		}
	}

	return "", fmt.Errorf("record not found: %s %s.%s", recordType, name, domain)
}

// doRequest executes a signed request to the Aliyun API.
func (a *AliyunProvider) doRequest(ctx context.Context, params map[string]string) (*http.Response, error) {
	// Add common parameters
	params["Format"] = "JSON"
	params["Version"] = "2015-01-09"
	params["AccessKeyId"] = a.accessKeyID
	params["SignatureMethod"] = "HMAC-SHA1"
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = fmt.Sprintf("%d", time.Now().UnixNano())
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Compute signature
	signature := a.sign(params)
	params["Signature"] = signature

	// Build query string
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}

	reqURL := a.baseURL + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	return a.httpClient.Do(req)
}

// sign generates an HMAC-SHA1 signature for the Aliyun API request.
func (a *AliyunProvider) sign(params map[string]string) string {
	// Sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build canonical query string
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(params[k]))
	}
	canonicalQS := strings.Join(pairs, "&")

	// Build string to sign
	stringToSign := "GET&" + percentEncode("/") + "&" + percentEncode(canonicalQS)

	// HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(a.accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
}

// percentEncode performs URL encoding compatible with Aliyun signature requirements.
func percentEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}
