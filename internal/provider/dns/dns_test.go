package dns

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// ========== DNSProvider Interface Tests (via Mock) ==========

func TestDNSProviderCreateAndGet(t *testing.T) {
	provider := NewMockProvider()

	err := provider.CreateRecord(context.Background(), &DNSRecord{
		Domain:   "example.com",
		Type:     "A",
		Name:     "www",
		Value:    "1.2.3.4",
		TTL:      300,
		Proxied:  false,
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

// ========== Cloudflare Provider Tests ==========

func TestCloudflareProviderConfig(t *testing.T) {
	cf := NewCloudflareProvider("test-api-token", "test@account.com")

	if cf.APIToken != "test-api-token" {
		t.Errorf("APIToken = %q", cf.APIToken)
	}
	if cf.AccountEmail != "test@account.com" {
		t.Errorf("AccountEmail = %q", cf.AccountEmail)
	}
}

func TestCloudflareProviderCreateRecordMock(t *testing.T) {
	cf := NewCloudflareProvider("test-token", "test@example.com")
	cf.SetBaseURL("http://localhost:0") // no real server

	// Cloudflare provider needs a real server for full test,
	// but we test the config and interface compliance here
	var _ DNSProvider = cf
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

func recordKey(domain, recordType, name string) string {
	return strings.ToLower(fmt.Sprintf("%s:%s:%s", domain, recordType, name))
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
