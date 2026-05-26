package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockNotifier struct {
	sentNotifications []Notification
}

func (m *mockNotifier) Name() string {
	return "mock"
}

func (m *mockNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	m.sentNotifications = append(m.sentNotifications, notification)
	return &NotifyResult{Provider: "mock", Success: true, Message: "sent"}, nil
}

func TestNewBotNotifier(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeAll)

	if bot.inner != mock {
		t.Error("inner notifier should be set")
	}
	if bot.mode != BotModeAll {
		t.Errorf("mode = %v, want %v", bot.mode, BotModeAll)
	}
}

func TestBotNotifierName(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeErrorOnly)

	name := bot.Name()
	expected := "mock(bot:error_only)"
	if name != expected {
		t.Errorf("Name() = %q, want %q", name, expected)
	}
}

func TestBotNotifierModeAll(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeAll)

	result, err := bot.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Error("should succeed")
	}
	if len(mock.sentNotifications) != 1 {
		t.Errorf("expected 1 notification sent, got %d", len(mock.sentNotifications))
	}
}

func TestBotNotifierModeQuiet(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeQuiet)

	result, err := bot.Send(context.Background(), DeployFailed("app", "srv", "error"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Error("should succeed (suppressed)")
	}
	if !contains(result.Message, "suppressed") {
		t.Errorf("Message = %q", result.Message)
	}
	if len(mock.sentNotifications) != 0 {
		t.Errorf("expected 0 notifications sent, got %d", len(mock.sentNotifications))
	}
}

func TestBotNotifierModeErrorOnly_Success(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeErrorOnly)

	result, err := bot.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Error("should succeed (suppressed)")
	}
	if len(mock.sentNotifications) != 0 {
		t.Errorf("expected 0 notifications sent for success status, got %d", len(mock.sentNotifications))
	}
}

func TestBotNotifierModeErrorOnly_Failure(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeErrorOnly)

	result, err := bot.Send(context.Background(), DeployFailed("app", "srv", "error"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Error("should succeed")
	}
	if len(mock.sentNotifications) != 1 {
		t.Errorf("expected 1 notification sent for failed status, got %d", len(mock.sentNotifications))
	}
}

func TestBotNotifierModeErrorOnly_Warning(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeErrorOnly)

	result, err := bot.Send(context.Background(), HealthCheckFailed("app", "srv", "url"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Error("should succeed")
	}
	if len(mock.sentNotifications) != 1 {
		t.Errorf("expected 1 notification sent for warning status, got %d", len(mock.sentNotifications))
	}
}

func TestBotNotifierModeErrorOnly_Info(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeErrorOnly)

	result, err := bot.Send(context.Background(), Notification{
		Type:    "info",
		AppName: "app",
		Status:  "info",
		Message: "info message",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Error("should succeed (suppressed)")
	}
	if len(mock.sentNotifications) != 0 {
		t.Errorf("expected 0 notifications sent for info status, got %d", len(mock.sentNotifications))
	}
}

func TestBotNotifierModeDigest(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeDigest)

	result, err := bot.Send(context.Background(), DeploySuccess("app1", "srv", "img"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Error("should succeed")
	}
	if !contains(result.Message, "buffered") {
		t.Errorf("Message = %q", result.Message)
	}
	if len(mock.sentNotifications) != 0 {
		t.Errorf("expected 0 notifications sent (buffered), got %d", len(mock.sentNotifications))
	}
}

func TestBotNotifierFlushDigest_Empty(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeDigest)

	result, err := bot.FlushDigest(context.Background())
	if err != nil {
		t.Fatalf("FlushDigest() error = %v", err)
	}
	if !result.Success {
		t.Error("should succeed")
	}
	if !contains(result.Message, "empty digest buffer") {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestBotNotifierFlushDigest_WithNotifications(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeDigest)

	bot.Send(context.Background(), DeploySuccess("app1", "srv", "img"))
	bot.Send(context.Background(), DeployFailed("app2", "srv", "error"))

	result, err := bot.FlushDigest(context.Background())
	if err != nil {
		t.Fatalf("FlushDigest() error = %v", err)
	}
	if !result.Success {
		t.Error("should succeed")
	}
	if len(mock.sentNotifications) != 1 {
		t.Errorf("expected 1 digest notification sent, got %d", len(mock.sentNotifications))
	}
	if !contains(mock.sentNotifications[0].Message, "DeployPilot Digest") {
		t.Errorf("digest message = %q", mock.sentNotifications[0].Message)
	}
}

func TestBotNotifierSetMode(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeAll)

	bot.SetMode(BotModeQuiet)
	if bot.GetMode() != BotModeQuiet {
		t.Errorf("mode = %v, want %v", bot.GetMode(), BotModeQuiet)
	}
}

func TestBotNotifierBufferedCount(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeDigest)

	if bot.BufferedCount() != 0 {
		t.Errorf("BufferedCount() = %d, want 0", bot.BufferedCount())
	}

	bot.Send(context.Background(), DeploySuccess("app1", "srv", "img"))
	bot.Send(context.Background(), DeploySuccess("app2", "srv", "img"))

	if bot.BufferedCount() != 2 {
		t.Errorf("BufferedCount() = %d, want 2", bot.BufferedCount())
	}

	bot.FlushDigest(context.Background())

	if bot.BufferedCount() != 0 {
		t.Errorf("BufferedCount() = %d, want 0 after flush", bot.BufferedCount())
	}
}

func TestBotNotifierStartAndStopDigestTimer(t *testing.T) {
	mock := &mockNotifier{}
	bot := NewBotNotifier(mock, BotModeDigest)
	bot.digestInterval = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	bot.StartDigestTimer(ctx)

	bot.Send(context.Background(), DeploySuccess("app", "srv", "img"))

	time.Sleep(200 * time.Millisecond)

	if len(mock.sentNotifications) != 1 {
		t.Errorf("expected 1 notification after timer flush, got %d", len(mock.sentNotifications))
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestBotNotifierWithRealNotifier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	wh := NewWebhookNotifier(server.URL, nil)
	bot := NewBotNotifier(wh, BotModeAll)

	result, err := bot.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, error: %s", result.Error)
	}
}

func TestBotNotifierModeDefault(t *testing.T) {
	mock := &mockNotifier{}
	bot := &BotNotifier{
		inner: mock,
		mode:  BotMode("unknown"),
	}

	result, err := bot.Send(context.Background(), DeploySuccess("app", "srv", "img"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !result.Success {
		t.Error("should succeed")
	}
	if len(mock.sentNotifications) != 1 {
		t.Errorf("expected 1 notification sent for default mode, got %d", len(mock.sentNotifications))
	}
}

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