package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// mockDeployer implements Deployer for testing.
type mockDeployer struct {
	deployFn func(ctx context.Context, cfg DeployConfig) (*ContainerStatus, error)
	statusFn func(ctx context.Context, name string) (*ContainerStatus, error)
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

func (m *mockDeployer) Stop(_ context.Context, _ string) error  { return nil }
func (m *mockDeployer) Remove(_ context.Context, _ string) error { return nil }
func (m *mockDeployer) GetContainerLogs(_ context.Context, _ string, _ int) (string, error) {
	return "logs", nil
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
