package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/monitor"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// bridgeAlertHandler implements monitor.AlertHandler to bridge alerts from the monitor
// package to the typed event bus and notification system.
type bridgeAlertHandler struct {
	db        *gorm.DB
	typedBus  TypedEventBus
	bridge    *Bridge
}

// newBridgeAlertHandler creates a new bridgeAlertHandler.
func newBridgeAlertHandler(db *gorm.DB, typedBus TypedEventBus, bridge *Bridge) *bridgeAlertHandler {
	return &bridgeAlertHandler{
		db:       db,
		typedBus: typedBus,
		bridge:   bridge,
	}
}

// OnAlert is called when a new alert fires.
// It publishes the alert to the typed event bus and sends notifications.
func (h *bridgeAlertHandler) OnAlert(alert *monitor.Alert) {
	slog.Warn("alert fired", "alert_id", alert.ID, "rule", alert.RuleName, "severity", alert.Severity, "message", alert.Message)

	// Publish to typed event bus
	if h.typedBus != nil {
		event := BusEvent{
			ID:        GenerateBusEventID(),
			Type:      EventAlert,
			Topic:     "alert:all",
			Payload: AlertEventPayload{
				AlertID:   alert.ID,
				RuleID:    alert.RuleID,
				RuleName:  alert.RuleName,
				Severity:  string(alert.Severity),
				Message:   alert.Message,
				Value:     alert.Value,
				Threshold: alert.Threshold,
				Status:    alert.Status,
			},
			Timestamp: time.Now(),
		}
		h.typedBus.Publish(event)
	}

	// Persist alert history
	if h.db != nil {
		h.saveAlertHistory(alert)
	}

	// Send notification asynchronously
	if h.bridge != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, err := h.bridge.SendNotification(ctx, "alert", "", "", string(alert.Severity), alert.Message)
			if err != nil {
				slog.Error("failed to send alert notification", "alert_id", alert.ID, "error", err)
			}
		}()
	}
}

// OnAlertResolved is called when a previously firing alert is resolved.
func (h *bridgeAlertHandler) OnAlertResolved(alert *monitor.Alert) {
	slog.Info("alert resolved", "alert_id", alert.ID, "rule", alert.RuleName)

	// Publish to typed event bus
	if h.typedBus != nil {
		event := BusEvent{
			ID:        GenerateBusEventID(),
			Type:      EventAlert,
			Topic:     "alert:all",
			Payload: AlertEventPayload{
				AlertID:   alert.ID,
				RuleID:    alert.RuleID,
				RuleName:  alert.RuleName,
				Severity:  string(alert.Severity),
				Message:   alert.Message,
				Value:     alert.Value,
				Threshold: alert.Threshold,
				Status:    "resolved",
			},
			Timestamp: time.Now(),
		}
		h.typedBus.Publish(event)
	}

	// Update alert history
	if h.db != nil {
		h.db.Model(&model.AlertHistory{}).
			Where("rule_id = ? AND status = ?", alert.RuleID, "firing").
			Update("status", "resolved").
			Update("resolved_at", time.Now())
	}
}

// saveAlertHistory persists an alert to the alert_histories table.
func (h *bridgeAlertHandler) saveAlertHistory(alert *monitor.Alert) {
	history := model.AlertHistory{
		ID:        alert.ID,
		RuleID:    alert.RuleID,
		RuleName:  alert.RuleName,
		Severity:  string(alert.Severity),
		Message:   alert.Message,
		Value:     alert.Value,
		Threshold: alert.Threshold,
		Status:    alert.Status,
		FiredAt:   alert.FiredAt,
	}
	if err := h.db.Create(&history).Error; err != nil {
		slog.Error("failed to save alert history", "alert_id", alert.ID, "error", err)
	}
}
