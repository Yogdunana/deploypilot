package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// PanelBTPanel implements panel operations for BT-Panel (BaoTa).
type PanelBTPanel struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// btPanelResponse represents the standard BT-Panel API response format.
type btPanelResponse struct {
	Status bool   `json:"status"`
	Msg    string `json:"msg"`
}

// NewPanelBTPanel creates a new BT-Panel API client.
func NewPanelBTPanel(baseURL, apiKey string) *PanelBTPanel {
	return &PanelBTPanel{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetHTTPClient allows injecting a custom HTTP client (useful for testing).
func (p *PanelBTPanel) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

// doRequest performs an authenticated HTTP request to the BT-Panel API.
func (p *PanelBTPanel) doRequest(ctx context.Context, method, path string, body interface{}) (*btPanelResponse, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("panel API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result btPanelResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Status {
		return nil, fmt.Errorf("panel API error: %s", result.Msg)
	}

	return &result, nil
}

// OpenFirewall opens a port via the BT-Panel API.
func (p *PanelBTPanel) OpenFirewall(ctx context.Context, port int, protocol string) error {
	slog.Info("BT-Panel: opening firewall port", "port", port, "protocol", protocol)

	reqBody := map[string]interface{}{
		"port": fmt.Sprintf("%d", port),
		"ps":   "deploypilot",
		"type": "port",
	}

	_, err := p.doRequest(ctx, http.MethodPost, "/firewall?action=AddAcceptPort", reqBody)
	if err != nil {
		return fmt.Errorf("failed to open firewall port %d/%s: %w", port, protocol, err)
	}

	slog.Info("BT-Panel: firewall port opened successfully", "port", port, "protocol", protocol)
	return nil
}

// CloseFirewall closes a port via the BT-Panel API.
func (p *PanelBTPanel) CloseFirewall(ctx context.Context, port int, protocol string) error {
	slog.Info("BT-Panel: closing firewall port", "port", port, "protocol", protocol)

	reqBody := map[string]interface{}{
		"port": fmt.Sprintf("%d", port),
		"ps":   "deploypilot",
		"type": "port",
	}

	_, err := p.doRequest(ctx, http.MethodPost, "/firewall?action=DelAcceptPort", reqBody)
	if err != nil {
		return fmt.Errorf("failed to close firewall port %d/%s: %w", port, protocol, err)
	}

	slog.Info("BT-Panel: firewall port closed successfully", "port", port, "protocol", protocol)
	return nil
}

// CreateReverseProxy creates a reverse proxy via the BT-Panel API.
func (p *PanelBTPanel) CreateReverseProxy(ctx context.Context, domain, targetURL string, port int) error {
	slog.Info("BT-Panel: creating reverse proxy", "domain", domain, "targetURL", targetURL, "port", port)

	reqBody := map[string]interface{}{
		"domain":    domain,
		"proxy":     targetURL,
		"proxyPort": fmt.Sprintf("%d", port),
		"siteName":  domain,
		"type":      "reverse_proxy",
	}

	_, err := p.doRequest(ctx, http.MethodPost, "/site?action=CreateReverseProxy", reqBody)
	if err != nil {
		return fmt.Errorf("failed to create reverse proxy for %s: %w", domain, err)
	}

	slog.Info("BT-Panel: reverse proxy created successfully", "domain", domain)
	return nil
}
