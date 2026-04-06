package errors

import "fmt"

// AppError is the unified application error type with code, message, suggestion, and cause.
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	Err        error  `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap implements errors.Unwrap for error chain traversal.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithCause sets the underlying error cause and returns the AppError for chaining.
func (e *AppError) WithCause(err error) *AppError {
	e.Err = err
	return e
}

// New creates a new AppError with the given code, message, and suggestion.
func New(code, message, suggestion string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
	}
}

// IsAppError checks if an error is an AppError.
func IsAppError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*AppError)
	return ok
}

// ErrorCode extracts the error code from an AppError, or returns empty string.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ""
}

// FormatForLog formats an AppError for structured logging.
func FormatForLog(err error) string {
	if appErr, ok := err.(*AppError); ok {
		if appErr.Err != nil {
			return fmt.Sprintf("[%s] %s | suggestion: %s", appErr.Code, appErr.Error(), appErr.Suggestion)
		}
		return fmt.Sprintf("[%s] %s | suggestion: %s", appErr.Code, appErr.Message, appErr.Suggestion)
	}
	return err.Error()
}

// ========== Predefined Errors (E001-E017) ==========

var (
	// E001: Deploy failed
	ErrDeployFailed = New("E001", "deployment failed", "check docker logs and container status, verify image exists")

	// E002: SSH connection failed
	ErrSSHConnectFailed = New("E002", "SSH connection failed", "verify server address, port, SSH key, and network connectivity")

	// E003: Container not found
	ErrContainerNotFound = New("E003", "container not found", "verify container name, check if it was removed or renamed")

	// E004: Docker image pull failed
	ErrImagePullFailed = New("E004", "docker image pull failed", "verify image name/tag, check registry credentials and network")

	// E005: Health check failed
	ErrHealthCheckFailed = New("E005", "health check failed", "verify the service is running, check health endpoint URL and port")

	// E006: Rollback failed
	ErrRollbackFailed = New("E006", "rollback failed", "manually stop the container and redeploy the previous image version")

	// E007: Configuration not found
	ErrConfigNotFound = New("E007", "configuration not found", "verify config file path, check environment variables for config location")

	// E008: Invalid configuration
	ErrConfigInvalid = New("E008", "invalid configuration", "check required fields, verify config file format and values")

	// E009: Database connection failed
	ErrDBConnectionFailed = New("E009", "database connection failed", "verify database URL, credentials, and that the database server is reachable")

	// E010: Database migration failed
	ErrMigrationFailed = New("E010", "database migration failed", "check migration files, verify database permissions, restore from backup if needed")

	// E011: Application not found
	ErrAppNotFound = New("E011", "application not found", "verify application ID, check if it was deleted")

	// E012: Application already exists
	ErrAppAlreadyExists = New("E012", "application already exists", "use a different name or update the existing application")

	// E013: Credential not found
	ErrCredentialNotFound = New("E013", "credential not found", "verify credential ID, check encryption key is correct")

	// E014: Credential decryption failed
	ErrCredentialDecryptFailed = New("E014", "credential decryption failed", "verify encryption key matches the one used during encryption")

	// E015: DNS record not found
	ErrDNSRecordNotFound = New("E015", "DNS record not found", "verify domain, record type, and record name")

	// E016: DNS provider error
	ErrDNSProviderError = New("E016", "DNS provider error", "verify API token, check provider status page, retry later")

	// E017: Permission denied
	ErrPermissionDenied = New("E017", "permission denied", "verify user role and permissions, contact administrator if needed")
)
