package cicd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ========== Gitea Actions Provider Tests ==========

func TestGiteaActionsName(t *testing.T) {
	g := NewGiteaActionsProvider("test-token", "test-owner", "https://gitea.com")
	if g.Name() != "gitea" {
		t.Errorf("Name() = %q, want %q", g.Name(), "gitea")
	}
}

func TestGiteaValidateEmpty(t *testing.T) {
	g := NewGiteaActionsProvider("", "owner", "")
	err := g.Validate()
	if err == nil {
		t.Error("Validate() should return error for empty token and baseURL")
	}
}

func TestGiteaValidateEmptyToken(t *testing.T) {
	g := NewGiteaActionsProvider("", "owner", "https://gitea.com")
	err := g.Validate()
	if err == nil {
		t.Error("Validate() should return error for empty token")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("error = %q, want to contain 'token'", err.Error())
	}
}

func TestGiteaValidateEmptyBaseURL(t *testing.T) {
	g := NewGiteaActionsProvider("token", "owner", "")
	err := g.Validate()
	if err == nil {
		t.Error("Validate() should return error for empty baseURL")
	}
	if !strings.Contains(err.Error(), "base URL") {
		t.Errorf("error = %q, want to contain 'base URL'", err.Error())
	}
}

func TestGiteaValidateValid(t *testing.T) {
	g := NewGiteaActionsProvider("test-token", "test-owner", "https://gitea.com")
	err := g.Validate()
	if err != nil {
		t.Errorf("Validate() unexpected error = %v", err)
	}
}

func TestGiteaTriggerBuild(t *testing.T) {
	workflowsCalled := false
	dispatchCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test-owner/test-repo/actions/workflows" {
			workflowsCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"workflows": []map[string]interface{}{
					{"id": 123, "name": "CI", "path": ".gitea/workflows/ci.yml", "state": "active"},
				},
			})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/test-owner/test-repo/actions/workflows/123/dispatches" {
			dispatchCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

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

func TestGiteaTriggerBuildCreated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test-owner/test-repo/actions/workflows" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"workflows": []map[string]interface{}{
					{"id": 1, "name": "CI", "state": "active"},
				},
			})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/test-owner/test-repo/actions/workflows/1/dispatches" {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	runID, err := g.TriggerBuild(context.Background(), "test-repo", "main")
	if err != nil {
		t.Fatalf("TriggerBuild() error = %v", err)
	}
	if runID != "triggered" {
		t.Errorf("runID = %q, want %q", runID, "triggered")
	}
}

func TestGiteaGetBuildStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test-owner/actions/runs/12345" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          12345,
				"status":      "completed",
				"conclusion":  "success",
				"head_sha":    "abc123",
				"head_branch": "main",
				"created_at":  "2024-01-01T00:00:00Z",
				"updated_at":  "2024-01-01T00:05:00Z",
				"logs_url":    "https://gitea.com/test-owner/test-repo/actions/runs/12345/logs",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

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

func TestGiteaGetBuildStatusInProgress(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          12345,
			"status":      "running",
			"conclusion":  "",
			"head_sha":    "abc123",
			"head_branch": "main",
			"created_at":  "2024-01-01T00:00:00Z",
			"updated_at":  "2024-01-01T00:05:00Z",
			"logs_url":    "",
		})
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	status, err := g.GetBuildStatus(context.Background(), "12345")
	if err != nil {
		t.Fatalf("GetBuildStatus() error = %v", err)
	}
	if status.Status != "running" {
		t.Errorf("Status = %q, want %q", status.Status, "running")
	}
}

func TestGiteaGetBuildStatusNoConclusion(t *testing.T) {
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

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

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

func TestGiteaGetBuildStatusTriggered(t *testing.T) {
	g := NewGiteaActionsProvider("test-token", "test-owner", "https://gitea.com")
	_, err := g.GetBuildStatus(context.Background(), "triggered")
	if err == nil {
		t.Error("GetBuildStatus() should return error for 'triggered' runID")
	}
}

func TestGiteaGetBuildStatusError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Not Found",
		})
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	_, err := g.GetBuildStatus(context.Background(), "99999")
	if err == nil {
		t.Error("GetBuildStatus() should return error for 404")
	}
}

func TestGiteaGetBuildStatusParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	_, err := g.GetBuildStatus(context.Background(), "12345")
	if err == nil {
		t.Error("GetBuildStatus() should return error for invalid JSON")
	}
}

func TestGiteaListWorkflows(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test-owner/test-repo/actions/workflows" {
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

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	workflows, err := g.ListWorkflows(context.Background(), "test-repo")
	if err != nil {
		t.Fatalf("ListWorkflows() error = %v", err)
	}
	if len(workflows) != 2 {
		t.Errorf("len(workflows) = %d, want 2 (only active)", len(workflows))
	}
}

func TestGiteaListWorkflowsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Not Found"})
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	_, err := g.ListWorkflows(context.Background(), "test-repo")
	if err == nil {
		t.Error("ListWorkflows() should return error for 404")
	}
}

func TestGiteaListWorkflowsParseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	_, err := g.ListWorkflows(context.Background(), "test-repo")
	if err == nil {
		t.Error("ListWorkflows() should return error for invalid JSON")
	}
}

func TestGiteaTriggerBuildNoWorkflows(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"workflows": []map[string]interface{}{},
		})
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	_, err := g.TriggerBuild(context.Background(), "test-repo", "main")
	if err == nil {
		t.Error("TriggerBuild() should return error when no workflows found")
	}
}

func TestGiteaTriggerBuildListError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Not Found"})
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	_, err := g.TriggerBuild(context.Background(), "test-repo", "main")
	if err == nil {
		t.Error("TriggerBuild() should return error when listing workflows fails")
	}
}

func TestGiteaTriggerBuildDispatchError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test-owner/test-repo/actions/workflows" {
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

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	_, err := g.TriggerBuild(context.Background(), "test-repo", "nonexistent-branch")
	if err == nil {
		t.Error("TriggerBuild() should return error when dispatch fails")
	}
}

func TestGiteaGetBuildLogs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test-owner/actions/runs/12345/logs" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("build log line 1\nbuild log line 2\n"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	logs, err := g.GetBuildLogs(context.Background(), "12345")
	if err != nil {
		t.Fatalf("GetBuildLogs() error = %v", err)
	}
	if !strings.Contains(logs, "build log line 1") {
		t.Errorf("logs should contain build output, got: %s", logs)
	}
}

func TestGiteaGetBuildLogsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Not Found"})
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	_, err := g.GetBuildLogs(context.Background(), "99999")
	if err == nil {
		t.Error("GetBuildLogs() should return error for 404")
	}
}

func TestGiteaTriggerBuildContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.TriggerBuild(ctx, "test-repo", "main")
	if err == nil {
		t.Error("TriggerBuild() should return error when context is cancelled")
	}
}

func TestGiteaGetBuildStatusContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.GetBuildStatus(ctx, "12345")
	if err == nil {
		t.Error("GetBuildStatus() should return error when context is cancelled")
	}
}

func TestGiteaGetBuildLogsContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.GetBuildLogs(ctx, "12345")
	if err == nil {
		t.Error("GetBuildLogs() should return error when context is cancelled")
	}
}

func TestGiteaListWorkflowsContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("test-token", "test-owner", ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.ListWorkflows(ctx, "test-repo")
	if err == nil {
		t.Error("ListWorkflows() should return error when context is cancelled")
	}
}

func TestGiteaTriggerBuildAuthHeader(t *testing.T) {
	var authHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"workflows": []map[string]interface{}{
					{"id": 1, "name": "CI", "state": "active"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	g := NewGiteaActionsProvider("my-secret-token", "test-owner", ts.URL)

	_, err := g.TriggerBuild(context.Background(), "test-repo", "main")
	if err != nil {
		t.Fatalf("TriggerBuild() error = %v", err)
	}
	if authHeader != "token my-secret-token" {
		t.Errorf("Authorization = %q, want %q", authHeader, "token my-secret-token")
	}
}
