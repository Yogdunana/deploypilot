package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DingTalkNotifier sends notifications via DingTalk (钉钉) Webhook.
type DingTalkNotifier struct {
	webhookURL string
	secret     string // optional, for signed webhook
	httpClient *http.Client
}

// NewDingTalkNotifier creates a new DingTalkNotifier.
func NewDingTalkNotifier(webhookURL, secret string) *DingTalkNotifier {
	return &DingTalkNotifier{
		webhookURL: webhookURL,
		secret:     secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (d *DingTalkNotifier) Name() string { return "dingtalk" }

// Send sends a notification via DingTalk Webhook.
func (d *DingTalkNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	title, text := d.formatMessage(notification)

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  text,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return &NotifyResult{Provider: "dingtalk", Success: false, Error: err.Error()}, nil
	}

	reqURL := d.webhookURL
	if d.secret != "" {
		reqURL = d.signURL(d.webhookURL, d.secret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return &NotifyResult{Provider: "dingtalk", Success: false, Error: err.Error()}, nil
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return &NotifyResult{Provider: "dingtalk", Success: false, Error: err.Error()}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return &NotifyResult{
			Provider: "dingtalk",
			Success:  false,
			Error:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)),
		}, nil
	}

	// DingTalk returns {"errcode":0,"errmsg":"ok"} on success
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err == nil {
		if errCode, ok := result["errcode"].(float64); ok && errCode != 0 {
			return &NotifyResult{
				Provider: "dingtalk",
				Success:  false,
				Error:    fmt.Sprintf("DingTalk error %v: %v", result["errcode"], result["errmsg"]),
			}, nil
		}
	}

	return &NotifyResult{
		Provider: "dingtalk",
		Success:  true,
		Message:  "dingtalk webhook delivered",
	}, nil
}

// formatMessage formats a notification as DingTalk Markdown.
func (d *DingTalkNotifier) formatMessage(n Notification) (string, string) {
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

	text := fmt.Sprintf("### %s %s %s\n\n", icon, n.Type, n.AppName)
	text += fmt.Sprintf("- **%s:** %s\n", FieldLabel(locale, "server"), n.Server)
	text += fmt.Sprintf("- **%s:** %s\n", FieldLabel(locale, "status"), n.Status)
	if n.Message != "" {
		text += fmt.Sprintf("- **%s:** %s\n", FieldLabel(locale, "message"), n.Message)
	}
	text += fmt.Sprintf("- **%s:** %s", FieldLabel(locale, "time"), n.Timestamp.Format(time.RFC3339))

	return title, text
}

// signURL generates a signed DingTalk webhook URL with HMAC-SHA256.
func (d *DingTalkNotifier) signURL(webhookURL, secret string) string {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	stringToSign := timestamp + "\n" + secret

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stringToSign))
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	sep := "?"
	if strings.Contains(webhookURL, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%stimestamp=%s&sign=%s", webhookURL, sep, timestamp, sign)
}
