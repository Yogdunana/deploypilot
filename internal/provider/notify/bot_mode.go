package notify

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BotMode defines how a bot-style notifier should behave.
type BotMode string

const (
	// BotModeAll sends all notifications.
	BotModeAll BotMode = "all"
	// BotModeErrorOnly sends only error/critical notifications.
	BotModeErrorOnly BotMode = "error_only"
	// BotModeQuiet suppresses all notifications (dry run mode).
	BotModeQuiet BotMode = "quiet"
	// BotModeDigest collects notifications and sends them in batches.
	BotModeDigest BotMode = "digest"
)

// BotNotifier wraps a Notifier with bot-mode behavior controls.
// It supports filtering by severity, digest/batching, and quiet mode.
type BotNotifier struct {
	inner     Notifier
	mode      BotMode
	digestInterval time.Duration
	digestBuffer   []Notification
	digestMu       sync.Mutex
	stopCh         chan struct{}
}

// NewBotNotifier creates a new BotNotifier wrapping an existing Notifier.
func NewBotNotifier(inner Notifier, mode BotMode) *BotNotifier {
	return &BotNotifier{
		inner:          inner,
		mode:           mode,
		digestInterval: 5 * time.Minute,
		stopCh:         make(chan struct{}),
	}
}

// Name returns the notifier name with bot mode suffix.
func (b *BotNotifier) Name() string {
	return fmt.Sprintf("%s(bot:%s)", b.inner.Name(), b.mode)
}

// Send applies bot-mode filtering before delegating to the inner notifier.
func (b *BotNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	switch b.mode {
	case BotModeQuiet:
		return &NotifyResult{
			Provider: b.inner.Name(),
			Success:  true,
			Message:  "suppressed (quiet mode)",
		}, nil

	case BotModeErrorOnly:
		if notification.Status == "success" || notification.Status == "info" {
			return &NotifyResult{
				Provider: b.inner.Name(),
				Success:  true,
				Message:  "suppressed (error-only mode)",
			}, nil
		}
		return b.inner.Send(ctx, notification)

	case BotModeDigest:
		b.digestMu.Lock()
		b.digestBuffer = append(b.digestBuffer, notification)
		b.digestMu.Unlock()
		return &NotifyResult{
			Provider: b.inner.Name(),
			Success:  true,
			Message:  fmt.Sprintf("buffered (%d in digest)", len(b.digestBuffer)),
		}, nil

	case BotModeAll:
		return b.inner.Send(ctx, notification)

	default:
		return b.inner.Send(ctx, notification)
	}
}

// FlushDigest sends all buffered notifications as a single digest message.
func (b *BotNotifier) FlushDigest(ctx context.Context) (*NotifyResult, error) {
	b.digestMu.Lock()
	if len(b.digestBuffer) == 0 {
		b.digestMu.Unlock()
		return &NotifyResult{
			Provider: b.inner.Name(),
			Success:  true,
			Message:  "nothing to flush (empty digest buffer)",
		}, nil
	}

	notifications := make([]Notification, len(b.digestBuffer))
	copy(notifications, b.digestBuffer)
	b.digestBuffer = nil
	b.digestMu.Unlock()

	// Build digest message
	var lines []string
	lines = append(lines, fmt.Sprintf("📋 **DeployPilot Digest** (%d events)", len(notifications)))
	lines = append(lines, "")
	for i, n := range notifications {
		lines = append(lines, fmt.Sprintf("%d. **[%s]** %s — %s", i+1, n.Status, n.AppName, truncate(n.Message, 80)))
	}

	digestNotification := Notification{
		Type:    "digest",
		Status:  "info",
		Message: joinLines(lines),
	}

	return b.inner.Send(ctx, digestNotification)
}

// SetMode changes the bot mode at runtime.
func (b *BotNotifier) SetMode(mode BotMode) {
	b.mode = mode
}

// GetMode returns the current bot mode.
func (b *BotNotifier) GetMode() BotMode {
	return b.mode
}

// BufferedCount returns the number of notifications in the digest buffer.
func (b *BotNotifier) BufferedCount() int {
	b.digestMu.Lock()
	defer b.digestMu.Unlock()
	return len(b.digestBuffer)
}

// StartDigestTimer starts a background timer that flushes the digest periodically.
func (b *BotNotifier) StartDigestTimer(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(b.digestInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.stopCh:
				return
			case <-ticker.C:
				_, _ = b.FlushDigest(ctx)
			}
		}
	}()
}

// StopDigestTimer stops the digest timer.
func (b *BotNotifier) StopDigestTimer() {
	close(b.stopCh)
}

// joinLines joins strings with newlines.
func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}
