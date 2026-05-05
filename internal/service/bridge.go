package service

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/agent"
	"github.com/Yogdunana/deploypilot/internal/bruteforce"
	"github.com/Yogdunana/deploypilot/internal/confirm"
	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/engine/healer"
	"github.com/Yogdunana/deploypilot/internal/license"
	"github.com/Yogdunana/deploypilot/internal/monitor"
	"github.com/Yogdunana/deploypilot/internal/plugin"
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

	// License engine (v2)
	LicenseEngine *license.Engine          // license validation and feature evaluation engine
	LicensePrivKey ed25519.PrivateKey      // only set for developer instances (license issuance)

	// Feature flag system
	featureFlagCache *FeatureFlagCache

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
		DB:               db,
		Executor:         executor,
		EncryptionKey:    encryptionKey,
		EventBus:         eventBus,
		ConfirmStore:     confirm.NewStore(),
		BFProtector:      bruteforce.New(bruteforce.DefaultConfig()),
		featureFlagCache: NewFeatureFlagCache(5 * time.Minute),
		tasks:            make(map[string]*taskInfo),
		backupApps:       make(map[string]string),
		portForwards:     make(map[string]*portForwardEntry),
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

// portForwardEntry tracks an active SSH port forward.
type portForwardEntry struct {
	ServerID   string
	LocalPort  int
	RemotePort int
	RemoteHost string
	Command    string
}

