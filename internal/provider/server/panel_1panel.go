package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// Panel1Client implements panel operations for 1Panel.
type Panel1Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// panel1Response represents the standard 1Panel API response format.
type panel1Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// panel1FirewallRule represents a 1Panel firewall rule.
type panel1FirewallRule struct {
	ID       json.Number `json:"id"`
	Protocol string      `json:"protocol"`
	Port     string      `json:"port"`
	Address  string      `json:"address"`
	Comment  string      `json:"comment"`
	Action   string      `json:"action"`
}

// panel1FirewallListData represents the data field of a firewall list response.
type panel1FirewallListData struct {
	Items []panel1FirewallRule `json:"items"`
	Total int                  `json:"total"`
}

// NewPanel1Client creates a new 1Panel API client.
func NewPanel1Client(baseURL, apiKey string) *Panel1Client {
	return &Panel1Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetHTTPClient allows injecting a custom HTTP client (useful for testing).
func (p *Panel1Client) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

// doRequest performs an authenticated HTTP request to the 1Panel API.
func (p *Panel1Client) doRequest(ctx context.Context, method, path string, body interface{}) (*panel1Response, error) {
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

	var result panel1Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("panel API error (code %d): %s", result.Code, result.Message)
	}

	return &result, nil
}

// OpenFirewall opens a port via the 1Panel API.
func (p *Panel1Client) OpenFirewall(ctx context.Context, port int, protocol string) error {
	slog.Info("1Panel: opening firewall port", "port", port, "protocol", protocol)

	reqBody := map[string]interface{}{
		"protocol": protocol,
		"port":     strconv.Itoa(port),
		"address":  "",
		"comment":  "deploypilot",
		"action":   "accept",
	}

	_, err := p.doRequest(ctx, http.MethodPost, "/api/v1/firewall/rules", reqBody)
	if err != nil {
		return fmt.Errorf("failed to open firewall port %d/%s: %w", port, protocol, err)
	}

	slog.Info("1Panel: firewall port opened successfully", "port", port, "protocol", protocol)
	return nil
}

// CloseFirewall closes a port via the 1Panel API.
// It first lists firewall rules to find the matching rule ID, then deletes it.
func (p *Panel1Client) CloseFirewall(ctx context.Context, port int, protocol string) error {
	slog.Info("1Panel: closing firewall port", "port", port, "protocol", protocol)

	// Step 1: List firewall rules to find the rule ID
	resp, err := p.doRequest(ctx, http.MethodGet, "/api/v1/firewall/rules", nil)
	if err != nil {
		return fmt.Errorf("failed to list firewall rules: %w", err)
	}

	var listData panel1FirewallListData
	if err := json.Unmarshal(resp.Data, &listData); err != nil {
		return fmt.Errorf("failed to parse firewall rules: %w", err)
	}

	// Find the matching rule
	portStr := strconv.Itoa(port)
	var ruleID string
	for _, rule := range listData.Items {
		if rule.Port == portStr && rule.Protocol == protocol && rule.Comment == "deploypilot" {
			ruleID = rule.ID.String()
			break
		}
	}

	if ruleID == "" {
		slog.Warn("1Panel: no matching firewall rule found to close", "port", port, "protocol", protocol)
		return nil
	}

	// Step 2: Delete the rule
	_, err = p.doRequest(ctx, http.MethodDelete, "/api/v1/firewall/rules/"+ruleID, nil)
	if err != nil {
		return fmt.Errorf("failed to close firewall port %d/%s: %w", port, protocol, err)
	}

	slog.Info("1Panel: firewall port closed successfully", "port", port, "protocol", protocol, "ruleID", ruleID)
	return nil
}

// CreateReverseProxy creates a reverse proxy via the 1Panel API.
func (p *Panel1Client) CreateReverseProxy(ctx context.Context, domain, targetURL string, port int) error {
	slog.Info("1Panel: creating reverse proxy", "domain", domain, "targetURL", targetURL, "port", port)

	reqBody := map[string]interface{}{
		"primaryDomain": domain,
		"proxy":         targetURL,
		"type":          "reverse_proxy",
	}

	_, err := p.doRequest(ctx, http.MethodPost, "/api/v1/websites/reverse_proxy", reqBody)
	if err != nil {
		return fmt.Errorf("failed to create reverse proxy for %s: %w", domain, err)
	}

	slog.Info("1Panel: reverse proxy created successfully", "domain", domain)
	return nil
}
