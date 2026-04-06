package errors

import (
	stderrors "errors"
	"fmt"
	"testing"
)

// ========== AppError Tests ==========

func TestNewAppError(t *testing.T) {
	err := New("E001", "deploy failed", "check docker logs")

	if err.Code != "E001" {
		t.Errorf("Code = %q, want E001", err.Code)
	}
	if err.Message != "deploy failed" {
		t.Errorf("Message = %q", err.Message)
	}
	if err.Suggestion != "check docker logs" {
		t.Errorf("Suggestion = %q", err.Suggestion)
	}
	if err.Err != nil {
		t.Errorf("Err should be nil, got %v", err.Err)
	}
}

func TestNewAppErrorWithCause(t *testing.T) {
	cause := stdlibError("connection refused")
	err := New("E002", "ssh failed", "check server address").WithCause(cause)

	if err.Err != cause {
		t.Errorf("Err = %v, want %v", err.Err, cause)
	}
}

func TestAppErrorImplementsError(t *testing.T) {
	var _ error = New("E001", "test", "fix it")
}

func TestAppErrorErrorMessage(t *testing.T) {
	err := New("E001", "deploy failed", "check logs")
	if err.Error() != "deploy failed" {
		t.Errorf("Error() = %q, want %q", err.Error(), "deploy failed")
	}
}

func TestAppErrorWithCauseErrorMessage(t *testing.T) {
	cause := stdlibError("timeout")
	err := New("E001", "ssh failed", "retry").WithCause(cause)

	msg := err.Error()
	if msg != "ssh failed: timeout" {
		t.Errorf("Error() = %q, want %q", msg, "ssh failed: timeout")
	}
}

func TestAppErrorUnwrap(t *testing.T) {
	cause := stdlibError("root cause")
	err := New("E001", "wrapper", "fix").WithCause(cause)

	if !stderrors.Is(err, cause) {
		t.Error("stderrors.Is should match root cause")
	}
}

func TestAppErrorUnwrapNil(t *testing.T) {
	err := New("E001", "no cause", "fix")

	if stderrors.Unwrap(err) != nil {
		t.Error("Unwrap should return nil when no cause")
	}
}

// ========== Predefined Errors ==========

func TestPredefinedErrorsExist(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		code string
	}{
		{"DeployFailed", ErrDeployFailed, "E001"},
		{"SSHConnectFailed", ErrSSHConnectFailed, "E002"},
		{"ContainerNotFound", ErrContainerNotFound, "E003"},
		{"ImagePullFailed", ErrImagePullFailed, "E004"},
		{"HealthCheckFailed", ErrHealthCheckFailed, "E005"},
		{"RollbackFailed", ErrRollbackFailed, "E006"},
		{"ConfigNotFound", ErrConfigNotFound, "E007"},
		{"ConfigInvalid", ErrConfigInvalid, "E008"},
		{"DBConnectionFailed", ErrDBConnectionFailed, "E009"},
		{"MigrationFailed", ErrMigrationFailed, "E010"},
		{"AppNotFound", ErrAppNotFound, "E011"},
		{"AppAlreadyExists", ErrAppAlreadyExists, "E012"},
		{"CredentialNotFound", ErrCredentialNotFound, "E013"},
		{"CredentialDecryptFailed", ErrCredentialDecryptFailed, "E014"},
		{"DNSRecordNotFound", ErrDNSRecordNotFound, "E015"},
		{"DNSProviderError", ErrDNSProviderError, "E016"},
		{"PermissionDenied", ErrPermissionDenied, "E017"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Fatalf("predefined error %s is nil", tc.name)
			}
			if tc.err.Code != tc.code {
				t.Errorf("Code = %q, want %q", tc.err.Code, tc.code)
			}
			if tc.err.Message == "" {
				t.Errorf("Message should not be empty for %s", tc.name)
			}
			if tc.err.Suggestion == "" {
				t.Errorf("Suggestion should not be empty for %s", tc.name)
			}
		})
	}
}

// ========== WithCause Chain ==========

func TestWithCauseChain(t *testing.T) {
	root := stdlibError("network unreachable")
	mid := New("E002", "ssh failed", "check network").WithCause(root)
	top := New("E001", "deploy failed", "retry deployment").WithCause(mid)

	if !stderrors.Is(top, root) {
		t.Error("stderrors.Is should traverse full chain to root")
	}
	if !stderrors.Is(top, mid) {
		t.Error("stderrors.Is should match intermediate error")
	}
}

// ========== Format ==========

func TestFormatForLog(t *testing.T) {
	err := New("E001", "deploy failed", "check docker logs")
	output := FormatForLog(err)

	if output != "[E001] deploy failed | suggestion: check docker logs" {
		t.Errorf("FormatForLog() = %q", output)
	}
}

func TestFormatForLogWithCause(t *testing.T) {
	cause := stdlibError("timeout after 30s")
	err := New("E001", "deploy failed", "check docker logs").WithCause(cause)
	output := FormatForLog(err)

	if output != "[E001] deploy failed: timeout after 30s | suggestion: check docker logs" {
		t.Errorf("FormatForLog() = %q", output)
	}
}

// ========== IsAppError ==========

func TestIsAppError(t *testing.T) {
	appErr := New("E001", "test", "fix")

	if !IsAppError(appErr) {
		t.Error("IsAppError should return true for AppError")
	}
	if IsAppError(stdlibError("plain error")) {
		t.Error("IsAppError should return false for plain error")
	}
	if IsAppError(nil) {
		t.Error("IsAppError should return false for nil")
	}
}

// ========== ErrorCode ==========

func TestErrorCode(t *testing.T) {
	appErr := New("E005", "health check failed", "check endpoint")

	if ErrorCode(appErr) != "E005" {
		t.Errorf("ErrorCode() = %q, want E005", ErrorCode(appErr))
	}
	if ErrorCode(stdlibError("plain")) != "" {
		t.Errorf("ErrorCode() for plain error should be empty")
	}
	if ErrorCode(nil) != "" {
		t.Errorf("ErrorCode() for nil should be empty")
	}
}

// stdlibError creates a standard library error (avoids package name conflict).
func stdlibError(msg string) error {
	return fmt.Errorf("%s", msg)
}
