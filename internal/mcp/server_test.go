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
	addServerFn    func(ctx context.Context, name, host string, port int, user string) (*ServerInfo, error)
	removeServerFn func(ctx context.Context, serverID string) error
	testServerFn   func(ctx context.Context, serverID string) (interface{}, error)
	createCredFn   func(ctx context.Context, tenantID, name, credType, plainValue string) (interface{}, error)
	listCredsFn    func(ctx context.Context, tenantID string) (interface{}, error)
	deleteCredFn   func(ctx context.Context, credID string) error
	dnsCreateFn    func(ctx context.Context, domain, recordType, name, value string) (interface{}, error)
	dnsDeleteFn    func(ctx context.Context, recordID string) error
	dnsListFn      func(ctx context.Context, domain string) (interface{}, error)
	notifySendFn   func(ctx context.Context, nType, appName, server, status, message string) (interface{}, error)
	templateListFn func(ctx context.Context) (interface{}, error)
	templateGetFn  func(ctx context.Context, tmplType string) (interface{}, error)
	getAppDetailFn func(ctx context.Context, appID string) (interface{}, error)
	updateAppFn    func(ctx context.Context, appID string, config map[string]interface{}) (interface{}, error)
	getTaskStatusFn func(ctx context.Context, taskID string) (interface{}, error)
	listTasksFn    func(ctx context.Context, limit int, statusFilter string) (interface{}, error)
	searchLogsFn   func(ctx context.Context, appID, keyword string, limit int) (interface{}, error)
	updateDNSFn    func(ctx context.Context, domain, subdomain, recordType, newValue string) (interface{}, error)
	updateCredFn   func(ctx context.Context, credID string, value string) (interface{}, error)
	updateServerFn func(ctx context.Context, serverID string, config map[string]interface{}) (interface{}, error)
	checkReadinessFn func(ctx context.Context, appConfig map[string]interface{}) (interface{}, error)
	batchDeployFn  func(ctx context.Context, apps []map[string]interface{}) (interface{}, error)
	batchBackupFn  func(ctx context.Context, appIDs []string) (interface{}, error)
	batchDNSFn     func(ctx context.Context, records []map[string]interface{}) (interface{}, error)
	checkSysUpdateFn func(ctx context.Context) (interface{}, error)
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

func (m *mockDeployer) AddServer(ctx context.Context, name, host string, port int, user string) (*ServerInfo, error) {
	if m.addServerFn != nil {
		return m.addServerFn(ctx, name, host, port, user)
	}
	return &ServerInfo{ID: "srv-new-001", Name: name, Host: host, Port: port, Status: "added"}, nil
}

func (m *mockDeployer) RemoveServer(ctx context.Context, serverID string) error {
	if m.removeServerFn != nil {
		return m.removeServerFn(ctx, serverID)
	}
	return nil
}

func (m *mockDeployer) TestServer(ctx context.Context, serverID string) (interface{}, error) {
	if m.testServerFn != nil {
		return m.testServerFn(ctx, serverID)
	}
	return map[string]interface{}{"server_id": serverID, "reachable": true, "latency": "15ms"}, nil
}

func (m *mockDeployer) CreateCredential(ctx context.Context, tenantID, name, credType, plainValue string) (interface{}, error) {
	if m.createCredFn != nil {
		return m.createCredFn(ctx, tenantID, name, credType, plainValue)
	}
	return map[string]interface{}{"id": "cred-new-001", "name": name, "type": credType}, nil
}

func (m *mockDeployer) ListCredentials(ctx context.Context, tenantID string) (interface{}, error) {
	if m.listCredsFn != nil {
		return m.listCredsFn(ctx, tenantID)
	}
	return []map[string]interface{}{
		{"id": "cred-001", "name": "ssh-key", "type": "ssh"},
		{"id": "cred-002", "name": "api-token", "type": "api_key"},
	}, nil
}

func (m *mockDeployer) DeleteCredential(ctx context.Context, credID string) error {
	if m.deleteCredFn != nil {
		return m.deleteCredFn(ctx, credID)
	}
	return nil
}

func (m *mockDeployer) DNSCreateRecord(ctx context.Context, domain, recordType, name, value string) (interface{}, error) {
	if m.dnsCreateFn != nil {
		return m.dnsCreateFn(ctx, domain, recordType, name, value)
	}
	return map[string]interface{}{"id": "rec-001", "domain": domain, "type": recordType, "name": name, "value": value}, nil
}

func (m *mockDeployer) DNSDeleteRecord(ctx context.Context, recordID string) error {
	if m.dnsDeleteFn != nil {
		return m.dnsDeleteFn(ctx, recordID)
	}
	return nil
}

func (m *mockDeployer) DNSListRecords(ctx context.Context, domain string) (interface{}, error) {
	if m.dnsListFn != nil {
		return m.dnsListFn(ctx, domain)
	}
	return []map[string]interface{}{
		{"id": "rec-001", "type": "A", "name": "@", "value": "1.2.3.4"},
	}, nil
}

func (m *mockDeployer) SendNotification(ctx context.Context, nType, appName, server, status, message string) (interface{}, error) {
	if m.notifySendFn != nil {
		return m.notifySendFn(ctx, nType, appName, server, status, message)
	}
	return map[string]interface{}{"sent": true, "type": nType, "app": appName}, nil
}

func (m *mockDeployer) ListTemplates(ctx context.Context) (interface{}, error) {
	if m.templateListFn != nil {
		return m.templateListFn(ctx)
	}
	return []map[string]interface{}{
		{"type": "node", "name": "Node.js"},
		{"type": "python", "name": "Python"},
		{"type": "go", "name": "Go"},
	}, nil
}

func (m *mockDeployer) GetTemplate(ctx context.Context, tmplType string) (interface{}, error) {
	if m.templateGetFn != nil {
		return m.templateGetFn(ctx, tmplType)
	}
	return map[string]interface{}{"type": tmplType, "name": "Node.js", "port": 3000, "image": "node:18-alpine"}, nil
}

func (m *mockDeployer) GetAppDetail(ctx context.Context, appID string) (interface{}, error) {
	if m.getAppDetailFn != nil {
		return m.getAppDetailFn(ctx, appID)
	}
	return map[string]interface{}{
		"id": appID, "name": "my-app", "image": "nginx:latest",
		"status": "running", "port": 8080, "server": "prod-01",
	}, nil
}

func (m *mockDeployer) UpdateApp(ctx context.Context, appID string, config map[string]interface{}) (interface{}, error) {
	if m.updateAppFn != nil {
		return m.updateAppFn(ctx, appID, config)
	}
	return map[string]interface{}{"id": appID, "updated": true, "config": config}, nil
}

func (m *mockDeployer) GetTaskStatus(ctx context.Context, taskID string) (interface{}, error) {
	if m.getTaskStatusFn != nil {
		return m.getTaskStatusFn(ctx, taskID)
	}
	return map[string]interface{}{"task_id": taskID, "status": "completed", "progress": 100}, nil
}

func (m *mockDeployer) ListTasks(ctx context.Context, limit int, statusFilter string) (interface{}, error) {
	if m.listTasksFn != nil {
		return m.listTasksFn(ctx, limit, statusFilter)
	}
	return []map[string]interface{}{
		{"task_id": "task-001", "status": "completed", "type": "deploy"},
		{"task_id": "task-002", "status": "running", "type": "backup"},
	}, nil
}

func (m *mockDeployer) SearchAppLogs(ctx context.Context, appID, keyword string, limit int) (interface{}, error) {
	if m.searchLogsFn != nil {
		return m.searchLogsFn(ctx, appID, keyword, limit)
	}
	return map[string]interface{}{"app_id": appID, "keyword": keyword, "matches": 2, "logs": []string{"line1 error", "line2 error"}}, nil
}

func (m *mockDeployer) UpdateDNSRecord(ctx context.Context, domain, subdomain, recordType, newValue string) (interface{}, error) {
	if m.updateDNSFn != nil {
		return m.updateDNSFn(ctx, domain, subdomain, recordType, newValue)
	}
	return map[string]interface{}{"domain": domain, "subdomain": subdomain, "type": recordType, "value": newValue, "updated": true}, nil
}

func (m *mockDeployer) UpdateCredential(ctx context.Context, credID string, value string) (interface{}, error) {
	if m.updateCredFn != nil {
		return m.updateCredFn(ctx, credID, value)
	}
	return map[string]interface{}{"id": credID, "updated": true}, nil
}

func (m *mockDeployer) UpdateServer(ctx context.Context, serverID string, config map[string]interface{}) (interface{}, error) {
	if m.updateServerFn != nil {
		return m.updateServerFn(ctx, serverID, config)
	}
	return map[string]interface{}{"id": serverID, "updated": true, "config": config}, nil
}

func (m *mockDeployer) CheckDeployReadiness(ctx context.Context, appConfig map[string]interface{}) (interface{}, error) {
	if m.checkReadinessFn != nil {
		return m.checkReadinessFn(ctx, appConfig)
	}
	return map[string]interface{}{"ready": true, "checks": []map[string]interface{}{
		{"name": "docker", "passed": true},
		{"name": "ports", "passed": true},
	}}, nil
}

func (m *mockDeployer) BatchDeploy(ctx context.Context, apps []map[string]interface{}) (interface{}, error) {
	if m.batchDeployFn != nil {
		return m.batchDeployFn(ctx, apps)
	}
	return map[string]interface{}{"total": len(apps), "succeeded": len(apps), "failed": 0}, nil
}

func (m *mockDeployer) BatchBackup(ctx context.Context, appIDs []string) (interface{}, error) {
	if m.batchBackupFn != nil {
		return m.batchBackupFn(ctx, appIDs)
	}
	return map[string]interface{}{"total": len(appIDs), "succeeded": len(appIDs), "failed": 0}, nil
}

func (m *mockDeployer) BatchDNS(ctx context.Context, records []map[string]interface{}) (interface{}, error) {
	if m.batchDNSFn != nil {
		return m.batchDNSFn(ctx, records)
	}
	return map[string]interface{}{"total": len(records), "succeeded": len(records), "failed": 0}, nil
}

func (m *mockDeployer) CheckSystemUpdate(ctx context.Context) (interface{}, error) {
	if m.checkSysUpdateFn != nil {
		return m.checkSysUpdateFn(ctx)
	}
	return map[string]interface{}{
		"current_version": "0.5.0",
		"latest_version":  "0.6.0",
		"update_available": true,
		"release_notes": "Added Web Dashboard",
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

// ========== manage_servers ==========

func TestAddServerSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleAddServer(context.Background(), mock, newRequest(map[string]interface{}{
		"name": "prod-server",
		"host": "1.2.3.4",
		"port": "22",
		"user": "root",
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
	server := parsed["server"].(map[string]interface{})
	if server["name"] != "prod-server" {
		t.Errorf("server.name = %v", server["name"])
	}
}

func TestAddServerMissingName(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleAddServer(context.Background(), mock, newRequest(map[string]interface{}{
		"host": "1.2.3.4",
	}))
	if !result.IsError {
		t.Error("should return error when name is missing")
	}
}

func TestAddServerFailure(t *testing.T) {
	mock := &mockDeployer{
		addServerFn: func(_ context.Context, _, _ string, _ int, _ string) (*ServerInfo, error) {
			return nil, fmt.Errorf("server already exists")
		},
	}
	result, _ := handleAddServer(context.Background(), mock, newRequest(map[string]interface{}{
		"name": "dup",
		"host": "1.1.1.1",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

func TestRemoveServerSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleRemoveServer(context.Background(), mock, newRequest(map[string]interface{}{
		"server_id": "srv-001",
	}))

	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestRemoveServerMissingID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleRemoveServer(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when server_id is missing")
	}
}

func TestTestServerSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleTestServer(context.Background(), mock, newRequest(map[string]interface{}{
		"server_id": "srv-001",
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

func TestTestServerFailure(t *testing.T) {
	mock := &mockDeployer{
		testServerFn: func(_ context.Context, _ string) (interface{}, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	result, _ := handleTestServer(context.Background(), mock, newRequest(map[string]interface{}{
		"server_id": "srv-001",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== manage_credentials ==========

func TestCreateCredentialSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleCreateCredential(context.Background(), mock, newRequest(map[string]interface{}{
		"tenant_id": "tenant-default",
		"name":      "my-ssh-key",
		"type":      "ssh",
		"value":     "ssh-rsa AAAA...",
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
	cred := parsed["credential"].(map[string]interface{})
	if cred["name"] != "my-ssh-key" {
		t.Errorf("credential.name = %v", cred["name"])
	}
}

func TestCreateCredentialMissingName(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleCreateCredential(context.Background(), mock, newRequest(map[string]interface{}{
		"tenant_id": "tenant-default",
		"type":      "ssh",
		"value":     "secret",
	}))
	if !result.IsError {
		t.Error("should return error when name is missing")
	}
}

func TestCreateCredentialFailure(t *testing.T) {
	mock := &mockDeployer{
		createCredFn: func(_ context.Context, _, _, _, _ string) (interface{}, error) {
			return nil, fmt.Errorf("encryption failed")
		},
	}
	result, _ := handleCreateCredential(context.Background(), mock, newRequest(map[string]interface{}{
		"tenant_id": "tenant-default",
		"name":      "my-key",
		"type":      "ssh",
		"value":     "secret",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

func TestListCredentialsSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleListCredentials(context.Background(), mock, newRequest(map[string]interface{}{
		"tenant_id": "tenant-default",
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

func TestListCredentialsFailure(t *testing.T) {
	mock := &mockDeployer{
		listCredsFn: func(_ context.Context, _ string) (interface{}, error) {
			return nil, fmt.Errorf("database error")
		},
	}
	result, _ := handleListCredentials(context.Background(), mock, newRequest(map[string]interface{}{
		"tenant_id": "tenant-default",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

func TestDeleteCredentialSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDeleteCredential(context.Background(), mock, newRequest(map[string]interface{}{
		"credential_id": "cred-001",
	}))

	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestDeleteCredentialMissingID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDeleteCredential(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when credential_id is missing")
	}
}

func TestDeleteCredentialFailure(t *testing.T) {
	mock := &mockDeployer{
		deleteCredFn: func(_ context.Context, _ string) error {
			return fmt.Errorf("credential not found")
		},
	}
	result, _ := handleDeleteCredential(context.Background(), mock, newRequest(map[string]interface{}{
		"credential_id": "nonexistent",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== dns_create_record ==========

func TestDNSCreateRecordSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDNSCreateRecord(context.Background(), mock, newRequest(map[string]interface{}{
		"domain":   "example.com",
		"type":     "A",
		"name":     "@",
		"value":    "1.2.3.4",
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

func TestDNSCreateRecordMissingDomain(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDNSCreateRecord(context.Background(), mock, newRequest(map[string]interface{}{
		"type": "A",
	}))
	if !result.IsError {
		t.Error("should return error when domain is missing")
	}
}

func TestDNSCreateRecordFailure(t *testing.T) {
	mock := &mockDeployer{
		dnsCreateFn: func(_ context.Context, _, _, _, _ string) (interface{}, error) {
			return nil, fmt.Errorf("DNS API error")
		},
	}
	result, _ := handleDNSCreateRecord(context.Background(), mock, newRequest(map[string]interface{}{
		"domain": "example.com", "type": "A", "name": "@", "value": "1.2.3.4",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== dns_delete_record ==========

func TestDNSDeleteRecordSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDNSDeleteRecord(context.Background(), mock, newRequest(map[string]interface{}{
		"record_id": "rec-001",
	}))
	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestDNSDeleteRecordMissingID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDNSDeleteRecord(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when record_id is missing")
	}
}

// ========== dns_list_records ==========

func TestDNSListRecordsSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleDNSListRecords(context.Background(), mock, newRequest(map[string]interface{}{
		"domain": "example.com",
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

func TestDNSListRecordsFailure(t *testing.T) {
	mock := &mockDeployer{
		dnsListFn: func(_ context.Context, _ string) (interface{}, error) {
			return nil, fmt.Errorf("API error")
		},
	}
	result, _ := handleDNSListRecords(context.Background(), mock, newRequest(map[string]interface{}{
		"domain": "example.com",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== send_notification ==========

func TestSendNotificationSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleSendNotification(context.Background(), mock, newRequest(map[string]interface{}{
		"type":    "deploy_success",
		"app":     "my-app",
		"server":  "prod",
		"status":  "success",
		"message": "deployed nginx:latest",
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

func TestSendNotificationMissingType(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleSendNotification(context.Background(), mock, newRequest(map[string]interface{}{
		"app": "my-app",
	}))
	if !result.IsError {
		t.Error("should return error when type is missing")
	}
}

func TestSendNotificationFailure(t *testing.T) {
	mock := &mockDeployer{
		notifySendFn: func(_ context.Context, _, _, _, _, _ string) (interface{}, error) {
			return nil, fmt.Errorf("send failed")
		},
	}
	result, _ := handleSendNotification(context.Background(), mock, newRequest(map[string]interface{}{
		"type": "deploy_failed", "app": "my-app", "server": "prod", "status": "failed", "message": "error",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== list_templates ==========

func TestListTemplatesSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleListTemplates(context.Background(), mock, newRequest(map[string]interface{}{}))

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

func TestListTemplatesFailure(t *testing.T) {
	mock := &mockDeployer{
		templateListFn: func(_ context.Context) (interface{}, error) {
			return nil, fmt.Errorf("template error")
		},
	}
	result, _ := handleListTemplates(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== get_template ==========

func TestGetTemplateSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleGetTemplate(context.Background(), mock, newRequest(map[string]interface{}{
		"type": "node",
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

func TestGetTemplateMissingType(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleGetTemplate(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when type is missing")
	}
}

func TestGetTemplateFailure(t *testing.T) {
	mock := &mockDeployer{
		templateGetFn: func(_ context.Context, _ string) (interface{}, error) {
			return nil, fmt.Errorf("template not found")
		},
	}
	result, _ := handleGetTemplate(context.Background(), mock, newRequest(map[string]interface{}{
		"type": "nonexistent",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== get_app_detail ==========

func TestGetAppDetailSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleGetAppDetail(context.Background(), mock, newRequest(map[string]interface{}{
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
	detail := parsed["app"].(map[string]interface{})
	if detail["id"] != "app-001" {
		t.Errorf("app.id = %v, want app-001", detail["id"])
	}
}

func TestGetAppDetailMissingID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleGetAppDetail(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when app_id is missing")
	}
}

func TestGetAppDetailFailure(t *testing.T) {
	mock := &mockDeployer{
		getAppDetailFn: func(_ context.Context, _ string) (interface{}, error) {
			return nil, fmt.Errorf("app not found")
		},
	}
	result, _ := handleGetAppDetail(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id": "nonexistent",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== update_app ==========

func TestUpdateAppSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleUpdateApp(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id": "app-001",
		"config": "{\"memory\": \"2GB\"}",
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

func TestUpdateAppMissingID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleUpdateApp(context.Background(), mock, newRequest(map[string]interface{}{
		"config": "{}",
	}))
	if !result.IsError {
		t.Error("should return error when app_id is missing")
	}
}

func TestUpdateAppFailure(t *testing.T) {
	mock := &mockDeployer{
		updateAppFn: func(_ context.Context, _ string, _ map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("update failed")
		},
	}
	result, _ := handleUpdateApp(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id": "app-001",
		"config": "{}",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== get_task_status ==========

func TestGetTaskStatusSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleGetTaskStatus(context.Background(), mock, newRequest(map[string]interface{}{
		"task_id": "task-001",
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
	task := parsed["task"].(map[string]interface{})
	if task["task_id"] != "task-001" {
		t.Errorf("task.task_id = %v", task["task_id"])
	}
}

func TestGetTaskStatusMissingID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleGetTaskStatus(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error when task_id is missing")
	}
}

func TestGetTaskStatusFailure(t *testing.T) {
	mock := &mockDeployer{
		getTaskStatusFn: func(_ context.Context, _ string) (interface{}, error) {
			return nil, fmt.Errorf("task not found")
		},
	}
	result, _ := handleGetTaskStatus(context.Background(), mock, newRequest(map[string]interface{}{
		"task_id": "nonexistent",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== list_tasks ==========

func TestListTasksSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleListTasks(context.Background(), mock, newRequest(map[string]interface{}{
		"limit": "10",
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

func TestListTasksWithFilter(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleListTasks(context.Background(), mock, newRequest(map[string]interface{}{
		"limit": "5",
		"status_filter": "running",
	}))

	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestListTasksFailure(t *testing.T) {
	mock := &mockDeployer{
		listTasksFn: func(_ context.Context, _ int, _ string) (interface{}, error) {
			return nil, fmt.Errorf("database error")
		},
	}
	result, _ := handleListTasks(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== search_app_logs ==========

func TestSearchAppLogsSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleSearchAppLogs(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id":  "app-001",
		"keyword": "error",
		"limit":   "10",
	}))
	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestSearchAppLogsMissingAppID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleSearchAppLogs(context.Background(), mock, newRequest(map[string]interface{}{
		"keyword": "error",
	}))
	if !result.IsError {
		t.Error("should return error when app_id is missing")
	}
}

func TestSearchAppLogsFailure(t *testing.T) {
	mock := &mockDeployer{
		searchLogsFn: func(_ context.Context, _, _ string, _ int) (interface{}, error) {
			return nil, fmt.Errorf("search failed")
		},
	}
	result, _ := handleSearchAppLogs(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id": "app-001", "keyword": "error",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== update_dns_record ==========

func TestUpdateDNSRecordSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleUpdateDNSRecord(context.Background(), mock, newRequest(map[string]interface{}{
		"domain": "example.com", "subdomain": "www", "type": "A", "new_value": "2.3.4.5",
	}))
	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestUpdateDNSRecordMissingDomain(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleUpdateDNSRecord(context.Background(), mock, newRequest(map[string]interface{}{
		"type": "A", "new_value": "1.2.3.4",
	}))
	if !result.IsError {
		t.Error("should return error when domain is missing")
	}
}

func TestUpdateDNSRecordFailure(t *testing.T) {
	mock := &mockDeployer{
		updateDNSFn: func(_ context.Context, _, _, _, _ string) (interface{}, error) {
			return nil, fmt.Errorf("DNS API error")
		},
	}
	result, _ := handleUpdateDNSRecord(context.Background(), mock, newRequest(map[string]interface{}{
		"domain": "example.com", "subdomain": "www", "type": "A", "new_value": "1.2.3.4",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== update_credential ==========

func TestUpdateCredentialSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleUpdateCredential(context.Background(), mock, newRequest(map[string]interface{}{
		"credential_id": "cred-001", "value": "new-secret",
	}))
	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestUpdateCredentialMissingID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleUpdateCredential(context.Background(), mock, newRequest(map[string]interface{}{
		"value": "secret",
	}))
	if !result.IsError {
		t.Error("should return error when credential_id is missing")
	}
}

func TestUpdateCredentialFailure(t *testing.T) {
	mock := &mockDeployer{
		updateCredFn: func(_ context.Context, _ string, _ string) (interface{}, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	result, _ := handleUpdateCredential(context.Background(), mock, newRequest(map[string]interface{}{
		"credential_id": "nonexistent", "value": "secret",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== update_server ==========

func TestUpdateServerSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleUpdateServer(context.Background(), mock, newRequest(map[string]interface{}{
		"server_id": "srv-001", "config": "{\"port\": 2222}",
	}))
	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestUpdateServerMissingID(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleUpdateServer(context.Background(), mock, newRequest(map[string]interface{}{
		"config": "{}",
	}))
	if !result.IsError {
		t.Error("should return error when server_id is missing")
	}
}

func TestUpdateServerFailure(t *testing.T) {
	mock := &mockDeployer{
		updateServerFn: func(_ context.Context, _ string, _ map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("server not found")
		},
	}
	result, _ := handleUpdateServer(context.Background(), mock, newRequest(map[string]interface{}{
		"server_id": "nonexistent", "config": "{}",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== check_deploy_readiness ==========

func TestCheckDeployReadinessSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleCheckDeployReadiness(context.Background(), mock, newRequest(map[string]interface{}{
		"app_config": "{\"repo\": \"github.com/user/repo\"}",
	}))
	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestCheckDeployReadinessFailure(t *testing.T) {
	mock := &mockDeployer{
		checkReadinessFn: func(_ context.Context, _ map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("readiness check failed")
		},
	}
	result, _ := handleCheckDeployReadiness(context.Background(), mock, newRequest(map[string]interface{}{
		"app_config": "{}",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== batch_deploy ==========

func TestBatchDeploySuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleBatchDeploy(context.Background(), mock, newRequest(map[string]interface{}{
		"apps": "[{\"repo\":\"github.com/a/b\"},{\"repo\":\"github.com/c/d\"}]",
	}))
	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestBatchDeployInvalidJSON(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleBatchDeploy(context.Background(), mock, newRequest(map[string]interface{}{
		"apps": "invalid json",
	}))
	if !result.IsError {
		t.Error("should return error for invalid JSON")
	}
}

func TestBatchDeployFailure(t *testing.T) {
	mock := &mockDeployer{
		batchDeployFn: func(_ context.Context, _ []map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("batch deploy failed")
		},
	}
	result, _ := handleBatchDeploy(context.Background(), mock, newRequest(map[string]interface{}{
		"apps": "[]",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== batch_backup ==========

func TestBatchBackupSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleBatchBackup(context.Background(), mock, newRequest(map[string]interface{}{
		"app_ids": "[\"app-001\",\"app-002\"]",
	}))
	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestBatchBackupInvalidJSON(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleBatchBackup(context.Background(), mock, newRequest(map[string]interface{}{
		"app_ids": "invalid",
	}))
	if !result.IsError {
		t.Error("should return error for invalid JSON")
	}
}

func TestBatchBackupFailure(t *testing.T) {
	mock := &mockDeployer{
		batchBackupFn: func(_ context.Context, _ []string) (interface{}, error) {
			return nil, fmt.Errorf("batch backup failed")
		},
	}
	result, _ := handleBatchBackup(context.Background(), mock, newRequest(map[string]interface{}{
		"app_ids": "[]",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== batch_dns ==========

func TestBatchDNSSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleBatchDNS(context.Background(), mock, newRequest(map[string]interface{}{
		"records": "[{\"domain\":\"example.com\",\"type\":\"A\",\"value\":\"1.2.3.4\"}]",
	}))
	text, _ := extractText(result)
	if !strings.Contains(text, "success") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestBatchDNSInvalidJSON(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleBatchDNS(context.Background(), mock, newRequest(map[string]interface{}{
		"records": "invalid",
	}))
	if !result.IsError {
		t.Error("should return error for invalid JSON")
	}
}

func TestBatchDNSFailure(t *testing.T) {
	mock := &mockDeployer{
		batchDNSFn: func(_ context.Context, _ []map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("batch DNS failed")
		},
	}
	result, _ := handleBatchDNS(context.Background(), mock, newRequest(map[string]interface{}{
		"records": "[]",
	}))
	if !result.IsError {
		t.Error("should return error on failure")
	}
}

// ========== check_system_update ==========

func TestCheckSystemUpdateSuccess(t *testing.T) {
	mock := &mockDeployer{}
	result, _ := handleCheckSystemUpdate(context.Background(), mock, newRequest(map[string]interface{}{}))

	text, err := extractText(result)
	if err != nil {
		t.Fatalf("extractText error = %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	if parsed["status"] != "success" {
		t.Errorf("status = %v, want success", parsed["status"])
	}
	info := parsed["update"].(map[string]interface{})
	if info["update_available"] != true {
		t.Errorf("update_available = %v, want true", info["update_available"])
	}
}

func TestCheckSystemUpdateFailure(t *testing.T) {
	mock := &mockDeployer{
		checkSysUpdateFn: func(_ context.Context) (interface{}, error) {
			return nil, fmt.Errorf("network error")
		},
	}
	result, _ := handleCheckSystemUpdate(context.Background(), mock, newRequest(map[string]interface{}{}))
	if !result.IsError {
		t.Error("should return error on failure")
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

func TestHandleDoctor(t *testing.T) {
        mock := &mockDeployer{}
        result, err := handleDoctor(context.Background(), mock, newRequest(map[string]interface{}{}))
        if err != nil {
                t.Fatalf("handleDoctor failed: %v", err)
        }
        if result.IsError {
                t.Errorf("doctor should not return error, got: %v", result)
        }
        if len(result.Content) == 0 {
                t.Fatal("doctor should return content")
        }
        text := result.Content[0].(mcp.TextContent).Text
        var parsed map[string]interface{}
        json.Unmarshal([]byte(text), &parsed)
        if parsed["status"] != "ok" {
                t.Errorf("status = %v, want ok", parsed["status"])
        }
        checks, ok := parsed["checks"].([]interface{})
        if !ok || len(checks) != 3 {
                t.Errorf("expected 3 checks, got %v", parsed["checks"])
        }
}
