package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ========== DingTalk Notifier Tests ==========

func TestNewDingTalkNotifier(t *testing.T) {
	n := NewDingTalkNotifier("https://oapi.dingtalk.com/robot/send?access_token=test", "secret123")
	if n.webhookURL != "https://oapi.dingtalk.com/robot/send?access_token=test" {
		t.Errorf("webhookURL = %q", n.webhookURL)
	}
	if n.secret != "secret123" {
		t.Errorf("secret = %q", n.secret)
	}
}

func TestDingTalkName(t *testing.T) {
	n := NewDingTalkNotifier("url", "secret")
	if n.Name() != "dingtalk" {
		t.Errorf("Name() = %q, want %q", n.Name(), "dingtalk")
	}
}

func TestDingTalkSend(t *testing.T) {
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

	n := NewDingTalkNotifier(server.URL, "")
	result, err := n.Send(context.Background(), DeploySuccess("my-app", "prod", "nginx:latest"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}
	if receivedBody["msgtype"] != "markdown" {
		t.Errorf("msgtype = %v", receivedBody["msgtype"])
	}
}

func TestDingTalkSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"errcode": 310000, "errmsg": "sign not match"})
	}))
	defer server.Close()

	n := NewDingTalkNotifier(server.URL, "")
	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail when errcode != 0")
	}
}

func TestDingTalkSign(t *testing.T) {
	n := NewDingTalkNotifier("https://example.com/webhook", "my-secret")
	signedURL := n.signURL("https://example.com/webhook", "my-secret")
	if !strings.Contains(signedURL, "timestamp=") {
		t.Error("signed URL should contain timestamp")
	}
	if !strings.Contains(signedURL, "sign=") {
		t.Error("signed URL should contain sign")
	}
}

func TestDingTalkFormatMessage(t *testing.T) {
	n := NewDingTalkNotifier("url", "secret")
	notification := DeploySuccess("my-app", "prod-server", "nginx:latest")
	title, text := n.formatMessage(notification)
	if !strings.Contains(title, "my-app") {
		t.Error("title should contain app name")
	}
	if !strings.Contains(text, "prod-server") {
		t.Error("text should contain server")
	}
	if !strings.Contains(text, "success") {
		t.Error("text should contain status")
	}
}

func TestDingTalkFormatMessageFailed(t *testing.T) {
	n := NewDingTalkNotifier("url", "secret")
	notification := DeployFailed("my-app", "prod", "error")
	_, text := n.formatMessage(notification)
	if !strings.Contains(text, "failed") {
		t.Error("text should contain failed status")
	}
}

func TestDingTalkHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	n := NewDingTalkNotifier(server.URL, "")
	result, _ := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if result.Success {
		t.Error("should fail for 500 response")
	}
}

func TestDingTalkSignWithSecret(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"errcode": 0, "errmsg": "ok"})
	}))
	defer server.Close()

	n := NewDingTalkNotifier(server.URL, "my-secret")
	_, err := n.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !strings.Contains(receivedQuery, "timestamp=") {
		t.Errorf("URL query should contain timestamp, got: %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "sign=") {
		t.Errorf("URL query should contain sign, got: %s", receivedQuery)
	}
}
