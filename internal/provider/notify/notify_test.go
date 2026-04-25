package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ========== Webhook Notifier ==========

func TestWebhookSuccess(t *testing.T) {
	var received Notification
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("Authorization = %q", auth)
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	wh := NewWebhookNotifier(server.URL, map[string]string{
		"Authorization": "Bearer test-token",
	})

	notification := DeploySuccess("my-app", "prod-server", "nginx:latest")
	result, err := wh.Send(context.Background(), notification)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}
	if received.AppName != "my-app" {
		t.Errorf("received AppName = %q", received.AppName)
	}
	if received.Type != "deploy_success" {
		t.Errorf("received Type = %q", received.Type)
	}
}

func TestWebhookServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	wh := NewWebhookNotifier(server.URL, nil)
	result, _ := wh.Send(context.Background(), DeploySuccess("app", "srv", "img"))

	if result.Success {
		t.Error("should fail for 500 response")
	}
	if !strings.Contains(result.Error, "500") {
		t.Errorf("error = %q", result.Error)
	}
}

func TestWebhookConnectionRefused(t *testing.T) {
	wh := NewWebhookNotifier("http://127.0.0.1:1", nil)
	result, _ := wh.Send(context.Background(), DeploySuccess("app", "srv", "img"))

	if result.Success {
		t.Error("should fail for connection refused")
	}
}

func TestWebhookName(t *testing.T) {
	wh := NewWebhookNotifier("", nil)
	if wh.Name() != "webhook" {
		t.Errorf("Name() = %q", wh.Name())
	}
}

func TestWebhookContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	wh := NewWebhookNotifier("http://127.0.0.1:1", nil)
	result, _ := wh.Send(ctx, DeploySuccess("app", "srv", "img"))

	if result.Success {
		t.Error("should fail with cancelled context")
	}
}

// ========== Email Notifier ==========

func TestEmailSuccess(t *testing.T) {
	var sentFrom string
	var sentMsg []byte

	email := NewEmailNotifier(EmailConfig{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		From:     "noreply@example.com",
	})
	email.SendFunc = func(from, to string, msg []byte) error {
		sentFrom = from
		sentMsg = msg
		return nil
	}

	notification := DeploySuccess("my-app", "prod-server", "nginx:latest")
	result, err := email.Send(context.Background(), notification)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if !result.Success {
		t.Error("should succeed")
	}
	if sentFrom != "noreply@example.com" {
		t.Errorf("From = %q", sentFrom)
	}
	msgStr := string(sentMsg)
	if !strings.Contains(msgStr, "my-app") {
		t.Error("email body should contain app name")
	}
	if !strings.Contains(msgStr, "DEPLOY_SUCCESS") {
		t.Error("email subject should contain notification type")
	}
}

func TestEmailFailure(t *testing.T) {
	email := NewEmailNotifier(EmailConfig{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		From:     "noreply@example.com",
	})
	email.SendFunc = func(_, _ string, _ []byte) error {
		return fmt.Errorf("SMTP connection refused")
	}

	result, _ := email.Send(context.Background(), DeploySuccess("app", "srv", "img"))

	if result.Success {
		t.Error("should fail when SMTP errors")
	}
	if !strings.Contains(result.Error, "SMTP") {
		t.Errorf("error = %q", result.Error)
	}
}

func TestEmailName(t *testing.T) {
	email := NewEmailNotifier(EmailConfig{})
	if email.Name() != "email" {
		t.Errorf("Name() = %q", email.Name())
	}
}

// ========== Multi Notifier ==========

func TestMultiNotifierAllSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	email := NewEmailNotifier(EmailConfig{From: "test@test.com"})
	email.SendFunc = func(_, _ string, _ []byte) error { return nil }

	wh := NewWebhookNotifier(server.URL, nil)
	multi := NewMultiNotifier(wh, email)

	results, err := multi.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("Provider %s failed: %s", r.Provider, r.Error)
		}
	}
}

func TestMultiNotifierPartialFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	email := NewEmailNotifier(EmailConfig{From: "test@test.com"})
	email.SendFunc = func(_, _ string, _ []byte) error { return nil }

	wh := NewWebhookNotifier(server.URL, nil)
	multi := NewMultiNotifier(wh, email)

	results, _ := multi.Send(context.Background(), DeploySuccess("app", "srv", "img"))

	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}
	if successCount != 1 {
		t.Errorf("Expected 1 success, got %d", successCount)
	}
	if failCount != 1 {
		t.Errorf("Expected 1 failure, got %d", failCount)
	}
}

func TestMultiNotifierName(t *testing.T) {
	multi := NewMultiNotifier()
	if multi.Name() != "multi" {
		t.Errorf("Name() = %q", multi.Name())
	}
}

// ========== Notification Builders ==========

func TestDeploySuccessBuilder(t *testing.T) {
	n := DeploySuccess("my-app", "prod", "nginx:latest")

	if n.Type != "deploy_success" {
		t.Errorf("Type = %q", n.Type)
	}
	if n.Status != "success" {
		t.Errorf("Status = %q", n.Status)
	}
	if n.AppName != "my-app" {
		t.Errorf("AppName = %q", n.AppName)
	}
	if n.Metadata["image"] != "nginx:latest" {
		t.Errorf("Metadata[image] = %q", n.Metadata["image"])
	}
}

func TestDeployFailedBuilder(t *testing.T) {
	n := DeployFailed("my-app", "prod", "port in use")

	if n.Type != "deploy_failed" {
		t.Errorf("Type = %q", n.Type)
	}
	if n.Status != "failed" {
		t.Errorf("Status = %q", n.Status)
	}
	if !strings.Contains(n.Message, "port in use") {
		t.Errorf("Message = %q", n.Message)
	}
}

func TestHealthCheckFailedBuilder(t *testing.T) {
	n := HealthCheckFailed("my-app", "prod", "http://localhost:8080/health")

	if n.Type != "health_check" {
		t.Errorf("Type = %q", n.Type)
	}
	if n.Status != "warning" {
		t.Errorf("Status = %q", n.Status)
	}
}

func TestRollbackBuilder(t *testing.T) {
	n := Rollback("my-app", "prod", "nginx:1.24", "health check failed")

	if n.Type != "rollback" {
		t.Errorf("Type = %q", n.Type)
	}
	if n.Metadata["old_image"] != "nginx:1.24" {
		t.Errorf("Metadata[old_image] = %q", n.Metadata["old_image"])
	}
}

// Suppress unused import
var _ = fmt.Errorf

// ===================== Additional Coverage: NewEmailNotifier =====================

func TestNewEmailNotifier_DefaultSendFunc(t *testing.T) {
	email := NewEmailNotifier(EmailConfig{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		Username: "user",
		Password: "pass",
		From:     "noreply@example.com",
	})
	if email == nil {
		t.Fatal("expected non-nil notifier")
	}
	if email.Name() != "email" {
		t.Errorf("Name() = %q, want %q", email.Name(), "email")
	}
	// SendFunc should be set (non-nil)
	if email.SendFunc == nil {
		t.Error("expected default SendFunc to be set")
	}
}

func TestNewEmailNotifier_EmptyConfig(t *testing.T) {
	email := NewEmailNotifier(EmailConfig{})
	if email == nil {
		t.Fatal("expected non-nil notifier")
	}
	if email.SendFunc == nil {
		t.Error("expected default SendFunc to be set even with empty config")
	}
}

func TestEmailNotifier_SendWithDefaultFunc(t *testing.T) {
	email := NewEmailNotifier(EmailConfig{
		SMTPHost: "127.0.0.1",
		SMTPPort: 19999,
		Username: "user",
		Password: "pass",
		From:     "test@example.com",
	})
	// Don't override SendFunc - use the default which calls smtp.SendMail
	// This will fail since there's no SMTP server, covering the error path
	result, err := email.Send(context.Background(), Notification{
		Type:      "deploy",
		AppName:   "myapp",
		Server:    "server1",
		Status:    "success",
		Message:   "deployed ok",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("Send should not return error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Success {
		t.Error("expected Success=false when SMTP connection fails")
	}
	if result.Error == "" {
		t.Error("expected non-empty error message")
	}
}
