package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SignWebhook computes an HMAC-SHA256 signature for a webhook payload.
// The signature format is "sha256=<hex>" where the HMAC input is "timestamp.body".
func SignWebhook(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.%s", timestamp, string(body))
	return fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))
}

// WebhookPayload is the structured envelope sent to outbound webhook endpoints.
type WebhookPayload struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	Topic     string      `json:"topic"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// FormatAdapter defines the interface for converting a WebhookPayload into
// a platform-specific byte representation and a content-type string.
type FormatAdapter interface {
	Format(payload WebhookPayload) ([]byte, string)
	Name() string
}

// --- Helper: extract common fields from payload ---

// payloadFields extracts commonly-used fields from a WebhookPayload's Payload.
func payloadFields(wp WebhookPayload) map[string]string {
	m, err := payloadToMap(wp.Payload)
	if err != nil {
		return map[string]string{}
	}
	fields := make(map[string]string)
	for _, key := range []string{"status", "message", "app_name", "server_name", "action", "severity"} {
		if v, ok := m[key].(string); ok {
			fields[key] = v
		}
	}
	return fields
}

// statusColor returns a Slack-compatible hex color based on status string.
func statusColor(status string) string {
	switch strings.ToLower(status) {
	case "success", "ok", "healthy", "up":
		return "#36a64f"
	case "fail", "failed", "error", "down", "critical":
		return "#e01e5a"
	default:
		return "#ffaa00"
	}
}

// statusColorInt returns a Discord-compatible integer color based on status string.
func statusColorInt(status string) int {
	switch strings.ToLower(status) {
	case "success", "ok", "healthy", "up":
		return 3066993
	case "fail", "failed", "error", "down", "critical":
		return 15158332
	default:
		return 16776960
	}
}

// ===================== JSON Formatter =====================

// JSONFormatter implements FormatAdapter as a stateless strategy.
type JSONFormatter struct{}

// Format marshals the WebhookPayload to indented JSON.
func (f *JSONFormatter) Format(payload WebhookPayload) ([]byte, string) {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return []byte(`{"error":"failed to marshal payload"}`), "application/json"
	}
	return data, "application/json"
}

// Name returns the formatter name.
func (f *JSONFormatter) Name() string { return "json" }

// ===================== Slack Formatter =====================

// SlackFormatter implements FormatAdapter for Slack webhook payloads.
type SlackFormatter struct{}

// slackAttachment represents a Slack message attachment.
type slackAttachment struct {
	Fallback string   `json:"fallback"`
	Color    string   `json:"color"`
	Title    string   `json:"title"`
	Text     string   `json:"text"`
	Fields   []slackField `json:"fields,omitempty"`
	Footer   string   `json:"footer,omitempty"`
	Ts       int64    `json:"ts,omitempty"`
}

type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// slackMessage is the top-level Slack Incoming Webhook payload.
type slackMessage struct {
	Text        string            `json:"text,omitempty"`
	Attachments []slackAttachment `json:"attachments"`
}

// Format produces a Slack Incoming Webhook compatible JSON body.
func (f *SlackFormatter) Format(payload WebhookPayload) ([]byte, string) {
	fields := payloadFields(payload)
	status := fields["status"]
	message := fields["message"]
	if message == "" {
		message = fmt.Sprintf("[%s] %s", payload.EventType, payload.Topic)
	}

	attachment := slackAttachment{
		Fallback: fmt.Sprintf("[%s] %s: %s", payload.EventType, payload.Topic, message),
		Color:    statusColor(status),
		Title:    fmt.Sprintf("[%s] %s", payload.EventType, payload.Topic),
		Text:     message,
		Ts:       payload.Timestamp.Unix(),
	}

	// Build optional fields
	var sf []slackField
	if v := fields["app_name"]; v != "" {
		sf = append(sf, slackField{Title: "App", Value: v, Short: true})
	}
	if v := fields["server_name"]; v != "" {
		sf = append(sf, slackField{Title: "Server", Value: v, Short: true})
	}
	if v := fields["action"]; v != "" {
		sf = append(sf, slackField{Title: "Action", Value: v, Short: true})
	}
	if v := fields["severity"]; v != "" {
		sf = append(sf, slackField{Title: "Severity", Value: v, Short: true})
	}
	attachment.Fields = sf
	attachment.Footer = fmt.Sprintf("Event: %s", payload.EventID)

	msg := slackMessage{
		Attachments: []slackAttachment{attachment},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return []byte(`{"text":"failed to format slack payload"}`), "application/json"
	}
	return data, "application/json"
}

// Name returns the formatter name.
func (f *SlackFormatter) Name() string { return "slack" }

// ===================== Discord Formatter =====================

// DiscordFormatter implements FormatAdapter for Discord webhook payloads.
type DiscordFormatter struct{}

// discordEmbed represents a Discord embed object.
type discordEmbed struct {
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Color       int             `json:"color"`
	Fields      []discordField  `json:"fields,omitempty"`
	Footer      *discordFooter  `json:"footer,omitempty"`
	Timestamp   string          `json:"timestamp"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordFooter struct {
	Text string `json:"text"`
}

// discordMessage is the top-level Discord Webhook payload.
type discordMessage struct {
	Embeds []discordEmbed `json:"embeds"`
}

// Format produces a Discord Webhook compatible JSON body.
func (f *DiscordFormatter) Format(payload WebhookPayload) ([]byte, string) {
	fields := payloadFields(payload)
	status := fields["status"]
	message := fields["message"]
	if message == "" {
		message = fmt.Sprintf("[%s] %s", payload.EventType, payload.Topic)
	}

	embed := discordEmbed{
		Title:       fmt.Sprintf("[%s] %s", payload.EventType, payload.Topic),
		Description: message,
		Color:       statusColorInt(status),
		Timestamp:   payload.Timestamp.Format(time.RFC3339),
		Footer:      &discordFooter{Text: fmt.Sprintf("Event: %s", payload.EventID)},
	}

	var df []discordField
	if v := fields["app_name"]; v != "" {
		df = append(df, discordField{Name: "App", Value: v, Inline: true})
	}
	if v := fields["server_name"]; v != "" {
		df = append(df, discordField{Name: "Server", Value: v, Inline: true})
	}
	if v := fields["action"]; v != "" {
		df = append(df, discordField{Name: "Action", Value: v, Inline: true})
	}
	if v := fields["severity"]; v != "" {
		df = append(df, discordField{Name: "Severity", Value: v, Inline: true})
	}
	embed.Fields = df

	msg := discordMessage{
		Embeds: []discordEmbed{embed},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return []byte(`{"embeds":[{"title":"Error","description":"failed to format discord payload"}]}`), "application/json"
	}
	return data, "application/json"
}

// Name returns the formatter name.
func (f *DiscordFormatter) Name() string { return "discord" }

// ===================== Teams Formatter =====================

// TeamsFormatter implements FormatAdapter for Microsoft Teams webhook payloads.
type TeamsFormatter struct{}

// teamsAttachment is the top-level Teams webhook payload.
type teamsAttachment struct {
	Type        string             `json:"type"`
	Attachments []teamsCardWrapper `json:"attachments"`
}

type teamsCardWrapper struct {
	ContentType string      `json:"contentType"`
	Content     teamsAdaptiveCard `json:"content"`
}

// teamsAdaptiveCard represents a Microsoft Adaptive Card.
type teamsAdaptiveCard struct {
	Type    string           `json:"type"`
	Version string           `json:"$version"`
	Body    []teamsCardElement `json:"body"`
}

// teamsCardElement is a union type for Adaptive Card body elements.
type teamsCardElement struct {
	Type       string              `json:"type"`
	Text       string              `json:"text,omitempty"`
	Weight     string              `json:"weight,omitempty"`
	Size       string              `json:"size,omitempty"`
	Color      string              `json:"color,omitempty"`
	Facts      []teamsFact         `json:"facts,omitempty"`
	IsSubtle   *bool               `json:"isSubtle,omitempty"`
	Separator  bool                `json:"separator,omitempty"`
}

type teamsFact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

// Format produces a Microsoft Teams Adaptive Card compatible JSON body.
func (f *TeamsFormatter) Format(payload WebhookPayload) ([]byte, string) {
	fields := payloadFields(payload)
	status := fields["status"]
	message := fields["message"]
	if message == "" {
		message = fmt.Sprintf("[%s] %s", payload.EventType, payload.Topic)
	}

	// Determine color based on status
	color := "warning"
	switch strings.ToLower(status) {
	case "success", "ok", "healthy", "up":
		color = "good"
	case "fail", "failed", "error", "down", "critical":
		color = "attention"
	}

	isSubtle := true
	cardBody := []teamsCardElement{
		{
			Type:   "TextBlock",
			Text:   fmt.Sprintf("[%s] %s", payload.EventType, payload.Topic),
			Size:   "Large",
			Weight: "Bolder",
		},
		{
			Type:     "TextBlock",
			Text:     message,
			Color:    color,
			Separator: true,
		},
	}

	// Build fact set
	var facts []teamsFact
	if v := fields["app_name"]; v != "" {
		facts = append(facts, teamsFact{Title: "App", Value: v})
	}
	if v := fields["server_name"]; v != "" {
		facts = append(facts, teamsFact{Title: "Server", Value: v})
	}
	if v := fields["action"]; v != "" {
		facts = append(facts, teamsFact{Title: "Action", Value: v})
	}
	if v := fields["severity"]; v != "" {
		facts = append(facts, teamsFact{Title: "Severity", Value: v})
	}
	if v := fields["status"]; v != "" {
		facts = append(facts, teamsFact{Title: "Status", Value: v})
	}
	facts = append(facts, teamsFact{Title: "Event ID", Value: payload.EventID})

	if len(facts) > 0 {
		cardBody = append(cardBody, teamsCardElement{
			Type:  "FactSet",
			Facts: facts,
		})
	}

	cardBody = append(cardBody, teamsCardElement{
		Type:      "TextBlock",
		Text:      payload.Timestamp.Format(time.RFC3339),
		IsSubtle:  &isSubtle,
		Size:      "Small",
		Separator: true,
	})

	card := teamsAdaptiveCard{
		Type:    "AdaptiveCard",
		Version: "1.4",
		Body:    cardBody,
	}

	attachment := teamsAttachment{
		Type: "message",
		Attachments: []teamsCardWrapper{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content:     card,
			},
		},
	}

	data, err := json.Marshal(attachment)
	if err != nil {
		return []byte(`{"type":"message","attachments":[{"contentType":"text/plain","content":"failed to format teams payload"}]}`), "application/json"
	}
	return data, "application/json"
}

// Name returns the formatter name.
func (f *TeamsFormatter) Name() string { return "teams" }

// ===================== Factory =====================

// GetFormatter returns a FormatAdapter for the given format name.
// Supported formats: "json", "slack", "discord", "teams".
// Returns JSONFormatter as default if the format is not recognized.
func GetFormatter(format string) FormatAdapter {
	switch strings.ToLower(format) {
	case "slack":
		return &SlackFormatter{}
	case "discord":
		return &DiscordFormatter{}
	case "teams":
		return &TeamsFormatter{}
	default:
		return &JSONFormatter{}
	}
}
