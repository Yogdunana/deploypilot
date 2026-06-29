package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// MockCredentialManager implements CredentialManager for testing
type mockCredentialManager struct {
	credentials map[string]interface{}
	createErr   error
	listErr     error
	deleteErr   error
	updateErr   error
}

func (m *mockCredentialManager) CreateCredential(ctx context.Context, tenantID, name, credType, plainValue string) (interface{}, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	id := "cred-" + name
	m.credentials[id] = map[string]interface{}{
		"id":        id,
		"tenant_id": tenantID,
		"name":      name,
		"type":      credType,
	}
	return m.credentials[id], nil
}

func (m *mockCredentialManager) ListCredentials(ctx context.Context, tenantID string) (interface{}, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	result := make([]map[string]interface{}, 0)
	for _, c := range m.credentials {
		cred := c.(map[string]interface{})
		if cred["tenant_id"] == tenantID {
			result = append(result, cred)
		}
	}
	return result, nil
}

func (m *mockCredentialManager) DeleteCredential(ctx context.Context, credID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.credentials[credID]; !ok {
		return errors.New("credential not found")
	}
	delete(m.credentials, credID)
	return nil
}

func (m *mockCredentialManager) UpdateCredential(ctx context.Context, credID string, value string) (interface{}, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if _, ok := m.credentials[credID]; !ok {
		return nil, errors.New("credential not found")
	}
	return map[string]interface{}{
		"id":      credID,
		"status":  "updated",
		"message": "credential value updated",
	}, nil
}

func TestHandleCreateCredential_Success(t *testing.T) {
	mock := &mockCredentialManager{
		credentials: make(map[string]interface{}),
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"tenant_id": "tenant-1",
				"name":      "db-password",
				"type":      "password",
				"value":     "secret123",
			},
		},
	}

	result, err := handleCreateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected success result")
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["status"] != "success" {
		t.Errorf("expected status=success, got %v", response["status"])
	}
}

func TestHandleCreateCredential_MissingTenantID(t *testing.T) {
	mock := &mockCredentialManager{}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name":  "db-password",
				"type":  "password",
				"value": "secret123",
			},
		},
	}

	result, err := handleCreateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing tenant_id")
	}
}

func TestHandleCreateCredential_MissingName(t *testing.T) {
	mock := &mockCredentialManager{}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"tenant_id": "tenant-1",
				"type":      "password",
				"value":     "secret123",
			},
		},
	}

	result, err := handleCreateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing name")
	}
}

func TestHandleCreateCredential_MissingType(t *testing.T) {
	mock := &mockCredentialManager{}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"tenant_id": "tenant-1",
				"name":      "db-password",
				"value":     "secret123",
			},
		},
	}

	result, err := handleCreateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing type")
	}
}

func TestHandleCreateCredential_MissingValue(t *testing.T) {
	mock := &mockCredentialManager{}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"tenant_id": "tenant-1",
				"name":      "db-password",
				"type":      "password",
			},
		},
	}

	result, err := handleCreateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing value")
	}
}

func TestHandleCreateCredential_ServiceError(t *testing.T) {
	mock := &mockCredentialManager{
		createErr: errors.New("database connection failed"),
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"tenant_id": "tenant-1",
				"name":      "db-password",
				"type":      "password",
				"value":     "secret123",
			},
		},
	}

	result, err := handleCreateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for service error")
	}
}

func TestHandleListCredentials_Success(t *testing.T) {
	mock := &mockCredentialManager{
		credentials: map[string]interface{}{
			"cred-1": map[string]interface{}{
				"id":        "cred-1",
				"tenant_id": "tenant-1",
				"name":      "password1",
				"type":      "password",
			},
			"cred-2": map[string]interface{}{
				"id":        "cred-2",
				"tenant_id": "tenant-1",
				"name":      "password2",
				"type":      "api-key",
			},
		},
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"tenant_id": "tenant-1",
			},
		},
	}

	result, err := handleListCredentials(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected success result")
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["status"] != "success" {
		t.Errorf("expected status=success, got %v", response["status"])
	}
}

func TestHandleListCredentials_MissingTenantID(t *testing.T) {
	mock := &mockCredentialManager{}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handleListCredentials(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing tenant_id")
	}
}

func TestHandleListCredentials_ServiceError(t *testing.T) {
	mock := &mockCredentialManager{
		listErr: errors.New("database error"),
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"tenant_id": "tenant-1",
			},
		},
	}

	result, err := handleListCredentials(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for service error")
	}
}

func TestHandleDeleteCredential_Success(t *testing.T) {
	mock := &mockCredentialManager{
		credentials: map[string]interface{}{
			"cred-1": map[string]interface{}{
				"id":        "cred-1",
				"tenant_id": "tenant-1",
				"name":      "password1",
				"type":      "password",
			},
		},
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"credential_id": "cred-1",
			},
		},
	}

	result, err := handleDeleteCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected success result")
	}

	// Verify credential was deleted
	if len(mock.credentials) != 0 {
		t.Errorf("expected credential to be deleted, but still has %d credentials", len(mock.credentials))
	}
}

func TestHandleDeleteCredential_MissingCredentialID(t *testing.T) {
	mock := &mockCredentialManager{}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handleDeleteCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing credential_id")
	}
}

func TestHandleDeleteCredential_NotFound(t *testing.T) {
	mock := &mockCredentialManager{
		deleteErr: errors.New("credential not found"),
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"credential_id": "non-existent",
			},
		},
	}

	result, err := handleDeleteCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for not found")
	}
}

func TestHandleUpdateCredential_Success(t *testing.T) {
	mock := &mockCredentialManager{
		credentials: map[string]interface{}{
			"cred-1": map[string]interface{}{
				"id":        "cred-1",
				"tenant_id": "tenant-1",
				"name":      "password1",
				"type":      "password",
			},
		},
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"credential_id": "cred-1",
				"value":        "new-secret",
			},
		},
	}

	result, err := handleUpdateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("expected success result")
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["status"] != "success" {
		t.Errorf("expected status=success, got %v", response["status"])
	}
}

func TestHandleUpdateCredential_MissingCredentialID(t *testing.T) {
	mock := &mockCredentialManager{}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"value": "new-secret",
			},
		},
	}

	result, err := handleUpdateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing credential_id")
	}
}

func TestHandleUpdateCredential_MissingValue(t *testing.T) {
	mock := &mockCredentialManager{}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"credential_id": "cred-1",
			},
		},
	}

	result, err := handleUpdateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing value")
	}
}

func TestHandleUpdateCredential_NotFound(t *testing.T) {
	mock := &mockCredentialManager{
		updateErr: errors.New("credential not found"),
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"credential_id": "non-existent",
				"value":         "new-secret",
			},
		},
	}

	result, err := handleUpdateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for not found")
	}
}

func TestHandleUpdateCredential_ServiceError(t *testing.T) {
	mock := &mockCredentialManager{
		updateErr: errors.New("encryption failed"),
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"credential_id": "cred-1",
				"value":         "new-secret",
			},
		},
	}

	result, err := handleUpdateCredential(context.Background(), mock, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for service error")
	}
}
