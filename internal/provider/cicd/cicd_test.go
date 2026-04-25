package cicd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ========== GitHub Actions Provider Tests ==========

func TestGitHubActionsName(t *testing.T) {
	g := NewGitHubActionsProvider("test-token", "test-owner")
	if g.Name() != "github-actions" {
		t.Errorf("Name() = %q, want %q", g.Name(), "github-actions")
	}
}

func TestGitHubActionsTriggerBuild(t *testing.T) {
	workflowsCalled := false
	dispatchCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/repos/test-owner/test-repo/actions/workflows" {
			workflowsCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"workflows": []map[string]interface{}{
					{"id": 123, "name": "CI", "path": ".github/workflows/ci.yml", "state": "active"},
				},
			})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/repos/test-owner/test-repo/actions/workflows/123/dispatches" {
			dispatchCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	runID, err := g.TriggerBuild(context.Background(), "test-repo", "main")
	if err != nil {
		t.Fatalf("TriggerBuild() error = %v", err)
	}
	if runID != "triggered" {
		t.Errorf("runID = %q, want %q", runID, "triggered")
	}
	if !workflowsCalled {
		t.Error("should have called workflows endpoint")
	}
	if !dispatchCalled {
		t.Error("should have called dispatch endpoint")
	}
}

func TestGitHubActionsGetBuildStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/repos/test-owner/actions/runs/12345" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          12345,
				"status":      "completed",
				"conclusion":  "success",
				"head_sha":    "abc123",
				"head_branch": "main",
				"created_at":  "2024-01-01T00:00:00Z",
				"updated_at":  "2024-01-01T00:05:00Z",
				"logs_url":    "https://example.com/logs",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	status, err := g.GetBuildStatus(context.Background(), "12345")
	if err != nil {
		t.Fatalf("GetBuildStatus() error = %v", err)
	}
	if status.RunID != "12345" {
		t.Errorf("RunID = %q, want %q", status.RunID, "12345")
	}
	if status.Status != "success" {
		t.Errorf("Status = %q, want %q", status.Status, "success")
	}
	if status.Commit != "abc123" {
		t.Errorf("Commit = %q, want %q", status.Commit, "abc123")
	}
	if status.Branch != "main" {
		t.Errorf("Branch = %q, want %q", status.Branch, "main")
	}
	if status.Duration != 300 {
		t.Errorf("Duration = %d, want 300", status.Duration)
	}
}

func TestGitHubActionsListWorkflows(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/repos/test-owner/test-repo/actions/workflows" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"workflows": []map[string]interface{}{
					{"id": 1, "name": "CI", "state": "active"},
					{"id": 2, "name": "Deploy", "state": "active"},
					{"id": 3, "name": "Old", "state": "disabled_manually"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	workflows, err := g.ListWorkflows(context.Background(), "test-repo")
	if err != nil {
		t.Fatalf("ListWorkflows() error = %v", err)
	}
	if len(workflows) != 2 {
		t.Errorf("len(workflows) = %d, want 2 (only active)", len(workflows))
	}
}

func TestGitHubActionsTriggerBuildNoWorkflows(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"workflows": []map[string]interface{}{},
		})
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	_, err := g.TriggerBuild(context.Background(), "test-repo", "main")
	if err == nil {
		t.Error("TriggerBuild() should return error when no workflows found")
	}
}

func TestGitHubActionsGetBuildStatusError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Not Found",
		})
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	_, err := g.GetBuildStatus(context.Background(), "99999")
	if err == nil {
		t.Error("GetBuildStatus() should return error for 404")
	}
}

func TestGitHubActionsSetBaseURL(t *testing.T) {
	g := NewGitHubActionsProvider("token", "owner")
	g.SetBaseURL("http://custom-url")
	if g.baseURL != "http://custom-url" {
		t.Errorf("baseURL = %q, want %q", g.baseURL, "http://custom-url")
	}
}

func TestGitHubActionsTriggerBuildDispatchError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/repos/test-owner/test-repo/actions/workflows" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"workflows": []map[string]interface{}{
					{"id": 1, "name": "CI", "state": "active"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "reference not found"})
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	_, err := g.TriggerBuild(context.Background(), "test-repo", "nonexistent-branch")
	if err == nil {
		t.Error("TriggerBuild() should return error when dispatch fails")
	}
}

func TestGitHubActionsGetBuildLogs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/repos/test-owner/actions/runs/12345/logs" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("build log line 1\nbuild log line 2\n"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	logs, err := g.GetBuildLogs(context.Background(), "12345")
	if err != nil {
		t.Fatalf("GetBuildLogs() error = %v", err)
	}
	if !strings.Contains(logs, "build log line 1") {
		t.Errorf("logs should contain build output, got: %s", logs)
	}
}

func TestGitHubActionsGetBuildLogsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Not Found"})
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	_, err := g.GetBuildLogs(context.Background(), "99999")
	if err == nil {
		t.Error("GetBuildLogs() should return error for 404")
	}
}

func TestGitHubActionsGetBuildStatusTriggered(t *testing.T) {
	g := NewGitHubActionsProvider("test-token", "test-owner")
	_, err := g.GetBuildStatus(context.Background(), "triggered")
	if err == nil {
		t.Error("GetBuildStatus() should return error for 'triggered' runID")
	}
}

func TestGitHubActionsListWorkflowsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Not Found"})
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	_, err := g.ListWorkflows(context.Background(), "test-repo")
	if err == nil {
		t.Error("ListWorkflows() should return error for 404")
	}
}

func TestGitHubActionsListWorkflowsParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	_, err := g.ListWorkflows(context.Background(), "test-repo")
	if err == nil {
		t.Error("ListWorkflows() should return error for invalid JSON")
	}
}

func TestGitHubActionsTriggerBuildListError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Not Found"})
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	_, err := g.TriggerBuild(context.Background(), "test-repo", "main")
	if err == nil {
		t.Error("TriggerBuild() should return error when listing workflows fails")
	}
}

func TestGitHubActionsGetBuildStatusParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	_, err := g.GetBuildStatus(context.Background(), "12345")
	if err == nil {
		t.Error("GetBuildStatus() should return error for invalid JSON")
	}
}

func TestGitHubActionsGetBuildStatusInProgress(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          12345,
			"status":      "in_progress",
			"conclusion":  "",
			"head_sha":    "abc123",
			"head_branch": "main",
			"created_at":  "2024-01-01T00:00:00Z",
			"updated_at":  "2024-01-01T00:05:00Z",
			"logs_url":    "https://example.com/logs",
		})
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	status, err := g.GetBuildStatus(context.Background(), "12345")
	if err != nil {
		t.Fatalf("GetBuildStatus() error = %v", err)
	}
	if status.Status != "in_progress" {
		t.Errorf("Status = %q, want %q", status.Status, "in_progress")
	}
}

func TestGitHubActionsGetBuildLogsRedirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test-owner/actions/runs/12345/logs" {
			w.Header().Set("Location", "http://example.com/logs.zip")
			w.WriteHeader(http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	// This tests the redirect path - the redirect URL points to an unreachable host,
	// so it will fail, but we verify the redirect handling code is exercised.
	_, err := g.GetBuildLogs(context.Background(), "12345")
	if err == nil {
		t.Log("GetBuildLogs with redirect succeeded")
	}
}

func TestGitHubActionsGetBuildLogsRedirectEmptyLocation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test-owner/actions/runs/12345/logs" {
			w.WriteHeader(http.StatusFound)
			// No Location header
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	// Redirect with empty location should fall through and read empty body
	logs, err := g.GetBuildLogs(context.Background(), "12345")
	if err != nil {
		t.Logf("GetBuildLogs with empty redirect location: %v", err)
	}
	_ = logs
}

func TestGitHubActionsGetBuildLogsRedirectSuccess(t *testing.T) {
	redirectCalled := false
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test-owner/actions/runs/12345/logs" {
			w.Header().Set("Location", ts.URL+"/redirect-logs")
			w.WriteHeader(http.StatusFound)
			return
		}
		if r.URL.Path == "/redirect-logs" {
			redirectCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("redirected log content"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	logs, err := g.GetBuildLogs(context.Background(), "12345")
	if err != nil {
		t.Fatalf("GetBuildLogs redirect success: %v", err)
	}
	if !redirectCalled {
		t.Error("expected redirect URL to be followed")
	}
	if logs != "redirected log content" {
		t.Errorf("expected 'redirected log content', got %q", logs)
	}
}

func TestGitHubActionsTriggerBuildContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		select {}
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.TriggerBuild(ctx, "test-repo", "main")
	if err == nil {
		t.Error("TriggerBuild() should return error when context is cancelled")
	}
}

func TestGitHubActionsGetBuildStatusContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.GetBuildStatus(ctx, "12345")
	if err == nil {
		t.Error("GetBuildStatus() should return error when context is cancelled")
	}
}

func TestGitHubActionsGetBuildLogsContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.GetBuildLogs(ctx, "12345")
	if err == nil {
		t.Error("GetBuildLogs() should return error when context is cancelled")
	}
}

func TestGitHubActionsListWorkflowsContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.ListWorkflows(ctx, "test-repo")
	if err == nil {
		t.Error("ListWorkflows() should return error when context is cancelled")
	}
}

func TestGitHubActionsGetBuildStatusNoConclusion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          12345,
			"status":      "queued",
			"conclusion":  nil,
			"head_sha":    "abc123",
			"head_branch": "main",
			"created_at":  "",
			"updated_at":  "",
			"logs_url":    "",
		})
	}))
	defer ts.Close()

	g := NewGitHubActionsProvider("test-token", "test-owner")
	g.SetBaseURL(ts.URL)

	status, err := g.GetBuildStatus(context.Background(), "12345")
	if err != nil {
		t.Fatalf("GetBuildStatus() error = %v", err)
	}
	if status.Status != "queued" {
		t.Errorf("Status = %q, want %q", status.Status, "queued")
	}
	if status.Duration != 0 {
		t.Errorf("Duration = %d, want 0 for empty timestamps", status.Duration)
	}
}
