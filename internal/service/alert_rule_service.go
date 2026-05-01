package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/monitor"
	"gorm.io/gorm"
)

// AlertRuleService provides CRUD operations for alert rules.
type AlertRuleService struct {
	db *gorm.DB
}

// NewAlertRuleService creates a new AlertRuleService.
func NewAlertRuleService(db *gorm.DB) *AlertRuleService {
	return &AlertRuleService{db: db}
}

// CreateAlertRule creates a new alert rule in the database.
func (s *AlertRuleService) CreateAlertRule(rule *model.AlertRuleRecord) error {
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d", time.Now().UnixNano())
	}
	return s.db.Create(rule).Error
}

// UpdateAlertRule updates an existing alert rule.
func (s *AlertRuleService) UpdateAlertRule(rule *model.AlertRuleRecord) error {
	return s.db.Save(rule).Error
}

// DeleteAlertRule deletes an alert rule by ID.
func (s *AlertRuleService) DeleteAlertRule(id string) error {
	return s.db.Delete(&model.AlertRuleRecord{}, "id = ?", id).Error
}

// GetAlertRule retrieves a single alert rule by ID.
func (s *AlertRuleService) GetAlertRule(id string) (*model.AlertRuleRecord, error) {
	var rule model.AlertRuleRecord
	if err := s.db.Where("id = ?", id).First(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListAlertRules returns all alert rules, optionally filtered by server_id.
func (s *AlertRuleService) ListAlertRules(serverID string) ([]model.AlertRuleRecord, error) {
	var rules []model.AlertRuleRecord
	query := s.db.Order("created_at DESC")
	if serverID != "" {
		query = query.Where("server_id = ?", serverID)
	}
	if err := query.Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// LoadAlertRulesFromDB loads all enabled alert rules from the database and converts them
// to monitor.AlertRule format for the AlertManager.
func (s *AlertRuleService) LoadAlertRulesFromDB() []monitor.AlertRule {
	var records []model.AlertRuleRecord
	if err := s.db.Where("enabled = ?", true).Find(&records).Error; err != nil {
		slog.Error("failed to load alert rules from database", "error", err)
		return nil
	}

	var rules []monitor.AlertRule
	for _, r := range records {
		var channels []string
		if r.NotifyChannels != "" {
			_ = json.Unmarshal([]byte(r.NotifyChannels), &channels)
		}

		rule := monitor.AlertRule{
			ID:             r.ID,
			Name:           r.Name,
			MetricType:     monitor.MetricType(r.MetricType),
			Condition:      r.Condition,
			Threshold:      r.Threshold,
			Severity:       monitor.AlertSeverity(r.Severity),
			Enabled:        r.Enabled,
			Cooldown:       time.Duration(r.CooldownSeconds) * time.Second,
			NotifyChannels: channels,
			ServerID:       r.ServerID,
		}
		rules = append(rules, rule)
	}
	return rules
}

// SyncRulesToAlertManager loads rules from DB and replaces the AlertManager's rules.
// It preserves the built-in default rules and appends DB rules.
func (s *AlertRuleService) SyncRulesToAlertManager(am *monitor.AlertManager) {
	dbRules := s.LoadAlertRulesFromDB()

	// Start with default built-in rules
	defaults := monitor.DefaultRules()
	allRules := make([]monitor.AlertRule, len(defaults))
	copy(allRules, defaults)

	// Append DB rules (they may override defaults by ID)
	for _, rule := range dbRules {
		found := false
		for i, existing := range allRules {
			if existing.ID == rule.ID {
				allRules[i] = rule
				found = true
				break
			}
		}
		if !found {
			allRules = append(allRules, rule)
		}
	}

	// Replace all rules in the AlertManager
	am.ReplaceRules(allRules)
	slog.Info("alert rules synced from database", "total", len(allRules), "db_rules", len(dbRules))
}
