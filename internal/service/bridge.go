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
	Step      string `json:"step"`     // preflight, pull, stop, run, health_check, done
	Status    string `json:"status"`   // running, success, failed
	Progress  int    `json:"progress"` // 0-100
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	TraceID   string `json:"trace_id,omitempty"`
}

// Bridge implements mcp.Deployer by wiring DB + Docker executor.
type Bridge struct {
	DB            *gorm.DB
	Executor      deployer.CommandExecutor // can be SSH client or local shell
	EncryptionKey []byte                   // AES-256 key for credential encryption
	Monitor       *monitor.Monitor         // monitoring system (lazy-initialized)
	healer        *healer.Healer           // self-healing engine (lazy-initialized)
	EventBus      EventBus                 // pub/sub for deploy progress events
	TypedBus      TypedEventBus            // typed event bus for alerts, notifications, system events
	TunnelManager TunnelManager            // agent reverse tunnel manager
	PluginMgr     *plugin.Manager          // plugin lifecycle manager
	UpgradeSvc    *UpgradeService          // system upgrade service
	ConfirmStore  *confirm.Store
	BFProtector   *bruteforce.Protector
	Cache         Cache                    // general-purpose cache (Redis or in-memory)
	Scheduler     *Scheduler               // scheduled task system
	uptimeSvc     *MonitorService          // uptime monitoring service

	// Task tracking (moved from package-level globals, Issue #117)
	taskMu      sync.RWMutex
	tasks       map[string]*taskInfo
	taskCounter int64

	// Backup tracking (moved from package-level globals, Issue #117)
	backupMu   sync.RWMutex
	backupApps map[string]string // backupID -> appID

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
// Durations are parsed from strings; invalid values fall back to defaults.
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

// getDNSProvider loads the first enabled DNS provider from DB and returns a DNSProvider interface.

// getRemoteExecutor creates an SSH executor for the given server.
// It looks up the server record, finds its credential, decrypts the password/key,
// and returns an SSH client that satisfies deployer.CommandExecutor.
func (b *Bridge) getRemoteExecutor(ctx context.Context, serverID string) (*sshClientExecutor, error) {
	row := make(map[string]interface{})
	if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	host := toString(row["host"])
	port := toInt(row["port"])

	// Look up credential if associated
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

	// SSH username: prefer server record field, fall back to env var
	username := toString(row["username"])
	if username == "" {
		username = os.Getenv("DEPLOYPILOT_SSH_DEFAULT_USER")
	}
	if username == "" {
		return nil, fmt.Errorf("SSH username not configured for server %s (configure DEPLOYPILOT_SSH_DEFAULT_USER or set server username)", serverID)
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
		return nil, fmt.Errorf("SSH connection failed to %s:%d: %w. "+
			"Suggestions: check host/port, verify security group allows TCP/%d, "+
			"confirm sshd is running, and ensure credentials are correct",
			host, port, err, port)
	}

	return &sshClientExecutor{Client: client}, nil
}

// RemoteExecutor is the interface for remote command execution (used by WebSocket terminal).
type RemoteExecutor interface {
	RunCommand(ctx context.Context, cmd string) (string, error)
	CreateInteractiveSession(ctx context.Context, termType string, rows, cols int) (InteractiveSession, error)
	Close() error
}

// InteractiveSession represents a persistent interactive SSH session with PTY.
type InteractiveSession interface {
	// StdinPipe returns a writer connected to the session's stdin.
	StdinPipe() io.Writer
	// SetWindowSize resizes the PTY.
	SetWindowSize(rows, cols int) error
	// Output returns a channel that receives stdout+stderr output.
	Output() <-chan []byte
	// Done returns a channel that is closed when the session exits.
	Done() <-chan struct{}
	// Close terminates the session.
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
	// Get stdin pipe
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		close(is.done)
		return is
	}
	is.stdin = stdinPipe

	// Pipe stdout and stderr to output channel
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

	// Start shell
	if err := session.Shell(); err != nil {
		close(is.done)
		return is
	}

	// Read stdout in goroutine
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

	// Read stderr in goroutine
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

	// Wait for session to exit
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


// ---------- 3. ListApps ----------


// GetAppDeploymentHistory returns the deployment history for an app by app ID.


// ---------- 16. CreateCredential ----------


// ---------- 38. BatchDNS ----------


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


// ---------- GetServersByTags ----------

// ---------- Phase 3.1: Compose Operations ----------

// ComposeDeploy deploys an app using docker-compose.
func (b *Bridge) ComposeDeploy(ctx context.Context, appID string) (string, error) {
	var app model.App
	if err := b.DB.First(&app, "id = ?", appID).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	executor, err := b.getRemoteExecutor(ctx, app.ServerID)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	workDir := fmt.Sprintf("/opt/deploypilot/apps/%s", app.Name)
	projectName := app.ComposeProjectName
	if projectName == "" {
		projectName = app.Name
	}

	// Prepend project name to compose content if not present
	composeContent := app.ComposeContent
	if !strings.Contains(composeContent, "name:") && !strings.Contains(composeContent, "version:") {
		// Wrap with project name
		composeContent = fmt.Sprintf("name: %s\n\n%s", projectName, composeContent)
	}

	// Parse env vars from JSON string
	var envVars map[string]string
	if app.EnvVars != "" {
		_ = json.Unmarshal([]byte(app.EnvVars), &envVars)
	}

	deployer := deployer.NewComposeDeployer(executor)
	out, err := deployer.ComposeUp(ctx, workDir, composeContent, envVars)
	if err != nil {
		return out, err
	}

	// Save deployment record
	if b.DB != nil {
		snapshotJSON, _ := json.Marshal(map[string]interface{}{
			"compose_content":     composeContent,
			"compose_project_name": projectName,
			"env_vars":            envVars,
		})
		record := &model.DeploymentRecord{
			ID:             generateID(),
			TenantID:       app.TenantID,
			ServerID:       app.ServerID,
			AppName:        app.Name,
			AppID:          app.ID,
			DeployType:     "compose_up",
			ConfigSnapshot: string(snapshotJSON),
			Status:         "success",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := b.DB.Create(record).Error; err != nil {
			slog.Error("failed to save compose deployment record", "error", err)
		}
	}

	return out, nil
}

// ComposeStop stops a compose deployment.
func (b *Bridge) ComposeStop(ctx context.Context, appID string) (string, error) {
	var app model.App
	if err := b.DB.First(&app, "id = ?", appID).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	executor, err := b.getRemoteExecutor(ctx, app.ServerID)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	workDir := fmt.Sprintf("/opt/deploypilot/apps/%s", app.Name)
	deployer := deployer.NewComposeDeployer(executor)
	return deployer.ComposeDown(ctx, workDir)
}

// ComposePs lists compose containers.
func (b *Bridge) ComposePs(ctx context.Context, appID string) (string, error) {
	var app model.App
	if err := b.DB.First(&app, "id = ?", appID).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	executor, err := b.getRemoteExecutor(ctx, app.ServerID)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	workDir := fmt.Sprintf("/opt/deploypilot/apps/%s", app.Name)
	deployer := deployer.NewComposeDeployer(executor)
	return deployer.ComposePs(ctx, workDir)
}

// ComposeLogs gets compose service logs.
func (b *Bridge) ComposeLogs(ctx context.Context, appID, service, tail string) (string, error) {
	var app model.App
	if err := b.DB.First(&app, "id = ?", appID).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	executor, err := b.getRemoteExecutor(ctx, app.ServerID)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	workDir := fmt.Sprintf("/opt/deploypilot/apps/%s", app.Name)
	deployer := deployer.NewComposeDeployer(executor)
	return deployer.ComposeLogs(ctx, workDir, service, tail)
}

// ComposeRestart restarts compose services.
func (b *Bridge) ComposeRestart(ctx context.Context, appID, service string) (string, error) {
	var app model.App
	if err := b.DB.First(&app, "id = ?", appID).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	executor, err := b.getRemoteExecutor(ctx, app.ServerID)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	workDir := fmt.Sprintf("/opt/deploypilot/apps/%s", app.Name)
	deployer := deployer.NewComposeDeployer(executor)
	return deployer.ComposeRestart(ctx, workDir, service)
}

// ---------- Phase 3.5: Preflight Visualization ----------

// RunPreflightFull runs all preflight checks without short-circuiting and returns a full report.
func (b *Bridge) RunPreflightFull(ctx context.Context, serverID string, portMappings string) (interface{}, error) {
	var executor CommandExecutor
	var host string
	var port int

	if serverID != "" {
		remoteExec, err := b.getRemoteExecutor(ctx, serverID)
		if err != nil {
			return nil, fmt.Errorf("failed to get remote executor for server %s: %w", serverID, err)
		}
		defer func() {
			if cerr := remoteExec.Close(); cerr != nil {
				slog.Warn("failed to close remote executor", "error", cerr)
			}
		}()
		executor = remoteExec

		// Look up server host/port for TCP check
		row := make(map[string]interface{})
		if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err == nil {
			host = toString(row["host"])
			port = toInt(row["port"])
		}
	} else {
		executor = b.Executor
	}

	cfg := PreflightConfig{
		Host:         host,
		Port:         port,
		Executor:     executor,
		PortMappings: portMappings,
	}

	result := RunPreflightFull(ctx, cfg)
	return result, nil
}

// ---------- Phase 3.3: ExecCommand ----------

func (b *Bridge) ExecCommand(ctx context.Context, serverID, command string, timeout int) (string, error) {
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	if serverID == "" {
		// Local execution
		return b.Executor.RunCommand(execCtx, command)
	}

	// Remote execution via SSH
	remoteExec, err := b.getRemoteExecutor(ctx, serverID)
	if err != nil {
		return "", fmt.Errorf("failed to get remote executor for server %s: %w", serverID, err)
	}
	defer func() {
		if cerr := remoteExec.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	return remoteExec.RunCommand(execCtx, command)
}

// ---------- Phase 3.3: ListImages ----------

func (b *Bridge) ListImages(ctx context.Context, serverID, filter string) (string, error) {
	dockerCmd := `docker images --format "{{.Repository}}:{{.Tag}}\t{{.Size}}\t{{.CreatedSince}}"`
	if filter != "" {
		dockerCmd += " | grep " + shellQuote(filter)
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if serverID == "" {
		return b.Executor.RunCommand(execCtx, dockerCmd)
	}

	remoteExec, err := b.getRemoteExecutor(ctx, serverID)
	if err != nil {
		return "", fmt.Errorf("failed to get remote executor for server %s: %w", serverID, err)
	}
	defer func() {
		if cerr := remoteExec.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	return remoteExec.RunCommand(execCtx, dockerCmd)
}

// ---------- Phase 3.3: PortForward ----------

// portForwardEntry tracks an active SSH port forward.
type portForwardEntry struct {
	ServerID   string
	LocalPort  int
	RemotePort int
	RemoteHost string
	Command    string
}

func (b *Bridge) PortForward(ctx context.Context, action, serverID string, localPort, remotePort int, remoteHost string) (string, error) {
	switch action {
	case "list":
		b.portForwardMu.RLock()
		defer b.portForwardMu.RUnlock()
		if len(b.portForwards) == 0 {
			return "No active port forwards.", nil
		}
		var lines []string
		for key, pf := range b.portForwards {
			lines = append(lines, fmt.Sprintf("  %s -> server=%s remote=%s:%d (key=%s)", pf.Command, pf.ServerID, pf.RemoteHost, pf.RemotePort, key))
		}
		return fmt.Sprintf("Active port forwards (%d):\n%s", len(b.portForwards), strings.Join(lines, "\n")), nil

	case "create":
		if serverID == "" {
			return "", fmt.Errorf("server_id is required for create action")
		}
		if localPort <= 0 || remotePort <= 0 {
			return "", fmt.Errorf("local_port and remote_port must be positive integers")
		}
		if remoteHost == "" {
			remoteHost = "127.0.0.1"
		}

		key := fmt.Sprintf("%s:%d", serverID, localPort)
		b.portForwardMu.Lock()
		defer b.portForwardMu.Unlock()

		if _, exists := b.portForwards[key]; exists {
			return "", fmt.Errorf("port forward already exists for %s (local port %d is already in use)", key, localPort)
		}

		// Get server info for SSH connection
		row := make(map[string]interface{})
		if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err != nil {
			return "", fmt.Errorf("server not found: %w", err)
		}

		host := toString(row["host"])
		port := toInt(row["port"])
		username := toString(row["username"])
		if username == "" {
			username = os.Getenv("DEPLOYPILOT_SSH_DEFAULT_USER")
		}
		if username == "" {
			return "", fmt.Errorf("SSH username not configured for server %s (configure DEPLOYPILOT_SSH_DEFAULT_USER or set server username)", serverID)
		}

		sshCmd := fmt.Sprintf("ssh -N -L %d:%s:%d -p %d %s@%s",
			localPort, remoteHost, remotePort, port, username, host)

		// Execute SSH tunnel in background
		execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		// We run the SSH command and let it run in the background
		// Since CommandExecutor.RunCommand is blocking, we launch it in a goroutine
		go func() {
			remoteExec, err := b.getRemoteExecutor(ctx, serverID)
			if err != nil {
				slog.Error("failed to create SSH tunnel", "error", err)
				return
			}
			defer func() {
				if cerr := remoteExec.Close(); cerr != nil {
					slog.Warn("failed to close remote executor", "error", cerr)
				}
			}()

			tunnelCmd := fmt.Sprintf("ssh -f -N -L %d:%s:%d -p %d %s@%s",
				localPort, remoteHost, remotePort, port, username, host)
			if _, err := remoteExec.RunCommand(execCtx, tunnelCmd); err != nil {
				slog.Error("SSH tunnel command failed", "error", err)
			}
			cancel()
		}()

		b.portForwards[key] = &portForwardEntry{
			ServerID:   serverID,
			LocalPort:  localPort,
			RemotePort: remotePort,
			RemoteHost: remoteHost,
			Command:    sshCmd,
		}

		return fmt.Sprintf("Port forward created: localhost:%d -> %s:%d:%d (server: %s)", localPort, remoteHost, remotePort, port, serverID), nil

	case "delete":
		if serverID == "" {
			return "", fmt.Errorf("server_id is required for delete action")
		}
		if localPort <= 0 {
			return "", fmt.Errorf("local_port must be a positive integer")
		}

		key := fmt.Sprintf("%s:%d", serverID, localPort)
		b.portForwardMu.Lock()
		defer b.portForwardMu.Unlock()

		entry, exists := b.portForwards[key]
		if !exists {
			return "", fmt.Errorf("port forward not found for %s", key)
		}

		// Kill the SSH tunnel process
		killCmd := fmt.Sprintf("pkill -f 'ssh.*-L %d:%s:%d'", localPort, shellQuote(entry.RemoteHost), entry.RemotePort)
		if _, err := b.Executor.RunCommand(ctx, killCmd); err != nil {
			slog.Warn("failed to kill SSH tunnel process", "error", err)
		}

		delete(b.portForwards, key)
		return fmt.Sprintf("Port forward deleted: %s", key), nil

	default:
		return "", fmt.Errorf("invalid action: %s (must be 'create', 'delete', or 'list')", action)
	}
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
