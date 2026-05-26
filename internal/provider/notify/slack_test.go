package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewSlackNotifier(t *testing.T) {
	n := NewSlackNotifier("https://hooks.slack.com/services/test")
	if n.WebhookURL != "https://hooks.slack.com/services/test" {
		t.Errorf("WebhookURL = %q", n.WebhookURL)
	}
	if n.Username != "DeployPilot" {
		t.Errorf("Username = %q", n.Username)
	}
	if n.IconEmoji != ":rocket:" {
		t.Errorf("IconEmoji = %q", n.IconEmoji)
	}
}

func TestSlackName(t *testing.T) {
	n := NewSlackNotifier("url")
	if n.Name() != "slack" {
		t.Errorf("Name() = %q, want %q", n.Name(), "slack")
	}
}

func TestSlackSendSuccess(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	n := NewSlackNotifier(server.URL)
	result, err := n.Send(context.Background(), DeploySuccess("my-app", "prod", "nginx:latest"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}

	attachments := receivedBody["attachments"].([]interface{})
	if len(attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(attachments))
	}
}

func TestSlackSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "token_expired"})
	}))
	defer server.Close()

	n := NewSlackNotifier(server.URL)
	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail when ok=false")
	}
	if !strings.Contains(result.Error, "token_expired") {
		t.Errorf("error = %q", result.Error)
	}
}

func TestSlackSendHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	n := NewSlackNotifier(server.URL)
	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail for 500 response")
	}
}

func TestSlackMapColor(t *testing.T) {
	n := NewSlackNotifier("")

	tests := []struct {
		status    string
		wantColor string
	}{
		{"success", "#36a64f"},
		{"failed", "#e01e5a"},
		{"critical", "#e01e5a"},
		{"warning", "#f2c744"},
		{"info", "#808080"},
		{"unknown", "#808080"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			color := n.mapColor(tt.status)
			if color != tt.wantColor {
				t.Errorf("mapColor(%q) = %q, want %q", tt.status, color, tt.wantColor)
			}
		})
	}
}

func TestSlackSendWithChannelOverride(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	n := NewSlackNotifier(server.URL)
	n.Channel = "#ops-alerts"

	result, err := n.Send(context.Background(), DeploySuccess("my-app", "prod", "nginx:latest"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false")
	}
	if receivedBody["channel"] != "#ops-alerts" {
		t.Errorf("channel = %v", receivedBody["channel"])
	}
}

func TestSlackSendWithoutServer(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	n := NewSlackNotifier(server.URL)

	notification := Notification{
		Type:    "deploy",
		AppName: "my-app",
		Server:  "",
		Status:  "success",
		Message: "deployed",
	}

	result, err := n.Send(context.Background(), notification)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false")
	}

	attachments := receivedBody["attachments"].([]interface{})
	if len(attachments) != 1 {
		t.Fatal("expected 1 attachment")
	}
	attachment := attachments[0].(map[string]interface{})
	fields := attachment["fields"].([]interface{})
	if len(fields) != 1 {
		t.Errorf("expected 1 field when no server, got %d", len(fields))
	}
}

func TestSlackContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n := NewSlackNotifier("http://127.0.0.1:1")
	result, _ := n.Send(ctx, DeploySuccess("app", "srv", "img"))

	if result.Success {
		t.Error("should fail with cancelled context")
	}
}