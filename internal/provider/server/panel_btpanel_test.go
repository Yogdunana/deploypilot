package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupBTPanelTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// POST /firewall?action=AddAcceptPort
	mux.HandleFunc("/firewall", func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")

		switch action {
		case "AddAcceptPort":
			resp := btPanelResponse{Status: true, Msg: "ok"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		case "DelAcceptPort":
			resp := btPanelResponse{Status: true, Msg: "ok"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	// POST /site?action=CreateReverseProxy
	mux.HandleFunc("/site", func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		if action == "CreateReverseProxy" {
			resp := btPanelResponse{Status: true, Msg: "ok"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	})

	return httptest.NewServer(mux)
}

func setupBTPanelErrorServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/firewall", func(w http.ResponseWriter, r *http.Request) {
		resp := btPanelResponse{Status: false, Msg: "permission denied"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func TestNewPanelBTPanel(t *testing.T) {
	p := NewPanelBTPanel("http://localhost:8888", "test-key")
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

func TestPanelBTPanel_OpenFirewall(t *testing.T) {
	server := setupBTPanelTestServer()
	defer server.Close()

	p := NewPanelBTPanel(server.URL, "test-api-key")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Fatalf("OpenFirewall() error = %v", err)
	}
}

func TestPanelBTPanel_OpenFirewall_APIError(t *testing.T) {
	server := setupBTPanelErrorServer()
	defer server.Close()

	p := NewPanelBTPanel(server.URL, "test-api-key")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err == nil {
		t.Fatal("OpenFirewall() should return error on API failure")
	}
}

func TestPanelBTPanel_CloseFirewall(t *testing.T) {
	server := setupBTPanelTestServer()
	defer server.Close()

	p := NewPanelBTPanel(server.URL, "test-api-key")
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Fatalf("CloseFirewall() error = %v", err)
	}
}

func TestPanelBTPanel_CloseFirewall_APIError(t *testing.T) {
	server := setupBTPanelErrorServer()
	defer server.Close()

	p := NewPanelBTPanel(server.URL, "test-api-key")
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err == nil {
		t.Fatal("CloseFirewall() should return error on API failure")
	}
}

func TestPanelBTPanel_CreateReverseProxy(t *testing.T) {
	server := setupBTPanelTestServer()
	defer server.Close()

	p := NewPanelBTPanel(server.URL, "test-api-key")
	err := p.CreateReverseProxy(context.TODO(), "example.com", "http://localhost:3000", 3000)
	if err != nil {
		t.Fatalf("CreateReverseProxy() error = %v", err)
	}
}

func TestPanelBTPanel_CreateReverseProxy_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := btPanelResponse{Status: false, Msg: "site already exists"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewPanelBTPanel(server.URL, "test-api-key")
	err := p.CreateReverseProxy(context.TODO(), "example.com", "http://localhost:3000", 3000)
	if err == nil {
		t.Fatal("CreateReverseProxy() should return error on API failure")
	}
}

func TestPanelBTPanel_OpenFirewall_SendsCorrectRequest(t *testing.T) {
	var receivedMethod, receivedPath, receivedBody string
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedAuth = r.Header.Get("Authorization")

		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)

		resp := btPanelResponse{Status: true, Msg: "ok"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewPanelBTPanel(server.URL, "my-secret-key")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Fatalf("OpenFirewall() error = %v", err)
	}

	if receivedMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", receivedMethod, http.MethodPost)
	}
	if receivedPath != "/firewall" {
		t.Errorf("path = %q, want %q", receivedPath, "/firewall")
	}
	if receivedAuth != "Bearer my-secret-key" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer my-secret-key")
	}

	var body map[string]interface{}
	_ = json.Unmarshal([]byte(receivedBody), &body)
	if body["port"] != "8080" {
		t.Errorf("port = %v, want %q", body["port"], "8080")
	}
	if body["ps"] != "deploypilot" {
		t.Errorf("ps = %v, want %q", body["ps"], "deploypilot")
	}
	if body["type"] != "port" {
		t.Errorf("type = %v, want %q", body["type"], "port")
	}
}

func TestPanelBTPanel_SetHTTPClient(t *testing.T) {
	p := NewPanelBTPanel("http://localhost:8888", "test-key")
	custom := &http.Client{}
	p.SetHTTPClient(custom)
	if p.httpClient != custom {
		t.Error("SetHTTPClient() did not set the client")
	}
}
