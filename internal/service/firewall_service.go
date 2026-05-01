package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/sandbox"
	"gorm.io/gorm"
)

// FirewallType represents the detected firewall backend.
type FirewallType string

const (
	FirewallTypeUFW      FirewallType = "ufw"
	FirewallTypeFirewalld FirewallType = "firewalld"
	FirewallTypeIptables  FirewallType = "iptables"
	FirewallTypeUnknown   FirewallType = "unknown"
)

// FirewallRule represents a single firewall rule.
type FirewallRule struct {
	ID          string `json:"id"`
	Chain       string `json:"chain"`
	Action      string `json:"action"` // ACCEPT, DROP, REJECT
	Protocol    string `json:"protocol"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Port        string `json:"port"`
	Target      string `json:"target"`
	Raw         string `json:"raw"`
}

// FirewallStatus represents the overall firewall status.
type FirewallStatus struct {
	Type      FirewallType `json:"type"`
	Enabled   bool         `json:"enabled"`
	Version   string       `json:"version,omitempty"`
	Rules     []FirewallRule `json:"rules"`
}

// FirewallService manages remote server firewalls via SSH.
type FirewallService struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewFirewallService creates a new FirewallService.
func NewFirewallService(db *gorm.DB, _ *sandbox.Sandbox) *FirewallService {
	return &FirewallService{
		db:     db,
		logger: slog.Default(),
	}
}

// DetectFirewall detects which firewall is active on the remote server.
func (f *FirewallService) DetectFirewall(ctx context.Context, serverID string) (*FirewallStatus, error) {
	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	// Check in order of preference
	if fw, err := f.detectUFW(ctx, exec); err == nil {
		return fw, nil
	}
	if fw, err := f.detectFirewalld(ctx, exec); err == nil {
		return fw, nil
	}
	if fw, err := f.detectIptables(ctx, exec); err == nil {
		return fw, nil
	}

	return &FirewallStatus{Type: FirewallTypeUnknown, Enabled: false}, nil
}

// GetStatus returns the current firewall status and rules.
func (f *FirewallService) GetStatus(ctx context.Context, serverID string) (*FirewallStatus, error) {
	fw, err := f.DetectFirewall(ctx, serverID)
	if err != nil {
		return nil, err
	}

	if fw.Type == FirewallTypeUnknown {
		return fw, nil
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	switch fw.Type {
	case FirewallTypeUFW:
		return f.getUFWStatus(ctx, exec)
	case FirewallTypeFirewalld:
		return f.getFirewalldStatus(ctx, exec)
	case FirewallTypeIptables:
		return f.getIptablesStatus(ctx, exec)
	default:
		return fw, nil
	}
}

// ========== Port Management ==========

// OpenPort opens a port on the remote firewall.
func (f *FirewallService) OpenPort(ctx context.Context, serverID, port, protocol string) error {
	fw, err := f.DetectFirewall(ctx, serverID)
	if err != nil {
		return err
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	if protocol == "" {
		protocol = "tcp"
	}

	switch fw.Type {
	case FirewallTypeUFW:
		return f.ufwAllow(ctx, exec, port, protocol)
	case FirewallTypeFirewalld:
		return f.firewalldOpenPort(ctx, exec, port, protocol)
	case FirewallTypeIptables:
		return f.iptablesOpenPort(ctx, exec, port, protocol)
	default:
		return fmt.Errorf("no supported firewall detected")
	}
}

// ClosePort closes a port on the remote firewall.
func (f *FirewallService) ClosePort(ctx context.Context, serverID, port, protocol string) error {
	fw, err := f.DetectFirewall(ctx, serverID)
	if err != nil {
		return err
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	if protocol == "" {
		protocol = "tcp"
	}

	switch fw.Type {
	case FirewallTypeUFW:
		return f.ufwDeny(ctx, exec, port, protocol)
	case FirewallTypeFirewalld:
		return f.firewalldClosePort(ctx, exec, port, protocol)
	case FirewallTypeIptables:
		return f.iptablesClosePort(ctx, exec, port, protocol)
	default:
		return fmt.Errorf("no supported firewall detected")
	}
}

// ========== IP Management ==========

// BlockIP blocks an IP address on the remote firewall.
func (f *FirewallService) BlockIP(ctx context.Context, serverID, ip string) error {
	fw, err := f.DetectFirewall(ctx, serverID)
	if err != nil {
		return err
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	switch fw.Type {
	case FirewallTypeUFW:
		return f.ufwDenyIP(ctx, exec, ip)
	case FirewallTypeFirewalld:
		return f.firewalldBlockIP(ctx, exec, ip)
	case FirewallTypeIptables:
		return f.iptablesBlockIP(ctx, exec, ip)
	default:
		return fmt.Errorf("no supported firewall detected")
	}
}

// UnblockIP unblocks an IP address on the remote firewall.
func (f *FirewallService) UnblockIP(ctx context.Context, serverID, ip string) error {
	fw, err := f.DetectFirewall(ctx, serverID)
	if err != nil {
		return err
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	switch fw.Type {
	case FirewallTypeUFW:
		return f.ufwAllowIP(ctx, exec, ip)
	case FirewallTypeFirewalld:
		return f.firewalldUnblockIP(ctx, exec, ip)
	case FirewallTypeIptables:
		return f.iptablesUnblockIP(ctx, exec, ip)
	default:
		return fmt.Errorf("no supported firewall detected")
	}
}

// ========== Quick Actions ==========

// AllowCommonPorts opens commonly needed ports (22, 80, 443, 8080).
func (f *FirewallService) AllowCommonPorts(ctx context.Context, serverID string) error {
	ports := []struct {
		port     string
		protocol string
	}{
		{"22", "tcp"},
		{"80", "tcp"},
		{"443", "tcp"},
		{"8080", "tcp"},
	}

	for _, p := range ports {
		if err := f.OpenPort(ctx, serverID, p.port, p.protocol); err != nil {
			f.logger.Warn("failed to open port", "port", p.port, "error", err)
			// Continue with other ports
		}
	}
	return nil
}

// ========== Detection ==========

func (f *FirewallService) detectUFW(ctx context.Context, exec *sshClientExecutor) (*FirewallStatus, error) {
	output, err := exec.RunCommand(ctx, "which ufw 2>/dev/null && ufw status 2>/dev/null")
	if err != nil || output == "" {
		return nil, fmt.Errorf("ufw not found")
	}
	enabled := strings.Contains(output, "Status: active")
	return &FirewallStatus{Type: FirewallTypeUFW, Enabled: enabled}, nil
}

func (f *FirewallService) detectFirewalld(ctx context.Context, exec *sshClientExecutor) (*FirewallStatus, error) {
	output, err := exec.RunCommand(ctx, "which firewall-cmd 2>/dev/null && firewall-cmd --state 2>/dev/null")
	if err != nil || !strings.Contains(output, "running") {
		return nil, fmt.Errorf("firewalld not found")
	}
	return &FirewallStatus{Type: FirewallTypeFirewalld, Enabled: true}, nil
}

func (f *FirewallService) detectIptables(ctx context.Context, exec *sshClientExecutor) (*FirewallStatus, error) {
	output, err := exec.RunCommand(ctx, "which iptables 2>/dev/null")
	if err != nil || output == "" {
		return nil, fmt.Errorf("iptables not found")
	}
	return &FirewallStatus{Type: FirewallTypeIptables, Enabled: true}, nil
}

// ========== UFW Operations ==========

func (f *FirewallService) getUFWStatus(ctx context.Context, exec *sshClientExecutor) (*FirewallStatus, error) {
	output, err := exec.RunCommand(ctx, "ufw status numbered 2>/dev/null")
	if err != nil {
		return nil, fmt.Errorf("failed to get ufw status: %w", err)
	}

	enabled := strings.Contains(output, "Status: active")
	rules := parseUFWOutput(output)

	return &FirewallStatus{
		Type:    FirewallTypeUFW,
		Enabled: enabled,
		Rules:   rules,
	}, nil
}

func (f *FirewallService) ufwAllow(ctx context.Context, exec *sshClientExecutor, port, protocol string) error {
	cmd := fmt.Sprintf("sudo ufw allow %s/%s", port, protocol)
	_, err := exec.RunCommand(ctx, cmd)
	return err
}

func (f *FirewallService) ufwDeny(ctx context.Context, exec *sshClientExecutor, port, protocol string) error {
	cmd := fmt.Sprintf("sudo ufw delete allow %s/%s", port, protocol)
	_, err := exec.RunCommand(ctx, cmd)
	return err
}

func (f *FirewallService) ufwDenyIP(ctx context.Context, exec *sshClientExecutor, ip string) error {
	cmd := fmt.Sprintf("sudo ufw deny from %s", ip)
	_, err := exec.RunCommand(ctx, cmd)
	return err
}

func (f *FirewallService) ufwAllowIP(ctx context.Context, exec *sshClientExecutor, ip string) error {
	cmd := fmt.Sprintf("sudo ufw delete deny from %s", ip)
	_, err := exec.RunCommand(ctx, cmd)
	return err
}

func parseUFWOutput(output string) []FirewallRule {
	var rules []FirewallRule
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Match lines like: "[ 1] 22/tcp                   ALLOW IN    Anywhere"
		re := regexp.MustCompile(`\[(\d+)\]\s+(\S+)\s+(ALLOW|DENY|REJECT)\s+IN\s+(.*)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 5 {
			rules = append(rules, FirewallRule{
				ID:     matches[1],
				Port:   matches[2],
				Action: matches[3],
				Target: strings.TrimSpace(matches[4]),
				Raw:    line,
			})
		}
	}
	return rules
}

// ========== firewalld Operations ==========

func (f *FirewallService) getFirewalldStatus(ctx context.Context, exec *sshClientExecutor) (*FirewallStatus, error) {
	output, err := exec.RunCommand(ctx, "sudo firewall-cmd --list-all 2>/dev/null")
	if err != nil {
		return nil, fmt.Errorf("failed to get firewalld status: %w", err)
	}

	rules := parseFirewalldOutput(output)
	return &FirewallStatus{
		Type:    FirewallTypeFirewalld,
		Enabled: true,
		Rules:   rules,
	}, nil
}

func (f *FirewallService) firewalldOpenPort(ctx context.Context, exec *sshClientExecutor, port, protocol string) error {
	cmd := fmt.Sprintf("sudo firewall-cmd --permanent --add-port=%s/%s && sudo firewall-cmd --reload", port, protocol)
	_, err := exec.RunCommand(ctx, cmd)
	return err
}

func (f *FirewallService) firewalldClosePort(ctx context.Context, exec *sshClientExecutor, port, protocol string) error {
	cmd := fmt.Sprintf("sudo firewall-cmd --permanent --remove-port=%s/%s && sudo firewall-cmd --reload", port, protocol)
	_, err := exec.RunCommand(ctx, cmd)
	return err
}

func (f *FirewallService) firewalldBlockIP(ctx context.Context, exec *sshClientExecutor, ip string) error {
	cmd := fmt.Sprintf("sudo firewall-cmd --permanent --add-rich-rule='rule family=ipv4 source address=%s reject' && sudo firewall-cmd --reload", ip)
	_, err := exec.RunCommand(ctx, cmd)
	return err
}

func (f *FirewallService) firewalldUnblockIP(ctx context.Context, exec *sshClientExecutor, ip string) error {
	cmd := fmt.Sprintf("sudo firewall-cmd --permanent --remove-rich-rule='rule family=ipv4 source address=%s reject' && sudo firewall-cmd --reload", ip)
	_, err := exec.RunCommand(ctx, cmd)
	return err
}

func parseFirewalldOutput(output string) []FirewallRule {
	var rules []FirewallRule
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ports:") {
			ports := strings.TrimSpace(strings.TrimPrefix(line, "ports:"))
			for _, port := range strings.Fields(ports) {
				parts := strings.Split(port, "/")
				proto := "tcp"
				if len(parts) == 2 {
					proto = parts[1]
				}
				rules = append(rules, FirewallRule{
					Port:     parts[0],
					Protocol: proto,
					Action:   "ACCEPT",
					Raw:      line,
				})
			}
		}
		if strings.Contains(line, "rich rule") {
			rules = append(rules, FirewallRule{
				Action: "REJECT",
				Raw:    line,
			})
		}
	}
	return rules
}

// ========== iptables Operations ==========

func (f *FirewallService) getIptablesStatus(ctx context.Context, exec *sshClientExecutor) (*FirewallStatus, error) {
	output, err := exec.RunCommand(ctx, "sudo iptables -L -n --line-numbers 2>/dev/null")
	if err != nil {
		return nil, fmt.Errorf("failed to get iptables status: %w", err)
	}

	rules := parseIptablesOutput(output)
	return &FirewallStatus{
		Type:    FirewallTypeIptables,
		Enabled: true,
		Rules:   rules,
	}, nil
}

func (f *FirewallService) iptablesOpenPort(ctx context.Context, exec *sshClientExecutor, port, protocol string) error {
	cmd := fmt.Sprintf("sudo iptables -A INPUT -p %s --dport %s -j ACCEPT", protocol, port)
	_, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}
	// Persist
	_, _ = exec.RunCommand(ctx, "sudo iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
	return nil
}

func (f *FirewallService) iptablesClosePort(ctx context.Context, exec *sshClientExecutor, port, protocol string) error {
	cmd := fmt.Sprintf("sudo iptables -D INPUT -p %s --dport %s -j ACCEPT", protocol, port)
	_, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}
	_, _ = exec.RunCommand(ctx, "sudo iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
	return nil
}

func (f *FirewallService) iptablesBlockIP(ctx context.Context, exec *sshClientExecutor, ip string) error {
	cmd := fmt.Sprintf("sudo iptables -A INPUT -s %s -j DROP", ip)
	_, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}
	_, _ = exec.RunCommand(ctx, "sudo iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
	return nil
}

func (f *FirewallService) iptablesUnblockIP(ctx context.Context, exec *sshClientExecutor, ip string) error {
	cmd := fmt.Sprintf("sudo iptables -D INPUT -s %s -j DROP", ip)
	_, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}
	_, _ = exec.RunCommand(ctx, "sudo iptables-save > /etc/iptables/rules.v4 2>/dev/null || true")
	return nil
}

func parseIptablesOutput(output string) []FirewallRule {
	var rules []FirewallRule
	lines := strings.Split(output, "\n")
	currentChain := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Chain ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentChain = parts[1]
			}
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 || currentChain == "" {
			continue
		}

		rule := FirewallRule{
			Chain: currentChain,
			Raw:   line,
		}

		for i, field := range fields {
			switch field {
			case "ACCEPT", "DROP", "REJECT":
				rule.Action = field
			case "tcp", "udp", "icmp":
				rule.Protocol = field
			case "dpt:":
				if i+1 < len(fields) {
					rule.Port = fields[i+1]
				}
			}
		}

		// Extract source
		for i, field := range fields {
			if field == "s" && i+1 < len(fields) && fields[i+1] != "0.0.0.0/0" {
				rule.Source = fields[i+1]
			}
		}

		rules = append(rules, rule)
	}
	return rules
}

// ========== Helpers ==========

func (f *FirewallService) getExecutor(ctx context.Context, serverID string) (*sshClientExecutor, error) {
	b := &Bridge{DB: f.db}
	return b.getRemoteExecutor(ctx, serverID)
}

// IsValidPort checks if a port number is valid.
func IsValidPort(port string) bool {
	p, err := strconv.Atoi(port)
	return err == nil && p >= 1 && p <= 65535
}

// IsValidIP checks if a string is a valid IPv4 address (basic check).
func IsValidIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return false
		}
	}
	return true
}
