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
					{"id": "zone-placeholder", "name": name},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// DNS records endpoint (using zone-placeholder as returned by getZoneID)
		if r.URL.Path == "/zones/zone-placeholder/dns_records" {
			switch r.Method {
			case "GET":
				resp := map[string]interface{}{
					"success": true,
					"result": []map[string]interface{}{
						{
							"id":      "record-placeholder",
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
						"id":      "record-placeholder",
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
						"id": "record-placeholder",
					},
				}
				json.NewEncoder(w).Encode(resp)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}

		// Record-specific endpoint (for GET/PUT/DELETE with record ID)
		if r.URL.Path == "/zones/zone-placeholder/dns_records/record-placeholder" {
			switch r.Method {
			case "GET":
				resp := map[string]interface{}{
					"success": true,
					"result": map[string]interface{}{
						"id":      "record-placeholder",
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
						"id":      "record-placeholder",
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
						"id": "record-placeholder",
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
