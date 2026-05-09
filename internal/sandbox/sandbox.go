// Package sandbox provides command execution sandboxing for DeployPilot.
// It intercepts all command execution and applies whitelist/blacklist rules
// to prevent dangerous operations.
package sandbox

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
)

// Mode defines the sandbox operation mode.
type Mode string

const (
	// ModeAllow allows only commands matching whitelist rules.
	ModeAllow Mode = "allow"
	// ModeDeny allows all commands except those matching blacklist rules.
	ModeDeny Mode = "deny"
	// ModeOff disables sandboxing entirely.
	ModeOff Mode = "off"
)

// Rule represents a single sandbox rule.
type Rule struct {
	ID          string `json:"id" yaml:"id"`
	Pattern     string `json:"pattern" yaml:"pattern"`
	Description string `json:"description" yaml:"description"`
	Enabled     bool   `json:"enabled" yaml:"enabled"`

	compiled *regexp.Regexp // pre-compiled regex
}

// Config holds the sandbox configuration.
type Config struct {
	Mode       Mode   `json:"mode" yaml:"mode"`
	LogBlocked bool   `json:"log_blocked" yaml:"log_blocked"`
	Rules      []Rule `json:"rules" yaml:"rules"`
}

// Sandbox validates commands against configured rules.
type Sandbox struct {
	mu     sync.RWMutex
	config Config
}

// New creates a new Sandbox with the given configuration.
func New(cfg Config) *Sandbox {
	s := &Sandbox{config: cfg}
	_ = s.compileRules() //nolint:errcheck // invalid rules are logged but not fatal
	return s
}

// DefaultConfig returns the default sandbox configuration.
// Default mode is "deny" with a blacklist of dangerous commands.
func DefaultConfig() Config {
	return Config{
		Mode:       ModeDeny,
		LogBlocked: true,
		Rules: []Rule{
			// Destructive file operations
			{
				ID:          "deny-rm-root",
				Pattern:     `rm\s+(-[rfRF]+\s+)?/\s*$`,
				Description: "Deny rm -rf / (root deletion)",
				Enabled:     true,
			},
			{
				ID:          "deny-rm-recursive-root",
				Pattern:     `rm\s+(-[rfRF]+\s+.*/)$`,
				Description: "Deny recursive rm starting from root directories",
				Enabled:     true,
			},
			{
				ID:          "deny-mkfs",
				Pattern:     `mkfs(\.\w+)?\s+/dev/`,
				Description: "Deny filesystem formatting",
				Enabled:     true,
			},
			{
				ID:          "deny-dd-destructive",
				Pattern:     `dd\s+.*of=/dev/`,
				Description: "Deny destructive dd writes to devices",
				Enabled:     true,
			},
			// System-critical operations
			{
				ID:          "deny-shutdown",
				Pattern:     `(?:^|\s)(?:shutdown|reboot|halt|poweroff)\b`,
				Description: "Deny system shutdown/reboot",
				Enabled:     true,
			},
			{
				ID:          "deny-systemctl-critical",
				Pattern:     `systemctl\s+(stop|disable|mask)\s+(docker|sshd|ssh|nginx|systemd|network)`,
				Description: "Deny stopping critical system services",
				Enabled:     true,
			},
			{
				ID:          "deny-iptables-flush",
				Pattern:     `iptables\s+(-F|--flush)`,
				Description: "Deny flushing all iptables rules",
				Enabled:     true,
			},
			{
				ID:          "deny-passwd",
				Pattern:     `passwd\s+(root|$)`,
				Description: "Deny changing root password",
				Enabled:     true,
			},
			{
				ID:          "deny-chmod-777",
				Pattern:     `chmod\s+(-R\s+)?777\s+/`,
				Description: "Deny chmod 777 on root paths",
				Enabled:     true,
			},
			{
				ID:          "deny-curl-pipe-sh",
				Pattern:     `curl\s+.*\|\s*(ba)?sh`,
				Description: "Deny curl | sh (remote script execution)",
				Enabled:     true,
			},
			{
				ID:          "deny-wget-pipe-sh",
				Pattern:     `wget\s+.*\|\s*(ba)?sh`,
				Description: "Deny wget | sh (remote script execution)",
				Enabled:     true,
			},
			{
				ID:          "deny-userdel",
				Pattern:     `userdel\s+(-r\s+)?(root|admin)`,
				Description: "Deny deleting root or admin users",
				Enabled:     true,
			},
			{
				ID:          "deny-kill-all",
				Pattern:     `kill\s+(-9\s+)?-1\b`,
				Description: "Deny kill -1 (kill all processes)",
				Enabled:     true,
			},
			{
				ID:          "deny-fdisk",
				Pattern:     `(fdisk|parted|mkpart)\s+/dev/`,
				Description: "Deny disk partitioning operations",
				Enabled:     true,
			},
			{
				ID:          "deny-crontab",
				Pattern:     `crontab\s+(-r|--remove)`,
				Description: "Deny removing all crontab entries",
				Enabled:     true,
			},
			{
				ID:          "deny-source-etc",
				Pattern:     `source\s+/etc/(passwd|shadow|sudoers)`,
				Description: "Deny sourcing sensitive system files",
				Enabled:     true,
			},
			{
				ID:          "deny-move-critical",
				Pattern:     `mv\s+(-f\s+)?/(bin|sbin|lib|usr|etc|boot|var)\s`,
				Description: "Deny moving critical system directories",
				Enabled:     true,
			},
			// Bypass prevention
			{
				ID:          "deny-base64-exec",
				Pattern:     `base64\s+(-d|--decode)`,
				Description: "Deny base64 decode (command obfuscation bypass)",
				Enabled:     true,
			},
			{
				ID:          "deny-xxd-exec",
				Pattern:     `xxd\s+(-r|--revert)`,
				Description: "Deny xxd reverse (command obfuscation bypass)",
				Enabled:     true,
			},
			{
				ID:          "deny-shell-escape",
				Pattern:     `(ba)?sh\s+(-c|--cmd)\s+['\"]`,
				Description: "Deny shell -c with quoted commands (escape bypass)",
				Enabled:     true,
			},
			{
				ID:          "deny-find-exec",
				Pattern:     `find\s+.*-exec\s+`,
				Description: "Deny find -exec (command execution bypass)",
				Enabled:     true,
			},
			{
				ID:          "deny-env-exec",
				Pattern:     `env\s+(-i\s+)?\S+\s*=.*\s+(ba)?sh`,
				Description: "Deny env variable execution bypass",
				Enabled:     true,
			},
		},
	}
}

// Validate checks if a command is allowed by the sandbox rules.
// Returns nil if the command is allowed, or an error describing why it was blocked.
func (s *Sandbox) Validate(cmd string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config.Mode == ModeOff {
		return nil
	}

	// Normalize: trim whitespace and collapse multiple spaces
	normalized := strings.TrimSpace(cmd)
	normalized = strings.Join(strings.Fields(normalized), " ")

	if normalized == "" {
		return nil
	}

	if s.config.Mode == ModeDeny {
		return s.checkBlacklist(normalized)
	}
	// ModeAllow
	return s.checkWhitelist(normalized)
}

// checkBlacklist returns error if command matches any enabled blacklist rule.
func (s *Sandbox) checkBlacklist(cmd string) error {
	for _, rule := range s.config.Rules {
		if !rule.Enabled || rule.compiled == nil {
			continue
		}
		if rule.compiled.MatchString(cmd) {
			if s.config.LogBlocked {
				slog.Warn("sandbox: command blocked",
					"rule_id", rule.ID,
					"rule", rule.Description,
					"command", truncate(cmd, 200),
				)
			}
			return &BlockedError{
				RuleID:      rule.ID,
				Rule:        rule.Description,
				Command:     cmd,
			}
		}
	}
	return nil
}

// checkWhitelist returns error if command does NOT match any enabled whitelist rule.
func (s *Sandbox) checkWhitelist(cmd string) error {
	for _, rule := range s.config.Rules {
		if !rule.Enabled || rule.compiled == nil {
			continue
		}
		if rule.compiled.MatchString(cmd) {
			return nil // matched a whitelist rule, allowed
		}
	}
	// No whitelist rule matched
	if s.config.LogBlocked {
		slog.Warn("sandbox: command not in whitelist",
			"command", truncate(cmd, 200),
		)
	}
	return &BlockedError{
		RuleID:  "whitelist",
		Rule:    "command does not match any whitelist rule",
		Command: cmd,
	}
}

// GetConfig returns a copy of the current sandbox configuration.
func (s *Sandbox) GetConfig() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// UpdateConfig replaces the sandbox configuration.
func (s *Sandbox) UpdateConfig(cfg Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg
	return s.compileRules()
}

// AddRule adds a new rule to the sandbox.
func (s *Sandbox) AddRule(rule Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.Rules = append(s.config.Rules, rule)
	return s.compileRules()
}

// RemoveRule removes a rule by ID.
func (s *Sandbox) RemoveRule(ruleID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.config.Rules {
		if r.ID == ruleID {
			s.config.Rules = append(s.config.Rules[:i], s.config.Rules[i+1:]...)
			break
		}
	}
}

// ToggleRule enables or disables a rule by ID.
func (s *Sandbox) ToggleRule(ruleID string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.config.Rules {
		if s.config.Rules[i].ID == ruleID {
			s.config.Rules[i].Enabled = enabled
			break
		}
	}
}

// compileRules pre-compiles all regex patterns.
func (s *Sandbox) compileRules() error {
	for i := range s.config.Rules {
		if s.config.Rules[i].Pattern == "" {
			continue
		}
		re, err := regexp.Compile(s.config.Rules[i].Pattern)
		if err != nil {
			slog.Error("sandbox: invalid rule pattern",
				"rule_id", s.config.Rules[i].ID,
				"pattern", s.config.Rules[i].Pattern,
				"error", err,
			)
			continue
		}
		s.config.Rules[i].compiled = re
	}
	return nil
}

// BlockedError is returned when a command is blocked by the sandbox.
type BlockedError struct {
	RuleID  string
	Rule    string
	Command string
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("command blocked by sandbox [%s]: %s", e.RuleID, e.Rule)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
