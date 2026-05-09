package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// WestDNSProvider implements DNSProvider for WestDNS (西部数码 / west.cn).
//
// API documentation: https://api.west.cn/CustomerCenter/doc/domain_v2.html
//
// Authentication uses HTTP Basic Auth with the API key as username and API
// secret as password, sent via application/x-www-form-urlencoded POST requests.
// The base URL is https://api.west.cn and DNS operations live under /domain/.
type WestDNSProvider struct {
	apiKey    string
	apiSecret string
	baseURL   string
	httpClient *http.Client
}

// NewWestDNSProvider creates a new WestDNS (西部数码) DNS provider.
func NewWestDNSProvider(apiKey, apiSecret string) *WestDNSProvider {
	return &WestDNSProvider{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   "https://api.west.cn",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL allows overriding the API base URL (for testing).
func (w *WestDNSProvider) SetBaseURL(u string) {
	w.baseURL = u
}

// Name returns the provider name.
func (w *WestDNSProvider) Name() string { return "west-dns" }

// westDNSAPIResponse represents the standard WestDNS API response.
// All endpoints return { "result": <int>, "clientid": "<string>", "data": {...} }
// where result=0 means success.
type westDNSAPIResponse struct {
	Result   int                    `json:"result"`
	ClientID string                 `json:"clientid"`
	Data     json.RawMessage        `json:"data"`
}

// westDNSRecord represents a single DNS record in the WestDNS API response.
type westDNSRecord struct {
	ID     int    `json:"id"`
	Item   string `json:"item"`   // host / subdomain name
	Value  string `json:"value"`  // record value
	Type   string `json:"type"`   // A, CNAME, MX, TXT, AAAA, SRV
	Level  int    `json:"level"`  // priority (1-100, default 10)
	TTL    int    `json:"ttl"`    // 60-86400 seconds
	Line   string `json:"line"`   // ISP line (default="", LTEL, LCNC, LMOB, LEDU, LSEO)
	Pause  int    `json:"pause"`  // 0=active, 1=paused
}

// westDNSRecordListData represents the paginated list data from getdnsrecord.
type westDNSRecordListData struct {
	Items      []westDNSRecord `json:"items"`
	Limit      int             `json:"limit"`
	Total      int             `json:"total"`
	PageNo     int             `json:"pageno"`
	TotalPages int             `json:"totalpages"`
}

// westDNSAddResult represents the data returned from adddnsrecord.
type westDNSAddResult struct {
	ID int `json:"id"`
}

// CreateRecord creates a DNS record via the WestDNS API.
// POST /domain/?act=adddnsrecord
func (w *WestDNSProvider) CreateRecord(ctx context.Context, req *DNSRecord) error {
	ttl := req.TTL
	if ttl <= 0 {
		ttl = 900
	}

	params := url.Values{
		"domain": {req.Domain},
		"host":   {req.Name},
		"type":   {req.Type},
		"value":  {req.Value},
		"ttl":    {strconv.Itoa(ttl)},
		"level":  {"10"},
	}

	resp, err := w.doPostForm(ctx, "/domain/", "adddnsrecord", params)
	if err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("westdns API error %d: %s", resp.StatusCode, string(body))
	}

	var result westDNSAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse create response: %w", err)
	}

	if result.Result != 0 {
		return fmt.Errorf("westdns error (code %d): failed to create record", result.Result)
	}

	return nil
}

// UpdateRecord updates a DNS record via the WestDNS API.
// First finds the record ID, then calls POST /domain/?act=moddnsrecord.
func (w *WestDNSProvider) UpdateRecord(ctx context.Context, req *DNSRecord) error {
	recordID, err := w.findRecordID(ctx, req.Domain, req.Type, req.Name)
	if err != nil {
		return fmt.Errorf("failed to find record: %w", err)
	}

	ttl := req.TTL
	if ttl <= 0 {
		ttl = 900
	}

	params := url.Values{
		"domain": {req.Domain},
		"id":     {strconv.Itoa(recordID)},
		"value":  {req.Value},
		"ttl":    {strconv.Itoa(ttl)},
	}

	resp, err := w.doPostForm(ctx, "/domain/", "moddnsrecord", params)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("westdns API error %d: %s", resp.StatusCode, string(body))
	}

	var result westDNSAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse update response: %w", err)
	}

	if result.Result != 0 {
		return fmt.Errorf("westdns error (code %d): failed to update record", result.Result)
	}

	return nil
}

// DeleteRecord deletes a DNS record via the WestDNS API.
// POST /domain/?act=deldnsrecord
func (w *WestDNSProvider) DeleteRecord(ctx context.Context, domain, recordType, name string) error {
	recordID, err := w.findRecordID(ctx, domain, recordType, name)
	if err != nil {
		return fmt.Errorf("failed to find record: %w", err)
	}

	params := url.Values{
		"domain": {domain},
		"id":     {strconv.Itoa(recordID)},
	}

	resp, err := w.doPostForm(ctx, "/domain/", "deldnsrecord", params)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("westdns API error %d: %s", resp.StatusCode, string(body))
	}

	var result westDNSAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse delete response: %w", err)
	}

	if result.Result != 0 {
		return fmt.Errorf("westdns error (code %d): failed to delete record", result.Result)
	}

	return nil
}

// GetRecord retrieves a single DNS record by finding it in the list.
// WestDNS does not have a single-record GET endpoint, so we use getdnsrecord
// and filter by type and host.
func (w *WestDNSProvider) GetRecord(ctx context.Context, domain, recordType, name string) (*DNSRecord, error) {
	recordID, err := w.findRecordID(ctx, domain, recordType, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find record: %w", err)
	}

	// Fetch all records and find the one with matching ID
	records, err := w.fetchAllRecords(ctx, domain)
	if err != nil {
		return nil, err
	}

	for _, r := range records {
		if r.ID == recordID {
			return &DNSRecord{
				Domain: domain,
				Type:   r.Type,
				Name:   r.Item,
				Value:  r.Value,
				TTL:    r.TTL,
			}, nil
		}
	}

	return nil, fmt.Errorf("record not found after fetch: %s %s.%s (id=%d)", recordType, name, domain, recordID)
}

// ListRecords lists all DNS records for a domain via the WestDNS API.
// POST /domain/?act=getdnsrecord
func (w *WestDNSProvider) ListRecords(ctx context.Context, domain string) ([]*DNSRecord, error) {
	records, err := w.fetchAllRecords(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	result := make([]*DNSRecord, 0, len(records))
	for _, r := range records {
		result = append(result, &DNSRecord{
			Domain: domain,
			Type:   r.Type,
			Name:   r.Item,
			Value:  r.Value,
			TTL:    r.TTL,
		})
	}

	return result, nil
}

// fetchAllRecords fetches all DNS records for a domain, handling pagination.
func (w *WestDNSProvider) fetchAllRecords(ctx context.Context, domain string) ([]westDNSRecord, error) {
	var allRecords []westDNSRecord
	pageNo := 1
	limit := 100

	for {
		params := url.Values{
			"domain": {domain},
			"limit":  {strconv.Itoa(limit)},
			"pageno": {strconv.Itoa(pageNo)},
		}

		resp, err := w.doPostForm(ctx, "/domain/", "getdnsrecord", params)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("westdns API error %d: %s", resp.StatusCode, string(body))
		}

		var result westDNSAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to parse list response: %w", err)
		}
		_ = resp.Body.Close()

		if result.Result != 0 {
			return nil, fmt.Errorf("westdns error (code %d): failed to list records", result.Result)
		}

		var data westDNSRecordListData
		if err := json.Unmarshal(result.Data, &data); err != nil {
			return nil, fmt.Errorf("failed to parse record data: %w", err)
		}

		allRecords = append(allRecords, data.Items...)

		if pageNo >= data.TotalPages || len(data.Items) == 0 {
			break
		}
		pageNo++
	}

	return allRecords, nil
}

// findRecordID finds a DNS record ID by domain, type, and host name.
func (w *WestDNSProvider) findRecordID(ctx context.Context, domain, recordType, name string) (int, error) {
	records, err := w.fetchAllRecords(ctx, domain)
	if err != nil {
		return 0, err
	}

	for _, r := range records {
		if strings.EqualFold(r.Type, recordType) && strings.EqualFold(r.Item, name) {
			return r.ID, nil
		}
	}

	return 0, fmt.Errorf("record not found: %s %s.%s", recordType, name, domain)
}

// doPostForm sends an authenticated POST request with form data to the WestDNS API.
// Authentication is done via HTTP Basic Auth (userid=apiKey, password=apiSecret).
func (w *WestDNSProvider) doPostForm(ctx context.Context, path, act string, params url.Values) (*http.Response, error) {
	reqURL := w.baseURL + path + "?act=" + act

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// WestDNS API uses HTTP Basic Auth: userid as username, userpwd as password
	req.SetBasicAuth(w.apiKey, w.apiSecret)

	return w.httpClient.Do(req)
}
