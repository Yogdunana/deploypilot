package mcp

import (
	"context"
	"testing"
)

func TestHandleBuildAndDeployWithEnvVars(t *testing.T) {
	tests := []struct {
		name      string
		envVars   string
		wantError bool
	}{
		{
			name:      "valid_json_env_vars",
			envVars:   `{"KEY1":"value1","KEY2":"value2"}`,
			wantError: false,
		},
		{
			name:      "empty_env_vars",
			envVars:   "",
			wantError: false,
		},
		{
			name:      "invalid_json_env_vars",
			envVars:   "not-json",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDeployer{}

			args := map[string]interface{}{
				"repo_url": "https://github.com/test/repo.git",
				"app_name": "test-app",
			}
			if tt.envVars != "" {
				args["env_vars"] = tt.envVars
			}

			result, _ := handleBuildAndDeploy(context.Background(), mock, newRequest(args))

			if tt.wantError {
				if !result.IsError {
					t.Error("expected error for invalid env_vars")
				}
			} else {
				if result.IsError {
					t.Errorf("unexpected error result: %v", result)
				}
			}
		})
	}
}

func TestHandleBuildAndDeployPushImage(t *testing.T) {
	tests := []struct {
		name    string
		pushVal string
		wantErr bool
	}{
		{"true", "true", false},
		{"false", "false", false},
		{"TRUE_uppercase", "TRUE", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDeployer{}

			args := map[string]interface{}{
				"repo_url":   "https://github.com/test/repo.git",
				"app_name":   "test-app",
				"push_image": tt.pushVal,
			}

			result, _ := handleBuildAndDeploy(context.Background(), mock, newRequest(args))

			if tt.wantErr && !result.IsError {
				t.Errorf("expected error")
			}
		})
	}
}

func TestHandleComposeDeployValidation(t *testing.T) {
	tests := []struct {
		name    string
		appID   string
		wantErr bool
	}{
		{
			name:    "valid_app_id",
			appID:   "app-123",
			wantErr: false,
		},
		{
			name:    "missing_app_id",
			appID:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDeployer{}

			args := map[string]interface{}{}
			if tt.appID != "" {
				args["app_id"] = tt.appID
			}

			result, _ := handleComposeDeploy(context.Background(), mock, newRequest(args))

			if tt.wantErr {
				if !result.IsError {
					t.Errorf("expected error result, got success")
				}
			}
		})
	}
}

func TestHandleComposeStopSuccess(t *testing.T) {
	mock := &mockDeployer{}

	result, err := handleComposeStop(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id": "app-123",
	}))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestHandleComposePsSuccess(t *testing.T) {
	mock := &mockDeployer{}

	result, err := handleComposePs(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id": "app-123",
	}))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestHandleComposeLogsValidation(t *testing.T) {
	tests := []struct {
		name    string
		service string
		tail    string
	}{
		{
			name:    "with_service_and_tail",
			service: "web",
			tail:    "100",
		},
		{
			name:    "without_params",
			service: "",
			tail:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDeployer{}

			args := map[string]interface{}{
				"app_id": "app-123",
			}
			if tt.service != "" {
				args["service"] = tt.service
			}
			if tt.tail != "" {
				args["tail"] = tt.tail
			}

			result, err := handleComposeLogs(context.Background(), mock, newRequest(args))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

func TestHandleComposeRestartSuccess(t *testing.T) {
	mock := &mockDeployer{}

	result, err := handleComposeRestart(context.Background(), mock, newRequest(map[string]interface{}{
		"app_id":  "app-123",
		"service": "web",
	}))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}
