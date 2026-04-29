package service

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// PreflightErrorCode represents a structured error code for preflight failures.
type PreflightErrorCode string

const (
	PreflightTCPUnreachable    PreflightErrorCode = "SSH_TCP_UNREACHABLE"
	PreflightSSHAuthFailed     PreflightErrorCode = "SSH_AUTH_FAILED"
	PreflightDockerUnavailable PreflightErrorCode = "REMOTE_DOCKER_UNAVAILABLE"
	PreflightPortInUse         PreflightErrorCode = "PORT_ALREADY_IN_USE"
	PreflightUnknownError      PreflightErrorCode = "PREFLIGHT_UNKNOWN_ERROR"
)

// PreflightResult holds the result of a preflight check.
type PreflightResult struct {
	Passed  bool             `json:"passed"`
	Checks  []PreflightCheck `json:"checks"`
	Code    PreflightErrorCode `json:"code,omitempty"`
	Message string           `json:"message,omitempty"`
}

// PreflightCheck holds the result of an individual check.
type PreflightCheck struct {
	Name       string `json:"name"`
	Passed     bool   `json:"passed"`
	Message    string `json:"message,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
	Category   string `json:"category,omitempty"`  // "network", "docker", "resource", "disk", "memory"
	Severity   string `json:"severity,omitempty"`  // "error", "warning", "info"
}

// PreflightConfig holds the configuration for preflight checks.
type PreflightConfig struct {
	Host         string
	Port         int
	Executor     CommandExecutor // for SSH/Docker checks; nil for local-only
	PortMappings string          // e.g. "18080:80" or "8080:80,3000:3000"
}

// CommandExecutor is the interface for running commands (local or remote).
type CommandExecutor interface {
	RunCommand(ctx context.Context, cmd string) (string, error)
}

// RunPreflight executes all preflight checks and returns the result.
func RunPreflight(ctx context.Context, cfg PreflightConfig) *PreflightResult {
	result := &PreflightResult{Passed: true}
	var checks []PreflightCheck

	// Check 1: TCP connectivity (only for remote deployments)
	if cfg.Host != "" && cfg.Port != 0 {
		check := checkTCP(ctx, cfg.Host, cfg.Port)
		checks = append(checks, check)
		if !check.Passed {
			result.Passed = false
			result.Code = PreflightTCPUnreachable
			result.Message = check.Message
			result.Checks = checks
			return result
		}
	}

	// Check 2: SSH authentication (only for remote deployments with executor)
	if cfg.Executor != nil && cfg.Host != "" {
		check := checkSSHAuth(ctx, cfg.Executor)
		checks = append(checks, check)
		if !check.Passed {
			result.Passed = false
			result.Code = PreflightSSHAuthFailed
			result.Message = check.Message
			result.Checks = checks
			return result
		}
	}

	// Check 3: Docker availability
	check := checkDocker(ctx, cfg.Executor)
	checks = append(checks, check)
	if !check.Passed {
		result.Passed = false
		result.Code = PreflightDockerUnavailable
		result.Message = check.Message
		result.Checks = checks
		return result
	}

	// Check 4: Port conflict (only if port mappings specified)
	if cfg.PortMappings != "" {
		check := checkPortConflict(ctx, cfg.Executor, cfg.PortMappings)
		checks = append(checks, check)
		if !check.Passed {
			result.Passed = false
			result.Code = PreflightPortInUse
			result.Message = check.Message
			result.Checks = checks
			return result
		}
	}

	result.Checks = checks
	return result
}

// checkTCP verifies TCP connectivity to host:port.
func checkTCP(ctx context.Context, host string, port int) PreflightCheck {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return PreflightCheck{
			Name:    "TCP Connectivity",
			Passed:  false,
			Message: fmt.Sprintf("Cannot connect to %s:%d — %v", host, port, err),
			Suggestion: fmt.Sprintf(
				"Check: 1) host and port are correct (cloud providers often use non-22 ports like 23196, 2222) "+
					"2) security group/firewall allows inbound TCP/%d 3) sshd is running: ss -tlnp | grep sshd "+
					"4) try manually: ssh -p %d root@%s",
				port, port, host,
			),
		}
	}
	if cerr := conn.Close(); cerr != nil {
		return PreflightCheck{
			Name:    "TCP Connectivity",
			Passed:  false,
			Message: fmt.Sprintf("TCP connection to %s:%d succeeded but close failed: %v", host, port, cerr),
		}
	}
	return PreflightCheck{
		Name:    "TCP Connectivity",
		Passed:  true,
		Message: fmt.Sprintf("TCP connection to %s:%d successful", host, port),
	}
}

// checkSSHAuth verifies SSH authentication by running a simple command.
func checkSSHAuth(ctx context.Context, executor CommandExecutor) PreflightCheck {
	out, err := executor.RunCommand(ctx, "echo ok")
	if err != nil {
		return PreflightCheck{
			Name:    "SSH Authentication",
			Passed:  false,
			Message: fmt.Sprintf("SSH auth failed: %v", err),
			Suggestion: "Check: 1) credential exists and is correct 2) DEPLOYPILOT_ENCRYPTION_KEY matches the key used to create the credential 3) credential type (password/ssh_key) matches server config",
		}
	}
	trimmed := strings.TrimSpace(out)
	if trimmed != "ok" {
		return PreflightCheck{
			Name:    "SSH Authentication",
			Passed:  false,
			Message: fmt.Sprintf("SSH auth returned unexpected output: %q", out),
			Suggestion: "SSH connection succeeded but command output was unexpected. The remote shell may have login messages interfering.",
		}
	}
	return PreflightCheck{
		Name:    "SSH Authentication",
		Passed:  true,
		Message: "SSH authentication successful",
	}
}

// checkDocker verifies Docker is available on the target.
func checkDocker(ctx context.Context, executor CommandExecutor) PreflightCheck {
	out, err := executor.RunCommand(ctx, "docker version --format '{{.Server.Version}}'")
	if err != nil {
		return PreflightCheck{
			Name:    "Docker Availability",
			Passed:  false,
			Message: fmt.Sprintf("Docker is not available: %v", err),
			Suggestion: "Install Docker (https://docs.docker.com/get-docker/) and ensure the daemon is running: sudo systemctl start docker && sudo usermod -aG docker $USER",
		}
	}
	version := strings.TrimSpace(out)
	return PreflightCheck{
		Name:    "Docker Availability",
		Passed:  true,
		Message: fmt.Sprintf("Docker is available (version %s)", version),
	}
}

// checkPortConflict checks if any mapped host ports are already in use.
func checkPortConflict(ctx context.Context, executor CommandExecutor, portMappings string) PreflightCheck {
	// Parse host ports from mappings like "18080:80" or "8080:80,3000:3000"
	hostPorts := parseHostPorts(portMappings)
	if len(hostPorts) == 0 {
		return PreflightCheck{
			Name:    "Port Conflict",
			Passed:  true,
			Message: "No host port mappings to check",
		}
	}

	// Check each port using ss or netstat
	for _, port := range hostPorts {
		cmd := fmt.Sprintf("ss -tlnp 2>/dev/null | grep ':%s ' || true", port)
		out, err := executor.RunCommand(ctx, cmd)
		if err == nil && strings.TrimSpace(out) != "" {
			return PreflightCheck{
				Name:    "Port Conflict",
				Passed:  false,
				Message: fmt.Sprintf("Port %s is already in use: %s", port, strings.TrimSpace(out)),
				Suggestion: fmt.Sprintf("Port %s is occupied. Choose a different host port or stop the existing service using it.", port),
			}
		}
	}

	return PreflightCheck{
		Name:    "Port Conflict",
		Passed:  true,
		Message: fmt.Sprintf("Host ports %v are available", hostPorts),
	}
}

// parseHostPorts extracts host port numbers from Docker port mapping strings.
func parseHostPorts(portMappings string) []string {
	var ports []string
	parts := strings.Split(portMappings, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Handle "hostPort:containerPort" or just "hostPort"
		if idx := strings.Index(part, ":"); idx >= 0 {
			part = part[:idx]
		}
		// Handle "hostPort:containerPort/protocol"
		if idx := strings.Index(part, "/"); idx >= 0 {
			part = part[:idx]
		}
		part = strings.TrimSpace(part)
		if part != "" {
			ports = append(ports, part)
		}
	}
	return ports
}

// parseDiskSize parses a human-readable disk size string (e.g. "50G", "200M", "1024K") into float64 GB.
func parseDiskSize(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}
	// Remove trailing non-numeric characters (unit suffix)
	suffix := ""
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] < '0' || s[i] > '9' {
			suffix = string(s[i]) + suffix
			s = s[:i]
		} else {
			break
		}
	}
	s = strings.TrimSpace(s)
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse size %q: %w", s, err)
	}
	switch strings.ToUpper(suffix) {
	case "T", "TB":
		return val * 1024, nil
	case "G", "GB":
		return val, nil
	case "M", "MB":
		return val / 1024, nil
	case "K", "KB":
		return val / (1024 * 1024), nil
	default:
		// Assume bytes
		return val / (1024 * 1024 * 1024), nil
	}
}

// checkDiskSpace verifies available disk space on the target.
func checkDiskSpace(ctx context.Context, cfg PreflightConfig) PreflightCheck {
	if cfg.Executor == nil {
		return PreflightCheck{
			Name:     "Disk Space",
			Category: "disk",
			Passed:   false,
			Message:  "No executor available to check disk space",
			Severity: "warning",
		}
	}

	out, err := cfg.Executor.RunCommand(ctx, "df -h / --output=avail | tail -1")
	if err != nil {
		return PreflightCheck{
			Name:       "Disk Space",
			Category:   "disk",
			Passed:     false,
			Message:    fmt.Sprintf("Failed to check disk space: %v", err),
			Severity:   "warning",
			Suggestion: "Ensure the 'df' command is available on the target system",
		}
	}

	availGB, err := parseDiskSize(strings.TrimSpace(out))
	if err != nil {
		return PreflightCheck{
			Name:       "Disk Space",
			Category:   "disk",
			Passed:     false,
			Message:    fmt.Sprintf("Failed to parse disk space output %q: %v", strings.TrimSpace(out), err),
			Severity:   "warning",
			Suggestion: "Ensure the 'df' command supports --output flag (coreutils >= 8.21)",
		}
	}

	if availGB < 1.0 {
		return PreflightCheck{
			Name:       "Disk Space",
			Category:   "disk",
			Passed:     false,
			Message:    fmt.Sprintf("Critical: only %.1fGB available disk space", availGB),
			Severity:   "error",
			Suggestion: "Clean up disk space or expand storage",
		}
	}
	if availGB < 5.0 {
		return PreflightCheck{
			Name:       "Disk Space",
			Category:   "disk",
			Passed:     true,
			Message:    fmt.Sprintf("Warning: %.1fGB available disk space (low)", availGB),
			Severity:   "warning",
			Suggestion: "Clean up disk space or expand storage",
		}
	}

	return PreflightCheck{
		Name:     "Disk Space",
		Category: "disk",
		Passed:   true,
		Message:  fmt.Sprintf("%.1fGB available disk space", availGB),
		Severity: "info",
	}
}

// checkMemory verifies available memory on the target.
func checkMemory(ctx context.Context, cfg PreflightConfig) PreflightCheck {
	if cfg.Executor == nil {
		return PreflightCheck{
			Name:     "Memory",
			Category: "memory",
			Passed:   false,
			Message:  "No executor available to check memory",
			Severity: "warning",
		}
	}

	out, err := cfg.Executor.RunCommand(ctx, "free -m | awk '/Mem:/{print $7}'")
	if err != nil {
		return PreflightCheck{
			Name:       "Memory",
			Category:   "memory",
			Passed:     false,
			Message:    fmt.Sprintf("Failed to check memory: %v", err),
			Severity:   "warning",
			Suggestion: "Ensure the 'free' command is available on the target system",
		}
	}

	availMB, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return PreflightCheck{
			Name:       "Memory",
			Category:   "memory",
			Passed:     false,
			Message:    fmt.Sprintf("Failed to parse memory output %q: %v", strings.TrimSpace(out), err),
			Severity:   "warning",
			Suggestion: "Ensure the 'free' command is available on the target system",
		}
	}

	if availMB < 128 {
		return PreflightCheck{
			Name:       "Memory",
			Category:   "memory",
			Passed:     false,
			Message:    fmt.Sprintf("Critical: only %dMB available memory", availMB),
			Severity:   "error",
			Suggestion: "Close unused processes or add more RAM",
		}
	}
	if availMB < 512 {
		return PreflightCheck{
			Name:       "Memory",
			Category:   "memory",
			Passed:     true,
			Message:    fmt.Sprintf("Warning: %dMB available memory (low)", availMB),
			Severity:   "warning",
			Suggestion: "Close unused processes or add more RAM",
		}
	}

	return PreflightCheck{
		Name:     "Memory",
		Category: "memory",
		Passed:   true,
		Message:  fmt.Sprintf("%dMB available memory", availMB),
		Severity: "info",
	}
}

// RunPreflightFull runs ALL preflight checks without short-circuiting on failure.
// It collects results from every check and returns a comprehensive report.
func RunPreflightFull(ctx context.Context, cfg PreflightConfig) PreflightResult {
	var checks []PreflightCheck

	// Check 1: TCP connectivity (only for remote deployments)
	if cfg.Host != "" && cfg.Port != 0 {
		checks = append(checks, checkTCP(ctx, cfg.Host, cfg.Port))
	}

	// Check 2: SSH authentication (only for remote deployments with executor)
	if cfg.Executor != nil && cfg.Host != "" {
		checks = append(checks, checkSSHAuth(ctx, cfg.Executor))
	}

	// Check 3: Docker availability
	checks = append(checks, checkDocker(ctx, cfg.Executor))

	// Check 4: Port conflict (only if port mappings specified)
	if cfg.PortMappings != "" {
		checks = append(checks, checkPortConflict(ctx, cfg.Executor, cfg.PortMappings))
	}

	// Check 5: Disk space
	checks = append(checks, checkDiskSpace(ctx, cfg))

	// Check 6: Memory
	checks = append(checks, checkMemory(ctx, cfg))

	// Overall result: failed if any check has severity "error" or Passed == false
	passed := true
	for _, c := range checks {
		if !c.Passed {
			passed = false
			break
		}
	}

	return PreflightResult{
		Passed: passed,
		Checks: checks,
	}
}
