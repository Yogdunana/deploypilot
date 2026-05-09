package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TelegramNotifier sends notifications via Telegram Bot API.
type TelegramNotifier struct {
	botToken   string
	chatID     string
	baseURL    string
	httpClient *http.Client
}

// NewTelegramNotifier creates a new TelegramNotifier.
func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		baseURL:   "https://api.telegram.org",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetBaseURL allows overriding the API base URL (for testing).
func (t *TelegramNotifier) SetBaseURL(u string) {
	t.baseURL = u
}

// Name returns the notifier name.
func (t *TelegramNotifier) Name() string { return "telegram" }

// Send sends a notification via Telegram Bot API.
func (t *TelegramNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	message := t.formatMessage(notification)

	payload := map[string]string{
		"chat_id":    t.chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return &NotifyResult{Provider: "telegram", Success: false, Error: err.Error()}, nil
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", t.baseURL, t.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return &NotifyResult{Provider: "telegram", Success: false, Error: err.Error()}, nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return &NotifyResult{Provider: "telegram", Success: false, Error: err.Error()}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &NotifyResult{
			Provider: "telegram",
			Success:  false,
			Error:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}, nil
	}

	return &NotifyResult{
		Provider: "telegram",
		Success:  true,
		Message:  fmt.Sprintf("telegram message delivered (HTTP %d)", resp.StatusCode),
	}, nil
}

// formatMessage formats a notification as a Telegram Markdown message.
func (t *TelegramNotifier) formatMessage(n Notification) string {
	locale := n.Metadata["locale"]
	if locale == "" {
		locale = "en"
	}

	var icon string
	switch n.Status {
	case "success":
		icon = "\u2705"
	case "failed":
		icon = "\u274c"
	case "warning":
		icon = "\u26a0\ufe0f"
	default:
		icon = "\u2139\ufe0f"
	}

	msg := fmt.Sprintf("%s *[%s]* %s\n", icon, n.Type, n.AppName)
	msg += fmt.Sprintf("*%s:* %s\n", FieldLabel(locale, "server"), n.Server)
	msg += fmt.Sprintf("*%s:* %s\n", FieldLabel(locale, "status"), n.Status)
	if n.Message != "" {
		msg += fmt.Sprintf("*%s:* %s\n", FieldLabel(locale, "message"), n.Message)
	}
	msg += fmt.Sprintf("*%s:* %s", FieldLabel(locale, "time"), n.Timestamp.Format(time.RFC3339))

	return msg
}
