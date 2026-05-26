package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewBarkNotifier(t *testing.T) {
	n := NewBarkNotifier("https://api.day.app", "test-device-key")
	if n.ServerURL != "https://api.day.app" {
		t.Errorf("ServerURL = %q", n.ServerURL)
	}
	if n.DeviceKey != "test-device-key" {
		t.Errorf("DeviceKey = %q", n.DeviceKey)
	}
	if n.Group != "DeployPilot" {
		t.Errorf("Group = %q", n.Group)
	}
}

func TestNewBarkNotifierDefaultServer(t *testing.T) {
	n := NewBarkNotifier("", "test-device-key")
	if n.ServerURL != "https://api.day.app" {
		t.Errorf("Default ServerURL = %q, want %q", n.ServerURL, "https://api.day.app")
	}
}

func TestBarkName(t *testing.T) {
	n := NewBarkNotifier("", "")
	if n.Name() != "bark" {
		t.Errorf("Name() = %q, want %q", n.Name(), "bark")
	}
}

func TestBarkSendSuccess(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 200, "message": "success"})
	}))
	defer server.Close()

	n := NewBarkNotifier(server.URL, "test-key")
	result, err := n.Send(context.Background(), DeploySuccess("my-app", "prod", "nginx:latest"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}
	if receivedBody["title"] == nil {
		t.Error("title should be set")
	}
	if !strings.Contains(receivedBody["title"].(string), "my-app") {
		t.Errorf("title = %v", receivedBody["title"])
	}
}

func TestBarkSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "message": "bad request"})
	}))
	defer server.Close()

	n := NewBarkNotifier(server.URL, "test-key")
	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail when code != 200")
	}
	if !strings.Contains(result.Error, "bad request") {
		t.Errorf("error = %q", result.Error)
	}
}

func TestBarkSendHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	n := NewBarkNotifier(server.URL, "test-key")
	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail for 500 response")
	}
}

func TestBarkMapLevel(t *testing.T) {
	n := NewBarkNotifier("", "")

	tests := []struct {
		status     string
		wantLevel  string
	}{
		{"failed", "timeSensitive"},
		{"critical", "timeSensitive"},
		{"warning", "active"},
		{"success", "passive"},
		{"info", "passive"},
		{"unknown", "passive"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			level := n.mapLevel(tt.status)
			if level != tt.wantLevel {
				t.Errorf("mapLevel(%q) = %q, want %q", tt.status, level, tt.wantLevel)
			}
		})
	}
}

func TestBarkContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	n := NewBarkNotifier("http://127.0.0.1:1", "test-key")
	result, _ := n.Send(ctx, DeploySuccess("app", "srv", "img"))

	if result.Success {
		t.Error("should fail with cancelled context")
	}
}

func TestBarkSendWithMetadata(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 200})
	}))
	defer server.Close()

	n := NewBarkNotifier(server.URL, "test-key")
	n.Archive = "1"
	n.Icon = "https://example.com/icon.png"

	notification := Notification{
		Type:      "deploy",
		AppName:   "my-app",
		Server:    "prod",
		Status:    "success",
		Message:   "deployed",
		Timestamp: time.Now(),
	}

	result, err := n.Send(context.Background(), notification)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false")
	}
	if receivedBody["isArchive"] != "1" {
		t.Errorf("isArchive = %v", receivedBody["isArchive"])
	}
}