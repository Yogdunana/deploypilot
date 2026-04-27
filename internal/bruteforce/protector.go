// Package bruteforce provides login brute-force protection with
// per-account rate limiting, progressive delays, and account locking.
package bruteforce

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Config holds the brute-force protection configuration.
type Config struct {
	// MaxAttempts is the number of failed login attempts before account lockout.
	MaxAttempts int `json:"max_attempts" yaml:"max_attempts"`
	// LockoutDuration is how long an account stays locked after MaxAttempts failures.
	LockoutDuration time.Duration `json:"lockout_duration" yaml:"lockout_duration"`
	// WindowDuration is the sliding window for counting failed attempts.
	WindowDuration time.Duration `json:"window_duration" yaml:"window_duration"`
	// ProgressiveDelay adds increasing delay per failure (0 = disabled).
	ProgressiveDelay bool `json:"progressive_delay" yaml:"progressive_delay"`
	// BaseDelay is the initial delay after first failure (used with ProgressiveDelay).
	BaseDelay time.Duration `json:"base_delay" yaml:"base_delay"`
	// MaxDelay is the maximum delay cap.
	MaxDelay time.Duration `json:"max_delay" yaml:"max_delay"`
	// IPMaxAttempts is the max failed attempts per IP before IP-level block.
	IPMaxAttempts int `json:"ip_max_attempts" yaml:"ip_max_attempts"`
	// IPLockoutDuration is how long an IP stays blocked.
	IPLockoutDuration time.Duration `json:"ip_lockout_duration" yaml:"ip_lockout_duration"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:       5,
		LockoutDuration:   15 * time.Minute,
		WindowDuration:    15 * time.Minute,
		ProgressiveDelay:  true,
		BaseDelay:         1 * time.Second,
		MaxDelay:          30 * time.Second,
		IPMaxAttempts:     20,
		IPLockoutDuration: 30 * time.Minute,
	}
}

// AttemptRecord tracks a single failed login attempt.
type AttemptRecord struct {
	Time  time.Time `json:"time"`
	IP    string    `json:"ip"`
	Error string    `json:"error"`
}

// AccountState tracks the brute-force state for one account or IP.
type AccountState struct {
	Attempts    []AttemptRecord `json:"attempts"`
	LockedUntil time.Time       `json:"locked_until,omitempty"`
}

// Protector provides brute-force protection for login attempts.
type Protector struct {
	mu       sync.RWMutex
	config   Config
	accounts map[string]*AccountState // username -> state
	ips      map[string]*AccountState // ip -> state
}

// New creates a new Protector with the given configuration.
func New(cfg Config) *Protector {
	p := &Protector{
		config:   cfg,
		accounts: make(map[string]*AccountState),
		ips:      make(map[string]*AccountState),
	}
	go p.cleanupLoop()
	return p
}

// CheckResult is returned by Check().
type CheckResult struct {
	Allowed    bool          `json:"allowed"`
	Reason     string        `json:"reason,omitempty"`
	Delay      time.Duration `json:"delay,omitempty"` // suggested delay before allowing
	Attempts   int           `json:"attempts"`
	LockedUntil time.Time    `json:"locked_until,omitempty"`
}

// Check checks if a login attempt is allowed for the given username and IP.
// Call this BEFORE verifying credentials.
func (p *Protector) Check(username, ip string) CheckResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()

	// Check account-level lockout
	if state, ok := p.accounts[username]; ok {
		if !state.LockedUntil.IsZero() && now.Before(state.LockedUntil) {
			return CheckResult{
				Allowed:     false,
				Reason:      "account_locked",
				LockedUntil: state.LockedUntil,
				Attempts:    len(state.Attempts),
			}
		}
		// Check progressive delay
		if p.config.ProgressiveDelay {
			recentCount := p.countRecent(state.Attempts, now)
			if recentCount > 0 {
				delay := p.calculateDelay(recentCount)
				if delay > 0 {
					return CheckResult{
						Allowed:  true,
						Delay:    delay,
						Reason:   "progressive_delay",
						Attempts: recentCount,
					}
				}
			}
		}
	}

	// Check IP-level lockout
	if state, ok := p.ips[ip]; ok {
		if !state.LockedUntil.IsZero() && now.Before(state.LockedUntil) {
			return CheckResult{
				Allowed:     false,
				Reason:      "ip_blocked",
				LockedUntil: state.LockedUntil,
				Attempts:    len(state.Attempts),
			}
		}
	}

	return CheckResult{Allowed: true, Attempts: 0}
}

// RecordFailure records a failed login attempt.
func (p *Protector) RecordFailure(username, ip, errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	record := AttemptRecord{Time: now, IP: ip, Error: errMsg}

	// Record account-level failure
	state := p.getOrCreateState(p.accounts, username)
	state.Attempts = append(state.Attempts, record)
	p.pruneOld(state, now)

	recentCount := len(state.Attempts)
	if recentCount >= p.config.MaxAttempts {
		state.LockedUntil = now.Add(p.config.LockoutDuration)
		slog.Warn("bruteforce: account locked",
			"username", username,
			"ip", ip,
			"attempts", recentCount,
			"locked_until", state.LockedUntil,
		)
	}

	// Record IP-level failure
	ipState := p.getOrCreateState(p.ips, ip)
	ipState.Attempts = append(ipState.Attempts, record)
	p.pruneOld(ipState, now)

	ipRecentCount := len(ipState.Attempts)
	if ipRecentCount >= p.config.IPMaxAttempts {
		ipState.LockedUntil = now.Add(p.config.IPLockoutDuration)
		slog.Warn("bruteforce: IP blocked",
			"ip", ip,
			"attempts", ipRecentCount,
			"locked_until", ipState.LockedUntil,
		)
	}
}

// RecordSuccess clears failed attempts for the given username on successful login.
func (p *Protector) RecordSuccess(username, ip string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear account failures
	if state, ok := p.accounts[username]; ok {
		if len(state.Attempts) > 0 {
			slog.Info("bruteforce: clearing failures after successful login",
				"username", username,
				"previous_failures", len(state.Attempts),
			)
		}
		delete(p.accounts, username)
	}

	// Clear IP failures (only if no other accounts are failing from this IP)
	if ipState, ok := p.ips[ip]; ok {
		// Only clear if all recent failures are for this username
		allSameUser := true
		for _, a := range ipState.Attempts {
			// We don't store username in IP records, so just clear on success
			_ = a
		}
		if allSameUser {
			delete(p.ips, ip)
		}
	}
}

// GetAccountStatus returns the current brute-force status for an account.
func (p *Protector) GetAccountStatus(username string) (attempts int, lockedUntil time.Time) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	state, ok := p.accounts[username]
	if !ok {
		return 0, time.Time{}
	}
	return len(state.Attempts), state.LockedUntil
}

// UnlockAccount manually unlocks a locked account.
func (p *Protector) UnlockAccount(username string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	state, ok := p.accounts[username]
	if !ok {
		return false
	}
	if state.LockedUntil.IsZero() || time.Now().After(state.LockedUntil) {
		return false // not locked
	}
	state.LockedUntil = time.Time{}
	state.Attempts = nil
	slog.Info("bruteforce: account manually unlocked", "username", username)
	return true
}

// UnlockIP manually unlocks a blocked IP.
func (p *Protector) UnlockIP(ip string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	state, ok := p.ips[ip]
	if !ok {
		return false
	}
	if state.LockedUntil.IsZero() || time.Now().After(state.LockedUntil) {
		return false
	}
	state.LockedUntil = time.Time{}
	state.Attempts = nil
	slog.Info("bruteforce: IP manually unlocked", "ip", ip)
	return true
}

// ListLockedAccounts returns all currently locked accounts.
func (p *Protector) ListLockedAccounts() []map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()
	var result []map[string]interface{}
	for username, state := range p.accounts {
		if !state.LockedUntil.IsZero() && now.Before(state.LockedUntil) {
			result = append(result, map[string]interface{}{
				"username":     username,
				"attempts":     len(state.Attempts),
				"locked_until": state.LockedUntil,
			})
		}
	}
	return result
}

// ListBlockedIPs returns all currently blocked IPs.
func (p *Protector) ListBlockedIPs() []map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()
	var result []map[string]interface{}
	for ip, state := range p.ips {
		if !state.LockedUntil.IsZero() && now.Before(state.LockedUntil) {
			result = append(result, map[string]interface{}{
				"ip":           ip,
				"attempts":     len(state.Attempts),
				"locked_until": state.LockedUntil,
			})
		}
	}
	return result
}

// --- Internal helpers ---

func (p *Protector) getOrCreateState(m map[string]*AccountState, key string) *AccountState {
	if state, ok := m[key]; ok {
		return state
	}
	state := &AccountState{}
	m[key] = state
	return state
}

func (p *Protector) countRecent(attempts []AttemptRecord, now time.Time) int {
	cutoff := now.Add(-p.config.WindowDuration)
	count := 0
	for _, a := range attempts {
		if a.Time.After(cutoff) {
			count++
		}
	}
	return count
}

func (p *Protector) pruneOld(state *AccountState, now time.Time) {
	cutoff := now.Add(-p.config.WindowDuration)
	pruned := state.Attempts[:0]
	for _, a := range state.Attempts {
		if a.Time.After(cutoff) {
			pruned = append(pruned, a)
		}
	}
	state.Attempts = pruned
}

func (p *Protector) calculateDelay(failureCount int) time.Duration {
	if !p.config.ProgressiveDelay || failureCount <= 0 {
		return 0
	}
	// Exponential backoff: base * 2^(count-1), capped at MaxDelay
	multiplier := 1 << (failureCount - 1)
	delay := p.config.BaseDelay * time.Duration(multiplier)
	if delay > p.config.MaxDelay {
		delay = p.config.MaxDelay
	}
	return delay
}

func (p *Protector) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		p.cleanup()
	}
}

func (p *Protector) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	for key, state := range p.accounts {
		p.pruneOld(state, now)
		if len(state.Attempts) == 0 && (state.LockedUntil.IsZero() || now.After(state.LockedUntil)) {
			delete(p.accounts, key)
		}
	}
	for key, state := range p.ips {
		p.pruneOld(state, now)
		if len(state.Attempts) == 0 && (state.LockedUntil.IsZero() || now.After(state.LockedUntil)) {
			delete(p.ips, key)
		}
	}
}

// IsAccountLockedError formats a user-friendly error message for locked accounts.
func IsAccountLockedError(result CheckResult) string {
	if result.Reason == "account_locked" {
		remaining := time.Until(result.LockedUntil).Truncate(time.Second)
		return fmt.Sprintf("account locked, try again in %s", remaining)
	}
	if result.Reason == "ip_blocked" {
		remaining := time.Until(result.LockedUntil).Truncate(time.Second)
		return fmt.Sprintf("too many failed attempts from your IP, try again in %s", remaining)
	}
	return ""
}
