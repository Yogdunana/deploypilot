package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ========== Telegram Notifier Tests ==========

func TestNewTelegramNotifier(t *testing.T) {
	n := NewTelegramNotifier("test-token", "test-chat-id")
	if n.botToken != "test-token" {
		t.Errorf("botToken = %q", n.botToken)
	}
	if n.chatID != "test-chat-id" {
		t.Errorf("chatID = %q", n.chatID)
	}
}

func TestTelegramName(t *testing.T) {
	n := NewTelegramNotifier("token", "chat-id")
	if n.Name() != "telegram" {
		t.Errorf("Name() = %q, want %q", n.Name(), "telegram")
	}
}

func TestTelegramSend(t *testing.T) {
	var receivedBody map[string]interface{}
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		capturedPath = r.URL.Path
		if !strings.Contains(r.URL.Path, "/sendMessage") {
			t.Errorf("Path = %q, should contain /sendMessage", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	n := NewTelegramNotifier("test-token", "test-chat-id")
	n.SetBaseURL(server.URL)

	result, err := n.Send(context.Background(), DeploySuccess("my-app", "prod", "nginx:latest"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}
	if receivedBody["chat_id"] != "test-chat-id" {
		t.Errorf("chat_id = %v", receivedBody["chat_id"])
	}
	text, _ := receivedBody["text"].(string)
	if !strings.Contains(text, "my-app") {
		t.Errorf("message should contain app name, got: %s", text)
	}
	if !strings.Contains(capturedPath, "/sendMessage") {
		t.Errorf("path should contain /sendMessage, got: %s", capturedPath)
	}
}

func TestTelegramSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "description": "Bad Request"})
	}))
	defer server.Close()

	n := NewTelegramNotifier("test-token", "test-chat-id")
	n.SetBaseURL(server.URL)

	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail for 400 response")
	}
}

func TestTelegramFormatMessage(t *testing.T) {
	n := NewTelegramNotifier("token", "chat-id")
	notification := DeploySuccess("my-app", "prod-server", "nginx:latest")
	msg := n.formatMessage(notification)
	if !strings.Contains(msg, "my-app") {
		t.Error("message should contain app name")
	}
	if !strings.Contains(msg, "prod-server") {
		t.Error("message should contain server")
	}
}

func TestTelegramFormatMessageFailed(t *testing.T) {
	n := NewTelegramNotifier("token", "chat-id")
	notification := DeployFailed("my-app", "prod-server", "build error")
	msg := n.formatMessage(notification)
	if !strings.Contains(msg, "failed") {
		t.Error("message should contain failed status")
	}
}

func TestTelegramFormatMessageWarning(t *testing.T) {
	n := NewTelegramNotifier("token", "chat-id")
	notification := HealthCheckFailed("my-app", "prod-server", "http://localhost")
	msg := n.formatMessage(notification)
	if !strings.Contains(msg, "warning") {
		t.Error("message should contain warning status")
	}
}
