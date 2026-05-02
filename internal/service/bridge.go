package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/Yogdunana/deploypilot/internal/agent"
	"github.com/Yogdunana/deploypilot/internal/bruteforce"
	"github.com/Yogdunana/deploypilot/internal/confirm"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/engine/healer"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/monitor"
	"github.com/Yogdunana/deploypilot/internal/plugin"
	"github.com/Yogdunana/deploypilot/internal/provider/server"
	"github.com/Yogdunana/deploypilot/internal/sandbox"
	"gorm.io/gorm"
)

// PreflightError represents a structured preflight failure.
type PreflightError struct {
	Code    PreflightErrorCode `json:"code"`
	Message string             `json:"message"`
	Checks  []PreflightCheck   `json:"checks"`
}

func (e *PreflightError) Error() string {
	return fmt.Sprintf("preflight failed [%s]: %s", e.Code, e.Message)
}

// PreflightCode returns the error code as a string (implements mcp.PreflightErrorInfo).
func (e *PreflightError) PreflightCode() string {
	return string(e.Code)
}

// PreflightMessage returns the error message (implements mcp.PreflightErrorInfo).
func (e *PreflightError) PreflightMessage() string {
	return e.Message
}

// PreflightChecks returns the individual check results (implements mcp.PreflightErrorInfo).
func (e *PreflightError) PreflightChecks() interface{} {
	return e.Checks
}

// DeployEvent represents a deploy progress event.
type DeployEvent struct {
	TaskID    string `json:"task_id"`
	AppID     string `json:"app_id"`
	Step      string `json:"step"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	TraceID   string `json:"trace_id,omitempty"`
}

// Bridge implements mcp.Deployer by wiring DB + Docker executor.
type Bridge struct {
	DB            *gorm.DB
	Executor      deployer.CommandExecutor
	EncryptionKey []byte
	Monitor       *monitor.Monitor
	healer        *healer.Healer
	EventBus      EventBus
	TypedBus      TypedEventBus
	TunnelManager TunnelManager
	PluginMgr     *plugin.Manager
	UpgradeSvc    *UpgradeService
	ConfirmStore  *confirm.Store
	BFProtector   *bruteforce.Protector
	Cache         Cache
	Scheduler     *Scheduler
	uptimeSvc     *MonitorService

	// Task tracking (moved from package-level globals, Issue #117)
	taskMu      sync.RWMutex
	tasks       map[string]*taskInfo
	taskCounter int64

	// Backup tracking (moved from package-level globals, Issue #117)
	backupMu   sync.RWMutex
	backupApps map[string]string

	// Port forwarding (moved from package-level globals, Issue #117)
	portForwardMu sync.RWMutex
	portForwards  map[string]*portForwardEntry
}

// TunnelManager defines the interface for agent reverse tunnel management.
type TunnelManager interface {
	HandleTunnel(w http.ResponseWriter, r *http.Request, serverID string)
	IsConnected(serverID string) bool
	ListAgents() []string
	ExecuteCommand(ctx context.Context, serverID, cmd string, timeout time.Duration) (*agent.CommandResult, error)
}

// CommandResult is an alias for agent.CommandResult for convenience.
type CommandResult = agent.CommandResult

// NewBridge creates a new Bridge that satisfies the mcp.Deployer interface.
func NewBridge(db *gorm.DB, executor deployer.CommandExecutor, encryptionKey []byte, eventBus EventBus) *Bridge {
	if eventBus == nil {
		eventBus = NewInMemoryEventBus()
	}
	return &Bridge{
		DB:            db,
		Executor:      executor,
		EncryptionKey: encryptionKey,
		EventBus:      eventBus,
		ConfirmStore:  confirm.NewStore(),
		BFProtector:   bruteforce.New(bruteforce.DefaultConfig()),
		tasks:         make(map[string]*taskInfo),
		backupApps:    make(map[string]string),
		portForwards:  make(map[string]*portForwardEntry),
	}
}

// SetCache sets the cache on the Bridge.
func (b *Bridge) SetCache(cache Cache) {
	b.Cache = cache
}

// SetTypedBus sets the typed event bus on the Bridge.
func (b *Bridge) SetTypedBus(bus TypedEventBus) {
	b.TypedBus = bus
}

// SetBruteForceConfig replaces the brute-force protector with one using the given config.
func (b *Bridge) SetBruteForceConfig(cfg bruteforce.Config) {
	b.BFProtector = bruteforce.New(cfg)
}

// BruteForceConfigFromMap creates a bruteforce.Config from raw config values.
func BruteForceConfigFromMap(maxAttempts, ipMaxAttempts int, lockoutDuration, windowDuration, baseDelay, maxDelay, ipLockoutDuration string, progressiveDelay bool) bruteforce.Config {
	return bruteforce.Config{
		MaxAttempts:       maxAttempts,
		LockoutDuration:   parseDuration(lockoutDuration, 15*time.Minute),
		WindowDuration:    parseDuration(windowDuration, 15*time.Minute),
		ProgressiveDelay:  progressiveDelay,
		BaseDelay:         parseDuration(baseDelay, 1*time.Second),
		MaxDelay:          parseDuration(maxDelay, 30*time.Second),
		IPMaxAttempts:     ipMaxAttempts,
		IPLockoutDuration: parseDuration(ipLockoutDuration, 30*time.Minute),
	}
}

// parseDuration parses a duration string, returning the fallback on error.
func parseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

// d returns a deployer.DockerDeployer backed by the bridge's executor.
func (b *Bridge) d() *deployer.DockerDeployer {
	return deployer.New(b.Executor)
}

// GetSandbox returns the sandbox instance if the executor is a SandboxExecutor.
func (b *Bridge) GetSandbox() interface {
	GetConfig() sandbox.Config
	Validate(cmd string) error
	AddRule(rule sandbox.Rule) error
	RemoveRule(ruleID string)
	ToggleRule(ruleID string, enabled bool)
	UpdateConfig(cfg sandbox.Config) error
} {
	if se, ok := b.Executor.(*deployer.SandboxExecutor); ok {
		return se.Sandbox()
	}
	return nil
}

// GetConfirmationStore returns the confirmation store.
func (b *Bridge) GetConfirmationStore() *confirm.Store {
	return b.ConfirmStore
}

// getRemoteExecutor creates an SSH executor for the given server.
func (b *Bridge) getRemoteExecutor(ctx context.Context, serverID string) (*sshClientExecutor, error) {
	row := make(map[string]interface{})
	if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	host := toString(row["host"])
	port := toInt(row["port"])

	var password, keyStr string
	if credID := toString(row["credential_id"]); credID != "" {
		credRow := make(map[string]interface{})
		if err := b.DB.Table("credentials").Where("id = ?", credID).Take(&credRow).Error; err == nil {
			encrypted := toString(credRow["encrypted_value"])
			if b.EncryptionKey != nil && encrypted != "" {
				if decrypted, err := crypto.Decrypt(b.EncryptionKey, encrypted); err == nil {
					password = decrypted
				}
			}
		}
	}

	username := toString(row["username"])
	if username == "" {
		username = os.Getenv("DEPLOYPILOT_SSH_DEFAULT_USER")
	}
	if username == "" {
		slog.Warn("SSH username not configured, falling back to root", "serverID", serverID)
		username = "root"
	}
	cfg := server.Config{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		KeyBytes: []byte(keyStr),
		Timeout:  30 * time.Second,
	}

	client, err := server.Connect(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("SSH connection failed to %s:%d: %w", host, port, err)
	}

	return &sshClientExecutor{Client: client}, nil
}

// RemoteExecutor is the interface for remote command execution.
type RemoteExecutor interface {
	RunCommand(ctx context.Context, cmd string) (string, error)
	CreateInteractiveSession(ctx context.Context, termType string, rows, cols int) (InteractiveSession, error)
	Close() error
}

// InteractiveSession represents a persistent interactive SSH session with PTY.
type InteractiveSession interface {
	StdinPipe() io.Writer
	SetWindowSize(rows, cols int) error
	Output() <-chan []byte
	Done() <-chan struct{}
	Close() error
}

// sshClientExecutor wraps server.Client to implement deployer.CommandExecutor.
type sshClientExecutor struct {
	Client *server.Client
}

func (e *sshClientExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	return e.Client.RunCommand(ctx, cmd)
}

func (e *sshClientExecutor) CreateInteractiveSession(ctx context.Context, termType string, rows, cols int) (InteractiveSession, error) {
	session, err := e.Client.CreateSession(ctx, true, termType, rows, cols)
	if err != nil {
		return nil, err
	}
	return newInteractiveSession(session), nil
}

func (e *sshClientExecutor) Close() error {
	return e.Client.Close()
}

// interactiveSession wraps an ssh.Session to implement InteractiveSession.
type interactiveSession struct {
	session *ssh.Session
	stdin   io.WriteCloser
	output  chan []byte
	done    chan struct{}
}

func newInteractiveSession(session *ssh.Session) *interactiveSession {
	is := &interactiveSession{
		session: session,
		output:  make(chan []byte, 256),
		done:    make(chan struct{}),
	}
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		close(is.done)
		return is
	}
	is.stdin = stdinPipe

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		close(is.done)
		return is
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		close(is.done)
		return is
	}

	if err := session.Shell(); err != nil {
		close(is.done)
		return is
	}

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				is.output <- data
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderrPipe.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				is.output <- data
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		_ = session.Wait()
		close(is.done)
	}()

	return is
}

func (is *interactiveSession) StdinPipe() io.Writer {
	return is.stdin
}

func (is *interactiveSession) SetWindowSize(rows, cols int) error {
	return is.session.WindowChange(rows, cols)
}

func (is *interactiveSession) Output() <-chan []byte {
	return is.output
}

func (is *interactiveSession) Done() <-chan struct{} {
	return is.done
}

func (is *interactiveSession) Close() error {
	_ = is.stdin.Close()
	return is.session.Close()
}

// generateID returns a unique deployment record ID.
func generateID() string {
	return fmt.Sprintf("dep-%d", time.Now().UnixNano())
}

// logPreflightResult logs structured preflight check results.
func logPreflightResult(containerName string, result *PreflightResult) {
	slog.Info("preflight result", "container", containerName, "passed", result.Passed, "code", result.Code, "message", result.Message)
	for _, c := range result.Checks {
		slog.Info("preflight check", "name", c.Name, "passed", c.Passed, "message", c.Message, "suggestion", c.Suggestion)
	}
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func toStringOrDefault(v interface{}, def string) string {
	s := toString(v)
	if s == "" {
		return def
	}
	return s
}

func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

func defaultVal(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

// ========== SchedulerService interface (stubs) ==========

func (b *Bridge) CreateScheduledTask(ctx context.Context, name, cronExpr, taskType, command string, serverID string) (interface{}, error) {
	return nil, fmt.Errorf("scheduled tasks: use Scheduler directly")
}

func (b *Bridge) ListScheduledTasks(ctx context.Context) (interface{}, error) {
	return nil, fmt.Errorf("scheduled tasks: use Scheduler directly")
}

func (b *Bridge) GetTaskExecutions(ctx context.Context, taskID string, limit int) (interface{}, error) {
	return nil, fmt.Errorf("scheduled tasks: use Scheduler directly")
}

func (b *Bridge) ToggleScheduledTask(ctx context.Context, taskID string, enabled bool) (interface{}, error) {
	return nil, fmt.Errorf("scheduled tasks: use Scheduler directly")
}

func (b *Bridge) DeleteScheduledTask(ctx context.Context, taskID string) (interface{}, error) {
	return nil, fmt.Errorf("scheduled tasks: use Scheduler directly")
}

// ========== MonitorService interface (stubs) ==========

func (b *Bridge) GetSystemMetrics(ctx context.Context) (interface{}, error) {
	if b.Monitor != nil {
		return b.Monitor.GetSystemMetrics(ctx)
	}
	return map[string]interface{}{
		"cpu": "0%", "memory": "0MB", "disk": "0MB", "load": "0.0 0.0 0.0",
	}, nil
}

func (b *Bridge) GetContainerMetrics(ctx context.Context, name string) (interface{}, error) {
	if b.Monitor != nil {
		return b.Monitor.GetContainerMetrics(ctx, name)
	}
	return map[string]interface{}{
		"name": name, "cpu": "0%", "memory": "0MB",
	}, nil
}

func (b *Bridge) ListAlerts(ctx context.Context) (interface{}, error) {
	if b.Monitor != nil {
		return b.Monitor.ListAlerts(ctx)
	}
	return []interface{}{}, nil
}

func (b *Bridge) ListAlertRules(ctx context.Context) (interface{}, error) {
	if b.Monitor != nil {
		return b.Monitor.ListAlertRules(ctx)
	}
	return []interface{}{}, nil
}

func (b *Bridge) GetRemoteSystemMetrics(ctx context.Context, serverID string) (interface{}, error) {
	exec, err := b.getRemoteExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	result := make(map[string]interface{})
	if output, err := exec.RunCommand(ctx, "free -m 2>/dev/null"); err == nil {
		result["memory"] = strings.TrimSpace(output)
	}
	if output, err := exec.RunCommand(ctx, "df -h --total 2>/dev/null | tail -1"); err == nil {
		result["disk"] = strings.TrimSpace(output)
	}
	if output, err := exec.RunCommand(ctx, "cat /proc/loadavg 2>/dev/null"); err == nil {
		result["load_average"] = strings.TrimSpace(output)
	}
	if output, err := exec.RunCommand(ctx, "uptime -p 2>/dev/null || uptime 2>/dev/null"); err == nil {
		result["uptime"] = strings.TrimSpace(output)
	}
	return result, nil
}

func (b *Bridge) QueryMetricHistory(ctx context.Context, metricType string, duration string) (interface{}, error) {
	return b.Monitor.QueryMetricHistory(ctx, metricType, duration)
}

func (b *Bridge) QueryAlertHistory(ctx context.Context, status string, limit int) (interface{}, error) {
	return b.Monitor.QueryAlertHistory(ctx, status, limit)
}
