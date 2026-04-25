package cicd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GiteaActionsProvider implements CICDProvider for Gitea Actions.
type GiteaActionsProvider struct {
	token      string // Gitea personal access token
	owner      string // repo owner
	baseURL    string // Gitea instance base URL, e.g. https://gitea.com
	httpClient *http.Client
}

// NewGiteaActionsProvider creates a new Gitea Actions provider.
func NewGiteaActionsProvider(token, owner, baseURL string) *GiteaActionsProvider {
	return &GiteaActionsProvider{
		token:   token,
		owner:   owner,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name.
func (g *GiteaActionsProvider) Name() string { return "gitea" }

// Validate checks that the token and baseURL are configured.
func (g *GiteaActionsProvider) Validate() error {
	if g.token == "" {
		return fmt.Errorf("gitea token is required")
	}
	if g.baseURL == "" {
		return fmt.Errorf("gitea base URL is required")
	}
	return nil
}

// TriggerBuild triggers a CI/CD build via Gitea Actions workflow dispatch.
func (g *GiteaActionsProvider) TriggerBuild(ctx context.Context, repo, branch string) (string, error) {
	// First, find a workflow to trigger
	workflows, err := g.ListWorkflows(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("failed to list workflows: %w", err)
	}
	if len(workflows) == 0 {
		return "", fmt.Errorf("no workflows found in %s/%s", g.owner, repo)
	}

	// Use the first workflow
	workflow := workflows[0]
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/actions/workflows/%s/dispatches", g.baseURL, g.owner, repo, workflow)

	payload := map[string]string{
		"ref": branch,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to trigger build: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gitea API error %d: %s", resp.StatusCode, string(respBody))
	}

	return "triggered", nil
}

// GetBuildStatus gets the status of a CI/CD build.
func (g *GiteaActionsProvider) GetBuildStatus(ctx context.Context, runID string) (*BuildStatus, error) {
	// For "triggered" runID, we need to find the latest run
	if runID == "triggered" {
		return nil, fmt.Errorf("build was just triggered; use a specific run ID to check status")
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/actions/runs/%s", g.baseURL, g.owner, runID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+g.token)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get build status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gitea API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID        int64  `json:"id"`
		Status    string `json:"status"`
		Conclusion string `json:"conclusion"`
		HeadSHA   string `json:"head_sha"`
		HeadBranch string `json:"head_branch"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		LogsURL   string `json:"logs_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	status := result.Status
	if result.Conclusion != "" {
		status = result.Conclusion
	}

	// Calculate duration
	var duration int
	if result.CreatedAt != "" && result.UpdatedAt != "" {
		created, _ := time.Parse(time.RFC3339, result.CreatedAt)
		updated, _ := time.Parse(time.RFC3339, result.UpdatedAt)
		if !created.IsZero() && !updated.IsZero() {
			duration = int(updated.Sub(created).Seconds())
		}
	}

	return &BuildStatus{
		RunID:    fmt.Sprintf("%d", result.ID),
		Status:   status,
		Commit:   result.HeadSHA,
		Branch:   result.HeadBranch,
		Duration: duration,
		LogsURL:  result.LogsURL,
	}, nil
}

// GetBuildLogs gets the logs for a CI/CD build.
func (g *GiteaActionsProvider) GetBuildLogs(ctx context.Context, runID string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/actions/runs/%s/logs", g.baseURL, g.owner, runID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "token "+g.token)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get build logs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gitea API error %d: %s", resp.StatusCode, string(respBody))
	}

	logs, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return string(logs), nil
}

// ListWorkflows lists available workflows for a repository.
func (g *GiteaActionsProvider) ListWorkflows(ctx context.Context, repo string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/actions/workflows", g.baseURL, g.owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+g.token)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gitea API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Workflows []struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Path  string `json:"path"`
			State string `json:"state"`
		} `json:"workflows"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	workflows := make([]string, 0, len(result.Workflows))
	for _, w := range result.Workflows {
		if w.State == "active" {
			workflows = append(workflows, fmt.Sprintf("%d", w.ID))
		}
	}

	return workflows, nil
}
