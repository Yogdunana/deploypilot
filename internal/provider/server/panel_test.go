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

// setupPanelProviderTestServer creates a mock HTTP server that handles both 1Panel and BT-Panel API calls.
func setupPanelProviderTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// 1Panel endpoints
	mux.HandleFunc("/api/v1/firewall/rules", func(w http.ResponseWriter, r *http.Request) {
		resp := panel1Response{Code: 200, Message: "success", Data: json.RawMessage(`{"id": 1}`)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/api/v1/websites/reverse_proxy", func(w http.ResponseWriter, r *http.Request) {
		resp := panel1Response{Code: 200, Message: "success", Data: json.RawMessage(`{"id": 1}`)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// BT-Panel endpoints
	mux.HandleFunc("/firewall", func(w http.ResponseWriter, r *http.Request) {
		resp := btPanelResponse{Status: true, Msg: "ok"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/site", func(w http.ResponseWriter, r *http.Request) {
		resp := btPanelResponse{Status: true, Msg: "ok"}
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

	panel := DetectPanel(context.TODO(), executor)
	if panel != PanelNone {
		t.Errorf("DetectPanel() = %q, want %q", panel, PanelNone)
	}
}

func TestDetectPanel_1Panel(t *testing.T) {
	executor := &mockCommandExecutor{
		output: map[string]string{
			"systemctl is-active 1panel 2>/dev/null || echo 'inactive'": "active",
		},
	}

	panel := DetectPanel(context.TODO(), executor)
	if panel != Panel1Panel {
		t.Errorf("DetectPanel() = %q, want %q", panel, Panel1Panel)
	}
}

func TestDetectPanel_BTPanel(t *testing.T) {
	executor := &mockCommandExecutor{
		output: map[string]string{
			"systemctl is-active 1panel 2>/dev/null || echo 'inactive'": "inactive",
			"systemctl is-active bt 2>/dev/null || echo 'inactive'":     "active",
		},
	}

	panel := DetectPanel(context.TODO(), executor)
	if panel != PanelBTPanel {
		t.Errorf("DetectPanel() = %q, want %q", panel, PanelBTPanel)
	}
}

func TestGetPanelInfo_None(t *testing.T) {
	p := NewPanelProvider(PanelNone, "", "")
	_, err := p.GetPanelInfo(context.TODO())
	if err == nil {
		t.Error("GetPanelInfo() should return error for PanelNone")
	}
}

func TestGetPanelInfo_1Panel(t *testing.T) {
	p := NewPanelProvider(Panel1Panel, "http://localhost:8888", "test-key")
	info, err := p.GetPanelInfo(context.TODO())
	if err != nil {
		t.Fatalf("GetPanelInfo() error = %v", err)
	}
	if info["type"] != "1panel" {
		t.Errorf("info[type] = %v, want %q", info["type"], "1panel")
	}
	if info["name"] != "1Panel" {
		t.Errorf("info[name] = %v, want %q", info["name"], "1Panel")
	}
	features, ok := info["features"].([]string)
	if !ok {
		t.Fatal("info[features] is not []string")
	}
	if len(features) == 0 {
		t.Error("1Panel should have features")
	}
}

func TestGetPanelInfo_BTPanel(t *testing.T) {
	p := NewPanelProvider(PanelBTPanel, "http://localhost:8888", "test-key")
	info, err := p.GetPanelInfo(context.TODO())
	if err != nil {
		t.Fatalf("GetPanelInfo() error = %v", err)
	}
	if info["type"] != "bt-panel" {
		t.Errorf("info[type] = %v, want %q", info["type"], "bt-panel")
	}
	if info["name"] != "BT-Panel (宝塔)" {
		t.Errorf("info[name] = %v, want %q", info["name"], "BT-Panel (宝塔)")
	}
	features, ok := info["features"].([]string)
	if !ok {
		t.Fatal("info[features] is not []string")
	}
	if len(features) == 0 {
		t.Error("BT-Panel should have features")
	}
}

func TestOpenFirewall_1Panel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()

	p := NewPanelProvider(Panel1Panel, server.URL, "test-key")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("OpenFirewall() error = %v", err)
	}
}

func TestOpenFirewall_BTPanel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()

	p := NewPanelProvider(PanelBTPanel, server.URL, "test-key")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("OpenFirewall() error = %v", err)
	}
}

func TestOpenFirewall_None(t *testing.T) {
	p := NewPanelProvider(PanelNone, "", "")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("OpenFirewall() for PanelNone should not error, got = %v", err)
	}
}

func TestOpenFirewall_MissingCredentials(t *testing.T) {
	p := NewPanelProvider(Panel1Panel, "", "")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err == nil {
		t.Error("OpenFirewall() should return error when credentials are missing")
	}
}

func TestCloseFirewall_1Panel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()

	p := NewPanelProvider(Panel1Panel, server.URL, "test-key")
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("CloseFirewall() error = %v", err)
	}
}

func TestCloseFirewall_BTPanel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()

	p := NewPanelProvider(PanelBTPanel, server.URL, "test-key")
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("CloseFirewall() error = %v", err)
	}
}

func TestCloseFirewall_None(t *testing.T) {
	p := NewPanelProvider(PanelNone, "", "")
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("CloseFirewall() for PanelNone should not error, got = %v", err)
	}
}

func TestCreateReverseProxy_1Panel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()

	p := NewPanelProvider(Panel1Panel, server.URL, "test-key")
	err := p.CreateReverseProxy(context.TODO(), "example.com", "http://localhost:3000", 3000)
	if err != nil {
		t.Errorf("CreateReverseProxy() error = %v", err)
	}
}

func TestCreateReverseProxy_BTPanel(t *testing.T) {
	server := setupPanelProviderTestServer()
	defer server.Close()

	p := NewPanelProvider(PanelBTPanel, server.URL, "test-key")
	err := p.CreateReverseProxy(context.TODO(), "example.com", "http://localhost:3000", 3000)
	if err != nil {
		t.Errorf("CreateReverseProxy() error = %v", err)
	}
}

func TestCreateReverseProxy_None(t *testing.T) {
	p := NewPanelProvider(PanelNone, "", "")
	err := p.CreateReverseProxy(context.TODO(), "example.com", "http://localhost:3000", 3000)
	if err != nil {
		t.Errorf("CreateReverseProxy() for PanelNone should not error, got = %v", err)
	}
}

func TestCreateReverseProxy_MissingCredentials(t *testing.T) {
	p := NewPanelProvider(PanelBTPanel, "", "")
	err := p.CreateReverseProxy(context.TODO(), "example.com", "http://localhost:3000", 3000)
	if err == nil {
		t.Error("CreateReverseProxy() should return error when credentials are missing")
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

	panel := DetectPanel(context.TODO(), executor)
	if panel != Panel1Panel {
		t.Errorf("DetectPanel() = %q, want %q (process fallback)", panel, Panel1Panel)
	}
}
