package service

import (
	"net"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/model"
)

// ===================== isPrivateIP Tests =====================

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ipStr    string
		expected bool
	}{
		// Private ranges
		{"10.0.0.0/8", "10.0.0.1", true},
		{"10.255.255.255", "10.255.255.255", true},
		{"10.1.2.3", "10.1.2.3", true},
		{"172.16.0.0/12", "172.16.0.1", true},
		{"172.31.255.255", "172.31.255.255", true},
		{"172.20.0.1", "172.20.0.1", true},
		{"192.168.0.0/16", "192.168.0.1", true},
		{"192.168.255.255", "192.168.255.255", true},
		{"192.168.1.100", "192.168.1.100", true},
		{"127.0.0.0/8", "127.0.0.1", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"127.255.255.255", "127.255.255.255", true},
		{"169.254.0.0/16", "169.254.0.1", true},
		{"169.254.169.254", "169.254.169.254", true},
		// IPv6 private ranges
		{"::1/128", "::1", true},
		{"fc00::/7", "fc00::1", true},
		{"fe80::/10", "fe80::1", true},
		// Public ranges
		{"8.8.8.8", "8.8.8.8", false},
		{"1.1.1.1", "1.1.1.1", false},
		{"8.8.4.4", "8.8.4.4", false},
		{"google.com", "142.250.80.46", false},
		{"Cloudflare 1.1.1.1", "1.1.1.1", false},
		// AWS metadata
		{"AWS EC2 metadata", "169.254.169.254", true}, // Should be blocked
		{"AWS metadata alternative", "169.254.169.253", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ipStr)
			if ip == nil {
				t.Skip("cannot parse IP address")
			}
			result := isPrivateIP(ip)
			if result != tc.expected {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tc.ipStr, result, tc.expected)
			}
		})
	}
}

func TestIsPrivateIP_InvalidIP(t *testing.T) {
	// Should not panic on nil IP
	ip := net.ParseIP("")
	if ip != nil {
		t.Error("expected nil IP for empty string")
	}
}

// ===================== validateWebhookURL Tests =====================

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		expectErr bool
		errMsg    string
	}{
		// Valid URLs
		{
			name:      "valid https URL",
			url:       "https://example.com/webhook",
			expectErr: false,
		},
		{
			name:      "valid http URL",
			url:       "http://example.com/webhook",
			expectErr: false,
		},
		{
			name:      "valid URL with port",
			url:       "https://example.com:8443/webhook",
			expectErr: false,
		},
		{
			name:      "valid URL with path",
			url:       "https://hooks.slack.com/services/abc/def/ghi",
			expectErr: false,
		},
		{
			name:      "valid IP address",
			url:       "https://142.250.80.46/webhook",
			expectErr: false,
		},
		{
			name:      "valid localhost",
			url:       "http://127.0.0.1:8080/webhook",
			expectErr: true, // Private IP after DNS resolution
			errMsg:    "private IP addresses are not allowed",
		},
		// Invalid URLs
		{
			name:      "invalid scheme",
			url:       "ftp://example.com/webhook",
			expectErr: true,
			errMsg:    "only http and https schemes are allowed",
		},
		{
			name:      "file scheme",
			url:       "file:///etc/passwd",
			expectErr: true,
			errMsg:    "only http and https schemes are allowed",
		},
		{
			name:      "empty host",
			url:       "http:///webhook",
			expectErr: true,
			errMsg:    "URL must have a host",
		},
		{
			name:      "AWS metadata IP",
			url:       "http://169.254.169.254/latest/meta-data/",
			expectErr: true,
			errMsg:    "metadata service access is not allowed",
		},
		{
			name:      "GCP metadata IP",
			url:       "http://metadata.google.internal/",
			expectErr: false, // DNS resolution fails, warning logged, URL allowed
		},
		{
			name:      "empty URL",
			url:       "",
			expectErr: true,
		},
		{
			name:      "invalid URL format",
			url:       "not-a-url",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWebhookURL(tc.url)
			if tc.expectErr {
				if err == nil {
					t.Errorf("expected error for URL %q, got nil", tc.url)
				} else if tc.errMsg != "" && err.Error() != tc.errMsg && !contains(err.Error(), tc.errMsg) {
					t.Errorf("error message = %q, want containing %q", err.Error(), tc.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for URL %q: %v", tc.url, err)
				}
			}
		})
	}
}

// ===================== matchesFilters Tests =====================

func TestMatchesFilters(t *testing.T) {
	svc := &OutboundWebhookService{}

	tests := []struct {
		name      string
		webhook   *model.OutboundWebhook
		event     BusEvent
		expectMatch bool
	}{
		{
			name: "no filters matches everything",
			webhook: &model.OutboundWebhook{
				EventTypes:     "",
				SeverityFilter: "",
				AppFilter:       "",
				ServerFilter:   "",
			},
			event: BusEvent{
				ID:      "evt-1",
				Type:    EventDeploy,
				Topic:   "deploy:app-1",
				Payload: map[string]interface{}{},
			},
			expectMatch: true,
		},
		{
			name: "event types filter matches",
			webhook: &model.OutboundWebhook{
				EventTypes: `["deploy","alert"]`,
			},
			event: BusEvent{
				ID:      "evt-2",
				Type:    EventDeploy,
				Topic:   "deploy:app-1",
				Payload: map[string]interface{}{},
			},
			expectMatch: true,
		},
		{
			name: "event types filter does not match",
			webhook:  &model.OutboundWebhook{
				EventTypes: `["alert"]`,
			},
			event: BusEvent{
				ID:      "evt-3",
				Type:    EventDeploy,
				Topic:   "deploy:app-1",
				Payload: map[string]interface{}{},
			},
			expectMatch: false,
		},
		{
			name: "severity filter matches",
			webhook:  &model.OutboundWebhook{
				SeverityFilter: `["critical","high"]`,
			},
			event: BusEvent{
				ID:      "evt-4",
				Type:    EventAlert,
				Topic:   "alert:server-1",
				Payload: map[string]interface{}{"severity": "critical"},
			},
			expectMatch: true,
		},
		{
			name: "severity filter does not match",
			webhook:  &model.OutboundWebhook{
				SeverityFilter: `["critical","high"]`,
			},
			event: BusEvent{
				ID:      "evt-5",
				Type:    EventAlert,
				Topic:   "alert:server-1",
				Payload: map[string]interface{}{"severity": "low"},
			},
			expectMatch: false,
		},
		{
			name: "severity filter missing in payload",
			webhook:  &model.OutboundWebhook{
				SeverityFilter: `["critical"]`,
			},
			event: BusEvent{
				ID:      "evt-6",
				Type:    EventAlert,
				Topic:   "alert:server-1",
				Payload: map[string]interface{}{"message": "test"},
			},
			expectMatch: false,
		},
		{
			name: "app filter matches",
			webhook:  &model.OutboundWebhook{
				AppFilter: `["myapp","yourapp"]`,
			},
			event: BusEvent{
				ID:      "evt-7",
				Type:    EventDeploy,
				Topic:   "deploy:myapp",
				Payload: map[string]interface{}{"app_name": "myapp"},
			},
			expectMatch: true,
		},
		{
			name: "app filter does not match",
			webhook:  &model.OutboundWebhook{
				AppFilter: `["myapp","yourapp"]`,
			},
			event: BusEvent{
				ID:      "evt-8",
				Type:    EventDeploy,
				Topic:   "deploy:otherapp",
				Payload: map[string]interface{}{"app_name": "otherapp"},
			},
			expectMatch: false,
		},
		{
			name: "server filter matches",
			webhook:  &model.OutboundWebhook{
				ServerFilter: `["server-1","server-2"]`,
			},
			event: BusEvent{
				ID:      "evt-9",
				Type:    EventServer,
				Topic:   "server:up",
				Payload: map[string]interface{}{"server_name": "server-1"},
			},
			expectMatch: true,
		},
		{
			name: "server filter does not match",
			webhook:  &model.OutboundWebhook{
				ServerFilter: `["server-1","server-2"]`,
			},
			event: BusEvent{
				ID:      "evt-10",
				Type:    EventServer,
				Topic:   "server:up",
				Payload: map[string]interface{}{"server_name": "server-3"},
			},
			expectMatch: false,
		},
		{
			name: "combined filters all match",
			webhook:  &model.OutboundWebhook{
				EventTypes:     `["deploy"]`,
				SeverityFilter: `["success"]`,
				AppFilter:      `["myapp"]`,
			},
			event: BusEvent{
				ID:      "evt-11",
				Type:    EventDeploy,
				Topic:   "deploy:myapp",
				Payload: map[string]interface{}{
					"app_name": "myapp",
					"status":   "success",
					"severity": "success",
				},
			},
			expectMatch: true,
		},
		{
			name: "combined filters one fails",
			webhook:  &model.OutboundWebhook{
				EventTypes: `["deploy"]`,
				AppFilter:  `["myapp"]`,
			},
			event: BusEvent{
				ID:      "evt-12",
				Type:    EventDeploy,
				Topic:   "deploy:otherapp",
				Payload: map[string]interface{}{
					"app_name": "otherapp",
				},
			},
			expectMatch: false,
		},
		{
			name: "invalid JSON in event types",
			webhook:  &model.OutboundWebhook{
				EventTypes: `not-json`,
			},
			event: BusEvent{
				ID:      "evt-13",
				Type:    EventDeploy,
				Topic:   "deploy:app-1",
				Payload: map[string]interface{}{},
			},
			expectMatch: true, // Invalid JSON means no filter applied
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := svc.matchesFilters(tc.webhook, tc.event)
			if result != tc.expectMatch {
				t.Errorf("matchesFilters() = %v, want %v", result, tc.expectMatch)
			}
		})
	}
}

// ===================== OutboundWebhookService CRUD Tests =====================

func TestNewOutboundWebhookService(t *testing.T) {
	// Just ensure it doesn't panic
	svc := NewOutboundWebhookService(nil, nil)
	if svc == nil {
		t.Error("NewOutboundWebhookService returned nil")
	}
}

// Helper

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
