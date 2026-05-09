package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setup1PanelTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// Firewall rules - list/create (exact path)
		case r.URL.Path == "/api/v1/firewall/rules":
			switch r.Method {
			case http.MethodPost:
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code": 200, "message": "ok", "data": map[string]interface{}{"id": 1},
				})
			case http.MethodGet:
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code": 200, "message": "ok",
					"data": map[string]interface{}{
						"items": []map[string]interface{}{
							{"id": 1, "protocol": "tcp", "port": "8080", "address": "", "comment": "deploypilot", "action": "accept"},
							{"id": 2, "protocol": "tcp", "port": "9090", "address": "", "comment": "other", "action": "accept"},
						},
						"total": 2,
					},
				})
			default:
				http.NotFound(w, r)
			}

		// Firewall rules - delete by ID (e.g., /api/v1/firewall/rules/1)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/firewall/rules/"):
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "message": "ok"})

		// Delete website by ID (e.g., /api/v1/websites/10)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/websites/") && r.URL.Path != "/api/v1/websites/reverse_proxy":
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "message": "ok"})

		// Create reverse proxy
		case r.URL.Path == "/api/v1/websites/reverse_proxy" && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "message": "ok", "data": map[string]interface{}{"id": 5}})

		// Website list (GET) or create (POST)
		case r.URL.Path == "/api/v1/websites":
			switch r.Method {
			case http.MethodGet:
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code": 200, "message": "ok",
					"data": map[string]interface{}{
						"items": []map[string]interface{}{
							{"id": 10, "primaryDomain": "app.example.com", "type": "reverse_proxy", "status": true, "remark": "my app", "ssl": false, "createdAt": "2026-04-28"},
							{"id": 11, "primaryDomain": "blog.example.com", "type": "static", "status": true, "remark": "blog", "ssl": true, "createdAt": "2026-04-27"},
						},
						"total": 2,
					},
				})
			case http.MethodPost:
				json.NewEncoder(w).Encode(map[string]interface{}{
					"code": 200, "message": "ok",
					"data": map[string]interface{}{"id": 12, "primaryDomain": "new.example.com", "type": "static", "status": true},
				})
			default:
				http.NotFound(w, r)
			}

		default:
			http.NotFound(w, r)
		}
	}))
}

func setup1PanelErrorServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 500, "message": "internal error"})
	}))
}

func TestNewPanel1Client(t *testing.T) {
	client := NewPanel1Client("http://localhost:4004", "test-key")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != "http://localhost:4004" {
		t.Errorf("expected baseURL http://localhost:4004, got %s", client.baseURL)
	}
	if client.apiKey != "test-key" {
		t.Errorf("expected apiKey test-key, got %s", client.apiKey)
	}
}

func TestPanel1Client_OpenFirewall(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if err := client.OpenFirewall(context.TODO(), 8080, "tcp"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPanel1Client_OpenFirewall_APIError(t *testing.T) {
	server := setup1PanelErrorServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if err := client.OpenFirewall(context.TODO(), 8080, "tcp"); err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestPanel1Client_CloseFirewall(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if err := client.CloseFirewall(context.TODO(), 8080, "tcp"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPanel1Client_CloseFirewall_NoMatchingRule(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if err := client.CloseFirewall(context.TODO(), 9999, "tcp"); err != nil {
		t.Fatalf("expected no error for non-matching rule, got %v", err)
	}
}

func TestPanel1Client_CloseFirewall_ListError(t *testing.T) {
	server := setup1PanelErrorServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if err := client.CloseFirewall(context.TODO(), 8080, "tcp"); err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestPanel1Client_CreateReverseProxy(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if err := client.CreateReverseProxy(context.TODO(), "app.example.com", "http://localhost:3000", 3000); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPanel1Client_CreateReverseProxy_APIError(t *testing.T) {
	server := setup1PanelErrorServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if err := client.CreateReverseProxy(context.TODO(), "app.example.com", "http://localhost:3000", 3000); err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestPanel1Client_DeleteReverseProxy(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if err := client.DeleteReverseProxy(context.TODO(), "app.example.com"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPanel1Client_DeleteReverseProxy_NotFound(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if err := client.DeleteReverseProxy(context.TODO(), "nonexistent.example.com"); err == nil {
		t.Fatal("expected error for non-existent domain")
	}
}

func TestPanel1Client_DeleteReverseProxy_APIError(t *testing.T) {
	server := setup1PanelErrorServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if err := client.DeleteReverseProxy(context.TODO(), "app.example.com"); err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestPanel1Client_CreateWebsite(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	info, err := client.CreateWebsite(context.TODO(), "new.example.com", "static", "test site")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil website info")
	}
	if info.PrimaryDomain != "new.example.com" {
		t.Errorf("expected domain new.example.com, got %s", info.PrimaryDomain)
	}
	if info.Type != "static" {
		t.Errorf("expected type static, got %s", info.Type)
	}
}

func TestPanel1Client_CreateWebsite_APIError(t *testing.T) {
	server := setup1PanelErrorServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if _, err := client.CreateWebsite(context.TODO(), "new.example.com", "static", "test site"); err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestPanel1Client_GetWebsiteList(t *testing.T) {
	server := setup1PanelTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	websites, err := client.GetWebsiteList(context.TODO())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(websites) != 2 {
		t.Fatalf("expected 2 websites, got %d", len(websites))
	}
	if websites[0].PrimaryDomain != "app.example.com" {
		t.Errorf("expected first domain app.example.com, got %s", websites[0].PrimaryDomain)
	}
	if websites[0].Type != "reverse_proxy" {
		t.Errorf("expected first type reverse_proxy, got %s", websites[0].Type)
	}
	if websites[1].PrimaryDomain != "blog.example.com" {
		t.Errorf("expected second domain blog.example.com, got %s", websites[1].PrimaryDomain)
	}
	if !websites[1].SSL {
		t.Error("expected second website to have SSL enabled")
	}
}

func TestPanel1Client_GetWebsiteList_APIError(t *testing.T) {
	server := setup1PanelErrorServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	if _, err := client.GetWebsiteList(context.TODO()); err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestPanel1Client_OpenFirewall_SendsCorrectRequest(t *testing.T) {
	var receivedMethod, receivedPath, receivedAuth string
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedAuth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "message": "ok", "data": map[string]interface{}{"id": 1}})
	}))
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-api-key")
	_ = client.OpenFirewall(context.TODO(), 443, "tcp")
	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", receivedMethod)
	}
	if receivedPath != "/api/v1/firewall/rules" {
		t.Errorf("expected path /api/v1/firewall/rules, got %s", receivedPath)
	}
	if receivedAuth != "Bearer test-api-key" {
		t.Errorf("expected Authorization Bearer test-api-key, got %s", receivedAuth)
	}
	if receivedBody["port"] != "443" {
		t.Errorf("expected port 443, got %v", receivedBody["port"])
	}
	if receivedBody["protocol"] != "tcp" {
		t.Errorf("expected protocol tcp, got %v", receivedBody["protocol"])
	}
	if receivedBody["comment"] != "deploypilot" {
		t.Errorf("expected comment deploypilot, got %v", receivedBody["comment"])
	}
}

func TestPanel1Client_SetHTTPClient(t *testing.T) {
	client := NewPanel1Client("http://localhost:4004", "test-key")
	customClient := &http.Client{}
	client.SetHTTPClient(customClient)
	if client.httpClient != customClient {
		t.Error("expected httpClient to be replaced")
	}
}

func TestPanel1Client_GetInfo(t *testing.T) {
	client := NewPanel1Client("http://localhost:4004", "test-key")
	info := client.GetInfo()
	if info["name"] != "1Panel" {
		t.Errorf("expected name 1Panel, got %v", info["name"])
	}
	features, ok := info["features"].([]string)
	if !ok {
		t.Fatal("expected features to be []string")
	}
	if len(features) == 0 {
		t.Error("expected non-empty features list")
	}
}

func TestPanelProvider_DeleteReverseProxy_NilClient(t *testing.T) {
	provider := NewPanelProvider(nil)
	if err := provider.DeleteReverseProxy(context.TODO(), "example.com"); err != nil {
		t.Errorf("expected nil error for nil client, got %v", err)
	}
}

func TestPanelProvider_CreateWebsite_NilClient(t *testing.T) {
	provider := NewPanelProvider(nil)
	if _, err := provider.CreateWebsite(context.TODO(), "example.com", "static", "test"); err == nil {
		t.Error("expected error for nil client")
	}
}

func TestPanelProvider_GetWebsiteList_NilClient(t *testing.T) {
	provider := NewPanelProvider(nil)
	if _, err := provider.GetWebsiteList(context.TODO()); err == nil {
		t.Error("expected error for nil client")
	}
}
