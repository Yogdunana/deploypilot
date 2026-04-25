package dns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// ========== WestDNS Provider Tests ==========

func TestNewWestDNSProvider(t *testing.T) {
	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	if p.apiKey != "test-api-key" {
		t.Errorf("apiKey = %q, want %q", p.apiKey, "test-api-key")
	}
	if p.apiSecret != "test-api-secret" {
		t.Errorf("apiSecret = %q, want %q", p.apiSecret, "test-api-secret")
	}
	if p.baseURL != "https://api.west.cn" {
		t.Errorf("baseURL = %q, want %q", p.baseURL, "https://api.west.cn")
	}
}

func TestWestDNSName(t *testing.T) {
	p := NewWestDNSProvider("key", "secret")
	if p.Name() != "west-dns" {
		t.Errorf("Name() = %q, want %q", p.Name(), "west-dns")
	}
}

func TestWestDNSSetBaseURL(t *testing.T) {
	p := NewWestDNSProvider("key", "secret")
	p.SetBaseURL("http://custom-url")
	if p.baseURL != "http://custom-url" {
		t.Errorf("baseURL = %q, want %q", p.baseURL, "http://custom-url")
	}
}

// newWestDNSTestServer creates a mock HTTP server that simulates the WestDNS API.
func newWestDNSTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	// Predefined records for the mock server
	records := []westDNSRecord{
		{ID: 1001, Item: "www", Value: "1.2.3.4", Type: "A", Level: 10, TTL: 900, Line: "", Pause: 0},
		{ID: 1002, Item: "api", Value: "5.6.7.8", Type: "A", Level: 10, TTL: 900, Line: "", Pause: 0},
		{ID: 1003, Item: "@", Value: "mail.example.com", Type: "MX", Level: 10, TTL: 3600, Line: "", Pause: 0},
	}
	nextID := 1004

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Basic Auth
		apiKey, apiSecret, ok := r.BasicAuth()
		if !ok || apiKey != "test-api-key" || apiSecret != "test-api-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{"result": -1})
			return
		}

		act := r.URL.Query().Get("act")
		w.Header().Set("Content-Type", "application/json")

		switch act {
		case "getdnsrecord":
			_ = r.ParseForm()
			_ = r.FormValue("domain")

			data := westDNSRecordListData{
				Items:      records,
				Limit:      100,
				Total:      len(records),
				PageNo:     1,
				TotalPages: 1,
			}
			dataBytes, _ := json.Marshal(data)
			json.NewEncoder(w).Encode(westDNSAPIResponse{
				Result:   0,
				ClientID: "test-client",
				Data:     dataBytes,
			})

		case "adddnsrecord":
			_ = r.ParseForm()
			host := r.FormValue("host")
			recordType := r.FormValue("type")
			value := r.FormValue("value")
			ttlStr := r.FormValue("ttl")
			ttl := 900
			if ttlStr != "" {
				if v, err := strconv.Atoi(ttlStr); err == nil {
					ttl = v
				}
			}

			newRecord := westDNSRecord{
				ID:    nextID,
				Item:  host,
				Value: value,
				Type:  recordType,
				Level: 10,
				TTL:   ttl,
				Line:  "",
				Pause: 0,
			}
			records = append(records, newRecord)
			nextID++

			addResult := westDNSAddResult{ID: newRecord.ID}
			dataBytes, _ := json.Marshal(addResult)
			json.NewEncoder(w).Encode(westDNSAPIResponse{
				Result:   0,
				ClientID: "test-client",
				Data:     dataBytes,
			})

		case "moddnsrecord":
			_ = r.ParseForm()
			idStr := r.FormValue("id")
			newValue := r.FormValue("value")
			newTTLStr := r.FormValue("ttl")

			id, _ := strconv.Atoi(idStr)
			newTTL := 900
			if newTTLStr != "" {
				if v, err := strconv.Atoi(newTTLStr); err == nil {
					newTTL = v
				}
			}

			found := false
			for i := range records {
				if records[i].ID == id {
					records[i].Value = newValue
					records[i].TTL = newTTL
					found = true
					break
				}
			}
			if !found {
				json.NewEncoder(w).Encode(westDNSAPIResponse{
					Result:   -1,
					ClientID: "test-client",
				})
				return
			}

			json.NewEncoder(w).Encode(westDNSAPIResponse{
				Result:   0,
				ClientID: "test-client",
			})

		case "deldnsrecord":
			_ = r.ParseForm()
			idStr := r.FormValue("id")
			id, _ := strconv.Atoi(idStr)

			found := false
			for i, rec := range records {
				if rec.ID == id {
					records = append(records[:i], records[i+1:]...)
					found = true
					break
				}
			}
			if !found {
				json.NewEncoder(w).Encode(westDNSAPIResponse{
					Result:   -1,
					ClientID: "test-client",
				})
				return
			}

			json.NewEncoder(w).Encode(westDNSAPIResponse{
				Result:   0,
				ClientID: "test-client",
			})

		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"result": -1})
		}
	}))
}

func TestWestDNSCreateRecord(t *testing.T) {
	ts := newWestDNSTestServer(t)
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	err := p.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "new", Value: "9.9.9.9", TTL: 600,
	})
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}
}

func TestWestDNSListRecords(t *testing.T) {
	ts := newWestDNSTestServer(t)
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	records, err := p.ListRecords(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if len(records) != 3 {
		t.Errorf("len(records) = %d, want 3", len(records))
	}
}

func TestWestDNSGetRecord(t *testing.T) {
	ts := newWestDNSTestServer(t)
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	record, err := p.GetRecord(context.Background(), "example.com", "A", "www")
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if record.Value != "1.2.3.4" {
		t.Errorf("Value = %q, want %q", record.Value, "1.2.3.4")
	}
	if record.Type != "A" {
		t.Errorf("Type = %q, want %q", record.Type, "A")
	}
	if record.Name != "www" {
		t.Errorf("Name = %q, want %q", record.Name, "www")
	}
}

func TestWestDNSUpdateRecord(t *testing.T) {
	ts := newWestDNSTestServer(t)
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	err := p.UpdateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "10.10.10.10", TTL: 600,
	})
	if err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}
}

func TestWestDNSDeleteRecord(t *testing.T) {
	ts := newWestDNSTestServer(t)
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	err := p.DeleteRecord(context.Background(), "example.com", "A", "www")
	if err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
}

// ========== Error Cases ==========

func TestWestDNSCreateRecordHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	err := p.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("CreateRecord() should return error on 500")
	}
}

func TestWestDNSListRecordsHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	_, err := p.ListRecords(context.Background(), "example.com")
	if err == nil {
		t.Error("ListRecords() should return error on 500")
	}
}

func TestWestDNSGetRecordHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	_, err := p.GetRecord(context.Background(), "example.com", "A", "www")
	if err == nil {
		t.Error("GetRecord() should return error on 500")
	}
}

func TestWestDNSUpdateRecordHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	err := p.UpdateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("UpdateRecord() should return error on 500")
	}
}

func TestWestDNSDeleteRecordHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	err := p.DeleteRecord(context.Background(), "example.com", "A", "www")
	if err == nil {
		t.Error("DeleteRecord() should return error on 500")
	}
}

func TestWestDNSListRecordsParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	_, err := p.ListRecords(context.Background(), "example.com")
	if err == nil {
		t.Error("ListRecords() should return error for invalid JSON")
	}
}

func TestWestDNSListRecordsAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(westDNSAPIResponse{
			Result:   -1,
			ClientID: "test-client",
		})
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	_, err := p.ListRecords(context.Background(), "example.com")
	if err == nil {
		t.Error("ListRecords() should return error when API returns non-zero result")
	}
}

func TestWestDNSGetRecordNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := westDNSRecordListData{
			Items:      []westDNSRecord{},
			Limit:      100,
			Total:      0,
			PageNo:     1,
			TotalPages: 1,
		}
		dataBytes, _ := json.Marshal(data)
		json.NewEncoder(w).Encode(westDNSAPIResponse{
			Result:   0,
			ClientID: "test-client",
			Data:     dataBytes,
		})
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	_, err := p.GetRecord(context.Background(), "example.com", "A", "nonexistent")
	if err == nil {
		t.Error("GetRecord() should return error when record not found")
	}
}

func TestWestDNSUpdateRecordNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := westDNSRecordListData{
			Items:      []westDNSRecord{},
			Limit:      100,
			Total:      0,
			PageNo:     1,
			TotalPages: 1,
		}
		dataBytes, _ := json.Marshal(data)
		json.NewEncoder(w).Encode(westDNSAPIResponse{
			Result:   0,
			ClientID: "test-client",
			Data:     dataBytes,
		})
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	err := p.UpdateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "nonexistent", Value: "1.1.1.1", TTL: 300,
	})
	if err == nil {
		t.Error("UpdateRecord() should return error when record not found")
	}
}

func TestWestDNSDeleteRecordNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := westDNSRecordListData{
			Items:      []westDNSRecord{},
			Limit:      100,
			Total:      0,
			PageNo:     1,
			TotalPages: 1,
		}
		dataBytes, _ := json.Marshal(data)
		json.NewEncoder(w).Encode(westDNSAPIResponse{
			Result:   0,
			ClientID: "test-client",
			Data:     dataBytes,
		})
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	err := p.DeleteRecord(context.Background(), "example.com", "A", "nonexistent")
	if err == nil {
		t.Error("DeleteRecord() should return error when record not found")
	}
}

func TestWestDNSAuthFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"result": -1})
	}))
	defer ts.Close()

	p := NewWestDNSProvider("wrong-key", "wrong-secret")
	p.SetBaseURL(ts.URL)

	err := p.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("CreateRecord() should return error on auth failure")
	}
}

func TestWestDNSContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.CreateRecord(ctx, &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("should return error when context is cancelled")
	}
}

func TestWestDNSCreateRecordAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(westDNSAPIResponse{
			Result:   -1,
			ClientID: "test-client",
		})
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	err := p.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("CreateRecord() should return error when API returns non-zero result")
	}
}

func TestWestDNSGetRecordDataParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(westDNSAPIResponse{
			Result:   0,
			ClientID: "test-client",
			Data:     json.RawMessage(`"not an object"`),
		})
	}))
	defer ts.Close()

	p := NewWestDNSProvider("test-api-key", "test-api-secret")
	p.SetBaseURL(ts.URL)

	_, err := p.GetRecord(context.Background(), "example.com", "A", "www")
	if err == nil {
		t.Error("GetRecord() should return error when data is not valid JSON object")
	}
}
