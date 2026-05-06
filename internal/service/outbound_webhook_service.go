package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/util"
)

var webhookSem = semaphore.NewWeighted(20) // max 20 concurrent webhook deliveries

const (
	webhookRetentionDays = 7
	maxRetryDelay        = 30 * time.Second
)

// OutboundWebhookService manages outbound webhook CRUD, delivery, retry, and event subscription.
type OutboundWebhookService struct {
	db     *gorm.DB
	bus    TypedEventBus
	ctx    context.Context
	cancel context.CancelFunc
}

// isPrivateIP checks if the given IP address is a private/reserved IP.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// validateWebhookURL checks if the URL is safe to call (SSRF protection).
func validateWebhookURL(webhookURL string) error {
	parsed, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only http and https schemes are allowed")
	}

	// Check host is not empty
	if parsed.Hostname() == "" {
		return fmt.Errorf("URL must have a host")
	}

	// Block metadata service
	if strings.Contains(parsed.Hostname(), "169.254.169.254") {
		return fmt.Errorf("metadata service access is not allowed")
	}

	// Resolve hostname and check IP
	ips, err := net.LookupIP(parsed.Hostname())
	if err != nil {
		// If we can't resolve, we'll allow it but log a warning
		slog.Warn("could not resolve webhook URL hostname", "hostname", parsed.Hostname())
		return nil
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("private IP addresses are not allowed")
		}
	}

	return nil
}

// NewOutboundWebhookService creates a new OutboundWebhookService.
func NewOutboundWebhookService(db *gorm.DB, bus TypedEventBus) *OutboundWebhookService {
	return &OutboundWebhookService{
		db:  db,
		bus: bus,
	}
}

// --- CRUD Methods ---

// Create generates a UUID and inserts the webhook into the database.
func (s *OutboundWebhookService) Create(ctx context.Context, webhook *model.OutboundWebhook) error {
	webhook.ID = uuid.New().String()
	return s.db.WithContext(ctx).Create(webhook).Error
}

// GetByID retrieves a webhook by its ID.
func (s *OutboundWebhookService) GetByID(ctx context.Context, id string) (*model.OutboundWebhook, error) {
	var webhook model.OutboundWebhook
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&webhook).Error; err != nil {
		return nil, err
	}
	return &webhook, nil
}

// List returns a paginated list of webhooks for a given tenant with total count.
func (s *OutboundWebhookService) List(ctx context.Context, tenantID string, page, pageSize int) ([]model.OutboundWebhook, int64, error) {
	var webhooks []model.OutboundWebhook
	var total int64

	query := s.db.WithContext(ctx).Model(&model.OutboundWebhook{})
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&webhooks).Error; err != nil {
		return nil, 0, err
	}

	return webhooks, total, nil
}

// Update updates an existing webhook.
func (s *OutboundWebhookService) Update(ctx context.Context, webhook *model.OutboundWebhook) error {
	return s.db.WithContext(ctx).Save(webhook).Error
}

// Delete removes a webhook by its ID.
func (s *OutboundWebhookService) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&model.OutboundWebhook{}).Error
}

// --- Delivery Methods ---

// Deliver sends a webhook payload to the configured endpoint.
func (s *OutboundWebhookService) Deliver(ctx context.Context, webhook *model.OutboundWebhook, event BusEvent) error {
	// Build WebhookPayload from BusEvent
	wp := WebhookPayload{
		EventID:   event.ID,
		EventType: string(event.Type),
		Topic:     event.Topic,
		Timestamp: event.Timestamp,
		Payload:   event.Payload,
	}

	// Get format adapter
	formatter := GetFormatter(webhook.Format)
	body, contentType := formatter.Format(wp)

	// Compute timestamp and signature
	timestamp := time.Now().Unix()
	signature := SignWebhook(webhook.Secret, timestamp, body)

	// Validate webhook URL (SSRF protection)
	if err := validateWebhookURL(webhook.URL); err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}

	// Create HTTP request
	timeout := time.Duration(webhook.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	httpCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, webhook.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("User-Agent", "DeployPilot-Webhook/1.0")

	// Send request
	startTime := time.Now()
	resp, err := util.DefaultClient.Do(req)
	latency := time.Since(startTime)

	// Read response body for error recording
	var respBody string
	var statusCode int
	if err != nil {
		statusCode = 0
		respBody = err.Error()
	} else {
		defer func() { _ = resp.Body.Close() }()
		statusCode = resp.StatusCode
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
		respBody = string(respBytes)
	}

	// Create delivery record
	delivery := &model.WebhookDelivery{
		ID:            uuid.New().String(),
		WebhookID:     webhook.ID,
		TenantID:      webhook.TenantID,
		EventID:       event.ID,
		EventType:     string(event.Type),
		StatusCode:    statusCode,
		LatencyMs:     int(latency.Milliseconds()),
		Attempt:       1,
		Success:       statusCode >= 200 && statusCode < 300,
		ErrorResponse: respBody,
		RequestBody:   string(body),
	}

	if err := s.db.WithContext(ctx).Create(delivery).Error; err != nil {
		slog.Error("failed to record webhook delivery", "webhook_id", webhook.ID, "error", err)
	}

	// Update webhook last delivery info
	now := time.Now()
	webhook.LastDeliveryAt = &now
	if delivery.Success {
		webhook.LastStatus = "success"
	} else {
		webhook.LastStatus = "failed"
	}
	if err := s.db.WithContext(ctx).Save(webhook).Error; err != nil {
		slog.Error("failed to save webhook delivery status", "webhook_id", webhook.ID, "error", err)
	}

	// Log result
	if delivery.Success {
		slog.Info("webhook delivered successfully",
			"webhook_id", webhook.ID,
			"event_id", event.ID,
			"status_code", statusCode,
			"latency_ms", latency.Milliseconds(),
		)
	} else {
		slog.Warn("webhook delivery failed",
			"webhook_id", webhook.ID,
			"event_id", event.ID,
			"status_code", statusCode,
			"error", respBody,
		)
		// Spawn retry goroutine on failure
		if webhook.MaxRetries > 0 {
			go s.retryDelivery(webhook, delivery)
		}
	}

	return nil
}

// retryDelivery implements exponential backoff retry for failed webhook deliveries.
func (s *OutboundWebhookService) retryDelivery(webhook *model.OutboundWebhook, delivery *model.WebhookDelivery) {
	attempt := delivery.Attempt

	for attempt < webhook.MaxRetries {
		// Exponential backoff: delay = min(2^attempt * time.Second, 30*time.Second)
		delay := 1 << uint(attempt)
		if delay > 30 {
			delay = 30
		}

		select {
		case <-s.ctx.Done():
			slog.Info("webhook retry cancelled: service shutting down",
				"webhook_id", webhook.ID,
				"event_id", delivery.EventID,
			)
			return
		case <-time.After(time.Duration(delay) * time.Second):
		}

		attempt++

		// Re-send the same request body
		timeout := time.Duration(webhook.Timeout) * time.Second
		if timeout == 0 {
			timeout = 10 * time.Second
		}

		ctx, cancel := context.WithTimeout(s.ctx, timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader([]byte(delivery.RequestBody)))
		if err != nil {
			slog.Error("retry: failed to create request", "webhook_id", webhook.ID, "attempt", attempt, "error", err)
			continue
		}

		timestamp := time.Now().Unix()
		signature := SignWebhook(webhook.Secret, timestamp, []byte(delivery.RequestBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Signature", signature)
		req.Header.Set("X-Webhook-Timestamp", strconv.FormatInt(timestamp, 10))
		req.Header.Set("User-Agent", "DeployPilot-Webhook/1.0")

		startTime := time.Now()
		resp, err := util.DefaultClient.Do(req)
		latency := time.Since(startTime)

		var respBody string
		var statusCode int
		if err != nil {
			statusCode = 0
			respBody = err.Error()
		} else {
			defer func() { _ = resp.Body.Close() }()
			statusCode = resp.StatusCode
			respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
			respBody = string(respBytes)
		}

		success := statusCode >= 200 && statusCode < 300

		// Update delivery record
		delivery.Attempt = attempt
		delivery.StatusCode = statusCode
		delivery.LatencyMs = int(latency.Milliseconds())
		delivery.Success = success
		delivery.ErrorResponse = respBody
		if err := s.db.Save(delivery).Error; err != nil {
			slog.Error("failed to save webhook delivery retry", "delivery_id", delivery.ID, "error", err)
		}

		// Update webhook last delivery info
		now := time.Now()
		webhook.LastDeliveryAt = &now
		if success {
			webhook.LastStatus = "success"
		} else {
			webhook.LastStatus = "failed"
		}
		if err := s.db.Save(webhook).Error; err != nil {
			slog.Error("failed to save webhook after retry", "webhook_id", webhook.ID, "error", err)
		}

		if success {
			slog.Info("webhook retry succeeded",
				"webhook_id", webhook.ID,
				"event_id", delivery.EventID,
				"attempt", attempt,
				"status_code", statusCode,
			)
			return
		}

		slog.Warn("webhook retry failed",
			"webhook_id", webhook.ID,
			"event_id", delivery.EventID,
			"attempt", attempt,
			"status_code", statusCode,
		)
	}

	slog.Error("webhook delivery exhausted all retries",
		"webhook_id", webhook.ID,
		"event_id", delivery.EventID,
		"max_retries", webhook.MaxRetries,
	)
}

// TestDelivery sends a synthetic test event to a webhook and returns the delivery record.
func (s *OutboundWebhookService) TestDelivery(ctx context.Context, webhookID string) (*model.WebhookDelivery, error) {
	webhook, err := s.GetByID(ctx, webhookID)
	if err != nil {
		return nil, fmt.Errorf("webhook not found: %w", err)
	}

	testEvent := BusEvent{
		ID:        fmt.Sprintf("test-%d", time.Now().UnixNano()),
		Type:      EventSystem,
		Topic:     "system:test",
		Payload:   map[string]interface{}{"message": "Test webhook delivery"},
		Timestamp: time.Now(),
	}

	// Build WebhookPayload
	wp := WebhookPayload{
		EventID:   testEvent.ID,
		EventType: string(testEvent.Type),
		Topic:     testEvent.Topic,
		Timestamp: testEvent.Timestamp,
		Payload:   testEvent.Payload,
	}

	formatter := GetFormatter(webhook.Format)
	body, contentType := formatter.Format(wp)

	timestamp := time.Now().Unix()
	signature := SignWebhook(webhook.Secret, timestamp, body)

	timeout := time.Duration(webhook.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	httpCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, webhook.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("User-Agent", "DeployPilot-Webhook/1.0")

	startTime := time.Now()
	resp, err := util.DefaultClient.Do(req)
	latency := time.Since(startTime)

	var respBody string
	var statusCode int
	if err != nil {
		statusCode = 0
		respBody = err.Error()
	} else {
		defer func() { _ = resp.Body.Close() }()
		statusCode = resp.StatusCode
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
		respBody = string(respBytes)
	}

	delivery := &model.WebhookDelivery{
		ID:            uuid.New().String(),
		WebhookID:     webhook.ID,
		TenantID:      webhook.TenantID,
		EventID:       testEvent.ID,
		EventType:     string(testEvent.Type),
		StatusCode:    statusCode,
		LatencyMs:     int(latency.Milliseconds()),
		Attempt:       1,
		Success:       statusCode >= 200 && statusCode < 300,
		ErrorResponse: respBody,
		RequestBody:   string(body),
	}

	if err := s.db.WithContext(ctx).Create(delivery).Error; err != nil {
		return nil, fmt.Errorf("failed to record test delivery: %w", err)
	}

	return delivery, nil
}

// --- Event Subscription ---

// Start subscribes to all event types and delivers matching webhooks.
func (s *OutboundWebhookService) Start() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	eventTypes := []EventType{
		EventDeploy, EventAlert, EventNotify, EventSystem,
		EventUser, EventServer, EventSecurity, EventAudit, EventBackup,
	}

	for _, et := range eventTypes {
		go s.listenEventType(et)
	}

	slog.Info("outbound webhook service started", "event_types", len(eventTypes))
}

// Stop cancels the service context and stops all event listeners.
func (s *OutboundWebhookService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// listenEventType subscribes to a specific event type and delivers to matching webhooks.
func (s *OutboundWebhookService) listenEventType(eventType EventType) {
	ch := s.bus.SubscribeType(s.ctx, eventType)
	for {
		select {
		case <-s.ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			s.handleEvent(event)
		}
	}
}

// handleEvent loads all enabled webhooks and delivers to those that match the event.
func (s *OutboundWebhookService) handleEvent(event BusEvent) {
	var webhooks []model.OutboundWebhook
	if err := s.db.WithContext(s.ctx).Where("enabled = ?", true).Find(&webhooks).Error; err != nil {
		slog.Error("failed to load webhooks for event", "event_id", event.ID, "error", err)
		return
	}

	for i := range webhooks {
		wh := &webhooks[i]
		if s.matchesFilters(wh, event) {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("panic recovered in webhook delivery", "webhook_id", wh.ID, "panic", r)
					}
				}()
				if err := webhookSem.Acquire(s.ctx, 1); err != nil {
					slog.Warn("webhook delivery skipped: context cancelled", "webhook_id", wh.ID)
					return
				}
				defer webhookSem.Release(1)
				_ = s.Deliver(s.ctx, wh, event)
			}()
		}
	}
}

// matchesFilters checks if an event matches a webhook's filter configuration.
func (s *OutboundWebhookService) matchesFilters(webhook *model.OutboundWebhook, event BusEvent) bool {
	// Check event types filter
	if webhook.EventTypes != "" {
		var types []string
		if err := json.Unmarshal([]byte(webhook.EventTypes), &types); err == nil && len(types) > 0 {
			found := false
			for _, t := range types {
				if t == string(event.Type) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Build payload map for field-based filters
	payloadMap, err := payloadToMap(event.Payload)
	if err != nil {
		slog.Warn("failed to parse event payload for filter matching", "event_id", event.ID, "error", err)
		return false
	}

	// Check severity filter
	if webhook.SeverityFilter != "" {
		var severities []string
		if err := json.Unmarshal([]byte(webhook.SeverityFilter), &severities); err == nil && len(severities) > 0 {
			severity, _ := payloadMap["severity"].(string)
			if severity == "" {
				return false
			}
			found := false
			for _, sev := range severities {
				if sev == severity {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check app filter
	if webhook.AppFilter != "" {
		var apps []string
		if err := json.Unmarshal([]byte(webhook.AppFilter), &apps); err == nil && len(apps) > 0 {
			appName, _ := payloadMap["app_name"].(string)
			if appName == "" {
				return false
			}
			found := false
			for _, app := range apps {
				if app == appName {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check server filter
	if webhook.ServerFilter != "" {
		var servers []string
		if err := json.Unmarshal([]byte(webhook.ServerFilter), &servers); err == nil && len(servers) > 0 {
			serverName, _ := payloadMap["server_name"].(string)
			if serverName == "" {
				return false
			}
			found := false
			for _, srv := range servers {
				if srv == serverName {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

// --- Cleanup ---

// StartCleanupLoop runs a periodic cleanup that deletes old webhook delivery records.
func (s *OutboundWebhookService) StartCleanupLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().AddDate(0, 0, -webhookRetentionDays)
				result := s.db.WithContext(ctx).
					Where("created_at < ?", cutoff).
					Delete(&model.WebhookDelivery{})
				if result.Error != nil {
					slog.Error("failed to cleanup old webhook deliveries", "error", result.Error)
				} else if result.RowsAffected > 0 {
					slog.Info("cleaned up old webhook deliveries", "deleted", result.RowsAffected)
				}
			}
		}
	}()

	slog.Info("webhook delivery cleanup loop started")
}
