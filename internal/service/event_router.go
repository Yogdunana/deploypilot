package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// --- Extended Event Types ---

const (
	EventUser     EventType = "user"     // login, logout, register, 2fa
	EventServer   EventType = "server"   // server up, down, health check
	EventSecurity EventType = "security" // brute force, suspicious activity
	EventAudit    EventType = "audit"    // config change, permission change
	EventBackup   EventType = "backup"   // backup start, success, fail
)

// --- Structured Event Payloads ---

// UserEventPayload is the payload for user-related events.
type UserEventPayload struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Action    string `json:"action"` // login, logout, register, password_change, 2fa_enabled
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
}

// DeployEventPayload is the payload for deploy-related events.
type DeployEventPayload struct {
	AppID      string `json:"app_id"`
	AppName    string `json:"app_name"`
	ServerID   string `json:"server_id,omitempty"`
	ServerName string `json:"server_name,omitempty"`
	Action     string `json:"action"` // start, success, fail, rollback
	Status     string `json:"status"`
	Duration   int64  `json:"duration_ms,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ServerEventPayload is the payload for server-related events.
type ServerEventPayload struct {
	ServerID   string  `json:"server_id"`
	ServerName string  `json:"server_name"`
	Action     string  `json:"action"` // up, down, health_check, cpu_high, memory_high, disk_high
	Metric     string  `json:"metric,omitempty"`
	Value      float64 `json:"value,omitempty"`
	Threshold  float64 `json:"threshold,omitempty"`
	Message    string  `json:"message,omitempty"`
}

// SecurityEventPayload is the payload for security-related events.
type SecurityEventPayload struct {
	Action    string `json:"action"` // brute_force, suspicious_login, permission_denied, ip_blocked
	IPAddress string `json:"ip_address"`
	Username  string `json:"username,omitempty"`
	Detail    string `json:"detail,omitempty"`
	Severity  string `json:"severity"` // low, medium, high, critical
}

// BackupEventPayload is the payload for backup-related events.
type BackupEventPayload struct {
	AppID    string `json:"app_id,omitempty"`
	AppName  string `json:"app_name,omitempty"`
	Action   string `json:"action"` // start, success, fail
	Location string `json:"location,omitempty"` // local, s3, oss
	Size     int64  `json:"size_bytes,omitempty"`
	Message  string `json:"message,omitempty"`
}

// --- Event Router ---

// EventRouteRule defines a routing rule for events.
// Events matching the rule's conditions are forwarded to specified channels.
type EventRouteRule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Enabled     bool      `json:"enabled"`
	EventType   EventType `json:"event_type"`            // match event type
	TopicPrefix string    `json:"topic_prefix,omitempty"` // match topic prefix (e.g. "alert:")
	Severity    string    `json:"severity,omitempty"`     // match severity in payload (low/medium/high/critical)
	Channels    []string  `json:"channels"`              // target notification channels
	Conditions  string    `json:"conditions,omitempty"`   // JSON: additional match conditions
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EventRouter routes events based on configured rules.
// It subscribes to the TypedEventBus and forwards matching events to notification channels.
type EventRouter struct {
	bus        TypedEventBus
	notifySvc  *NotificationService
	rules      []EventRouteRule
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	logger     *slog.Logger
}

// NewEventRouter creates a new EventRouter that subscribes to all event types.
func NewEventRouter(bus TypedEventBus, notifySvc *NotificationService) *EventRouter {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventRouter{
		bus:       bus,
		notifySvc: notifySvc,
		rules:     make([]EventRouteRule, 0),
		ctx:       ctx,
		cancel:    cancel,
		logger:    slog.Default(),
	}
}

// Start begins listening for events and routing them based on rules.
func (r *EventRouter) Start() {
	eventTypes := []EventType{
		EventDeploy, EventAlert, EventNotify, EventSystem,
		EventUser, EventServer, EventSecurity, EventAudit, EventBackup,
	}
	for _, et := range eventTypes {
		go r.listenType(et)
	}
	r.logger.Info("event router started", "types", len(eventTypes))
}

// Stop shuts down the event router.
func (r *EventRouter) Stop() {
	r.cancel()
}

// SetRules replaces the current routing rules.
func (r *EventRouter) SetRules(rules []EventRouteRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = rules
}

// GetRules returns the current routing rules.
func (r *EventRouter) GetRules() []EventRouteRule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]EventRouteRule, len(r.rules))
	copy(out, r.rules)
	return out
}

// AddRule adds a routing rule.
func (r *EventRouter) AddRule(rule EventRouteRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = append(r.rules, rule)
}

// RemoveRule removes a routing rule by ID.
func (r *EventRouter) RemoveRule(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, rule := range r.rules {
		if rule.ID == id {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			return
		}
	}
}

// listenType subscribes to a specific event type and routes matching events.
func (r *EventRouter) listenType(eventType EventType) {
	ch := r.bus.SubscribeType(r.ctx, eventType)
	for {
		select {
		case <-r.ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			r.routeEvent(event)
		}
	}
}

// routeEvent checks all rules against the event and forwards to matching channels.
func (r *EventRouter) routeEvent(event BusEvent) {
	r.mu.RLock()
	rules := make([]EventRouteRule, len(r.rules))
	copy(rules, r.rules)
	r.mu.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if r.matchRule(rule, event) {
			r.forwardToChannels(event, rule.Channels)
		}
	}
}

// matchRule checks if an event matches a routing rule.
func (r *EventRouter) matchRule(rule EventRouteRule, event BusEvent) bool {
	// Match event type
	if rule.EventType != "" && rule.EventType != event.Type {
		return false
	}

	// Match topic prefix
	if rule.TopicPrefix != "" && !strings.HasPrefix(event.Topic, rule.TopicPrefix) {
		return false
	}

	// Match severity from payload (if rule specifies severity)
	if rule.Severity != "" {
		severity := extractSeverity(event.Payload)
		if severity != "" && severity != rule.Severity {
			return false
		}
	}

	// Match additional conditions
	if rule.Conditions != "" {
		if !matchConditions(rule.Conditions, event.Payload) {
			return false
		}
	}

	return true
}

// forwardToChannels sends a notification for the event to the specified channels.
func (r *EventRouter) forwardToChannels(event BusEvent, channels []string) {
	if r.notifySvc == nil || len(channels) == 0 {
		return
	}

	payloadMap, err := payloadToMap(event.Payload)
	if err != nil {
		r.logger.Error("failed to marshal event payload", "event_id", event.ID, "error", err)
		return
	}

	message := fmt.Sprintf("[%s] %s", event.Type, event.Topic)
	if msg, ok := payloadMap["message"].(string); ok && msg != "" {
		message = msg
	}

	// Determine notification type and status from event
	notifType := "system"
	status := "info"
	switch event.Type {
	case EventAlert:
		notifType = "health_check"
		if sev, ok := payloadMap["severity"].(string); ok {
			status = sev
		}
	case EventDeploy:
		notifType = "deploy_success"
		if s, ok := payloadMap["status"].(string); ok {
			status = s
		}
	case EventSecurity:
		notifType = "security"
		status = "warning"
	case EventUser:
		notifType = "system"
	}

	appName := ""
	if name, ok := payloadMap["app_name"].(string); ok {
		appName = name
	}
	if name, ok := payloadMap["server_name"].(string); ok && appName == "" {
		appName = name
	}

	// Convert channels to comma-separated string for notification service
	channelsStr := strings.Join(channels, ",")

	// Send via notification service
	go func() {
		_, err := r.notifySvc.SendToChannels(r.ctx, notifType, appName, "", status, message, channelsStr)
		if err != nil {
			r.logger.Error("failed to route event notification",
				"event_id", event.ID, "channels", channelsStr, "error", err)
		}
	}()
}

// extractSeverity attempts to extract a severity field from a payload.
func extractSeverity(payload interface{}) string {
	m, ok := payload.(map[string]interface{})
	if !ok {
		return ""
	}
	if sev, ok := m["severity"].(string); ok {
		return sev
	}
	return ""
}

// matchConditions checks additional JSON conditions against a payload.
func matchConditions(conditionsJSON string, payload interface{}) bool {
	var conditions map[string]interface{}
	if err := json.Unmarshal([]byte(conditionsJSON), &conditions); err != nil {
		return false
	}

	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		// Try to marshal/unmarshal
		data, err := json.Marshal(payload)
		if err != nil {
			return false
		}
		if err := json.Unmarshal(data, &payloadMap); err != nil {
			return false
		}
	}

	for key, expected := range conditions {
		actual, ok := payloadMap[key]
		if !ok {
			return false
		}
		// Simple equality check
		if fmt.Sprintf("%v", expected) != fmt.Sprintf("%v", actual) {
			return false
		}
	}
	return true
}

// payloadToMap converts a payload interface to a map.
func payloadToMap(payload interface{}) (map[string]interface{}, error) {
	if m, ok := payload.(map[string]interface{}); ok {
		return m, nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
