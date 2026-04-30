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

// BTPanelClient implements panel operations for BT Panel.
type BTPanelClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	cookie     string
}

// btPanelResponse represents the standard BT Panel API response format.
type btPanelResponse struct {
	Status  bool        `json:"status"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
	Code    int         `json:"code,omitempty"`
}

// NewBTPanelClient creates a new BT Panel API client.
func NewBTPanelClient(baseURL, username, password string) *BTPanelClient {
	return &BTPanelClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetHTTPClient allows injecting a custom HTTP client (useful for testing).
func (p *BTPanelClient) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

// login authenticates with the BT Panel API and stores the cookie.
func (p *BTPanelClient) login(ctx context.Context) error {
	if p.cookie != "" {
		return nil
	}
	reqBody := map[string]interface{}{
		"username": p.username,
		"password": p.password,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}
	url := p.baseURL + "/login"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("BT Panel login failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "bt_user_token" {
			p.cookie = cookie.Name + "=" + cookie.Value
			break
		}
	}
	if p.cookie == "" {
		return fmt.Errorf("failed to get BT Panel auth cookie")
	}
	return nil
}

// doRequest performs an authenticated HTTP request to the BT Panel API.
func (p *BTPanelClient) doRequest(ctx context.Context, method, path string, body interface{}) (*btPanelResponse, error) {
	if err := p.login(ctx); err != nil {
		return nil, err
	}
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", p.cookie)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BT Panel API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var result btPanelResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if !result.Status {
		return nil, fmt.Errorf("BT Panel API error: %s", result.Msg)
	}
	return &result, nil
}

// OpenFirewall opens a port via the BT Panel API.
func (p *BTPanelClient) OpenFirewall(ctx context.Context, port int, protocol string) error {
	slog.Info("BT Panel: opening firewall port", "port", port, "protocol", protocol)
	reqBody := map[string]interface{}{
		"port":     strconv.Itoa(port),
		"protocol": protocol,
		"ps":       "deploypilot",
		"status":   "1",
	}
	_, err := p.doRequest(ctx, http.MethodPost, "/firewall/add", reqBody)
	if err != nil {
		return fmt.Errorf("failed to open firewall port %d/%s: %w", port, protocol, err)
	}
	slog.Info("BT Panel: firewall port opened successfully", "port", port, "protocol", protocol)
	return nil
}

// CloseFirewall closes a port via the BT Panel API.
func (p *BTPanelClient) CloseFirewall(ctx context.Context, port int, protocol string) error {
	slog.Info("BT Panel: closing firewall port", "port", port, "protocol", protocol)
	resp, err := p.doRequest(ctx, http.MethodGet, "/firewall", nil)
	if err != nil {
		return fmt.Errorf("failed to list firewall rules: %w", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid firewall rules data format")
	}
	rules, ok := data["list"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid firewall rules list format")
	}
	portStr := strconv.Itoa(port)
	var ruleID string
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}
		if ruleMap["port"] == portStr && ruleMap["protocol"] == protocol && ruleMap["ps"] == "deploypilot" {
			if id, ok := ruleMap["id"].(string); ok {
				ruleID = id
				break
			}
		}
	}
	if ruleID == "" {
		slog.Warn("BT Panel: no matching firewall rule found to close", "port", port, "protocol", protocol)
		return nil
	}
	reqBody := map[string]interface{}{
		"id": ruleID,
	}
	_, err = p.doRequest(ctx, http.MethodPost, "/firewall/del", reqBody)
	if err != nil {
		return fmt.Errorf("failed to close firewall port %d/%s: %w", port, protocol, err)
	}
	slog.Info("BT Panel: firewall port closed successfully", "port", port, "protocol", protocol, "ruleID", ruleID)
	return nil
}

// GetInfo returns information about the BT Panel client.
func (p *BTPanelClient) GetInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":     "BT-Panel (宝塔)",
		"features": []string{"website_management", "database", "ftp", "ssl", "cron"},
	}
}

// CreateReverseProxy creates a reverse proxy via the BT Panel API.
func (p *BTPanelClient) CreateReverseProxy(ctx context.Context, domain, targetURL string, port int) error {
	slog.Info("BT Panel: creating reverse proxy", "domain", domain, "targetURL", targetURL, "port", port)
	reqBody := map[string]interface{}{
		"domain":   domain,
		"upstream": targetURL,
		"type":     "reverse_proxy",
		"status":   "1",
	}
	_, err := p.doRequest(ctx, http.MethodPost, "/site/add", reqBody)
	if err != nil {
		return fmt.Errorf("failed to create reverse proxy for %s: %w", domain, err)
	}
	slog.Info("BT Panel: reverse proxy created successfully", "domain", domain)
	return nil
}

// DeleteReverseProxy deletes a reverse proxy website via the BT Panel API.
func (p *BTPanelClient) DeleteReverseProxy(ctx context.Context, domain string) error {
	slog.Info("BT Panel: deleting reverse proxy", "domain", domain)
	resp, err := p.doRequest(ctx, http.MethodGet, "/site/list", nil)
	if err != nil {
		return fmt.Errorf("failed to list sites: %w", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid site list data format")
	}
	sites, ok := data["list"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid site list format")
	}
	var siteID string
	for _, s := range sites {
		siteMap, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		if siteMap["domain"] == domain {
			if id, ok := siteMap["id"]; ok {
				switch v := id.(type) {
				case string:
					siteID = v
				case float64:
					siteID = strconv.Itoa(int(v))
				}
				break
			}
		}
	}
	if siteID == "" {
		slog.Warn("BT Panel: no matching site found to delete", "domain", domain)
		return fmt.Errorf("site not found for domain: %s", domain)
	}
	reqBody := map[string]interface{}{
		"id":      siteID,
		"webname": domain,
	}
	_, err = p.doRequest(ctx, http.MethodPost, "/site/delete", reqBody)
	if err != nil {
		return fmt.Errorf("failed to delete site for %s: %w", domain, err)
	}
	slog.Info("BT Panel: reverse proxy deleted successfully", "domain", domain, "siteID", siteID)
	return nil
}

// CreateWebsite creates a new website via the BT Panel API.
func (p *BTPanelClient) CreateWebsite(ctx context.Context, domain, websiteType, remark string) (*WebsiteInfo, error) {
	slog.Info("BT Panel: creating website", "domain", domain, "type", websiteType)
	reqBody := map[string]interface{}{
		"domain": domain,
		"type":   websiteType,
		"ps":     remark,
		"status": "1",
	}
	_, err := p.doRequest(ctx, http.MethodPost, "/site/add", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create website for %s: %w", domain, err)
	}
	slog.Info("BT Panel: website created successfully", "domain", domain)
	return &WebsiteInfo{
		PrimaryDomain: domain,
		Type:          websiteType,
		Remark:        remark,
		Status:        true,
	}, nil
}

// GetWebsiteList returns a list of websites managed by BT Panel.
func (p *BTPanelClient) GetWebsiteList(ctx context.Context) ([]WebsiteInfo, error) {
	slog.Info("BT Panel: listing websites")
	resp, err := p.doRequest(ctx, http.MethodGet, "/site/list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list websites: %w", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid website list data format")
	}
	sites, ok := data["list"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid website list format")
	}
	websites := make([]WebsiteInfo, 0, len(sites))
	for _, s := range sites {
		siteMap, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		var siteID string
		if id, ok := siteMap["id"]; ok {
			switch v := id.(type) {
			case string:
				siteID = v
			case float64:
				siteID = strconv.Itoa(int(v))
			}
		}
		var domain string
		if d, ok := siteMap["domain"].(string); ok {
			domain = d
		}
		var siteType string
		if t, ok := siteMap["type"].(string); ok {
			siteType = t
		}
		var remark string
		if ps, ok := siteMap["ps"].(string); ok {
			remark = ps
		}
		var status bool
		if st, ok := siteMap["status"].(bool); ok {
			status = st
		}
		websites = append(websites, WebsiteInfo{
			ID:            siteID,
			PrimaryDomain: domain,
			Type:          siteType,
			Status:        status,
			Remark:        remark,
		})
	}
	slog.Info("BT Panel: website list retrieved", "count", len(websites))
	return websites, nil
}
