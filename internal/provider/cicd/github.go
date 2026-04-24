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

// GitHubActionsProvider implements CICDProvider for GitHub Actions.
type GitHubActionsProvider struct {
	token      string // GitHub personal access token
	owner      string // repo owner
	baseURL    string
	httpClient *http.Client
}

// NewGitHubActionsProvider creates a new GitHub Actions provider.
func NewGitHubActionsProvider(token, owner string) *GitHubActionsProvider {
	return &GitHubActionsProvider{
		token:   token,
		owner:   owner,
		baseURL: "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL allows overriding the API base URL (for testing).
func (g *GitHubActionsProvider) SetBaseURL(u string) {
	g.baseURL = u
}

// Name returns the provider name.
func (g *GitHubActionsProvider) Name() string { return "github-actions" }

// TriggerBuild triggers a CI/CD build via GitHub Actions workflow dispatch.
// Note: GitHub dispatches return 204 with no body, so we return "triggered" as runID.
func (g *GitHubActionsProvider) TriggerBuild(ctx context.Context, repo, branch string) (string, error) {
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
	url := fmt.Sprintf("%s/repos/%s/%s/actions/workflows/%s/dispatches", g.baseURL, g.owner, repo, workflow)

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

	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to trigger build: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(respBody))
	}

	return "triggered", nil
}

// GetBuildStatus gets the status of a CI/CD build.
func (g *GitHubActionsProvider) GetBuildStatus(ctx context.Context, runID string) (*BuildStatus, error) {
	// For "triggered" runID, we need to find the latest run
	if runID == "triggered" {
		return nil, fmt.Errorf("build was just triggered; use a specific run ID to check status")
	}

	url := fmt.Sprintf("%s/repos/%s/actions/runs/%s", g.baseURL, g.owner, runID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get build status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(respBody))
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
func (g *GitHubActionsProvider) GetBuildLogs(ctx context.Context, runID string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/actions/runs/%s/logs", g.baseURL, g.owner, runID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get build logs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Follow redirect if needed
	if resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		if location != "" {
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
			if err != nil {
				return "", err
			}
			req.Header.Set("Authorization", "Bearer "+g.token)
			resp, err = g.httpClient.Do(req)
			if err != nil {
				return "", fmt.Errorf("failed to download logs: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
		}
	}

	logs, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return string(logs), nil
}

// ListWorkflows lists available workflows for a repository.
func (g *GitHubActionsProvider) ListWorkflows(ctx context.Context, repo string) ([]string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/workflows", g.baseURL, g.owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Workflows []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Path string `json:"path"`
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
