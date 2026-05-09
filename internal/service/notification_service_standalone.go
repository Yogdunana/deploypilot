package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/Yogdunana/deploypilot/internal/provider/notify"
	"gorm.io/gorm"
)

// NotificationService provides a standalone notification service
// that can be used by the EventRouter and other components independently of Bridge.
type NotificationService struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{
		db:     db,
		logger: slog.Default(),
	}
}

// SendToChannels sends a notification to specific channels by name.
// channels is a comma-separated list of channel names (e.g. "webhook,email,dingtalk").
func (s *NotificationService) SendToChannels(ctx context.Context, nType, appName, server, status, message, channels string) (map[string]interface{}, error) {
	if s.db == nil {
		return map[string]interface{}{"status": "no_db"}, nil
	}

	targetChannels := strings.Split(channels, ",")
	notifiers, err := s.loadNotifiersForChannels(ctx, targetChannels)
	if err != nil {
		s.logger.Error("failed to load notifiers", "error", err)
	}

	if len(notifiers) == 0 {
		return map[string]interface{}{
			"status":  "no_matching_channels",
			"channels": targetChannels,
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
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	return map[string]interface{}{
		"status":        "sent",
		"total":         len(notifiers),
		"success_count": successCount,
		"results":       results,
	}, nil
}

// SendToAll sends a notification to all enabled notification channels.
func (s *NotificationService) SendToAll(ctx context.Context, nType, appName, server, status, message string) (map[string]interface{}, error) {
	if s.db == nil {
		return map[string]interface{}{"status": "no_db"}, nil
	}

	notifiers, err := s.loadAllNotifiers(ctx)
	if err != nil {
		s.logger.Error("failed to load notifiers", "error", err)
	}

	if len(notifiers) == 0 {
		return map[string]interface{}{
			"status":  "no_providers",
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
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	return map[string]interface{}{
		"status":        "sent",
		"total":         len(notifiers),
		"success_count": successCount,
		"results":       results,
	}, nil
}

// loadNotifiersForChannels loads notifiers matching specific channel names.
func (s *NotificationService) loadNotifiersForChannels(ctx context.Context, channels []string) ([]notify.Notifier, error) {
	channelSet := make(map[string]bool, len(channels))
	for _, c := range channels {
		channelSet[strings.TrimSpace(c)] = true
	}

	var providers []model.Provider
	if err := s.db.Where("type = ? AND enabled = ?", "notify", true).Find(&providers).Error; err != nil {
		return nil, err
	}

	return s.providersToNotifiers(providers, channelSet)
}

// loadAllNotifiers loads all enabled notification providers.
func (s *NotificationService) loadAllNotifiers(ctx context.Context) ([]notify.Notifier, error) {
	var providers []model.Provider
	if err := s.db.Where("type = ? AND enabled = ?", "notify", true).Find(&providers).Error; err != nil {
		return nil, err
	}
	return s.providersToNotifiers(providers, nil)
}

// providersToNotifiers converts provider records to Notifier instances.
func (s *NotificationService) providersToNotifiers(providers []model.Provider, channelFilter map[string]bool) ([]notify.Notifier, error) {
	var notifiers []notify.Notifier
	for _, p := range providers {
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(p.Config), &config); err != nil {
			s.logger.Error("failed to parse notify provider config", "provider", p.Name, "error", err)
			continue
		}
		channel, _ := config["channel"].(string)
		if channel == "" {
			continue
		}
		// Apply channel filter if set
		if channelFilter != nil && !channelFilter[channel] {
			continue
		}

		desc, ok := plugin.Global().GetDescriptor("notify", channel)
		if !ok {
			s.logger.Error("no plugin registered for notify channel", "channel", channel, "provider", p.Name)
			continue
		}
		instance, err := plugin.Global().CreateInstance(fmt.Sprintf("notify-%s", p.ID), desc, config)
		if err != nil {
			s.logger.Error("failed to create notify instance", "provider", p.Name, "error", err)
			continue
		}
		notifier, ok := instance.(notify.Notifier)
		if !ok {
			continue
		}
		notifiers = append(notifiers, notifier)
	}
	return notifiers, nil
}
