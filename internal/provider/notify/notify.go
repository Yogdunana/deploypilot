package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/i18n"
)

// Notification represents a deployment notification.
type Notification struct {
	Type      string            `json:"type"`       // deploy_success, deploy_failed, health_check, rollback
	AppName   string            `json:"app_name"`
	Server    string            `json:"server"`
	Status    string            `json:"status"`     // success, failed, warning
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NotifyResult represents the result of a notification send.
type NotifyResult struct {
	Provider string `json:"provider"`
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Notifier defines the interface for notification providers.
type Notifier interface {
	Send(ctx context.Context, notification Notification) (*NotifyResult, error)
	Name() string
}

// ========== Webhook Notifier ==========

// WebhookNotifier sends notifications via HTTP webhook.
type WebhookNotifier struct {
	URL        string
	Headers    map[string]string
	HTTPClient *http.Client
}

// NewWebhookNotifier creates a new WebhookNotifier.
func NewWebhookNotifier(url string, headers map[string]string) *WebhookNotifier {
	return &WebhookNotifier{
		URL:     url,
		Headers: headers,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (w *WebhookNotifier) Name() string { return "webhook" }

// Send sends a notification via webhook POST.
func (w *WebhookNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	payload, err := json.Marshal(notification)
	if err != nil {
		return &NotifyResult{Provider: "webhook", Success: false, Error: err.Error()}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(payload))
	if err != nil {
		return &NotifyResult{Provider: "webhook", Success: false, Error: err.Error()}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range w.Headers {
		req.Header.Set(key, value)
	}

	resp, err := w.HTTPClient.Do(req)
	if err != nil {
		return &NotifyResult{Provider: "webhook", Success: false, Error: err.Error()}, nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &NotifyResult{
			Provider: "webhook",
			Success:  true,
			Message:  fmt.Sprintf("webhook delivered (HTTP %d)", resp.StatusCode),
		}, nil
	}

	return &NotifyResult{
		Provider: "webhook",
		Success:  false,
		Error:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
	}, nil
}

// ========== Email Notifier ==========

// EmailConfig holds SMTP configuration.
type EmailConfig struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
}

// EmailNotifier sends notifications via email.
type EmailNotifier struct {
	Config    EmailConfig
	SendFunc  func(from, to string, msg []byte) error // for testing
}

// NewEmailNotifier creates a new EmailNotifier.
func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{
		Config: config,
		SendFunc: func(from, to string, msg []byte) error {
			addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
			auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)
			return smtp.SendMail(addr, auth, from, []string{to}, msg)
		},
	}
}

// Name returns the notifier name.
func (e *EmailNotifier) Name() string { return "email" }

// Send sends a notification via email.
func (e *EmailNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	subject := fmt.Sprintf("[DeployPilot] %s: %s - %s",
		strings.ToUpper(notification.Type), notification.AppName, notification.Status)

	body := fmt.Sprintf("App: %s\nServer: %s\nStatus: %s\nTime: %s\n\n%s",
		notification.AppName,
		notification.Server,
		notification.Status,
		notification.Timestamp.Format(time.RFC3339),
		notification.Message)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		e.Config.From, notification.AppName+"-team", subject, body)

	err := e.SendFunc(e.Config.From, e.Config.From, []byte(msg))
	if err != nil {
		return &NotifyResult{Provider: "email", Success: false, Error: err.Error()}, nil
	}

	return &NotifyResult{Provider: "email", Success: true, Message: "email sent"}, nil
}

// ========== Multi Notifier ==========

// MultiNotifier sends to multiple notifiers.
type MultiNotifier struct {
	notifiers []Notifier
}

// NewMultiNotifier creates a new MultiNotifier.
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

// Name returns the notifier name.
func (m *MultiNotifier) Name() string { return "multi" }

// Send sends to all notifiers and returns all results.
func (m *MultiNotifier) Send(ctx context.Context, notification Notification) ([]*NotifyResult, error) {
	var results []*NotifyResult
	for _, n := range m.notifiers {
		result, err := n.Send(ctx, notification)
		if err != nil {
			result = &NotifyResult{Provider: n.Name(), Success: false, Error: err.Error()}
		}
		results = append(results, result)
	}
	return results, nil
}

// ========== Notification Builder ==========

// DeploySuccess creates a deploy success notification.
func DeploySuccess(appName, server, image string) Notification {
	return DeploySuccessLocale("en", appName, server, image)
}

// DeploySuccessLocale creates a deploy success notification with i18n support.
func DeploySuccessLocale(locale, appName, server, image string) Notification {
	return Notification{
		Type:      "deploy_success",
		AppName:   appName,
		Server:    server,
		Status:    "success",
		Message:   i18n.Tf(locale, "notify.deploy_success", image, server),
		Timestamp: time.Now(),
		Metadata:  map[string]string{"image": image},
	}
}

// DeployFailed creates a deploy failure notification.
func DeployFailed(appName, server, reason string) Notification {
	return DeployFailedLocale("en", appName, server, reason)
}

// DeployFailedLocale creates a deploy failure notification with i18n support.
func DeployFailedLocale(locale, appName, server, reason string) Notification {
	return Notification{
		Type:      "deploy_failed",
		AppName:   appName,
		Server:    server,
		Status:    "failed",
		Message:   i18n.Tf(locale, "notify.deploy_failed", reason),
		Timestamp: time.Now(),
		Metadata:  map[string]string{"reason": reason},
	}
}

// HealthCheckFailed creates a health check failure notification.
func HealthCheckFailed(appName, server, target string) Notification {
	return HealthCheckFailedLocale("en", appName, server, target)
}

// HealthCheckFailedLocale creates a health check failure notification with i18n support.
func HealthCheckFailedLocale(locale, appName, server, target string) Notification {
	return Notification{
		Type:      "health_check",
		AppName:   appName,
		Server:    server,
		Status:    "warning",
		Message:   i18n.Tf(locale, "notify.health_check_failed", appName, target),
		Timestamp: time.Now(),
		Metadata:  map[string]string{"target": target},
	}
}

// Rollback creates a rollback notification.
func Rollback(appName, server, oldImage, reason string) Notification {
	return RollbackLocale("en", appName, server, oldImage, reason)
}

// RollbackLocale creates a rollback notification with i18n support.
func RollbackLocale(locale, appName, server, oldImage, reason string) Notification {
	return Notification{
		Type:      "rollback",
		AppName:   appName,
		Server:    server,
		Status:    "warning",
		Message:   i18n.Tf(locale, "notify.rollback", appName, oldImage, reason),
		Timestamp: time.Now(),
		Metadata:  map[string]string{"old_image": oldImage, "reason": reason},
	}
}

// FieldLabel returns a localized field label for notifications.
func FieldLabel(locale, field string) string {
	return i18n.T(locale, "notify.field."+field)
}
