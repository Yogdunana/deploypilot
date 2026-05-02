package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GrafanaClient wraps the Grafana HTTP API.
type GrafanaClient struct {
	baseURL    string
	apiKey     string
	adminUser  string
	adminPass  string
	httpClient *http.Client
}

// NewGrafanaClient creates a new GrafanaClient with the given connection parameters.
func NewGrafanaClient(rawURL, apiKey, adminUser, adminPass string) (*GrafanaClient, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Grafana URL %q: %w", rawURL, err)
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "http"
	}
	return &GrafanaClient{
		baseURL:   strings.TrimRight(parsed.String(), "/"),
		apiKey:    apiKey,
		adminUser: adminUser,
		adminPass: adminPass,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// doRequest is the internal helper for all Grafana API calls.
// It returns the response body, HTTP status code, and any error.
func (c *GrafanaClient) doRequest(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(context.Background(), method, reqURL, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, reqURL, err)
	}

	// Prefer API key auth; fall back to basic auth.
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	} else if c.adminUser != "" {
		req.SetBasicAuth(c.adminUser, c.adminPass)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request %s %s: %w", method, reqURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// TestConnection verifies connectivity by fetching the current organization.
func (c *GrafanaClient) TestConnection() (map[string]interface{}, error) {
	body, status, err := c.doRequest(http.MethodGet, "/api/org", nil)
	if err != nil {
		return nil, fmt.Errorf("test connection: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("test connection returned status %d: %s", status, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal org response: %w", err)
	}
	return result, nil
}

// EnsureDatasource creates or updates the "DeployPilot" Prometheus datasource.
// It returns the datasource UID.
func (c *GrafanaClient) EnsureDatasource(metricsURL string) (string, error) {
	// First, try to find an existing datasource by name.
	body, status, err := c.doRequest(http.MethodGet, "/api/datasources/name/DeployPilot", nil)
	if err != nil {
		return "", fmt.Errorf("lookup datasource: %w", err)
	}

	if status == http.StatusOK {
		var existing map[string]interface{}
		if err := json.Unmarshal(body, &existing); err != nil {
			return "", fmt.Errorf("unmarshal existing datasource: %w", err)
		}
		uid, _ := existing["uid"].(string)
		if uid != "" {
			// Update the existing datasource URL.
			idVal, _ := existing["id"].(float64)
			updatePayload := map[string]interface{}{
				"id":   int(idVal),
				"name": "DeployPilot",
				"type": "prometheus",
				"url":  metricsURL,
				"access": "proxy",
				"isDefault": false,
				"jsonData": map[string]interface{}{
					"httpMethod": "POST",
					"timeInterval": "15s",
				},
			}
			_, updStatus, updErr := c.doRequest(http.MethodPut, "/api/datasources/"+strconv.FormatInt(int64(idVal), 10), updatePayload)
			if updErr != nil {
				return "", fmt.Errorf("update datasource: %w", updErr)
			}
			if updStatus != http.StatusOK {
				return "", fmt.Errorf("update datasource returned status %d", updStatus)
			}
			return uid, nil
		}
	}

	// Create a new datasource.
	createPayload := map[string]interface{}{
		"name":      "DeployPilot",
		"type":      "prometheus",
		"url":       metricsURL,
		"access":    "proxy",
		"isDefault": false,
		"jsonData": map[string]interface{}{
			"httpMethod":   "POST",
			"timeInterval": "15s",
		},
	}
	body, status, err = c.doRequest(http.MethodPost, "/api/datasources", createPayload)
	if err != nil {
		return "", fmt.Errorf("create datasource: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return "", fmt.Errorf("create datasource returned status %d: %s", status, string(body))
	}

	var created map[string]interface{}
	if err := json.Unmarshal(body, &created); err != nil {
		return "", fmt.Errorf("unmarshal created datasource: %w", err)
	}
	uid, _ := created["uid"].(string)
	if uid == "" {
		return "", fmt.Errorf("created datasource has no UID")
	}
	return uid, nil
}

// EnsureFolder creates the "DeployPilot" folder if it does not exist.
// It returns the folder ID.
func (c *GrafanaClient) EnsureFolder() (int, error) {
	// Try to find existing folder.
	body, status, err := c.doRequest(http.MethodGet, "/api/folders", nil)
	if err != nil {
		return 0, fmt.Errorf("list folders: %w", err)
	}

	if status == http.StatusOK {
		var folders []map[string]interface{}
		if err := json.Unmarshal(body, &folders); err != nil {
			return 0, fmt.Errorf("unmarshal folders: %w", err)
		}
		for _, f := range folders {
			if title, _ := f["title"].(string); title == "DeployPilot" {
				if idVal, ok := f["id"].(float64); ok {
					return int(idVal), nil
				}
			}
		}
	}

	// Create the folder.
	createPayload := map[string]interface{}{
		"title": "DeployPilot",
	}
	body, status, err = c.doRequest(http.MethodPost, "/api/folders", createPayload)
	if err != nil {
		return 0, fmt.Errorf("create folder: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return 0, fmt.Errorf("create folder returned status %d: %s", status, string(body))
	}

	var created map[string]interface{}
	if err := json.Unmarshal(body, &created); err != nil {
		return 0, fmt.Errorf("unmarshal created folder: %w", err)
	}
	idVal, _ := created["id"].(float64)
	return int(idVal), nil
}

// UpsertDashboard creates or updates a Grafana dashboard.
// If overwrite is true, existing dashboards with the same UID will be replaced.
func (c *GrafanaClient) UpsertDashboard(dashboardJSON map[string]interface{}, folderID int, overwrite bool) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"dashboard": dashboardJSON,
		"folderId":  folderID,
		"overwrite": overwrite,
	}

	body, status, err := c.doRequest(http.MethodPost, "/api/dashboards/db", payload)
	if err != nil {
		return nil, fmt.Errorf("upsert dashboard: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("upsert dashboard returned status %d: %s", status, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal dashboard response: %w", err)
	}
	return result, nil
}

// CreateAnnotation adds an annotation to Grafana with the given tags, text, and time range.
func (c *GrafanaClient) CreateAnnotation(tags []string, text string, timestamp int64, timeEnd int64) error {
	payload := map[string]interface{}{
		"time":     timestamp,
		"timeEnd":  timeEnd,
		"tags":     tags,
		"text":     text,
	}

	_, status, err := c.doRequest(http.MethodPost, "/api/annotations", payload)
	if err != nil {
		return fmt.Errorf("create annotation: %w", err)
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return fmt.Errorf("create annotation returned status %d", status)
	}
	return nil
}

// DeleteDashboard deletes a dashboard by its UID.
func (c *GrafanaClient) DeleteDashboard(uid string) error {
	_, status, err := c.doRequest(http.MethodDelete, "/api/dashboards/uid/"+uid, nil)
	if err != nil {
		return fmt.Errorf("delete dashboard %s: %w", uid, err)
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("delete dashboard %s returned status %d", uid, status)
	}
	return nil
}
