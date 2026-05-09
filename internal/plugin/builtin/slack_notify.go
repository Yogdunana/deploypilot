package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/gin-gonic/gin"
)

// SlackNotifyPlugin sends Slack notifications for deploy events.
// It reads the webhook URL from plugin config.
type SlackNotifyPlugin struct {
	webhookURL string
	client     *http.Client
}

// NewSlackNotifyPlugin creates a new SlackNotifyPlugin.
func NewSlackNotifyPlugin() plugin.EventPlugin {
	return &SlackNotifyPlugin{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *SlackNotifyPlugin) Name() string {
	return "slack-notify"
}

func (p *SlackNotifyPlugin) Version() string {
	return "1.0.0"
}

func (p *SlackNotifyPlugin) Description() string {
	return "Sends Slack notifications for deploy events via incoming webhook"
}

func (p *SlackNotifyPlugin) Init(_ context.Context, config map[string]interface{}) error {
	if config == nil {
		return fmt.Errorf("slack-notify: config is required")
	}
	webhookURL, _ := config["webhook_url"].(string)
	if webhookURL == "" {
		return fmt.Errorf("slack-notify: webhook_url is required in config")
	}
	p.webhookURL = webhookURL
	slog.Info("slack-notify plugin initialized", "webhook_url", webhookURL)
	return nil
}

func (p *SlackNotifyPlugin) Start() error {
	slog.Info("slack-notify plugin started")
	return nil
}

func (p *SlackNotifyPlugin) Stop() error {
	slog.Info("slack-notify plugin stopped")
	return nil
}

func (p *SlackNotifyPlugin) OnEvent(event plugin.BusEvent) {
	// Only handle deploy events
	if event.Type != "deploy" {
		return
	}

	payload := map[string]interface{}{
		"text": fmt.Sprintf(
			"[DeployPilot] Deploy event: %s (topic: %s)",
			event.Topic, event.Type,
		),
	}

	if event.Payload != nil {
		payload["attachments"] = []map[string]interface{}{
			{
				"color":  "#36a64f",
				"fields": []map[string]interface{}{{"value": fmt.Sprintf("%v", event.Payload)}},
			},
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("slack-notify: failed to marshal payload", "error", err)
		return
	}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.webhookURL, bytes.NewReader(body))
	if err != nil {
		slog.Error("slack-notify: failed to create request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		slog.Error("slack-notify: failed to send notification", "error", err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		slog.Error("slack-notify: webhook returned non-success status",
			"status_code", resp.StatusCode,
		)
		return
	}

	slog.Info("slack-notify: notification sent", "topic", event.Topic)
}

func (p *SlackNotifyPlugin) RegisterAPIRoutes(_ *gin.RouterGroup) {
	// No custom API routes for this plugin
}
