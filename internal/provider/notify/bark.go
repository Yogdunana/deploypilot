package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BarkNotifier sends push notifications via Bark (iOS push notification service).
// See: https://github.com/Finb/bark-server
type BarkNotifier struct {
	ServerURL string
	DeviceKey string
	Group     string
	Icon      string
	Sound     string
	Level     string // active, timeSensitive, passive
	Archive   string // "1" to archive
	Client    *http.Client
}

// NewBarkNotifier creates a new BarkNotifier.
func NewBarkNotifier(serverURL, deviceKey string) *BarkNotifier {
	if serverURL == "" {
		serverURL = "https://api.day.app"
	}
	return &BarkNotifier{
		ServerURL: serverURL,
		DeviceKey: deviceKey,
		Group:     "DeployPilot",
		Sound:     "default",
		Level:     "active",
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (b *BarkNotifier) Name() string { return "bark" }

// barkRequest is the request body for Bark API.
type barkRequest struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Group    string `json:"group,omitempty"`
	Sound    string `json:"sound,omitempty"`
	Icon     string `json:"icon,omitempty"`
	Level    string `json:"level,omitempty"`
	IsArchive string `json:"isArchive,omitempty"`
	URL      string `json:"url,omitempty"`
}

// barkResponse is the response from Bark API.
type barkResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Timestamp int64 `json:"timestamp"`
}

// Send sends a notification via Bark push.
func (b *BarkNotifier) Send(ctx context.Context, notification Notification) (*NotifyResult, error) {
	// Map severity to Bark level
	level := b.mapLevel(notification.Status)

	reqBody := barkRequest{
		Title:    fmt.Sprintf("[%s] %s", notification.Type, notification.AppName),
		Body:     notification.Message,
		Group:    b.Group,
		Sound:    b.Sound,
		Icon:     b.Icon,
		Level:    level,
		IsArchive: b.Archive,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return &NotifyResult{Provider: "bark", Success: false, Error: err.Error()}, nil
	}

	url := fmt.Sprintf("%s/%s", b.ServerURL, b.DeviceKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return &NotifyResult{Provider: "bark", Success: false, Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.Client.Do(req)
	if err != nil {
		return &NotifyResult{Provider: "bark", Success: false, Error: err.Error()}, nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var barkResp barkResponse
	if err := json.NewDecoder(resp.Body).Decode(&barkResp); err != nil {
		return &NotifyResult{Provider: "bark", Success: false, Error: fmt.Sprintf("decode response failed: %v", err)}, nil
	}

	if barkResp.Code == 200 {
		return &NotifyResult{
			Provider: "bark",
			Success:  true,
			Message:  "bark push delivered",
		}, nil
	}

	return &NotifyResult{
		Provider: "bark",
		Success:  false,
		Error:    fmt.Sprintf("bark error (code %d): %s", barkResp.Code, barkResp.Message),
	}, nil
}

// mapLevel maps notification status to Bark notification level.
func (b *BarkNotifier) mapLevel(status string) string {
	switch status {
	case "failed", "critical":
		return "timeSensitive"
	case "warning":
		return "active"
	default:
		return "passive"
	}
}
