package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/plugin"
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
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(p.Config), &config); err != nil {
			slog.Error("failed to parse notify provider config", "provider", p.Name, "error", err)
			continue
		}
		channel, _ := config["channel"].(string)
		if channel == "" {
			slog.Error("notify provider missing channel", "provider", p.Name)
			continue
		}
		desc, ok := plugin.Global().GetDescriptor("notify", channel)
		if !ok {
			slog.Error("no plugin registered for notify channel", "channel", channel, "provider", p.Name)
			continue
		}
		instance, err := plugin.Global().CreateInstance(fmt.Sprintf("notify-%s", p.ID), desc, config)
		if err != nil {
			slog.Error("failed to create notify instance", "provider", p.Name, "error", err)
			continue
		}
		notifier, ok := instance.(notify.Notifier)
		if !ok {
			slog.Error("notify plugin does not implement Notifier", "provider", p.Name)
			continue
		}
		notifiers = append(notifiers, notifier)
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
