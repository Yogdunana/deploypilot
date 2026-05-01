package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackNotifier sends notifications via Slack Incoming Webhook.
// See: https://api.slack.com/messaging/webhooks
type SlackNotifier struct {
	WebhookURL string
	Channel    string // optional override, e.g. "#ops-alerts"
	Username   string // optional bot name override
	IconEmoji  string // optional, e.g. ":rocket:"
	Client     *http.Client
}

// NewSlackNotifier creates a new SlackNotifier.
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		WebhookURL: webhookURL,
		Username:   "DeployPilot",
		IconEmoji:  ":rocket:",
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (s *SlackNotifier) Name() string { return "slack" }

// slackPayload is the request body for Slack Incoming Webhook.
type slackPayload struct {
	Channel   string             `json:"channel,omitempty"`
	Username  string             `json:"username,omitempty"`
	IconEmoji string             `json:"icon_emoji,omitempty"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
	Text      string             `json:"text,omitempty"`
}

// slackAttachment represents a Slack message attachment.
type slackAttachment struct {
	Color    string   `json:"color,omitempty"`
	Title    string   `json:"title,omitempty"`
	Text     string   `json:"text,omitempty"`
	Fields   []slackField `json:"fields,omitempty"`
	Footer   string   `json:"footer,omitempty"`
	TS       int64    `json:"ts,omitempty"`
}

// slackField represents a field in a Slack attachment.
type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// slackResponse is the response from Slack Incoming Webhook.
type slackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// Send sends a notification via Slack Incoming Webhook.
func (s *SlackNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	color := s.mapColor(notification.Status)

	attachment := slackAttachment{
		Color: color,
		Title: fmt.Sprintf("[%s] %s", notification.Type, notification.AppName),
		Text:  notification.Message,
		Fields: []slackField{
			{Title: "Status", Value: notification.Status, Short: true},
			{Title: "Server", Value: notification.Server, Short: true},
		},
		Footer: "DeployPilot",
		TS:     time.Now().Unix(),
	}

	// Add app name field if server is empty
	if notification.Server == "" {
		attachment.Fields = []slackField{
			{Title: "Status", Value: notification.Status, Short: true},
		}
	}

	payload := slackPayload{
		Channel:     s.Channel,
		Username:    s.Username,
		IconEmoji:   s.IconEmoji,
		Attachments: []slackAttachment{attachment},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return &NotifyResult{Provider: "slack", Success: false, Error: err.Error()}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(data))
	if err != nil {
		return &NotifyResult{Provider: "slack", Success: false, Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return &NotifyResult{Provider: "slack", Success: false, Error: err.Error()}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	var slackResp slackResponse
	if err := json.NewDecoder(resp.Body).Decode(&slackResp); err != nil {
		return &NotifyResult{Provider: "slack", Success: false, Error: fmt.Sprintf("decode response failed: %v", err)}, nil
	}

	if slackResp.OK {
		return &NotifyResult{Provider: "slack", Success: true, Message: "slack notification delivered"}, nil
	}

	return &NotifyResult{
		Provider: "slack",
		Success:  false,
		Error:    fmt.Sprintf("slack error: %s", slackResp.Error),
	}, nil
}

// mapColor maps notification status to Slack attachment color.
func (s *SlackNotifier) mapColor(status string) string {
	switch status {
	case "success":
		return "#36a64f" // green
	case "failed", "critical":
		return "#e01e5a" // red
	case "warning":
		return "#f2c744" // yellow
	default:
		return "#808080" // grey
	}
}
