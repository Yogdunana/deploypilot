package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// AlertEngine provides enhanced alert management with silence periods,
// escalation policies, and alert grouping.
type AlertEngine struct {
	db        *gorm.DB
	notifySvc *NotificationService
	logger    *slog.Logger
	stopCh    chan struct{}
}

// NewAlertEngine creates a new AlertEngine.
func NewAlertEngine(db *gorm.DB, notifySvc *NotificationService) *AlertEngine {
	return &AlertEngine{
		db:        db,
		notifySvc: notifySvc,
		logger:    slog.Default(),
		stopCh:    make(chan struct{}),
	}
}

// ========== Silence Period Management ==========

// IsSilenced checks if an alert should be suppressed based on active silence rules.
func (e *AlertEngine) IsSilenced(severity, serverID string) (bool, string) {
	if e.db == nil {
		return false, ""
	}

	var silences []model.AlertSilence
	now := time.Now()
	if err := e.db.Where("starts_at <= ? AND ends_at >= ?", now, now).Find(&silences).Error; err != nil {
		e.logger.Error("failed to check silence periods", "error", err)
		return false, ""
	}

	for _, s := range silences {
		if e.matchesSilence(s, severity, serverID) {
			return true, s.Name
		}
	}
	return false, ""
}

// matchesSilence checks if a silence rule matches the given alert attributes.
func (e *AlertEngine) matchesSilence(s model.AlertSilence, severity, serverID string) bool {
	if s.Matchers == "" {
		return true // empty matchers = match all
	}

	var matchers map[string]interface{}
	if err := json.Unmarshal([]byte(s.Matchers), &matchers); err != nil {
		return false
	}

	// Check severity match
	if sevMatchers, ok := matchers["severity"]; ok {
		switch v := sevMatchers.(type) {
		case string:
			if v != "" && v != severity {
				return false
			}
		case []interface{}:
			found := false
			for _, item := range v {
				if s, ok := item.(string); ok && s == severity {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check server_id match
	if sidMatchers, ok := matchers["server_id"]; ok {
		if s, ok := sidMatchers.(string); ok && s != "" && s != serverID {
			return false
		}
	}

	return true
}

// CreateSilence creates a new silence period.
func (e *AlertEngine) CreateSilence(silence *model.AlertSilence) error {
	if silence.ID == "" {
		silence.ID = fmt.Sprintf("silence-%d", time.Now().UnixNano())
	}
	return e.db.Create(silence).Error
}

// ListSilences returns active and upcoming silence periods.
func (e *AlertEngine) ListSilences() ([]model.AlertSilence, error) {
	var silences []model.AlertSilence
	err := e.db.Where("ends_at >= ?", time.Now()).Order("starts_at ASC").Find(&silences).Error
	return silences, err
}

// DeleteSilence deletes a silence period by ID.
func (e *AlertEngine) DeleteSilence(id string) error {
	return e.db.Delete(&model.AlertSilence{}, "id = ?", id).Error
}

// ========== Escalation Management ==========

// EscalationStep defines a single step in an escalation policy.
type EscalationStep struct {
	AfterMinutes int      `json:"after_minutes"`
	Severity     string   `json:"severity"`
	Channels     []string `json:"channels"`
	Message      string   `json:"message,omitempty"`
}

// CheckEscalations checks if any active alerts need escalation.
// Should be called periodically (e.g., every minute).
func (e *AlertEngine) CheckEscalations(ctx context.Context) {
	if e.db == nil {
		return
	}

	// Load active escalation policies
	var policies []model.AlertEscalation
	if err := e.db.Where("enabled = ?", true).Find(&policies).Error; err != nil {
		e.logger.Error("failed to load escalation policies", "error", err)
		return
	}

	// Load active (firing) alert groups
	var groups []model.AlertGroup
	if err := e.db.Where("status = ?", "firing").Find(&groups).Error; err != nil {
		e.logger.Error("failed to load alert groups", "error", err)
		return
	}

	now := time.Now()
	for _, policy := range policies {
		var steps []EscalationStep
		if err := json.Unmarshal([]byte(policy.Steps), &steps); err != nil {
			continue
		}

		// Parse rule IDs
		var ruleIDs []string
		if policy.RuleIDs != "" {
			_ = json.Unmarshal([]byte(policy.RuleIDs), &ruleIDs)
		}

		for _, group := range groups {
			// Check if this group matches the policy's rule scope
			if len(ruleIDs) > 0 && !e.stringSliceContains(ruleIDs, group.RuleID) {
				continue
			}

			// Check each escalation step
			for _, step := range steps {
				elapsed := now.Sub(group.FirstAlertAt).Minutes()
				if elapsed >= float64(step.AfterMinutes) {
					// Check if we already escalated to this level
					if group.Severity == step.Severity {
						continue
					}

					// Escalate: update severity and send notification
					e.escalateAlert(ctx, &group, step, policy)
					break
				}
			}
		}
	}
}

// escalateAlert performs the escalation: updates severity and sends notification.
func (e *AlertEngine) escalateAlert(ctx context.Context, group *model.AlertGroup, step EscalationStep, policy model.AlertEscalation) {
	// Update group severity
	if err := e.db.Model(group).Updates(map[string]interface{}{
		"severity":  step.Severity,
		"alert_count": gorm.Expr("alert_count + 1"),
		"updated_at": time.Now(),
	}).Error; err != nil {
		slog.Error("failed to escalate alert group severity", "group", group.GroupKey, "error", err)
	}

	message := fmt.Sprintf("🚨 ALERT ESCALATION: %s (group: %s, severity: %s)",
		group.GroupKey, group.RuleID, step.Severity)
	if step.Message != "" {
		message = step.Message
	}

	e.logger.Warn("alert escalated",
		"group", group.GroupKey,
		"rule", group.RuleID,
		"severity", step.Severity,
		"policy", policy.Name,
	)

	// Send escalation notification
	if e.notifySvc != nil && len(step.Channels) > 0 {
		channels := strings.Join(step.Channels, ",")
		go func() {
			defer func() {
				if rv := recover(); rv != nil {
					e.logger.Error("panic recovered in escalation notification", "group_key", group.GroupKey, "panic", rv)
				}
			}()
			_, err := e.notifySvc.SendToChannels(ctx, "alert", group.GroupKey, "", step.Severity, message, channels)
			if err != nil {
				e.logger.Error("failed to send escalation notification", "error", err)
			}
		}()
	}
}

// CreateEscalation creates a new escalation policy.
func (e *AlertEngine) CreateEscalation(esc *model.AlertEscalation) error {
	if esc.ID == "" {
		esc.ID = fmt.Sprintf("esc-%d", time.Now().UnixNano())
	}
	return e.db.Create(esc).Error
}

// ListEscalations returns all escalation policies.
func (e *AlertEngine) ListEscalations() ([]model.AlertEscalation, error) {
	var escalations []model.AlertEscalation
	err := e.db.Order("created_at DESC").Find(&escalations).Error
	return escalations, err
}

// DeleteEscalation deletes an escalation policy by ID.
func (e *AlertEngine) DeleteEscalation(id string) error {
	return e.db.Delete(&model.AlertEscalation{}, "id = ?", id).Error
}

// ========== Alert Grouping ==========

// GroupAlert creates or updates an alert group for deduplication.
func (e *AlertEngine) GroupAlert(ruleID, groupKey, severity string) (*model.AlertGroup, error) {
	if e.db == nil {
		return nil, nil
	}

	var group model.AlertGroup
	err := e.db.Where("group_key = ? AND rule_id = ? AND status = ?", groupKey, ruleID, "firing").First(&group).Error

	if err == gorm.ErrRecordNotFound {
		// Create new group
		group = model.AlertGroup{
			ID:           fmt.Sprintf("ag-%d", time.Now().UnixNano()),
			GroupKey:     groupKey,
			RuleID:       ruleID,
			Severity:     severity,
			AlertCount:   1,
			FirstAlertAt: time.Now(),
			LastAlertAt:  time.Now(),
			Status:       "firing",
		}
		err = e.db.Create(&group).Error
		return &group, err
	}

	if err != nil {
		return nil, err
	}

	// Update existing group
	if err := e.db.Model(&group).Updates(map[string]interface{}{
		"alert_count": gorm.Expr("alert_count + 1"),
		"last_alert_at": time.Now(),
		"updated_at":   time.Now(),
	}).Error; err != nil {
		slog.Error("failed to update alert group", "group", group.GroupKey, "error", err)
	}

	return &group, nil
}

// ResolveGroup marks an alert group as resolved.
func (e *AlertEngine) ResolveGroup(ruleID, groupKey string) error {
	if e.db == nil {
		return nil
	}
	now := time.Now()
	return e.db.Model(&model.AlertGroup{}).
		Where("group_key = ? AND rule_id = ? AND status = ?", groupKey, ruleID, "firing").
		Updates(map[string]interface{}{
			"status":     "resolved",
			"resolved_at": now,
			"updated_at": now,
		}).Error
}

// ListActiveGroups returns all currently firing alert groups.
func (e *AlertEngine) ListActiveGroups() ([]model.AlertGroup, error) {
	var groups []model.AlertGroup
	err := e.db.Where("status = ?", "firing").Order("last_alert_at DESC").Find(&groups).Error
	return groups, err
}

// ========== Background Worker ==========

// Start begins the background escalation checker.
func (e *AlertEngine) Start() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-e.stopCh:
				return
			case <-ticker.C:
				e.CheckEscalations(context.Background())
			}
		}
	}()
	e.logger.Info("alert engine started")
}

// Stop shuts down the alert engine.
func (e *AlertEngine) Stop() {
	close(e.stopCh)
}

// ========== Helpers ==========

func (e *AlertEngine) stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
