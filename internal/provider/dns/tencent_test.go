package dns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ========== Tencent Provider Tests ==========

func TestNewTencentProvider(t *testing.T) {
	p := NewTencentProvider("test-secret-id", "test-secret-key")
	if p.secretID != "test-secret-id" {
		t.Errorf("secretID = %q, want %q", p.secretID, "test-secret-id")
	}
	if p.secretKey != "test-secret-key" {
		t.Errorf("secretKey = %q, want %q", p.secretKey, "test-secret-key")
	}
	if p.baseURL != "https://dnspod.tencentcloudapi.com" {
		t.Errorf("baseURL = %q, want default", p.baseURL)
	}
}

func TestTencentName(t *testing.T) {
	p := NewTencentProvider("id", "key")
	if p.Name() != "tencent" {
		t.Errorf("Name() = %q, want %q", p.Name(), "tencent")
	}
}

func TestTencentSignHelpers(t *testing.T) {
	// Test sha256Hex
	result := sha256Hex([]byte("hello"))
	if len(result) != 64 {
		t.Errorf("sha256Hex length = %d, want 64", len(result))
	}

	// Test hmacSHA256
	mac := hmacSHA256([]byte("key"), "data")
	if len(mac) != 32 {
		t.Errorf("hmacSHA256 length = %d, want 32", len(mac))
	}
}

func newTencentTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		action, _ := reqBody["Action"].(string)

		switch action {
		case "DescribeRecordList":
			resp := map[string]interface{}{
				"Response": map[string]interface{}{
					"RecordCountInfo": map[string]interface{}{
						"TotalCount": 2,
					},
					"RecordList": []map[string]interface{}{
						{
							"RecordId": 1001,
							"Name":     "www",
							"Type":     "A",
							"Value":    "1.2.3.4",
							"TTL":      300,
						},
						{
							"RecordId": 1002,
							"Name":     "api",
							"Type":     "A",
							"Value":    "5.6.7.8",
							"TTL":      300,
						},
					},
					"RequestId": "test-req-id",
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "CreateRecord":
			resp := map[string]interface{}{
				"Response": map[string]interface{}{
					"RecordId":  1003,
					"RequestId": "test-req-id",
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "DescribeRecord":
			resp := map[string]interface{}{
				"Response": map[string]interface{}{
					"RecordInfo": map[string]interface{}{
						"RecordId": 1001,
						"Name":     "www",
						"Type":     "A",
						"Value":    "1.2.3.4",
						"TTL":      300,
						"Status":   "ENABLE",
					},
					"RequestId": "test-req-id",
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "ModifyRecord":
			resp := map[string]interface{}{
				"Response": map[string]interface{}{
					"RecordId":  1001,
					"RequestId": "test-req-id",
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "DeleteRecord":
			resp := map[string]interface{}{
				"Response": map[string]interface{}{
					"RecordId":  1001,
					"RequestId": "test-req-id",
				},
			}
			json.NewEncoder(w).Encode(resp)
		default:
			resp := map[string]interface{}{
				"Response": map[string]interface{}{
					"Error": map[string]interface{}{
						"Code":    "InvalidParameter",
						"Message": "unknown action",
					},
					"RequestId": "test-req-id",
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
}

func TestTencentCreateRecord(t *testing.T) {
	ts := newTencentTestServer(t)
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	err := p.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "new", Value: "9.9.9.9", TTL: 300,
	})
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}
}

func TestTencentListRecords(t *testing.T) {
	ts := newTencentTestServer(t)
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	records, err := p.ListRecords(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Errorf("len(records) = %d, want 2", len(records))
	}
}

func TestTencentGetRecord(t *testing.T) {
	ts := newTencentTestServer(t)
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	record, err := p.GetRecord(context.Background(), "example.com", "A", "www")
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if record.Value != "1.2.3.4" {
		t.Errorf("Value = %q, want %q", record.Value, "1.2.3.4")
	}
}

func TestTencentUpdateRecord(t *testing.T) {
	ts := newTencentTestServer(t)
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	err := p.UpdateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "9.9.9.9", TTL: 600,
	})
	if err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}
}

func TestTencentDeleteRecord(t *testing.T) {
	ts := newTencentTestServer(t)
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	err := p.DeleteRecord(context.Background(), "example.com", "A", "www")
	if err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
}

func TestTencentCreateRecordError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	err := p.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("CreateRecord() should return error on 500")
	}
}

func TestTencentSetBaseURL(t *testing.T) {
	p := NewTencentProvider("id", "key")
	p.SetBaseURL("http://custom-url")
	if p.baseURL != "http://custom-url" {
		t.Errorf("baseURL = %q, want %q", p.baseURL, "http://custom-url")
	}
}

func TestTencentListRecordsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	_, err := p.ListRecords(context.Background(), "example.com")
	if err == nil {
		t.Error("ListRecords() should return error on 500")
	}
}

func TestTencentGetRecordError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	_, err := p.GetRecord(context.Background(), "example.com", "A", "www")
	if err == nil {
		t.Error("GetRecord() should return error on 500")
	}
}

func TestTencentUpdateRecordError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	err := p.UpdateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("UpdateRecord() should return error on 500")
	}
}

func TestTencentDeleteRecordError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	err := p.DeleteRecord(context.Background(), "example.com", "A", "www")
	if err == nil {
		t.Error("DeleteRecord() should return error on 500")
	}
}

func TestTencentListRecordsAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Response": map[string]interface{}{
				"Error":     map[string]interface{}{"Code": "Forbidden", "Message": "access denied"},
				"RequestId": "test-req-id",
			},
		})
	}))
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	_, err := p.ListRecords(context.Background(), "example.com")
	if err == nil {
		t.Error("ListRecords() should return error when API returns error code")
	}
}

func TestTencentGetRecordNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Response": map[string]interface{}{
				"RecordCountInfo": map[string]interface{}{"TotalCount": 0},
				"RecordList":      []map[string]interface{}{},
				"RequestId":       "test-req-id",
			},
		})
	}))
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	_, err := p.GetRecord(context.Background(), "example.com", "A", "nonexistent")
	if err == nil {
		t.Error("GetRecord() should return error when record not found")
	}
}

func TestTencentGetRecordParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	_, err := p.GetRecord(context.Background(), "example.com", "A", "www")
	if err == nil {
		t.Error("GetRecord() should return error when response is not valid JSON")
	}
}

func TestTencentCreateRecordAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Response": map[string]interface{}{
				"Error":     map[string]interface{}{"Code": "InvalidParameter", "Message": "invalid domain"},
				"RequestId": "test-req-id",
			},
		})
	}))
	defer ts.Close()

	p := NewTencentProvider("test-id", "test-key")
	p.SetBaseURL(ts.URL)

	err := p.CreateRecord(context.Background(), &DNSRecord{
		Domain: "example.com", Type: "A", Name: "www", Value: "1.2.3.4", TTL: 300,
	})
	if err == nil {
		t.Error("CreateRecord() should return error when API returns error code")
	}
}
