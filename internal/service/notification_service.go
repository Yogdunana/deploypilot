package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/provider/notify"
)

// getNotifiers loads all enabled notification providers from DB.
func (b *Bridge) getNotifiers(ctx context.Context) ([]notify.Notifier, error) {
	if b.DB == nil {
		return nil, nil // No DB = no notifiers, just log
	}
	var providers []model.Provider
	err := b.DB.Where("type = ? AND enabled = ?", "notify", true).Find(&providers).Error
	if err != nil {
		return nil, err
	}
	var notifiers []notify.Notifier
	for _, p := range providers {
		var cfg struct {
			Channel    string            `json:"channel"` // webhook, email, telegram, dingtalk, feishu
			URL        string            `json:"url"`
			Headers    map[string]string `json:"headers"`
			SMTPHost   string            `json:"smtp_host"`
			SMTPPort   int               `json:"smtp_port"`
			Username   string            `json:"username"`
			Password   string            `json:"password"`
			From       string            `json:"from"`
			BotToken   string            `json:"bot_token"`
			ChatID     string            `json:"chat_id"`
			WebhookURL string            `json:"webhook_url"`
			Secret     string            `json:"secret"`
		}
		if err := json.Unmarshal([]byte(p.Config), &cfg); err != nil {
			slog.Error("failed to parse notify provider config", "provider", p.Name, "error", err)
			continue
		}
		switch cfg.Channel {
		case "webhook":
			notifiers = append(notifiers, notify.NewWebhookNotifier(cfg.URL, cfg.Headers))
		case "email":
			notifiers = append(notifiers, notify.NewEmailNotifier(notify.EmailConfig{
				SMTPHost: cfg.SMTPHost,
				SMTPPort: cfg.SMTPPort,
				Username: cfg.Username,
				Password: cfg.Password,
				From:     cfg.From,
			}))
		case "telegram":
			notifiers = append(notifiers, notify.NewTelegramNotifier(cfg.BotToken, cfg.ChatID))
		case "dingtalk":
			notifiers = append(notifiers, notify.NewDingTalkNotifier(cfg.WebhookURL, cfg.Secret))
		case "feishu":
			notifiers = append(notifiers, notify.NewFeishuNotifier(cfg.WebhookURL))
		case "wecom":
			notifiers = append(notifiers, notify.NewWeComNotifier(cfg.WebhookURL))
		}
	}
	return notifiers, nil
}

// ---------- 22. SendNotification ----------

func (b *Bridge) SendNotification(ctx context.Context, nType, appName, server, status, message string) (interface{}, error) {
	slog.Info("notification sent", "type", nType, "app", appName, "server", server, "status", status, "message", message)

	notifiers, err := b.getNotifiers(ctx)
	if err != nil {
		slog.Error("failed to load notifiers", "error", err)
	}

	if len(notifiers) == 0 {
		return map[string]interface{}{
			"status":  "logged",
			"type":    nType,
			"app":     appName,
			"message": "no notification providers configured",
		}, nil
	}

	notification := notify.Notification{
		Type:      nType,
		AppName:   appName,
		Server:    server,
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	}

	multi := notify.NewMultiNotifier(notifiers...)
	results, err := multi.Send(ctx, notification)
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		}, nil
	}

	// Count successes
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	return map[string]interface{}{
		"status":          "sent",
		"type":            nType,
		"app":             appName,
		"total_notifiers": len(notifiers),
		"success_count":   successCount,
		"results":         results,
	}, nil
}
