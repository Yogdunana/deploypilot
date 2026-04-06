package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// mockDeployer implements Deployer for testing.
type mockDeployer struct {
	deployFn       func(ctx context.Context, cfg DeployConfig) (*ContainerStatus, error)
	statusFn       func(ctx context.Context, name string) (*ContainerStatus, error)
	listAppsFn     func(ctx context.Context) ([]ContainerStatus, error)
	listServersFn  func(ctx context.Context) ([]ServerInfo, error)
	createAppFn    func(ctx context.Context, cfg CreateAppConfig) (string, error)
	deleteAppFn    func(ctx context.Context, appID string) error
	rollbackFn     func(ctx context.Context, containerName, previousImage string) (*ContainerStatus, error)
	backupFn       func(ctx context.Context, appID string) (string, error)
	restoreFn      func(ctx context.Context, backupID string) (*ContainerStatus, error)
	getLogsFn      func(ctx context.Context, name string, tail int) (string, error)
	detectEnvFn    func(ctx context.Context, level int, ports []int, services []string) (interface{}, error)
	healthCheckFn  func(ctx context.Context, target, healthType string) (interface{}, error)
}

func (m *mockDeployer) Deploy(ctx context.Context, cfg DeployConfig) (*ContainerStatus, error) {
	if m.deployFn != nil {
		return m.deployFn(ctx, cfg)
	}
	return &ContainerStatus{ID: "abc123", Name: cfg.ContainerName, Image: cfg.Image, Status: "running"}, nil
}

func (m *mockDeployer) GetContainerStatus(ctx context.Context, name string) (*ContainerStatus, error) {
	if m.statusFn != nil {
		return m.statusFn(ctx, name)
	}
	return &ContainerStatus{ID: "abc123", Name: name, Image: "nginx:latest", Status: "running"}, nil
}

func (m *mockDeployer) ListApps(ctx context.Context) ([]ContainerStatus, error) {
	if m.listAppsFn != nil {
		return m.listAppsFn(ctx)
	}
	return []ContainerStatus{
		{ID: "a1", Name: "web-app", Image: "nginx:latest", Status: "running"},
		{ID: "a2", Name: "api-app", Image: "node:18", Status: "stopped"},
	}, nil
}

func (m *mockDeployer) ListServers(ctx context.Context) ([]ServerInfo, error) {
	if m.listServersFn != nil {
		return m.listServersFn(ctx)
	}
	return []ServerInfo{
		{ID: "s1", Name: "prod-server", Host: "1.2.3.4", Status: "reachable"},
		{ID: "s2", Name: "staging-server", Host: "5.6.7.8", Status: "unknown"},
	}, nil
}

func (m *mockDeployer) CreateApp(ctx context.Context, cfg CreateAppConfig) (string, error) {
	if m.createAppFn != nil {
		return m.createAppFn(ctx, cfg)
	}
	return "app-new-001", nil
}

func (m *mockDeployer) DeleteApp(ctx context.Context, appID string) error {
	if m.deleteAppFn != nil {
		return m.deleteAppFn(ctx, appID)
	}
	return nil
}

func (m *mockDeployer) Stop(_ context.Context, _ string) error  { return nil }
func (m *mockDeployer) Remove(_ context.Context, _ string) error { return nil }

func (m *mockDeployer) Rollback(ctx context.Context, containerName, previousImage string) (*ContainerStatus, error) {
	if m.rollbackFn != nil {
		return m.rollbackFn(ctx, containerName, previousImage)
	}
	return &ContainerStatus{ID: "rb-001", Name: containerName, Image: previousImage, Status: "running"}, nil
}

func (m *mockDeployer) Backup(ctx context.Context, appID string) (string, error) {
	if m.backupFn != nil {
		return m.backupFn(ctx, appID)
	}
	return "backup-001", nil
}

func (m *mockDeployer) Restore(ctx context.Context, backupID string) (*ContainerStatus, error) {
	if m.restoreFn != nil {
		return m.restoreFn(ctx, backupID)
	}
	return &ContainerStatus{ID: "restore-001", Name: "restored-app", Image: "nginx:latest", Status: "running"}, nil
}

func (m *mockDeployer) GetContainerLogs(ctx context.Context, name string, tail int) (string, error) {
	if m.getLogsFn != nil {
		return m.getLogsFn(ctx, name, tail)
	}
	return "mock log line 1\nmock log line 2", nil
}

func (m *mockDeployer) DetectEnv(ctx context.Context, level int, ports []int, services []string) (interface{}, error) {
	if m.detectEnvFn != nil {
		return m.detectEnvFn(ctx, level, ports, services)
	}
	return map[string]interface{}{
		"os":     map[string]string{"goos": "linux", "arch": "amd64"},
		"docker": map[string]interface{}{"installed": true, "running": true},
		"level":  level,
	}, nil
}

func (m *mockDeployer) HealthCheck(ctx context.Context, target, healthType string) (interface{}, error) {
	if m.healthCheckFn != nil {
		return m.healthCheckFn(ctx, target, healthType)
	}
	return map[string]interface{}{
		"healthy":  true,
		"target":   target,
		"type":     healthType,
		"attempts": 1,
	}, nil
}

// extractText gets the text content from a CallToolResult.
func extractText(result *mcp.CallToolResult) (string, error) {
	if result.IsError {
		return "", fmt.Errorf("tool error: %s", result.Content[0].(mcp.TextContent).Text)
	}
	if len(result.Content) > 0 {
		if tc, ok := result.Content[0].(mcp.TextContent); ok {
			return tc.Text, nil
		}
	}
	return "", fmt.Errorf("no text content in result")
}

func newRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

// ========== deploy_app ==========

func TestDeployAppSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDeployApp(context.Background(), mock, newRequest(map[string]interface{}{
		"image": "nginx:latest", "container_name": "my-app", "ports": "8080:80",
	}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
	c := parsed["container"].(map[string]interface{})
	if c["name"] != "my-app" {
		t.Errorf("name = %v, want my-app", c["name"])
	}
}

func TestDeployAppWithEnvVars(t *testing.T) {
	var captured DeployConfig
	mock := &mockDeployer{
		deployFn: func(_ context.Context, cfg DeployConfig) (*ContainerStatus, error) {
			captured = cfg
			return &ContainerStatus{ID: "x", Name: cfg.ContainerName, Image: cfg.Image, Status: "running"}, nil
		},
	}

	envJSON, _ := json.Marshal(map[string]string{"DB_HOST": "localhost"})
	handleDeployApp(context.Background(), mock, newRequest(map[string]interface{}{
		"image": "nginx:latest", "container_name": "my-app", "env_vars": string(envJSON),
	}))
	if captured.EnvVars["DB_HOST"] != "localhost" {
		t.Errorf("EnvVars.DB_HOST = %q", captured.EnvVars["DB_HOST"])
	}
}

func TestDeployAppWithLabels(t *testing.T) {
	var captured DeployConfig
	mock := &mockDeployer{
		deployFn: func(_ context.Context, cfg DeployConfig) (*ContainerStatus, error) {
			captured = cfg
			return &ContainerStatus{ID: "x", Name: cfg.ContainerName, Image: cfg.Image, Status: "running"}, nil
		},
	}

	labelsJSON, _ := json.Marshal(map[string]string{"app": "myapp"})
	handleDeployApp(context.Background(), mock, newRequest(map[string]interface{}{
		"image": "nginx:latest", "container_name": "my-app", "labels": string(labelsJSON),
	}))
	if captured.Labels["app"] != "myapp" {
		t.Errorf("Labels.app = %q", captured.Labels["app"])
	}
}

func TestDeployAppMissingImage(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDeployApp(context.Background(), mock, newRequest(map[string]interface{}{
		"container_name": "my-app",
	}))
	if !result.IsError {
		t.Error("should return error when image is missing")
	}
}

func TestDeployAppMissingContainerName(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDeployApp(context.Background(), mock, newRequest(map[string]interface{}{
		"image": "nginx:latest",
	}))
	if !result.IsError {
		t.Error("should return error when container_name is missing")
	}
}

func TestDeployAppInvalidEnvVars(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDeployApp(context.Background(), mock, newRequest(map[string]interface{}{
		"image": "nginx:latest", "container_name": "my-app", "env_vars": "not-json",
	}))
	if !result.IsError {
		t.Error("should return error with invalid env_vars")
	}
}

func TestDeployAppDeployFailure(t *testing.T) {
	mock := &mockDeployer{
		deployFn: func(_ context.Context, _ DeployConfig) (*ContainerStatus, error) {
			return nil, fmt.Errorf("docker pull failed")
		},
	}
	result, _ := handleDeployApp(context.Background(), mock, newRequest(map[string]interface{}{
		"image": "nginx:latest", "container_name": "my-app",
	}))
	if !result.IsError {
		t.Error("should return error when deploy fails")
	}
}

// ========== get_deploy_status ==========

func TestGetDeployStatusSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleGetDeployStatus(context.Background(), mock, newRequest(map[string]interface{}{
		"container_name": "my-app",
	}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v", parsed["status"])
	}
}

func TestGetDeployStatusNotFound(t *testing.T) {
	mock := &mockDeployer{
		statusFn: func(_ context.Context, _ string) (*ContainerStatus, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	result, _ := handleGetDeployStatus(context.Background(), mock, newRequest(map[string]interface{}{
		"container_name": "nonexistent",
	}))
	if !result.IsError {
		t.Error("should return error for nonexistent container")
	}
}

func TestGetDeployStatusMissingName(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleGetDeployStatus(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when container_name is missing")
	}
}

// ========== NewServer ==========

func TestNewServerNotNil(t *testing.T) {
	s := NewServer(&mockDeployer{})
	if s == nil {
		t.Error("NewServer() returned nil")
	}
}

// ========== list_apps ==========

func TestListAppsSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleListApps(context.Background(), mock, newRequest(map[string]interface{}{}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
	apps := parsed["apps"].([]interface{})
	if len(apps) != 2 {
		t.Errorf("apps count = %d, want 2", len(apps))
	}
}

func TestListAppsEmpty(t *testing.T) {
	mock := &mockDeployer{
		listAppsFn: func(_ context.Context) ([]ContainerStatus, error) {
			return []ContainerStatus{}, nil
		},
	}
	result, _ := handleListApps(context.Background(), mock, newRequest(map[string]interface{}{}))

	text, _ := extractText(result)
	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	apps := parsed["apps"].([]interface{})
	if len(apps) != 0 {
		t.Errorf("apps count = %d, want 0", len(apps))
	}
}

func TestListAppsFailure(t *testing.T) {
	mock := &mockDeployer{
		listAppsFn: func(_ context.Context) ([]ContainerStatus, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	result, _ := handleListApps(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== rollback ==========

func TestRollbackSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleRollback(context.Background(), mock, newRequest(map[string]interface{}{
		"container_name": "my-app",
		"previous_image": "nginx:1.24",
	}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
	container := parsed["container"].(map[string]interface{})
	if container["image"] != "nginx:1.24" {
		t.Errorf("container.image = %v, want nginx:1.24", container["image"])
	}
}

func TestRollbackMissingName(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleRollback(context.Background(), mock, newRequest(map[string]interface{}{
		"previous_image": "nginx:1.24",
	}))
	if !result.IsError {
		t.Error("should return error when container_name is missing")
	}
}

func TestRollbackFailure(t *testing.T) {
	mock := &mockDeployer{
		rollbackFn: func(_ context.Context, _, _ string) (*ContainerStatus, error) {
			return nil, fmt.Errorf("rollback failed")
		},
	}
	result, _ := handleRollback(context.Background(), mock, newRequest(map[string]interface{}{
		"container_name": "my-app",
		"previous_image": "nginx:1.24",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== backup ==========

func TestBackupSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleBackup(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id": "app-001",
	}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
	backup := parsed["backup"].(map[string]interface{})
	if backup["id"] != "backup-001" {
		t.Errorf("backup.id = %v, want backup-001", backup["id"])
	}
}

func TestBackupMissingAppID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleBackup(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when app_id is missing")
	}
}

func TestBackupFailure(t *testing.T) {
	mock := &mockDeployer{
		backupFn: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("backup failed")
		},
	}
	result, _ := handleBackup(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id": "app-001",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== restore ==========

func TestRestoreSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleRestore(context.Background(), mock, newRequest(map[string]interface{}{
		"backup_id": "backup-001",
	}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
}

func TestRestoreMissingBackupID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleRestore(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when backup_id is missing")
	}
}

func TestRestoreFailure(t *testing.T) {
	mock := &mockDeployer{
		restoreFn: func(_ context.Context, _ string) (*ContainerStatus, error) {
			return nil, fmt.Errorf("restore failed")
		},
	}
	result, _ := handleRestore(context.Background(), mock, newRequest(map[string]interface{}{
		"backup_id": "backup-001",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== get_app_logs ==========

func TestGetAppLogsSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleGetAppLogs(context.Background(), mock, newRequest(map[string]interface{}{
		"container_name": "my-app",
	}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
	logs := parsed["logs"].(string)
	if !strings.Contains(logs, "mock log") {
		t.Errorf("logs = %q, want to contain 'mock log'", logs)
	}
}

func TestGetAppLogsMissingName(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleGetAppLogs(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when container_name is missing")
	}
}

func TestGetAppLogsFailure(t *testing.T) {
	mock := &mockDeployer{
		getLogsFn: func(_ context.Context, _ string, _ int) (string, error) {
			return "", fmt.Errorf("container not found")
		},
	}
	result, _ := handleGetAppLogs(context.Background(), mock, newRequest(map[string]interface{}{
		"container_name": "nonexistent",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== detect_env ==========

func TestDetectEnvSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDetectEnv(context.Background(), mock, newRequest(map[string]interface{}{
		"level": "2",
	}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
	if parsed["environment"] == nil {
		t.Error("environment should not be nil")
	}
}

func TestDetectEnvWithPorts(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDetectEnv(context.Background(), mock, newRequest(map[string]interface{}{
		"level": "3",
		"ports": "8080,3000",
	}))

	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestDetectEnvFailure(t *testing.T) {
	mock := &mockDeployer{
		detectEnvFn: func(_ context.Context, _ int, _ []int, _ []string) (interface{}, error) {
			return nil, fmt.Errorf("detection failed")
		},
	}
	result, _ := handleDetectEnv(context.Background(), mock, newRequest(map[string]interface{}{
		"level": "1",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== health_check ==========

func TestHealthCheckSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleHealthCheck(context.Background(), mock, newRequest(map[string]interface{}{
		"target": "http://localhost:8080/health",
		"type":   "http",
	}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
	health := parsed["health"].(map[string]interface{})
	if health["healthy"] != true {
		t.Errorf("healthy = %v, want true", health["healthy"])
	}
}

func TestHealthCheckMissingTarget(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleHealthCheck(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when target is missing")
	}
}

func TestHealthCheckFailure(t *testing.T) {
	mock := &mockDeployer{
		healthCheckFn: func(_ context.Context, _, _ string) (interface{}, error) {
			return nil, fmt.Errorf("health check failed")
		},
	}
	result, _ := handleHealthCheck(context.Background(), mock, newRequest(map[string]interface{}{
		"target": "http://localhost:9999/health",
		"type":   "http",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

func TestHealthCheckTCP(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleHealthCheck(context.Background(), mock, newRequest(map[string]interface{}{
		"target": "tcp://localhost:3306",
		"type":   "tcp",
	}))

	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

// ========== list_servers ==========

func TestListServersSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleListServers(context.Background(), mock, newRequest(map[string]interface{}{}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
	servers := parsed["servers"].([]interface{})
	if len(servers) != 2 {
		t.Errorf("servers count = %d, want 2", len(servers))
	}
}

func TestListServersEmpty(t *testing.T) {
	mock := &mockDeployer{
		listServersFn: func(_ context.Context) ([]ServerInfo, error) {
			return []ServerInfo{}, nil
		},
	}
	result, _ := handleListServers(context.Background(), mock, newRequest(map[string]interface{}{}))

	text, _ := extractText(result)
	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	servers := parsed["servers"].([]interface{})
	if len(servers) != 0 {
		t.Errorf("servers count = %d, want 0", len(servers))
	}
}

func TestListServersFailure(t *testing.T) {
	mock := &mockDeployer{
		listServersFn: func(_ context.Context) ([]ServerInfo, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	result, _ := handleListServers(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== create_app ==========

func TestCreateAppSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleCreateApp(context.Background(), mock, newRequest(map[string]interface{}{
		"name":     "my-app",
		"repo_url": "https://github.com/user/repo",
		"branch":   "main",
	}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
	app := parsed["app"].(map[string]interface{})
	if app["id"] != "app-new-001" {
		t.Errorf("app.id = %v, want app-new-001", app["id"])
	}
}

func TestCreateAppMissingName(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleCreateApp(context.Background(), mock, newRequest(map[string]interface{}{
		"repo_url": "https://github.com/user/repo",
	}))
	if !result.IsError {
		t.Error("should return error when name is missing")
	}
}

func TestCreateAppMissingRepoURL(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleCreateApp(context.Background(), mock, newRequest(map[string]interface{}{
		"name": "my-app",
	}))
	if !result.IsError {
		t.Error("should return error when repo_url is missing")
	}
}

func TestCreateAppCapturesConfig(t *testing.T) {
	var captured CreateAppConfig
	mock := &mockDeployer{
		createAppFn: func(_ context.Context, cfg CreateAppConfig) (string, error) {
			captured = cfg
			return "app-001", nil
		},
	}

	handleCreateApp(context.Background(), mock, newRequest(map[string]interface{}{
		"name":       "my-app",
		"repo_url":   "https://github.com/user/repo",
		"branch":     "develop",
		"tech_stack": "docker",
		"deploy_mode": "api",
	}))

	if captured.Name != "my-app" {
		t.Errorf("Name = %q", captured.Name)
	}
	if captured.Branch != "develop" {
		t.Errorf("Branch = %q, want develop", captured.Branch)
	}
	if captured.TechStack != "docker" {
		t.Errorf("TechStack = %q", captured.TechStack)
	}
}

func TestCreateAppFailure(t *testing.T) {
	mock := &mockDeployer{
		createAppFn: func(_ context.Context, _ CreateAppConfig) (string, error) {
			return "", fmt.Errorf("duplicate name")
		},
	}
	result, _ := handleCreateApp(context.Background(), mock, newRequest(map[string]interface{}{
		"name":     "my-app",
		"repo_url": "https://github.com/user/repo",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== delete_app ==========

func TestDeleteAppSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDeleteApp(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id": "app-001",
	}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
}

func TestDeleteAppMissingID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDeleteApp(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when app_id is missing")
	}
}

func TestDeleteAppFailure(t *testing.T) {
	mock := &mockDeployer{
		deleteAppFn: func(_ context.Context, _ string) error {
			return fmt.Errorf("app not found")
		},
	}
	result, _ := handleDeleteApp(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id": "nonexistent",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}
