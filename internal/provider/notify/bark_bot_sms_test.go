package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ========== Bark Notifier Tests ==========

func TestBarkNotifierName(t *testing.T) {
	b := NewBarkNotifier("", "test-key")
	if b.Name() != "bark" {
		t.Errorf("Name() = %q, want %q", b.Name(), "bark")
	}
}

func TestBarkNotifierSendSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		var req barkRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Title == "" {
			t.Error("Title should not be empty")
		}
		if req.Body == "" {
			t.Error("Body should not be empty")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(barkResponse{
			Code:      200,
			Message:   "success",
			Timestamp: time.Now().Unix(),
		})
	}))
	defer server.Close()

	b := NewBarkNotifier(server.URL, "test-key")
	result, err := b.Send(context.Background(), DeploySuccess("my-app", "server", "img:latest"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}
}

func TestBarkNotifierSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(barkResponse{
			Code:    400,
			Message: "invalid device key",
		})
	}))
	defer server.Close()

	b := NewBarkNotifier(server.URL, "invalid-key")
	result, err := b.Send(context.Background(), DeploySuccess("app", "server", "img"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.Success {
		t.Error("Success should be false for error code 400")
	}
}

func TestBarkNotifierMapLevel(t *testing.T) {
	b := NewBarkNotifier("", "test-key")
	tests := []struct {
		status string
		want   string
	}{
		{"failed", "timeSensitive"},
		{"critical", "timeSensitive"},
		{"warning", "active"},
		{"success", "passive"},
		{"info", "passive"},
		{"unknown", "passive"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := b.mapLevel(tt.status); got != tt.want {
				t.Errorf("mapLevel(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

// ========== Bot Notifier Tests ==========

func TestBotNotifierName(t *testing.T) {
	mock := &mockNotifier{name: "test"}
	bot := NewBotNotifier(mock, BotModeAll)
	if bot.Name() != "test(bot:all)" {
		t.Errorf("Name() = %q, want test(bot:all)", bot.Name())
	}
}

func TestBotNotifierAllMode(t *testing.T) {
	mock := &mockNotifier{name: "test"}
	bot := NewBotNotifier(mock, BotModeAll)
	n := DeploySuccess("app", "server", "img")
	result, err := bot.Send(context.Background(), n)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !mock.sent {
		t.Error("All mode should send all notifications")
	}
	if !result.Success {
		t.Error("Result should be success")
	}
}

func TestBotNotifierQuietMode(t *testing.T) {
	mock := &mockNotifier{name: "test"}
	bot := NewBotNotifier(mock, BotModeQuiet)
	n := DeploySuccess("app", "server", "img")
	result, err := bot.Send(context.Background(), n)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if mock.sent {
		t.Error("Quiet mode should not send notifications")
	}
	if !result.Success {
		t.Error("Result should be success (suppressed)")
	}
}

func TestBotNotifierErrorOnlyMode(t *testing.T) {
	mock := &mockNotifier{name: "test"}
	bot := NewBotNotifier(mock, BotModeErrorOnly)

	// Success should be suppressed
	successN := DeploySuccess("app", "server", "img")
	_, err := bot.Send(context.Background(), successN)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if mock.sent {
		t.Error("Error-only mode should suppress success notifications")
	}
	mock.sent = false

	// Failed should be sent
	failedN := DeployFailed("app", "server", "error")
	_, err = bot.Send(context.Background(), failedN)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !mock.sent {
		t.Error("Error-only mode should send failed notifications")
	}
}

func TestBotNotifierDigestMode(t *testing.T) {
	mock := &mockNotifier{name: "test"}
	bot := NewBotNotifier(mock, BotModeDigest)

	// Add some notifications
	n1 := DeploySuccess("app1", "srv1", "img1")
	n2 := DeployFailed("app2", "srv2", "err2")

	_, err := bot.Send(context.Background(), n1)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	_, err = bot.Send(context.Background(), n2)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if bot.BufferedCount() != 2 {
		t.Errorf("BufferedCount() = %d, want 2", bot.BufferedCount())
	}
	if mock.sent {
		t.Error("Digest mode should buffer notifications, not send immediately")
	}

	// Flush digest
	_, err = bot.FlushDigest(context.Background())
	if err != nil {
		t.Fatalf("FlushDigest() error = %v", err)
	}
	if !mock.sent {
		t.Error("FlushDigest() should send the digest")
	}
	if bot.BufferedCount() != 0 {
		t.Errorf("BufferedCount() after flush = %d, want 0", bot.BufferedCount())
	}
}

func TestBotNotifierSetGetMode(t *testing.T) {
	mock := &mockNotifier{name: "test"}
	bot := NewBotNotifier(mock, BotModeAll)
	if bot.GetMode() != BotModeAll {
		t.Errorf("GetMode() = %q, want %q", bot.GetMode(), BotModeAll)
	}

	bot.SetMode(BotModeQuiet)
	if bot.GetMode() != BotModeQuiet {
		t.Errorf("GetMode() after SetMode = %q, want %q", bot.GetMode(), BotModeQuiet)
	}
}

func TestBotNotifierFlushEmptyDigest(t *testing.T) {
	mock := &mockNotifier{name: "test"}
	bot := NewBotNotifier(mock, BotModeDigest)
	result, err := bot.FlushDigest(context.Background())
	if err != nil {
		t.Fatalf("FlushDigest() error = %v", err)
	}
	if !result.Success {
		t.Error("Flush empty digest should be success")
	}
	if mock.sent {
		t.Error("Flush empty digest should not send anything")
	}
}

// ========== SMS Notifier Tests ==========

func TestSMSNotifierName(t *testing.T) {
	s := NewSMSNotifier("id", "secret", "sign", "tpl")
	if s.Name() != "sms" {
		t.Errorf("Name() = %q, want %q", s.Name(), "sms")
	}
}

func TestGenericSMSNotifierSendSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q, want Bearer test-key", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := NewGenericSMSNotifier(server.URL, "test-key", "phone")
	n := DeploySuccess("app", "server", "img")
	n.Metadata = map[string]string{"phone": "13800138000"}

	result, err := s.Send(context.Background(), n)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}
}

func TestGenericSMSNotifierSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	s := NewGenericSMSNotifier(server.URL, "", "")
	result, err := s.Send(context.Background(), DeploySuccess("app", "server", "img"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.Success {
		t.Error("Success should be false for 400 response")
	}
}

func TestSMSNotifierUnknownProvider(t *testing.T) {
	s := &SMSNotifier{Provider: "unknown"}
	result, err := s.Send(context.Background(), DeploySuccess("app", "server", "img"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.Success {
		t.Error("Success should be false for unknown provider")
	}
}

func TestSMSNotifierExtractPhone(t *testing.T) {
	s := NewSMSNotifier("id", "secret", "sign", "tpl")
	tests := []struct {
		name     string
		metadata map[string]string
		want     string
	}{
		{"phone field", map[string]string{"phone": "123456"}, "123456"},
		{"mobile field", map[string]string{"mobile": "789012"}, "789012"},
		{"both fields", map[string]string{"phone": "111", "mobile": "222"}, "111"},
		{"no fields", map[string]string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := Notification{Metadata: tt.metadata}
			if got := s.extractPhone(n); got != tt.want {
				t.Errorf("extractPhone() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"shorter than max", "hello", 10, "hello"},
		{"equal to max", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hello..."},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncate(tt.s, tt.maxLen); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

// ========== Mock Notifier ==========

type mockNotifier struct {
	name string
	sent bool
}

func (m *mockNotifier) Name() string {
	return m.name
}

func (m *mockNotifier) Send(ctx context.Context, n Notification) (*NotifyResult, error) {
	m.sent = true
	return &NotifyResult{Provider: m.name, Success: true}, nil
}
