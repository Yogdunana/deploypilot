package config

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// ChangeCallback is called when config changes.
type ChangeCallback func(oldCfg, newCfg *Config)

// Manager manages configuration with hot-reload support.
type Manager struct {
	mu        sync.RWMutex
	config    *Config
	configPath string
	version   int
	callbacks []ChangeCallback
	closed    bool
	closeCh   chan struct{}
}

// NewManager creates a new config Manager and loads the initial config.
func NewManager(configPath string) *Manager {
	mgr := &Manager{
		configPath: configPath,
		closeCh:    make(chan struct{}),
	}

	cfg, err := Load(configPath)
	if err != nil {
		// Use defaults if file doesn't exist yet
		cfg = &Config{}
	}
	mgr.config = cfg
	mgr.version = 1

	// Start file watcher goroutine
	go mgr.watchLoop()

	return mgr
}

// GetConfig returns the current config (thread-safe).
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetConfigPath returns the config file path.
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// GetVersion returns the current config version (increments on each reload).
func (m *Manager) GetVersion() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.version
}

// Reload re-reads the config file and updates the in-memory config.
// If the new config is invalid, the old config is preserved and an error is returned.
func (m *Manager) Reload() error {
	newCfg, err := Load(m.configPath)
	if err != nil {
		return fmt.Errorf("reload failed: %w", err)
	}

	m.mu.Lock()
	oldCfg := m.config
	m.config = newCfg
	m.version++
	callbacks := make([]ChangeCallback, len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.mu.Unlock()

	// Fire callbacks outside the lock
	for _, cb := range callbacks {
		cb(oldCfg, newCfg)
	}

	return nil
}

// OnChange registers a callback to be called when config changes.
func (m *Manager) OnChange(cb ChangeCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, cb)
}

// Close stops the file watcher.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.closeCh)
	}
}

// watchLoop polls the config file for changes using content hash.
func (m *Manager) watchLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastHash := m.fileHash()
	_ = lastHash // suppress unused in non-debug mode

	for {
		select {
		case <-m.closeCh:
			return
		case <-ticker.C:
			currentHash := m.fileHash()
			if currentHash != "" && currentHash != lastHash {
				lastHash = currentHash
				if err := m.Reload(); err != nil {
					fmt.Fprintf(os.Stderr, "[config] reload error: %v\n", err)
				}
			}
		}
	}
}

// fileHash returns the SHA256 hash of the config file.
func (m *Manager) fileHash() string {
	f, err := os.Open(m.configPath)
	if err != nil {
		return ""
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "[config] failed to close config file: %v\n", cerr)
		}
	}()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
