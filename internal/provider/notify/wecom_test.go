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

// ========== WeCom Notifier Tests ==========

func TestNewWeComNotifier(t *testing.T) {
	n := NewWeComNotifier("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-token")
	if n.webhookURL != "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-token" {
		t.Errorf("webhookURL = %q", n.webhookURL)
	}
}

func TestWeComName(t *testing.T) {
	n := NewWeComNotifier("url")
	if n.Name() != "wecom" {
		t.Errorf("Name() = %q, want %q", n.Name(), "wecom")
	}
}

func TestWeComValidateEmpty(t *testing.T) {
	n := NewWeComNotifier("")
	err := n.Validate()
	if err == nil {
		t.Error("Validate() should return error for empty webhook URL")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("error = %q, want 'required'", err.Error())
	}
}

func TestWeComValidateValid(t *testing.T) {
	n := NewWeComNotifier("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test")
	err := n.Validate()
	if err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}
}

func TestWeComSend(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"errcode": 0, "errmsg": "ok"})
	}))
	defer server.Close()

	n := NewWeComNotifier(server.URL)
	result, err := n.Send(context.Background(), DeploySuccess("my-app", "prod", "nginx:latest"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}
	if receivedBody["msgtype"] != "markdown" {
		t.Errorf("msgtype = %v, want markdown", receivedBody["msgtype"])
	}
}

func TestWeComSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"errcode": 93000, "errmsg": "invalid webhook url"})
	}))
	defer server.Close()

	n := NewWeComNotifier(server.URL)
	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail when errcode != 0")
	}
}

func TestWeComHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	n := NewWeComNotifier(server.URL)
	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail for 500 response")
	}
	if !strings.Contains(result.Error, "500") {
		t.Errorf("error = %q", result.Error)
	}
}

func TestWeComFormatMessage(t *testing.T) {
	n := NewWeComNotifier("url")

	title, text := n.formatMessage(DeploySuccess("my-app", "prod", "nginx:latest"))
	if !strings.Contains(title, "deploy_success") {
		t.Errorf("title = %q, want to contain deploy_success", title)
	}
	if !strings.Contains(title, "my-app") {
		t.Errorf("title = %q, want to contain my-app", title)
	}
	if !strings.Contains(text, "prod") {
		t.Errorf("text = %q, want to contain prod", text)
	}
	if !strings.Contains(text, "success") {
		t.Errorf("text = %q, want to contain success", text)
	}
}

func TestWeComFormatMessageFailed(t *testing.T) {
	n := NewWeComNotifier("url")

	_, text := n.formatMessage(DeployFailed("my-app", "prod", "build error"))
	if !strings.Contains(text, "\u274c") {
		t.Errorf("text = %q, want to contain failed icon", text)
	}
	if !strings.Contains(text, "build error") {
		t.Errorf("text = %q, want to contain build error", text)
	}
}

func TestWeComFormatMessageWarning(t *testing.T) {
	n := NewWeComNotifier("url")

	_, text := n.formatMessage(HealthCheckFailed("my-app", "prod", "http://localhost"))
	if !strings.Contains(text, "\u26a0\ufe0f") {
		t.Errorf("text = %q, want to contain warning icon", text)
	}
}

func TestWeComFormatMessageNoMessage(t *testing.T) {
	n := NewWeComNotifier("url")

	notification := Notification{
		Type:      "deploy",
		AppName:   "my-app",
		Server:    "prod",
		Status:    "success",
		Message:   "",
		Timestamp: time.Now(),
	}
	_, text := n.formatMessage(notification)
	// Should not contain message field when empty
	if strings.Contains(text, "**Message:**") {
		t.Errorf("text should not contain message field when empty, got: %s", text)
	}
}

func TestWeComSendContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer server.Close()

	n := NewWeComNotifier(server.URL)

	// Use a short timeout instead of immediate cancel to ensure
	// the HTTP client has time to start the request and observe the deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := n.Send(ctx, DeploySuccess("app", "srv", "img"))
	if err == nil {
		t.Error("Send() should return error when context times out")
	}
}
