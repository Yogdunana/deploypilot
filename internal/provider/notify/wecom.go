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

// WeComNotifier sends notifications via WeCom (企业微信) Webhook.
type WeComNotifier struct {
	webhookURL string
	httpClient *http.Client
}

// NewWeComNotifier creates a new WeComNotifier.
func NewWeComNotifier(webhookURL string) *WeComNotifier {
	return &WeComNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (w *WeComNotifier) Name() string { return "wecom" }

// Validate checks that the webhook URL is configured.
func (w *WeComNotifier) Validate() error {
	if w.webhookURL == "" {
		return fmt.Errorf("wecom webhook URL is required")
	}
	return nil
}

// Send sends a notification via WeCom Webhook using markdown format.
func (w *WeComNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	title, text := w.formatMessage(notification)

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": fmt.Sprintf("## %s\n> %s", title, text),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return &NotifyResult{Provider: "wecom", Success: false, Error: err.Error()}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.webhookURL, bytes.NewReader(body))
	if err != nil {
		return &NotifyResult{Provider: "wecom", Success: false, Error: err.Error()}, nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return &NotifyResult{Provider: "wecom", Success: false, Error: err.Error()}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &NotifyResult{
			Provider: "wecom",
			Success:  false,
			Error:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}, nil
	}

	// WeCom returns {"errcode":0,"errmsg":"ok"} on success
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err == nil {
		if errCode, ok := result["errcode"].(float64); ok && errCode != 0 {
			return &NotifyResult{
				Provider: "wecom",
				Success:  false,
				Error:    fmt.Sprintf("WeCom error %v: %v", result["errcode"], result["errmsg"]),
			}, nil
		}
	}

	return &NotifyResult{
		Provider: "wecom",
		Success:  true,
		Message:  "wecom webhook delivered",
	}, nil
}

// formatMessage formats a notification as WeCom Markdown.
func (w *WeComNotifier) formatMessage(n Notification) (string, string) {
	locale := n.Metadata["locale"]
	if locale == "" {
		locale = "en"
	}

	title := fmt.Sprintf("[%s] %s - %s", n.Type, n.AppName, n.Status)

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

	text := fmt.Sprintf("%s **%s %s**\n\n", icon, n.Type, n.AppName)
	text += fmt.Sprintf("- **%s:** %s\n", FieldLabel(locale, "server"), n.Server)
	text += fmt.Sprintf("- **%s:** %s\n", FieldLabel(locale, "status"), n.Status)
	if n.Message != "" {
		text += fmt.Sprintf("- **%s:** %s\n", FieldLabel(locale, "message"), n.Message)
	}
	text += fmt.Sprintf("- **%s:** %s", FieldLabel(locale, "time"), n.Timestamp.Format(time.RFC3339))

	return title, text
}
