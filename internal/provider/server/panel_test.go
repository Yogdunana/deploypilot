package server

import (
	"context"
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

func TestOpenFirewall(t *testing.T) {
	p := NewPanelProvider(Panel1Panel, "http://localhost:8888", "test-key")
	err := p.OpenFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("OpenFirewall() error = %v", err)
	}
}

func TestCloseFirewall(t *testing.T) {
	p := NewPanelProvider(PanelBTPanel, "http://localhost:8888", "test-key")
	err := p.CloseFirewall(context.TODO(), 8080, "tcp")
	if err != nil {
		t.Errorf("CloseFirewall() error = %v", err)
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
