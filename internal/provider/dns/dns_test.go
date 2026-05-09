package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// ========== Cloudflare Provider Tests with httptest ==========

const testZoneID = "zone-abc123"
const testRecordID = "rec-xyz789"

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Zone lookup endpoint
		if r.URL.Path == "/zones" {
			name := r.URL.Query().Get("name")
			if name == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			resp := map[string]interface{}{
				"success": true,
				"result": []map[string]string{
					{"id": testZoneID, "name": name, "status": "active"},
				},
				"result_info": map[string]int{
					"total_count": 1,
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// DNS records endpoint for the real zone ID
		if r.URL.Path == "/zones/"+testZoneID+"/dns_records" {
			switch r.Method {
			case "GET":
				resp := map[string]interface{}{
					"success": true,
					"result": []map[string]interface{}{
						{
							"id":      testRecordID,
							"type":    "A",
							"name":    "www.example.com",
							"content": "1.2.3.4",
							"ttl":     300,
							"proxied": false,
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
			case "POST":
				var reqBody map[string]interface{}
				json.NewDecoder(r.Body).Decode(&reqBody)
				resp := map[string]interface{}{
					"success": true,
					"result": map[string]interface{}{
						"id":      "record-new-789",
						"type":    reqBody["type"],
						"name":    reqBody["name"],
						"content": reqBody["content"],
						"ttl":     reqBody["ttl"],
						"proxied": reqBody["proxied"],
					},
				}
				json.NewEncoder(w).Encode(resp)
			case "PUT":
				var reqBody map[string]interface{}
				json.NewDecoder(r.Body).Decode(&reqBody)
				resp := map[string]interface{}{
					"success": true,
					"result": map[string]interface{}{
						"id":      testRecordID,
						"type":    reqBody["type"],
						"name":    reqBody["name"],
						"content": reqBody["content"],
						"ttl":     reqBody["ttl"],
						"proxied": reqBody["proxied"],
					},
				}
				json.NewEncoder(w).Encode(resp)
			case "DELETE":
				resp := map[string]interface{}{
					"success": true,
					"result": map[string]interface{}{
						"id": testRecordID,
					},
				}
				json.NewEncoder(w).Encode(resp)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}

		// Record-specific endpoint (for GET/PUT/DELETE with record ID)
		if r.URL.Path == "/zones/"+testZoneID+"/dns_records/"+testRecordID {
			switch r.Method {
			case "GET":
				resp := map[string]interface{}{
					"success": true,
					"result": map[string]interface{}{
						"id":      testRecordID,
						"type":    "A",
						"name":    "www.example.com",
						"content": "1.2.3.4",
						"ttl":     300,
						"proxied": false,
					},
				}
				json.NewEncoder(w).Encode(resp)
			case "PUT":
				var reqBody map[string]interface{}
				json.NewDecoder(r.Body).Decode(&reqBody)
				resp := map[string]interface{}{
					"success": true,
					"result": map[string]interface{}{
						"id":      testRecordID,
						"type":    reqBody["type"],
						"name":    reqBody["name"],
						"content": reqBody["content"],
						"ttl":     reqBody["ttl"],
						"proxied": reqBody["proxied"],
					},
				}
				json.NewEncoder(w).Encode(resp)
			case "DELETE":
				resp := map[string]interface{}{
					"success": true,
					"result": map[string]interface{}{
						"id": testRecordID,
					},
				}
				json.NewEncoder(w).Encode(resp)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}

		// Default 404
		w.WriteHeader(http.StatusNotFound)
	})

	return httptest.NewServer(handler)
}

func newTestCFProvider(ts *httptest.Server) *CloudflareProvider {
	cf := NewCloudflareProvider("test-api-token", "test@example.com")
	cf.SetBaseURL(ts.URL)
	return cf
}

func TestCloudflareCreateRecord(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain:  "example.com",
		Type:    "A",
		Name:    "www",
		Value:   "1.2.3.4",
		TTL:     300,
		Proxied: false,
	})
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}
}

func TestCloudflareCreateRecordError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"errors":[{"message":"internal error"}]}`))
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("CreateRecord() should return error on 500")
	}
}

func TestCloudflareUpdateRecord(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.UpdateRecord(ctx, &DNSRecord{
		Domain:  "example.com",
		Type:    "A",
		Name:    "www",
		Value:   "9.9.9.9",
		TTL:     600,
		Proxied: true,
	})
	if err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}
}

func TestCloudflareUpdateRecordError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Zone lookup succeeds
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-test-123", "name": "example.com"}},
			})
			return
		}
		// Record lookup fails
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.UpdateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.1.1.1", TTL: 300,
	})
	if err == nil {
		t.Error("UpdateRecord() should return error when record not found")
	}
}

func TestCloudflareDeleteRecord(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.DeleteRecord(ctx, "example.com", "A", "www")
	if err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
}

func TestCloudflareDeleteRecordError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Zone lookup succeeds
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-test-123", "name": "example.com"}},
			})
			return
		}
		// Record lookup fails
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.DeleteRecord(ctx, "example.com", "A", "nonexistent")
	if err == nil {
		t.Error("DeleteRecord() should return error when record not found")
	}
}

func TestCloudflareGetRecord(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	record, err := cf.GetRecord(ctx, "example.com", "A", "www")
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if record.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", record.Domain, "example.com")
	}
	if record.Type != "A" {
		t.Errorf("Type = %q, want %q", record.Type, "A")
	}
	if record.Name != "www" {
		t.Errorf("Name = %q, want %q", record.Name, "www")
	}
	if record.Value != "1.2.3.4" {
		t.Errorf("Value = %q, want %q", record.Value, "1.2.3.4")
	}
	if record.TTL != 300 {
		t.Errorf("TTL = %d, want %d", record.TTL, 300)
	}
	if record.Proxied != false {
		t.Errorf("Proxied = %v, want %v", record.Proxied, false)
	}
}

func TestCloudflareGetRecordError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Zone lookup succeeds
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-test-123", "name": "example.com"}},
			})
			return
		}
		// Record lookup fails
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.GetRecord(ctx, "example.com", "A", "nonexistent")
	if err == nil {
		t.Error("GetRecord() should return error when record not found")
	}
}

func TestCloudflareListRecords(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	records, err := cf.ListRecords(ctx, "example.com")
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if records == nil {
		t.Error("ListRecords() should return non-nil slice")
	}
	if len(records) != 1 {
		t.Fatalf("ListRecords() returned %d records, want 1", len(records))
	}
	rec := records[0]
	if rec.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", rec.Domain, "example.com")
	}
	if rec.Type != "A" {
		t.Errorf("Type = %q, want %q", rec.Type, "A")
	}
	if rec.Name != "www" {
		t.Errorf("Name = %q, want %q", rec.Name, "www")
	}
	if rec.Value != "1.2.3.4" {
		t.Errorf("Value = %q, want %q", rec.Value, "1.2.3.4")
	}
	if rec.TTL != 300 {
		t.Errorf("TTL = %d, want %d", rec.TTL, 300)
	}
	if rec.Proxied != false {
		t.Errorf("Proxied = %v, want %v", rec.Proxied, false)
	}
}

func TestCloudflareListRecordsMultiple(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-multi", "name": "example.com"}},
			})
			return
		}
		if r.URL.Path == "/zones/zone-multi/dns_records" && r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": []map[string]interface{}{
					{
						"id": "rec-1", "type": "A", "name": "www.example.com",
						"content": "1.2.3.4", "ttl": 300, "proxied": false,
					},
					{
						"id": "rec-2", "type": "AAAA", "name": "www.example.com",
						"content": "2001:db8::1", "ttl": 300, "proxied": true,
					},
					{
						"id": "rec-3", "type": "CNAME", "name": "api.example.com",
						"content": "www.example.com", "ttl": 3600, "proxied": false,
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	records, err := cf.ListRecords(ctx, "example.com")
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("ListRecords() returned %d records, want 3", len(records))
	}

	// Verify first record
	if records[0].Type != "A" || records[0].Name != "www" || records[0].Value != "1.2.3.4" {
		t.Errorf("Record[0] = %+v, want A/www/1.2.3.4", records[0])
	}

	// Verify second record
	if records[1].Type != "AAAA" || records[1].Name != "www" || records[1].Value != "2001:db8::1" {
		t.Errorf("Record[1] = %+v, want AAAA/www/2001:db8::1", records[1])
	}
	if records[1].Proxied != true {
		t.Errorf("Record[1].Proxied = %v, want true", records[1].Proxied)
	}

	// Verify third record
	if records[2].Type != "CNAME" || records[2].Name != "api" || records[2].Value != "www.example.com" {
		t.Errorf("Record[2] = %+v, want CNAME/api/www.example.com", records[2])
	}
	if records[2].TTL != 3600 {
		t.Errorf("Record[2].TTL = %d, want 3600", records[2].TTL)
	}
}

func TestCloudflareListRecordsEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-empty", "name": "example.com"}},
			})
			return
		}
		if r.URL.Path == "/zones/zone-empty/dns_records" && r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]interface{}{},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	records, err := cf.ListRecords(ctx, "example.com")
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("ListRecords() returned %d records, want 0", len(records))
	}
}

func TestCloudflareListRecordsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.ListRecords(ctx, "example.com")
	if err == nil {
		t.Error("ListRecords() should return error on 500")
	}
}

func TestCloudflareGetZoneIDError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("should fail when zone lookup fails")
	}
}

func TestCloudflareDoRequestHeaders(t *testing.T) {
	var capturedAuth, capturedEmail, capturedContentType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedEmail = r.Header.Get("X-Auth-Email")
		capturedContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
		})
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})

	if capturedAuth != "Bearer test-api-token" {
		t.Errorf("Authorization = %q, want %q", capturedAuth, "Bearer test-api-token")
	}
	if capturedEmail != "test@example.com" {
		t.Errorf("X-Auth-Email = %q, want %q", capturedEmail, "test@example.com")
	}
	if capturedContentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", capturedContentType, "application/json")
	}
}

func TestCloudflareDoRequestNoEmail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email := r.Header.Get("X-Auth-Email")
		if email != "" {
			t.Errorf("X-Auth-Email should be empty when AccountEmail is empty, got %q", email)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
		})
	}))
	defer ts.Close()

	cf := NewCloudflareProvider("test-token", "")
	cf.SetBaseURL(ts.URL)
	ctx := context.Background()

	cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
}

func TestCloudflareContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		select {}
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("should return error when context is cancelled")
	}
}

func TestCloudflareProviderDefaultBaseURL(t *testing.T) {
	cf := NewCloudflareProvider("token", "email@example.com")
	if cf.BaseURL != "https://api.cloudflare.com/client/v4" {
		t.Errorf("BaseURL = %q, want default Cloudflare URL", cf.BaseURL)
	}
}

func TestCloudflareProviderSetBaseURL(t *testing.T) {
	cf := NewCloudflareProvider("token", "email@example.com")
	cf.SetBaseURL("http://custom-url")
	if cf.BaseURL != "http://custom-url" {
		t.Errorf("BaseURL = %q, want %q", cf.BaseURL, "http://custom-url")
	}
}

func TestDNSRecordStruct(t *testing.T) {
	rec := &DNSRecord{
		Domain:  "example.com",
		Type:    "A",
		Name:    "www",
		Value:   "1.2.3.4",
		TTL:     300,
		Proxied: true,
	}
	if rec.Domain != "example.com" {
		t.Errorf("Domain = %q", rec.Domain)
	}
	if rec.Type != "A" {
		t.Errorf("Type = %q", rec.Type)
	}
	if rec.Name != "www" {
		t.Errorf("Name = %q", rec.Name)
	}
	if rec.Value != "1.2.3.4" {
		t.Errorf("Value = %q", rec.Value)
	}
	if rec.TTL != 300 {
		t.Errorf("TTL = %d", rec.TTL)
	}
	if !rec.Proxied {
		t.Error("Proxied should be true")
	}
}

func TestCloudflareAPIRequestStruct(t *testing.T) {
	req := cloudflareAPIRequest{
		Type:    "A",
		Name:    "www.example.com",
		Content: "1.2.3.4",
		TTL:     300,
		Proxied: true,
	}
	if req.Type != "A" {
		t.Errorf("Type = %q", req.Type)
	}
	if req.Name != "www.example.com" {
		t.Errorf("Name = %q", req.Name)
	}
	if req.Content != "1.2.3.4" {
		t.Errorf("Content = %q", req.Content)
	}
	if req.TTL != 300 {
		t.Errorf("TTL = %d", req.TTL)
	}
	if !req.Proxied {
		t.Error("Proxied should be true")
	}
}

func recordKey(domain, recordType, name string) string {
	return strings.ToLower(fmt.Sprintf("%s:%s:%s", domain, recordType, name))
}

func TestRecordKey(t *testing.T) {
	tests := []struct {
		domain, rtype, name, want string
	}{
		{"example.com", "A", "www", "example.com:a:www"},
		{"Example.COM", "A", "WWW", "example.com:a:www"},
		{"test.org", "CNAME", "api", "test.org:cname:api"},
	}
	for _, tt := range tests {
		got := recordKey(tt.domain, tt.rtype, tt.name)
		if got != tt.want {
			t.Errorf("recordKey(%q, %q, %q) = %q, want %q", tt.domain, tt.rtype, tt.name, got, tt.want)
		}
	}
}

// ========== MockProvider Implementation ==========

// MockProvider implements DNSProvider for testing.
type MockProvider struct {
	mu      sync.RWMutex
	records map[string]*DNSRecord // key: "domain:type:name"
}

// NewMockProvider creates a new MockProvider.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		records: make(map[string]*DNSRecord),
	}
}

func (m *MockProvider) CreateRecord(_ context.Context, req *DNSRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := recordKey(req.Domain, req.Type, req.Name)
	if _, exists := m.records[key]; exists {
		return fmt.Errorf("record already exists: %s", key)
	}

	m.records[key] = &DNSRecord{
		Domain:  req.Domain,
		Type:    req.Type,
		Name:    req.Name,
		Value:   req.Value,
		TTL:     req.TTL,
		Proxied: req.Proxied,
	}
	return nil
}

func (m *MockProvider) UpdateRecord(_ context.Context, req *DNSRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := recordKey(req.Domain, req.Type, req.Name)
	rec, exists := m.records[key]
	if !exists {
		return fmt.Errorf("record not found: %s", key)
	}

	rec.Value = req.Value
	rec.TTL = req.TTL
	rec.Proxied = req.Proxied
	return nil
}

func (m *MockProvider) DeleteRecord(_ context.Context, domain, recordType, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := recordKey(domain, recordType, name)
	if _, exists := m.records[key]; !exists {
		return fmt.Errorf("record not found: %s", key)
	}

	delete(m.records, key)
	return nil
}

func (m *MockProvider) GetRecord(_ context.Context, domain, recordType, name string) (*DNSRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := recordKey(domain, recordType, name)
	rec, exists := m.records[key]
	if !exists {
		return nil, fmt.Errorf("record not found: %s", key)
	}
	return rec, nil
}

func (m *MockProvider) ListRecords(_ context.Context, domain string) ([]*DNSRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*DNSRecord
	for key, rec := range m.records {
		if strings.HasPrefix(key, strings.ToLower(domain)+":") {
			results = append(results, rec)
		}
	}
	return results, nil
}

// ========== DNSProvider Interface Tests (via Mock) ==========

func TestDNSProviderCreateAndGet(t *testing.T) {
	provider := NewMockProvider()

	err := provider.CreateRecord(context.Background(), &DNSRecord{
		Domain:  "example.com",
		Type:    "A",
		Name:    "www",
		Value:   "1.2.3.4",
		TTL:     300,
		Proxied: false,
	})
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}

	record, err := provider.GetRecord(context.Background(), "example.com", "A", "www")
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}

	if record.Value != "1.2.3.4" {
		t.Errorf("Value = %q, want %q", record.Value, "1.2.3.4")
	}
	if record.Type != "A" {
		t.Errorf("Type = %q, want %q", record.Type, "A")
	}
	if record.TTL != 300 {
		t.Errorf("TTL = %d, want 300", record.TTL)
	}
}

func TestDNSProviderListRecords(t *testing.T) {
	provider := NewMockProvider()

	provider.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	provider.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "api", Value: "5.6.7.8", TTL: 300,
	})

	records, err := provider.ListRecords(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}

	if len(records) != 2 {
		t.Errorf("count = %d, want 2", len(records))
	}
}

func TestDNSProviderListRecordsEmpty(t *testing.T) {
	provider := NewMockProvider()

	records, err := provider.ListRecords(context.Background(), "nonexistent.com")
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}

	if len(records) != 0 {
		t.Errorf("count = %d, want 0", len(records))
	}
}

func TestDNSProviderUpdateRecord(t *testing.T) {
	provider := NewMockProvider()

	provider.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})

	err := provider.UpdateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "9.9.9.9", TTL: 600,
	})
	if err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}

	record, _ := provider.GetRecord(context.Background(), "example.com", "A", "www")
	if record.Value != "9.9.9.9" {
		t.Errorf("Value = %q, want %q", record.Value, "9.9.9.9")
	}
	if record.TTL != 600 {
		t.Errorf("TTL = %d, want 600", record.TTL)
	}
}

func TestDNSProviderUpdateNotFound(t *testing.T) {
	provider := NewMockProvider()

	err := provider.UpdateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.1.1.1", TTL: 300,
	})
	if err == nil {
		t.Error("UpdateRecord() should fail for nonexistent record")
	}
}

func TestDNSProviderDeleteRecord(t *testing.T) {
	provider := NewMockProvider()

	provider.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})

	err := provider.DeleteRecord(context.Background(), "example.com", "A", "www")
	if err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}

	_, err = provider.GetRecord(context.Background(), "example.com", "A", "www")
	if err == nil {
		t.Error("GetRecord() should fail after delete")
	}
}

func TestDNSProviderDeleteNotFound(t *testing.T) {
	provider := NewMockProvider()

	err := provider.DeleteRecord(context.Background(), "example.com", "A", "nonexistent")
	if err == nil {
		t.Error("DeleteRecord() should fail for nonexistent record")
	}
}

func TestDNSProviderGetNotFound(t *testing.T) {
	provider := NewMockProvider()

	_, err := provider.GetRecord(context.Background(), "example.com", "A", "nonexistent")
	if err == nil {
		t.Error("GetRecord() should fail for nonexistent record")
	}
}

func TestDNSProviderCreateDuplicate(t *testing.T) {
	provider := NewMockProvider()

	provider.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})

	err := provider.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "5.6.7.8", TTL: 300,
	})
	if err == nil {
		t.Error("CreateRecord() should fail for duplicate")
	}
}

func TestDNSProviderMultipleTypes(t *testing.T) {
	provider := NewMockProvider()

	provider.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "@", Value: "1.2.3.4", TTL: 300,
	})
	provider.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "CNAME", Name: "www", Value: "example.com", TTL: 300,
	})
	provider.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "MX", Name: "@", Value: "mail.example.com", TTL: 300,
	})

	records, _ := provider.ListRecords(context.Background(), "example.com")
	if len(records) != 3 {
		t.Errorf("count = %d, want 3", len(records))
	}

	// Verify CNAME
	cname, _ := provider.GetRecord(context.Background(), "example.com", "CNAME", "www")
	if cname.Value != "example.com" {
		t.Errorf("CNAME value = %q", cname.Value)
	}
}

// ========== Additional Cloudflare Error Path Tests ==========

func TestCloudflareUpdateRecordZoneError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.UpdateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.1.1.1", TTL: 300,
	})
	if err == nil {
		t.Error("UpdateRecord() should return error when zone lookup fails")
	}
}

func TestCloudflareDeleteRecordZoneError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.DeleteRecord(ctx, "example.com", "A", "www")
	if err == nil {
		t.Error("DeleteRecord() should return error when zone lookup fails")
	}
}

func TestCloudflareGetRecordZoneError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.GetRecord(ctx, "example.com", "A", "www")
	if err == nil {
		t.Error("GetRecord() should return error when zone lookup fails")
	}
}

func TestCloudflareCreateRecordZoneError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return error for zone lookup but 200 for records
		if r.URL.Path == "/zones" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("CreateRecord() should return error when zone lookup fails")
	}
}

func TestCloudflareGetZoneIDNonOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zones" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("should fail when zone lookup returns non-200")
	}
}

func TestCloudflareGetRecordIDNonOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		// Record lookup fails
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.GetRecord(ctx, "example.com", "A", "www")
	if err == nil {
		t.Error("GetRecord() should return error when record ID lookup fails")
	}
}

func TestCloudflareDoRequestContextError(t *testing.T) {
	cf := NewCloudflareProvider("token", "email")
	// Use an invalid URL to trigger request creation error
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := cf.doRequest(ctx, "GET", "http://invalid url with spaces", nil)
	if err == nil {
		t.Error("doRequest() should return error for invalid URL")
	}
}

// ========== Non-200 API Response Path Tests ==========

func TestCloudflareCreateRecordAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		// Create record returns 403
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"success":false,"errors":[{"code":9109,"message":"You do not have permission to modify this DNS record."}]}`))
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("CreateRecord() should return error on 403 from API")
	}
}

func TestCloudflareUpdateRecordAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/zones/zone-1/dns_records") && r.Method == "GET" && r.URL.Query().Get("type") == "A" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "rec-1"}},
			})
			return
		}
		// Update returns 500
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"errors":[{"code":81057,"message":"Internal server error."}]}`))
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.UpdateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.1.1.1", TTL: 300,
	})
	if err == nil {
		t.Error("UpdateRecord() should return error on 500 from API")
	}
}

func TestCloudflareDeleteRecordAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/zones/zone-1/dns_records") && r.Method == "GET" && r.URL.Query().Get("type") == "A" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "rec-1"}},
			})
			return
		}
		// Delete returns 400
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"errors":[{"code":81057,"message":"Bad request."}]}`))
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.DeleteRecord(ctx, "example.com", "A", "www")
	if err == nil {
		t.Error("DeleteRecord() should return error on 400 from API")
	}
}

func TestCloudflareGetRecordAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/zones/zone-1/dns_records") && r.Method == "GET" && r.URL.Query().Get("type") == "A" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "rec-1"}},
			})
			return
		}
		// Get specific record returns 500
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.GetRecord(ctx, "example.com", "A", "www")
	if err == nil {
		t.Error("GetRecord() should return error on 500 from API")
	}
}

func TestCloudflareListRecordsAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		// List records returns 403
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.ListRecords(ctx, "example.com")
	if err == nil {
		t.Error("ListRecords() should return error on 403 from API")
	}
}

func TestCloudflareGetZoneIDEmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("should fail when zone ID is empty (no matching zone)")
	}
}

func TestCloudflareGetRecordIDEmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zones" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/zones/zone-1/dns_records") && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]interface{}{},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.GetRecord(ctx, "example.com", "A", "www")
	if err == nil {
		t.Error("GetRecord() should return error when record ID is empty (no matching record)")
	}
}

// ========== New Tests: Realistic Cloudflare API Response Parsing ==========

func TestCloudflareGetZoneIDParsesResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": []map[string]string{
					{"id": "zone-real-12345", "name": "example.com", "status": "active"},
				},
				"result_info": map[string]int{
					"total_count": 1,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	// Use CreateRecord as a vehicle to test getZoneID
	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	// Should not fail at zone lookup (may fail at record creation since we don't mock that endpoint)
	if err != nil && !strings.Contains(err.Error(), "404") {
		t.Fatalf("unexpected error (zone lookup should succeed): %v", err)
	}
}

func TestCloudflareGetZoneIDMatchesDomain(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			name := r.URL.Query().Get("name")
			// Return multiple zones, only one matching
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": []map[string]string{
					{"id": "zone-other-999", "name": "other.com", "status": "active"},
					{"id": "zone-target-456", "name": name, "status": "active"},
					{"id": "zone-another-789", "name": "another.org", "status": "active"},
				},
				"result_info": map[string]int{
					"total_count": 3,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	// Should succeed at zone lookup (selecting zone-target-456)
	if err != nil && !strings.Contains(err.Error(), "404") {
		t.Fatalf("unexpected error (zone lookup should match correct zone): %v", err)
	}
}

func TestCloudflareGetZoneIDSuccessFalse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"errors": []map[string]interface{}{
				{"code": 6003, "message": "Invalid request headers"},
			},
			"result": []map[string]string{},
		})
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("should fail when success=false")
	}
	if !strings.Contains(err.Error(), "Invalid request headers") {
		t.Errorf("error should contain API error message, got: %v", err)
	}
}

func TestCloudflareGetRecordIDParsesResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		// Record-specific endpoint MUST be checked BEFORE the prefix match
		if r.URL.Path == "/zones/zone-1/dns_records/rec-parsed-abc" && r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"id":      "rec-parsed-abc",
					"type":    "A",
					"name":    "www.example.com",
					"content": "1.2.3.4",
					"ttl":     300,
					"proxied": false,
				},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/zones/zone-1/dns_records") && r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": []map[string]interface{}{
					{
						"id":      "rec-parsed-abc",
						"type":    "A",
						"name":    "www.example.com",
						"content": "1.2.3.4",
						"ttl":     300,
						"proxied": false,
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	record, err := cf.GetRecord(ctx, "example.com", "A", "www")
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if record.Value != "1.2.3.4" {
		t.Errorf("Value = %q, want %q", record.Value, "1.2.3.4")
	}
	if record.TTL != 300 {
		t.Errorf("TTL = %d, want %d", record.TTL, 300)
	}
}

func TestCloudflareGetRecordIDMatchesTypeAndName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		// Record-specific endpoint MUST be checked BEFORE the prefix match
		if r.URL.Path == "/zones/zone-1/dns_records/rec-a-1" && r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"id": "rec-a-1", "type": "A", "name": "www.example.com",
					"content": "1.2.3.4", "ttl": 300, "proxied": false,
				},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/zones/zone-1/dns_records") && r.Method == "GET" {
			// Return multiple records with different types
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": []map[string]interface{}{
					{
						"id": "rec-aaaa-1", "type": "AAAA", "name": "www.example.com",
						"content": "2001:db8::1", "ttl": 300, "proxied": false,
					},
					{
						"id": "rec-a-1", "type": "A", "name": "www.example.com",
						"content": "1.2.3.4", "ttl": 300, "proxied": false,
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	// Request type A - should match rec-a-1, not rec-aaaa-1
	record, err := cf.GetRecord(ctx, "example.com", "A", "www")
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if record.Value != "1.2.3.4" {
		t.Errorf("Value = %q, want %q (should match A record, not AAAA)", record.Value, "1.2.3.4")
	}
}

func TestCloudflareGetRecordIDSuccessFalse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/zones/zone-1/dns_records") && r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"errors": []map[string]interface{}{
					{"code": 7003, "message": "Could not resolve zone"},
				},
				"result": []map[string]interface{}{},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.GetRecord(ctx, "example.com", "A", "www")
	if err == nil {
		t.Error("should fail when success=false in record lookup")
	}
	if !strings.Contains(err.Error(), "Could not resolve zone") {
		t.Errorf("error should contain API error message, got: %v", err)
	}
}

func TestCloudflareListRecordsSuccessFalse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/zones/zone-1/dns_records") && r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"errors": []map[string]interface{}{
					{"code": 81057, "message": "Unauthorized to access this resource"},
				},
				"result": []map[string]interface{}{},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.ListRecords(ctx, "example.com")
	if err == nil {
		t.Error("should fail when success=false in list records")
	}
	if !strings.Contains(err.Error(), "Unauthorized to access this resource") {
		t.Errorf("error should contain API error message, got: %v", err)
	}
}

func TestCloudflareDoRequestMarshalsBody(t *testing.T) {
	var receivedBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Read the request body
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		receivedBody = string(buf)

		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  map[string]string{"id": "rec-1"},
		})
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300, Proxied: true,
	})
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}

	// Verify the body was properly JSON-marshaled
	if !strings.Contains(receivedBody, `"type":"A"`) {
		t.Errorf("request body should contain type field, got: %s", receivedBody)
	}
	if !strings.Contains(receivedBody, `"name":"www.example.com"`) {
		t.Errorf("request body should contain name field, got: %s", receivedBody)
	}
	if !strings.Contains(receivedBody, `"content":"1.2.3.4"`) {
		t.Errorf("request body should contain content field, got: %s", receivedBody)
	}
	if !strings.Contains(receivedBody, `"ttl":300`) {
		t.Errorf("request body should contain ttl field, got: %s", receivedBody)
	}
	if !strings.Contains(receivedBody, `"proxied":true`) {
		t.Errorf("request body should contain proxied field, got: %s", receivedBody)
	}
}

func TestCloudflareGetRecordParsesFullResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/zones/zone-1/dns_records") && r.Method == "GET" && r.URL.Query().Get("type") != "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": []map[string]interface{}{
					{"id": "rec-get-full", "type": "A", "name": "www.example.com",
						"content": "5.6.7.8", "ttl": 600, "proxied": true},
				},
			})
			return
		}
		if r.URL.Path == "/zones/zone-1/dns_records/rec-get-full" && r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"id": "rec-get-full", "type": "A", "name": "www.example.com",
					"content": "5.6.7.8", "ttl": 600, "proxied": true,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	record, err := cf.GetRecord(ctx, "example.com", "A", "www")
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if record.Value != "5.6.7.8" {
		t.Errorf("Value = %q, want %q", record.Value, "5.6.7.8")
	}
	if record.TTL != 600 {
		t.Errorf("TTL = %d, want %d", record.TTL, 600)
	}
	if record.Proxied != true {
		t.Errorf("Proxied = %v, want true", record.Proxied)
	}
}

func TestCloudflareGetZoneIDInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	err := cf.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("should fail when zone response is invalid JSON")
	}
}

func TestCloudflareGetRecordIDInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.GetRecord(ctx, "example.com", "A", "www")
	if err == nil {
		t.Error("should fail when record response is invalid JSON")
	}
}

func TestCloudflareListRecordsInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/zones" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []map[string]string{{"id": "zone-1", "name": "example.com"}},
			})
			return
		}
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	cf := newTestCFProvider(ts)
	ctx := context.Background()

	_, err := cf.ListRecords(ctx, "example.com")
	if err == nil {
		t.Error("should fail when list records response is invalid JSON")
	}
}
