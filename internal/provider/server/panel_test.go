package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockCommandExecutor implements CommandExecutor for panel tests.
type mockCommandExecutor struct {
	output map[string]string
	err    error
}

func (m *mockCommandExecutor) RunCommand(_ context.Context, cmd string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if out, ok := m.output[cmd]; ok {
		return out, nil
	}
	return "", nil
}

// mockPanelClient implements PanelClient for tests.
type mockPanelClient struct {
	name     string
	features []string
}

func (m *mockPanelClient) OpenFirewall(_ context.Context, _ int, _ string) error { return nil }
func (m *mockPanelClient) CloseFirewall(_ context.Context, _ int, _ string) error { return nil }
func (m *mockPanelClient) CreateReverseProxy(_ context.Context, _, _ string, _ int) error { return nil }
func (m *mockPanelClient) DeleteReverseProxy(_ context.Context, _ string) error             { return nil }
func (m *mockPanelClient) CreateWebsite(_ context.Context, _, _, _ string) (*WebsiteInfo, error) {
	return &WebsiteInfo{PrimaryDomain: "mock.local", Type: "static", Status: true}, nil
}
func (m *mockPanelClient) GetWebsiteList(_ context.Context) ([]WebsiteInfo, error) {
	return []WebsiteInfo{{PrimaryDomain: "mock.local", Type: "static", Status: true}}, nil
}
func (m *mockPanelClient) GetInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":     m.name,
		"features": m.features,
	}
}

// setupPanelProviderTestServer creates a mock HTTP server that handles both 1Panel and BT-Panel API calls.
func setupPanelProviderTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// 1Panel endpoints
	mux.HandleFunc("/api/v1/firewall/rules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200, "message": "ok",
				"data": map[string]interface{}{
					"items": []map[string]interface{}{
						{"id": 1, "protocol": "tcp", "port": "8080", "address": "", "comment": "deploypilot", "action": "accept"},
					},
					"total": 1,
				},
			})
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "message": "ok", "data": map[string]interface{}{"id": 1}})
		}
	})
	mux.HandleFunc("/api/v1/websites/reverse_proxy", func(w http.ResponseWriter, r *http.Request) {
		resp := panel1Response{Code: 200, Message: "success", Data: json.RawMessage(`{"id": 1}`)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/api/v1/websites", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			resp := panel1Response{Code: 200, Message: "success", Data: json.RawMessage(`{"items":[],"total":0}`)}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	})

	// BT-Panel endpoints
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "bt_user_token=test-token; Path=/")
		resp := btPanelResponse{Status: true, Msg: "Login successful"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/firewall", func(w http.ResponseWriter, r *http.Request) {
		resp := btPanelResponse{Status: true, Msg: "ok", Data: map[string]interface{}{
			"list": []map[string]interface{}{
				{
					"id":       "1",
					"port":     "8080",
					"protocol": "tcp",
					"ps":       "deploypilot",
				},
			},
		}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/firewall/add", func(w http.ResponseWriter, r *http.Request) {
		resp := btPanelResponse{Status: true, Msg: "Firewall rule added"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/firewall/del", func(w http.ResponseWriter, r *http.Request) {
		resp := btPanelResponse{Status: true, Msg: "Firewall rule deleted"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/site/add", func(w http.ResponseWriter, r *http.Request) {
		resp := btPanelResponse{Status: true, Msg: "Site added"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/site/list", func(w http.ResponseWriter, r *http.Request) {
		resp := btPanelResponse{Status: true, Msg: "ok", Data: map[string]interface{}{
			"list": []interface{}{},
		}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

func TestDetectPanel_None(t *testing.T) {
	executor := &mockCommandExecutor{
		output: map[string]string{
			"systemctl is-active 1panel 2>/dev/null || echo 'inactive'": "inactive",
			"systemctl is-active bt 2>/dev/null || echo 'inactive'":     "inactive",
			"ps aux | grep -E '1panel|BT-Panel|bt' | grep -v grep || true": "",
		},
	}
	client := DetectPanel(context.TODO(), executor)
	if client != nil {
		t.Errorf("DetectPanel() = %v, want nil", client)
	}
}

func TestDetectPanel_1Panel(t *testing.T) {
	executor := &mockCommandExecutor{
		output: map[string]string{
			"systemctl is-active 1panel 2>/dev/null || echo 'inactive'": "active",
		},
	}
	client := DetectPanel(context.TODO(), executor)
	if client == nil {
		t.Fatal("DetectPanel() returned nil, want PanelClient")
	}
	info := client.GetInfo()
	if info["name"] != "1Panel" {
		t.Errorf("detected panel name = %v, want 1Panel", info["name"])
	}
}

func TestDetectPanel_BTPanel(t *testing.T) {
	executor := &mockCommandExecutor{
		output: map[string]string{
			"systemctl is-active 1panel 2>/dev/null || echo 'inactive'": "inactive",
			"systemctl is-active bt 2>/dev/null || echo 'inactive'":     "active",
		},
	}
	client := DetectPanel(context.TODO(), executor)
	if client == nil {
		t.Fatal("DetectPanel() returned nil, want PanelClient")
	}
	info := client.GetInfo()
	if info["name"] != "BT-Panel (宝塔)" {
		t.Errorf("detected panel name = %v, want BT-Panel (宝塔)", info["name"])
	}
}

func TestGetPanelInfo_None(t *testing.T) {
	p := NewPanelProvider(nil)
	_, err := p.GetPanelInfo()
	if err == nil {
		t.Error("GetPanelInfo() should return error for nil client")
	}
}

func TestGetPanelInfo_WithClient(t *testing.T) {
	mock := &mockPanelClient{
		name:     "TestPanel",
		features: []string{"feature1", "feature2"},
	}
	p := NewPanelProvider(mock)
	info, err := p.GetPanelInfo()
	if err != nil {
		t.Fatalf("GetPanelInfo() error = %v", err)
	}
	if info["type"] != "detected" {
		t.Errorf("info[type] = %v, want %q", info["type"], "detected")
	}
	if info["name"] != "TestPanel" {
		t.Errorf("info[name] = %v, want %q", info["name"], "TestPanel")
	}
	features, ok := info["features"].([]string)
	if !ok {
		t.Fatal("info[features] is not []string")
	}
	if len(features) != 2 {
		t.Errorf("features count = %d, want 2", len(features))
	}
}

func TestOpenFirewall_1Panel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	p := NewPanelProvider(client)
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("OpenFirewall() error = %v", err)
	}
}

func TestOpenFirewall_BTPanel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()
	client := NewBTPanelClient(server.URL, "admin", "test-key")
	p := NewPanelProvider(client)
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("OpenFirewall() error = %v", err)
	}
}

func TestOpenFirewall_None(t *testing.T) {
	p := NewPanelProvider(nil)
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("OpenFirewall() for nil client should not error, got = %v", err)
	}
}

func TestCloseFirewall_1Panel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	p := NewPanelProvider(client)
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("CloseFirewall() error = %v", err)
	}
}

func TestCloseFirewall_BTPanel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()
	client := NewBTPanelClient(server.URL, "admin", "test-key")
	p := NewPanelProvider(client)
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("CloseFirewall() error = %v", err)
	}
}

func TestCloseFirewall_None(t *testing.T) {
	p := NewPanelProvider(nil)
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("CloseFirewall() for nil client should not error, got = %v", err)
	}
}

func TestCreateReverseProxy_1Panel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()
	client := NewPanel1Client(server.URL, "test-key")
	p := NewPanelProvider(client)
	err := p.CreateReverseProxy(context.TODO(), "example.com", "http://localhost:3000", 3000)
	if err != nil {
		t.Errorf("CreateReverseProxy() error = %v", err)
	}
}

func TestCreateReverseProxy_BTPanel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()
	client := NewBTPanelClient(server.URL, "admin", "test-key")
	p := NewPanelProvider(client)
	err := p.CreateReverseProxy(context.TODO(), "example.com", "http://localhost:3000", 3000)
	if err != nil {
		t.Errorf("CreateReverseProxy() error = %v", err)
	}
}

func TestCreateReverseProxy_None(t *testing.T) {
	p := NewPanelProvider(nil)
	err := p.CreateReverseProxy(context.TODO(), "example.com", "http://localhost:3000", 3000)
	if err != nil {
		t.Errorf("CreateReverseProxy() for nil client should not error, got = %v", err)
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "x", false},
		{"", "", true},
		{"a", "ab", false},
		{"ab", "a", true},
		{"ab", "b", true},
	}
	for _, tt := range tests {
		got := contains(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestDetectPanel_ProcessFallback(t *testing.T) {
	executor := &mockCommandExecutor{
		output: map[string]string{
			"systemctl is-active 1panel 2>/dev/null || echo 'inactive'": "inactive",
			"systemctl is-active bt 2>/dev/null || echo 'inactive'":     "inactive",
			"ps aux | grep -E '1panel|BT-Panel|bt' | grep -v grep || true": "root  1234  1panel serve",
		},
	}
	client := DetectPanel(context.TODO(), executor)
	if client == nil {
		t.Fatal("DetectPanel() returned nil (process fallback)")
	}
	info := client.GetInfo()
	if info["name"] != "1Panel" {
		t.Errorf("detected panel name = %v, want 1Panel (process fallback)", info["name"])
	}
}
