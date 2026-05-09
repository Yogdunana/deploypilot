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

// ========== Feishu Notifier Tests ==========

func TestNewFeishuNotifier(t *testing.T) {
	n := NewFeishuNotifier("https://open.feishu.cn/open-apis/bot/v2/hook/test-token")
	if n.webhookURL != "https://open.feishu.cn/open-apis/bot/v2/hook/test-token" {
		t.Errorf("webhookURL = %q", n.webhookURL)
	}
}

func TestFeishuName(t *testing.T) {
	n := NewFeishuNotifier("url")
	if n.Name() != "feishu" {
		t.Errorf("Name() = %q, want %q", n.Name(), "feishu")
	}
}

func TestFeishuSend(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
	}))
	defer server.Close()

	n := NewFeishuNotifier(server.URL)
	result, err := n.Send(context.Background(), DeploySuccess("my-app", "prod", "nginx:latest"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}
	if receivedBody["msg_type"] != "interactive" {
		t.Errorf("msg_type = %v", receivedBody["msg_type"])
	}
}

func TestFeishuSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 19021, "msg": "sign match fail"})
	}))
	defer server.Close()

	n := NewFeishuNotifier(server.URL)
	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail when code != 0")
	}
}

func TestFeishuBuildCard(t *testing.T) {
	n := NewFeishuNotifier("url")

	// Test success status
	card := n.buildCard(DeploySuccess("my-app", "prod", "nginx:latest"))
	cardMap := card["card"].(map[string]interface{})
	header := cardMap["header"].(map[string]interface{})
	if header["template"] != "green" {
		t.Errorf("header template = %v, want green for success", header["template"])
	}

	// Test failed status
	card = n.buildCard(DeployFailed("my-app", "prod", "error"))
	cardMap = card["card"].(map[string]interface{})
	header = cardMap["header"].(map[string]interface{})
	if header["template"] != "red" {
		t.Errorf("header template = %v, want red for failed", header["template"])
	}

	// Test warning status
	card = n.buildCard(HealthCheckFailed("my-app", "prod", "http://localhost"))
	cardMap = card["card"].(map[string]interface{})
	header = cardMap["header"].(map[string]interface{})
	if header["template"] != "orange" {
		t.Errorf("header template = %v, want orange for warning", header["template"])
	}
}

func TestFeishuHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	n := NewFeishuNotifier(server.URL)
	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail for 500 response")
	}
	if !strings.Contains(result.Error, "500") {
		t.Errorf("error = %q", result.Error)
	}
}

func TestFeishuBuildCardDefaultStatus(t *testing.T) {
	n := NewFeishuNotifier("url")

	// Test default/unknown status
	notification := Notification{
		Type:      "info",
		AppName:   "my-app",
		Server:    "prod",
		Status:    "info",
		Message:   "informational message",
		Timestamp: time.Now(),
	}
	card := n.buildCard(notification)
	cardMap := card["card"].(map[string]interface{})
	header := cardMap["header"].(map[string]interface{})
	if header["template"] != "blue" {
		t.Errorf("header template = %v, want blue for default", header["template"])
	}
}

func TestFeishuBuildCardWithMessage(t *testing.T) {
	n := NewFeishuNotifier("url")

	// Test card with message
	card := n.buildCard(DeployFailed("my-app", "prod", "build failed: exit code 1"))
	cardMap := card["card"].(map[string]interface{})
	elements := cardMap["elements"].([]map[string]interface{})
	// Should have 3 elements: fields (app+server), fields (status+time), message
	if len(elements) < 3 {
		t.Errorf("expected at least 3 elements, got %d", len(elements))
	}
}

func TestFeishuBuildCardNoMessage(t *testing.T) {
	n := NewFeishuNotifier("url")

	// Test card without message
	notification := Notification{
		Type:      "deploy",
		AppName:   "my-app",
		Server:    "prod",
		Status:    "success",
		Message:   "",
		Timestamp: time.Now(),
	}
	card := n.buildCard(notification)
	cardMap := card["card"].(map[string]interface{})
	elements := cardMap["elements"].([]map[string]interface{})
	// Should have 2 elements: fields (app+server), fields (status+time) - no message element
	if len(elements) != 2 {
		t.Errorf("expected 2 elements when no message, got %d", len(elements))
	}
}
