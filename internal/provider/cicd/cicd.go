package cicd

import "context"

// BuildStatus represents the status of a CI/CD build.
type BuildStatus struct {
	RunID    string `json:"run_id"`
	Status   string `json:"status"` // queued, in_progress, completed, success, failure
	Commit   string `json:"commit"`
	Branch   string `json:"branch"`
	Duration int    `json:"duration_seconds"`
	LogsURL  string `json:"logs_url"`
}

// CICDProvider defines the interface for CI/CD integrations.
type CICDProvider interface {
	Name() string
	TriggerBuild(ctx context.Context, repo, branch string) (runID string, err error)
	GetBuildStatus(ctx context.Context, runID string) (*BuildStatus, error)
	GetBuildLogs(ctx context.Context, runID string) (string, error)
	ListWorkflows(ctx context.Context, repo string) ([]string, error)
}
