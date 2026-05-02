package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ===================== SignWebhook Tests =====================

func TestSignWebhook(t *testing.T) {
	secret := "test-secret"
	timestamp := int64(1715000000)
	body := []byte(`{"test": "data"}`)
	sig := SignWebhook(secret, timestamp, body)
	if !strings.HasPrefix(sig, "sha256=") {
		t.Errorf("signature should start with sha256=, got %s", sig)
	}
	if len(sig) != 71 {
		t.Errorf("signature should be 71 chars, got %d", len(sig))
	}
}

func TestSignWebhook_Consistency(t *testing.T) {
	secret := "test-secret"
	timestamp := int64(1715000000)
	body := []byte(`{"test": "data"}`)
	sig1 := SignWebhook(secret, timestamp, body)
	sig2 := SignWebhook(secret, timestamp, body)
	if sig1 != sig2 {
		t.Errorf("signatures should be identical, got %s vs %s", sig1, sig2)
	}
}

func TestSignWebhook_DifferentSecrets(t *testing.T) {
	timestamp := int64(1715000000)
	body := []byte(`{"test": "data"}`)
	sig1 := SignWebhook("secret-a", timestamp, body)
	sig2 := SignWebhook("secret-b", timestamp, body)
	if sig1 == sig2 {
		t.Errorf("signatures with different secrets should differ, both got %s", sig1)
	}
}

// ===================== Formatter Tests =====================

func TestJSONFormatter(t *testing.T) {
	payload := WebhookPayload{
		EventID:   "test-123",
		EventType: "deploy",
		Topic:     "deploy:app-1",
		Timestamp: time.Now(),
		Payload:   map[string]string{"message": "test"},
	}
	f := JSONFormatter{}
	body, ct := f.Format(payload)
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["event_id"] != "test-123" {
		t.Errorf("expected event_id test-123, got %v", result["event_id"])
	}
	if result["event_type"] != "deploy" {
		t.Errorf("expected event_type deploy, got %v", result["event_type"])
	}
	if _, ok := result["timestamp"]; !ok {
		t.Errorf("expected timestamp field to be present")
	}
}

func TestSlackFormatter(t *testing.T) {
	payload := WebhookPayload{
		EventID:   "evt-slack-001",
		EventType: "deploy",
		Topic:     "deploy:app-1",
		Timestamp: time.Now(),
		Payload:   map[string]string{"status": "success", "message": "deployed OK"},
	}
	f := SlackFormatter{}
	body, ct := f.Format(payload)
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
	if !strings.Contains(string(body), `"attachments"`) {
		t.Errorf("expected output to contain 'attachments', got %s", string(body))
	}
	// Verify it is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := result["attachments"]; !ok {
		t.Error("expected 'attachments' key in parsed JSON")
	}
}

func TestDiscordFormatter(t *testing.T) {
	payload := WebhookPayload{
		EventID:   "evt-discord-001",
		EventType: "alert",
		Topic:     "alert:server-1",
		Timestamp: time.Now(),
		Payload:   map[string]string{"status": "error", "message": "CPU high"},
	}
	f := DiscordFormatter{}
	body, ct := f.Format(payload)
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
	if !strings.Contains(string(body), `"embeds"`) {
		t.Errorf("expected output to contain 'embeds', got %s", string(body))
	}
	// Verify it is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := result["embeds"]; !ok {
		t.Error("expected 'embeds' key in parsed JSON")
	}
}

func TestTeamsFormatter(t *testing.T) {
	payload := WebhookPayload{
		EventID:   "evt-teams-001",
		EventType: "deploy",
		Topic:     "deploy:app-2",
		Timestamp: time.Now(),
		Payload:   map[string]string{"status": "success", "message": "deployed OK"},
	}
	f := TeamsFormatter{}
	body, ct := f.Format(payload)
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
	if !strings.Contains(string(body), `"AdaptiveCard"`) {
		t.Errorf("expected output to contain 'AdaptiveCard', got %s", string(body))
	}
	// Verify it is valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

// ===================== GetFormatter Tests =====================

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		format     string
		expectName string
	}{
		{"json", "json"},
		{"slack", "slack"},
		{"discord", "discord"},
		{"teams", "teams"},
		{"JSON", "json"},    // case-insensitive
		{"Slack", "slack"},  // case-insensitive
		{"unknown", "json"}, // default fallback
		{"", "json"},        // empty string fallback
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			f := GetFormatter(tc.format)
			if f == nil {
				t.Fatalf("GetFormatter(%q) returned nil", tc.format)
			}
			if f.Name() != tc.expectName {
				t.Errorf("GetFormatter(%q): expected name %q, got %q", tc.format, tc.expectName, f.Name())
			}
		})
	}
}
