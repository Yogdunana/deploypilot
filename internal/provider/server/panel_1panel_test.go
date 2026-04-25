package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func setup1PanelTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// POST /api/v1/firewall/rules - create rule
	mux.HandleFunc("/api/v1/firewall/rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			_ = json.Unmarshal(body, &req)

			resp := panel1Response{
				Code:    200,
				Message: "success",
				Data:    json.RawMessage(`{"id": 1}`),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodGet {
			resp := panel1Response{
				Code:    200,
				Message: "success",
				Data: json.RawMessage(`{
					"items": [
						{"id": 1, "protocol": "tcp", "port": "8080", "address": "", "comment": "deploypilot", "action": "accept"},
						{"id": 2, "protocol": "udp", "port": "9090", "address": "", "comment": "other", "action": "accept"}
					],
					"total": 2
				}`),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodDelete {
			resp := panel1Response{
				Code:    200,
				Message: "success",
				Data:    nil,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	// POST /api/v1/websites/reverse_proxy
	mux.HandleFunc("/api/v1/websites/reverse_proxy", func(w http.ResponseWriter, r *http.Request) {
		resp := panel1Response{
			Code:    200,
			Message: "success",
			Data:    json.RawMessage(`{"id": 1}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func setup1PanelErrorServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/firewall/rules", func(w http.ResponseWriter, r *http.Request) {
		resp := panel1Response{
			Code:    500,
			Message: "internal error",
			Data:    nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func TestNewPanel1Panel(t *testing.T) {
	p := NewPanel1Panel("http://localhost:8888", "test-key")
	if p.baseURL != "http://localhost:8888" {
		t.Errorf("baseURL = %q, want %q", p.baseURL, "http://localhost:8888")
	}
	if p.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", p.apiKey, "test-key")
	}
	if p.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestPanel1Panel_OpenFirewall(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()

	p := NewPanel1Panel(server.URL, "test-api-key")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Fatalf("OpenFirewall() error = %v", err)
	}
}

func TestPanel1Panel_OpenFirewall_APIError(t *testing.T) {
	server := setup1PanelErrorServer()
	defer server.Close()

	p := NewPanel1Panel(server.URL, "test-api-key")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err == nil {
		t.Fatal("OpenFirewall() should return error on API failure")
	}
}

func TestPanel1Panel_CloseFirewall(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()

	p := NewPanel1Panel(server.URL, "test-api-key")
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Fatalf("CloseFirewall() error = %v", err)
	}
}

func TestPanel1Panel_CloseFirewall_NoMatchingRule(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()

	p := NewPanel1Panel(server.URL, "test-api-key")
	// Use a port that doesn't exist in the mock data
	err := p.CloseFirewall(context.TODO(), 9999, "tcp")
	if err != nil {
		t.Fatalf("CloseFirewall() with no matching rule should not error, got = %v", err)
	}
}

func TestPanel1Panel_CloseFirewall_ListError(t *testing.T) {
	server := setup1PanelErrorServer()
	defer server.Close()

	p := NewPanel1Panel(server.URL, "test-api-key")
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err == nil {
		t.Fatal("CloseFirewall() should return error when list fails")
	}
}

func TestPanel1Panel_CreateReverseProxy(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()

	p := NewPanel1Panel(server.URL, "test-api-key")
	err := p.CreateReverseProxy(context.TODO(), "example.com", "http://localhost:3000", 3000)
	if err != nil {
		t.Fatalf("CreateReverseProxy() error = %v", err)
	}
}

func TestPanel1Panel_OpenFirewall_SendsCorrectRequest(t *testing.T) {
	var receivedMethod, receivedPath, receivedBody string
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedAuth = r.Header.Get("Authorization")

		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)

		resp := panel1Response{Code: 200, Message: "success"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewPanel1Panel(server.URL, "my-secret-key")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Fatalf("OpenFirewall() error = %v", err)
	}

	if receivedMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", receivedMethod, http.MethodPost)
	}
	if receivedPath != "/api/v1/firewall/rules" {
		t.Errorf("path = %q, want %q", receivedPath, "/api/v1/firewall/rules")
	}
	if receivedAuth != "Bearer my-secret-key" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer my-secret-key")
	}

	var body map[string]interface{}
	_ = json.Unmarshal([]byte(receivedBody), &body)
	if body["port"] != strconv.Itoa(8080) {
		t.Errorf("port = %v, want %q", body["port"], "8080")
	}
	if body["protocol"] != "tcp" {
		t.Errorf("protocol = %v, want %q", body["protocol"], "tcp")
	}
	if body["comment"] != "deploypilot" {
		t.Errorf("comment = %v, want %q", body["comment"], "deploypilot")
	}
}

func TestPanel1Panel_SetHTTPClient(t *testing.T) {
	p := NewPanel1Panel("http://localhost:8888", "test-key")
	custom := &http.Client{}
	p.SetHTTPClient(custom)
	if p.httpClient != custom {
		t.Error("SetHTTPClient() did not set the client")
	}
}
