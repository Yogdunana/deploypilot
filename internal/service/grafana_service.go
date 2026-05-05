package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service/dashboards"
	"gorm.io/gorm"
)

// builtInDashboardList holds the definitions of all built-in dashboards.
var builtInDashboardList = []struct {
	name string
	fn   func() map[string]interface{}
}{
	{"DeployPilot - Uptime Monitor", dashboards.UptimeOverviewJSON},
	{"DeployPilot - Server Resources", dashboards.ServerResourcesJSON},
	{"DeployPilot - Deploy Statistics", dashboards.DeployStatsJSON},
	{"DeployPilot - Alert & Heartbeat", dashboards.AlertHeartbeatJSON},
}

// GrafanaService orchestrates Grafana integration: datasource provisioning,
// dashboard sync, and event annotation forwarding.
type GrafanaService struct {
	db     *gorm.DB
	cfg    *config.GrafanaConfig
	bus    TypedEventBus
	client *GrafanaClient
	dsUID  string // cached datasource UID
	mu     sync.RWMutex
}

// NewGrafanaService creates a GrafanaService from the given configuration.
// If bus is nil, annotation listening is disabled.
func NewGrafanaService(db *gorm.DB, cfg *config.GrafanaConfig, bus TypedEventBus) *GrafanaService {
	client, err := NewGrafanaClient(cfg.URL, cfg.APIKey, cfg.AdminUser, cfg.AdminPassword)
	if err != nil {
		slog.Warn("failed to create Grafana client", "error", err)
	}
	return &GrafanaService{
		db:     db,
		cfg:    cfg,
		bus:    bus,
		client: client,
	}
}

// GetStatus returns the current Grafana integration status.
func (s *GrafanaService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"enabled":  s.cfg.Enabled,
		"url":      s.cfg.URL,
		"connected": false,
	}

	if s.dsUID != "" {
		status["connected"] = true
		status["datasource_uid"] = s.dsUID
	}

	return status
}

// TestConnection verifies connectivity to the Grafana server.
func (s *GrafanaService) TestConnection() (map[string]interface{}, error) {
	if s.client == nil {
		return nil, fmt.Errorf("grafana client not initialized")
	}
	return s.client.TestConnection()
}

// SyncAll performs a full sync: datasource, folder, and all enabled dashboards.
// It returns the count of synced dashboards.
func (s *GrafanaService) SyncAll() (int, error) {
	if s.client == nil {
		return 0, fmt.Errorf("grafana client not initialized")
	}

	// 1. Ensure datasource
	metricsURL := s.cfg.PrometheusURL
	if envURL := os.Getenv("DEPLOYPILOT_PROMETHEUS_URL"); envURL != "" {
		metricsURL = envURL
	}
	if metricsURL == "" {
		metricsURL = "http://localhost:9090"
	}
	dsUID, err := s.client.EnsureDatasource(metricsURL)
	if err != nil {
		return 0, fmt.Errorf("ensure datasource: %w", err)
	}
	s.mu.Lock()
	s.dsUID = dsUID
	s.mu.Unlock()

	// 2. Ensure folder
	folderID, err := s.client.EnsureFolder()
	if err != nil {
		return 0, fmt.Errorf("ensure folder: %w", err)
	}

	// 3. Seed built-in dashboards into DB if not present
	s.seedBuiltInDashboards()

	// 4. Sync all enabled dashboards from DB
	var dashboards []model.GrafanaCustomDashboard
	if err := s.db.Where("enabled = ?", true).Find(&dashboards).Error; err != nil {
		return 0, fmt.Errorf("list dashboards: %w", err)
	}

	count := 0
	for _, d := range dashboards {
		var dashboardJSON map[string]interface{}
		if err := json.Unmarshal([]byte(d.JSON), &dashboardJSON); err != nil {
			slog.Warn("skip invalid dashboard JSON", "id", d.ID, "name", d.Name, "error", err)
			continue
		}

		// Inject the actual datasource UID
		s.injectDatasourceUID(dashboardJSON, dsUID)

		_, err := s.client.UpsertDashboard(dashboardJSON, folderID, true)
		if err != nil {
			slog.Warn("failed to sync dashboard", "name", d.Name, "error", err)
			continue
		}

		// Record sync log
		s.recordSyncLog("sync_dashboard", "success", d.Name)
		count++
	}

	return count, nil
}

// injectDatasourceUID recursively replaces "${datasource}" placeholder references
// in the dashboard JSON with the actual datasource UID.
func (s *GrafanaService) injectDatasourceUID(dashboardJSON map[string]interface{}, dsUID string) {
	for key, val := range dashboardJSON {
		switch v := val.(type) {
		case string:
			if v == "${datasource}" && (key == "datasource" || key == "uid" || key == "query") {
				dashboardJSON[key] = dsUID
			}
		case map[string]interface{}:
			// Check for datasource objects like {"type": "prometheus", "uid": "${datasource}"}
			if uid, ok := v["uid"].(string); ok && uid == "${datasource}" {
				v["uid"] = dsUID
			}
			// Recurse into nested maps
			s.injectDatasourceUID(v, dsUID)
		case []interface{}:
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					s.injectDatasourceUID(m, dsUID)
				}
			}
		}
	}
}

// ListDashboards returns all dashboards from the database.
// Built-in dashboards are seeded on first call if not present.
func (s *GrafanaService) ListDashboards() ([]model.GrafanaCustomDashboard, error) {
	s.seedBuiltInDashboards()

	var dashboards []model.GrafanaCustomDashboard
	if err := s.db.Order("created_at ASC").Find(&dashboards).Error; err != nil {
		return nil, fmt.Errorf("list dashboards: %w", err)
	}
	return dashboards, nil
}

// GetDashboard returns a single dashboard by ID.
func (s *GrafanaService) GetDashboard(id string) (*model.GrafanaCustomDashboard, error) {
	var dashboard model.GrafanaCustomDashboard
	if err := s.db.Where("id = ?", id).First(&dashboard).Error; err != nil {
		return nil, fmt.Errorf("dashboard not found: %w", err)
	}
	return &dashboard, nil
}

// CreateCustomDashboard validates the JSON and creates a new dashboard in the database.
func (s *GrafanaService) CreateCustomDashboard(name, jsonStr, tags string) (*model.GrafanaCustomDashboard, error) {
	// Validate JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("invalid dashboard JSON: %w", err)
	}

	// Generate UID from title if present
	uid := ""
	if title, ok := parsed["uid"].(string); ok && title != "" {
		uid = title
	}
	if uid == "" {
		uid = fmt.Sprintf("custom-%d", time.Now().UnixNano())
	}

	dashboard := model.GrafanaCustomDashboard{
		Name:      name,
		UID:       uid,
		JSON:      jsonStr,
		Tags:      tags,
		IsBuiltIn: false,
		Enabled:   true,
	}

	if err := s.db.Create(&dashboard).Error; err != nil {
		return nil, fmt.Errorf("create dashboard: %w", err)
	}

	return &dashboard, nil
}

// UpdateCustomDashboard updates an existing dashboard in the database.
func (s *GrafanaService) UpdateCustomDashboard(id, name, jsonStr, tags string, enabled bool) error {
	var dashboard model.GrafanaCustomDashboard
	if err := s.db.Where("id = ?", id).First(&dashboard).Error; err != nil {
		return fmt.Errorf("dashboard not found: %w", err)
	}

	// If JSON is provided, validate it
	if jsonStr != "" {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			return fmt.Errorf("invalid dashboard JSON: %w", err)
		}
		dashboard.JSON = jsonStr
	}

	if name != "" {
		dashboard.Name = name
	}
	if tags != "" {
		dashboard.Tags = tags
	}
	dashboard.Enabled = enabled

	if err := s.db.Save(&dashboard).Error; err != nil {
		return fmt.Errorf("update dashboard: %w", err)
	}

	return nil
}

// DeleteCustomDashboard deletes a dashboard from the database.
func (s *GrafanaService) DeleteCustomDashboard(id string) error {
	if err := s.db.Where("id = ?", id).Delete(&model.GrafanaCustomDashboard{}).Error; err != nil {
		return fmt.Errorf("delete dashboard: %w", err)
	}
	return nil
}

// ExportAll returns all dashboards JSON along with datasource configuration.
func (s *GrafanaService) ExportAll() (map[string]interface{}, error) {
	s.seedBuiltInDashboards()

	var dashboards []model.GrafanaCustomDashboard
	if err := s.db.Find(&dashboards).Error; err != nil {
		return nil, fmt.Errorf("list dashboards for export: %w", err)
	}

	exported := make([]map[string]interface{}, 0, len(dashboards))
	for _, d := range dashboards {
		var dashboardJSON map[string]interface{}
		if err := json.Unmarshal([]byte(d.JSON), &dashboardJSON); err != nil {
			slog.Warn("skip invalid dashboard JSON on export", "id", d.ID, "error", err)
			continue
		}
		exported = append(exported, map[string]interface{}{
			"name":       d.Name,
			"uid":        d.UID,
			"is_built_in": d.IsBuiltIn,
			"enabled":    d.Enabled,
			"tags":       d.Tags,
			"dashboard":  dashboardJSON,
		})
	}

	s.mu.RLock()
	dsUID := s.dsUID
	s.mu.RUnlock()

	result := map[string]interface{}{
		"datasource_uid": dsUID,
		"grafana_url":    s.cfg.URL,
		"dashboards":     exported,
		"exported_at":    time.Now().UTC().Format(time.RFC3339),
	}

	return result, nil
}

// PushAnnotation converts a BusEvent to a Grafana annotation and pushes it.
func (s *GrafanaService) PushAnnotation(event BusEvent) {
	if s.client == nil || !s.cfg.AnnotationsEnabled {
		return
	}

	timestamp := event.Timestamp.UnixMilli()
	var tags []string
	var text string

	switch event.Type {
	case EventDeploy:
		p := s.parseDeployPayload(event.Payload)
		tags = []string{"deploy", p.Status, p.AppName}
		text = fmt.Sprintf("Deploy %s to %s: %s (%dms)", p.AppName, p.ServerName, p.Status, p.Duration)

	case EventAlert:
		p := s.parseAlertPayload(event.Payload)
		tags = []string{"alert", p.Severity, p.RuleName}
		text = fmt.Sprintf("Alert %s: %s - %s", p.RuleName, p.Severity, p.Message)

	case EventServer:
		p := s.parseServerPayload(event.Payload)
		tags = []string{"server", p.Action}
		text = fmt.Sprintf("Server %s: %s", p.ServerName, p.Action)

	case EventSystem:
		tags = []string{"system"}
		text = fmt.Sprintf("System: %s", event.Topic)

	default:
		return
	}

	// Filter out empty tags
	filteredTags := make([]string, 0, len(tags))
	for _, t := range tags {
		if t != "" {
			filteredTags = append(filteredTags, t)
		}
	}

	if err := s.client.CreateAnnotation(filteredTags, text, timestamp, timestamp); err != nil {
		slog.Warn("failed to push Grafana annotation", "type", event.Type, "error", err)
	}
}

// StartAnnotationListener subscribes to deploy, alert, server, and system event
// types and forwards them as Grafana annotations.
func (s *GrafanaService) StartAnnotationListener(ctx context.Context) {
	if s.bus == nil || !s.cfg.AnnotationsEnabled {
		return
	}

	eventTypes := []EventType{EventDeploy, EventAlert, EventServer, EventSystem}
	for _, et := range eventTypes {
		go func(eventType EventType) {
			ch := s.bus.SubscribeType(ctx, eventType)
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-ch:
					if !ok {
						return
					}
					s.PushAnnotation(event)
				}
			}
		}(et)
	}
	slog.Info("Grafana annotation listener started", "event_types", len(eventTypes))
}

// seedBuiltInDashboards checks if built-in dashboards exist in the database
// and inserts them if not present.
func (s *GrafanaService) seedBuiltInDashboards() {
	for _, def := range builtInDashboardList {
		var existing model.GrafanaCustomDashboard
		err := s.db.Where("name = ? AND is_built_in = ?", def.name, true).First(&existing).Error
		if err == nil {
			continue // already exists
		}
		if err != gorm.ErrRecordNotFound {
			slog.Warn("failed to check built-in dashboard", "name", def.name, "error", err)
			continue
		}

		// Insert built-in dashboard
		dashJSON := def.fn()
		jsonBytes, err := json.Marshal(dashJSON)
		if err != nil {
			slog.Warn("failed to marshal built-in dashboard", "name", def.name, "error", err)
			continue
		}

		uid := ""
		if u, ok := dashJSON["uid"].(string); ok {
			uid = u
		}

		dashboard := model.GrafanaCustomDashboard{
			Name:      def.name,
			UID:       uid,
			JSON:      string(jsonBytes),
			Tags:      `["deploypilot", "auto-provisioned"]`,
			IsBuiltIn: true,
			Enabled:   true,
		}

		if err := s.db.Create(&dashboard).Error; err != nil {
			// Ignore duplicate key errors (race condition)
			if !strings.Contains(err.Error(), "UNIQUE") && !strings.Contains(err.Error(), "duplicate") {
				slog.Warn("failed to seed built-in dashboard", "name", def.name, "error", err)
			}
		}
	}
}

// recordSyncLog creates a sync log entry in the database.
func (s *GrafanaService) recordSyncLog(action, status, message string) {
	log := model.GrafanaSyncLog{
		Action:  action,
		Status:  status,
		Message: message,
	}
	if err := s.db.Create(&log).Error; err != nil {
		slog.Warn("failed to record grafana sync log", "error", err)
	}
}

// --- Payload parsers ---

type deployAnnotationPayload struct {
	AppName    string `json:"app_name"`
	ServerName string `json:"server_name"`
	Status     string `json:"status"`
	Duration   int64  `json:"duration_ms"`
}

type alertAnnotationPayload struct {
	RuleName string `json:"rule_name"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type serverAnnotationPayload struct {
	ServerName string `json:"server_name"`
	Action     string `json:"action"`
}

func (s *GrafanaService) parseDeployPayload(payload interface{}) deployAnnotationPayload {
	p := deployAnnotationPayload{}
	data, err := json.Marshal(payload)
	if err != nil {
		return p
	}
	if err := json.Unmarshal(data, &p); err != nil {
		slog.Warn("failed to unmarshal deploy annotation payload", "error", err)
	}
	return p
}

func (s *GrafanaService) parseAlertPayload(payload interface{}) alertAnnotationPayload {
	p := alertAnnotationPayload{}
	data, err := json.Marshal(payload)
	if err != nil {
		return p
	}
	if err := json.Unmarshal(data, &p); err != nil {
		slog.Warn("failed to unmarshal alert annotation payload", "error", err)
	}
	return p
}

func (s *GrafanaService) parseServerPayload(payload interface{}) serverAnnotationPayload {
	p := serverAnnotationPayload{}
	data, err := json.Marshal(payload)
	if err != nil {
		return p
	}
	if err := json.Unmarshal(data, &p); err != nil {
		slog.Warn("failed to unmarshal server annotation payload", "error", err)
	}
	return p
}
