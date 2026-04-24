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

// FeishuNotifier sends notifications via Feishu (飞书/Lark) Webhook.
type FeishuNotifier struct {
	webhookURL string
	httpClient *http.Client
}

// NewFeishuNotifier creates a new FeishuNotifier.
func NewFeishuNotifier(webhookURL string) *FeishuNotifier {
	return &FeishuNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (f *FeishuNotifier) Name() string { return "feishu" }

// Send sends a notification via Feishu Webhook using interactive card format.
func (f *FeishuNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	payload := f.buildCard(notification)

	body, err := json.Marshal(payload)
	if err != nil {
		return &NotifyResult{Provider: "feishu", Success: false, Error: err.Error()}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.webhookURL, bytes.NewReader(body))
	if err != nil {
		return &NotifyResult{Provider: "feishu", Success: false, Error: err.Error()}, nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return &NotifyResult{Provider: "feishu", Success: false, Error: err.Error()}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &NotifyResult{
			Provider: "feishu",
			Success:  false,
			Error:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}, nil
	}

	// Feishu returns {"code":0,"msg":"success"} on success
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err == nil {
		if code, ok := result["code"].(float64); ok && code != 0 {
			return &NotifyResult{
				Provider: "feishu",
				Success:  false,
				Error:    fmt.Sprintf("Feishu error %v: %v", result["code"], result["msg"]),
			}, nil
		}
	}

	return &NotifyResult{
		Provider: "feishu",
		Success:  true,
		Message:  "feishu webhook delivered",
	}, nil
}

// buildCard builds an interactive card payload for Feishu.
func (f *FeishuNotifier) buildCard(n Notification) map[string]interface{} {
	// Color based on status
	var headerColor string
	switch n.Status {
	case "success":
		headerColor = "green"
	case "failed":
		headerColor = "red"
	case "warning":
		headerColor = "orange"
	default:
		headerColor = "blue"
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

	// Build elements
	elements := []map[string]interface{}{
		{
			"tag": "div",
			"fields": []map[string]interface{}{
				{"is_short": true, "text": map[string]string{"tag": "lark_md", "content": fmt.Sprintf("**App:**\n%s", n.AppName)}},
				{"is_short": true, "text": map[string]string{"tag": "lark_md", "content": fmt.Sprintf("**Server:**\n%s", n.Server)}},
			},
		},
		{
			"tag": "div",
			"fields": []map[string]interface{}{
				{"is_short": true, "text": map[string]string{"tag": "lark_md", "content": fmt.Sprintf("**Status:**\n%s", n.Status)}},
				{"is_short": true, "text": map[string]string{"tag": "lark_md", "content": fmt.Sprintf("**Time:**\n%s", n.Timestamp.Format("15:04:05"))}},
			},
		},
	}

	if n.Message != "" {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]string{"tag": "lark_md", "content": fmt.Sprintf("**Message:**\n%s", n.Message)},
		})
	}

	return map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]string{
					"tag":     "plain_text",
					"content": fmt.Sprintf("%s %s - %s", icon, n.Type, n.AppName),
				},
				"template": headerColor,
			},
			"elements": elements,
		},
	}
}
