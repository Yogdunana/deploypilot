package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ToolboxScript represents a saved script in the toolbox.
type ToolboxScript struct {
	ID          string `gorm:"primaryKey" json:"id"`
	TenantID    string `gorm:"index" json:"tenant_id"`
	Name        string `gorm:"not null;size:200" json:"name"`
	Description string `gorm:"size:500" json:"description"`
	Category    string `gorm:"size:50;index" json:"category"` // network, disk, security, system, custom
	Script      string `gorm:"type:text;not null" json:"script"`
	IsBuiltIn   bool   `gorm:"default:false" json:"is_built_in"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	CreatedAt   string `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   string `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ToolboxScript) TableName() string { return "toolbox_scripts" }

// ScriptExecutionResult represents the result of running a script.
type ScriptExecutionResult struct {
	ScriptID  string `json:"script_id"`
	ScriptName string `json:"script_name"`
	ExitCode  int    `json:"exit_code"`
	Output    string `json:"output"`
	Duration  string `json:"duration"`
	ServerID  string `json:"server_id"`
}

// SystemInfo represents detected system environment information.
type SystemInfo struct {
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	Kernel       string            `json:"kernel"`
	Hostname     string            `json:"hostname"`
	CPU          string            `json:"cpu"`
	Memory       string            `json:"memory"`
	Disk         string            `json:"disk"`
	Docker       bool              `json:"docker"`
	Services     map[string]bool   `json:"services"`
	OpenPorts    []string          `json:"open_ports"`
	PackageMgr   string            `json:"package_manager"`
	Environment  map[string]string `json:"environment"`
}

// ToolboxService provides common operations scripts and system detection.
type ToolboxService struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewToolboxService creates a new ToolboxService.
func NewToolboxService(db *gorm.DB) *ToolboxService {
	return &ToolboxService{
		db:     db,
		logger: slog.Default(),
	}
}

// DetectEnvironment detects the system environment of a remote server.
func (t *ToolboxService) DetectEnvironment(ctx context.Context, serverID string) (*SystemInfo, error) {
	exec, err := t.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	info := &SystemInfo{
		Services: make(map[string]bool),
	}

	// OS info
	if output, err := exec.RunCommand(ctx, "cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d'\"' -f2 || uname -s"); err == nil {
		info.OS = strings.TrimSpace(output)
	}

	// Architecture
	if output, err := exec.RunCommand(ctx, "uname -m"); err == nil {
		info.Arch = strings.TrimSpace(output)
	}

	// Kernel
	if output, err := exec.RunCommand(ctx, "uname -r"); err == nil {
		info.Kernel = strings.TrimSpace(output)
	}

	// Hostname
	if output, err := exec.RunCommand(ctx, "hostname"); err == nil {
		info.Hostname = strings.TrimSpace(output)
	}

	// CPU info
	if output, err := exec.RunCommand(ctx, "nproc 2>/dev/null && cat /proc/cpuinfo | grep 'model name' | head -1 | cut -d: -f2 | xargs"); err == nil {
		info.CPU = strings.TrimSpace(output)
	}

	// Memory
	if output, err := exec.RunCommand(ctx, "free -h | grep Mem | awk '{print $2}'"); err == nil {
		info.Memory = strings.TrimSpace(output)
	}

	// Disk
	if output, err := exec.RunCommand(ctx, "df -h / | tail -1 | awk '{print $2 \" total, \" $3 \" used, \" $4 \" available\"}'"); err == nil {
		info.Disk = strings.TrimSpace(output)
	}

	// Docker
	if output, err := exec.RunCommand(ctx, "docker --version 2>/dev/null"); err == nil && output != "" {
		info.Docker = true
	}

	// Services detection
	services := []string{"nginx", "apache2", "mysql", "mysqld", "postgresql", "redis-server", "docker", "supervisord", "cron", "sshd"}
	for _, svc := range services {
		if output, err := exec.RunCommand(ctx, fmt.Sprintf("systemctl is-active %s 2>/dev/null", util.ShellQuote(svc))); err == nil {
			info.Services[svc] = strings.TrimSpace(output) == "active"
		}
	}

	// Open ports
	if output, err := exec.RunCommand(ctx, "ss -tlnp 2>/dev/null | awk '{print $4}' | grep -oE '[0-9]+$' | sort -un | head -20"); err == nil {
		for _, port := range strings.Split(output, "\n") {
			port = strings.TrimSpace(port)
			if port != "" {
				info.OpenPorts = append(info.OpenPorts, port)
			}
		}
	}

	// Package manager
	for _, pm := range []string{"apt", "yum", "dnf", "pacman", "apk"} {
		if output, err := exec.RunCommand(ctx, fmt.Sprintf("which %s 2>/dev/null", util.ShellQuote(pm))); err == nil && output != "" {
			info.PackageMgr = pm
			break
		}
	}

	return info, nil
}

// RunScript runs a script on a remote server.
func (t *ToolboxService) RunScript(ctx context.Context, serverID, script string) (*ScriptExecutionResult, error) {
	exec, err := t.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	// Validate script - block dangerous commands
	if err := t.validateScript(script); err != nil {
		return nil, err
	}

	output, err := exec.RunCommand(ctx, script)
	exitCode := 0
	if err != nil {
		exitCode = 1
	}

	return &ScriptExecutionResult{
		ExitCode: exitCode,
		Output:   output,
		ServerID: serverID,
	}, nil
}

// RunBuiltInScript runs a built-in toolbox script by category and name.
func (t *ToolboxService) RunBuiltInScript(ctx context.Context, serverID, category, name string) (*ScriptExecutionResult, error) {
	script, err := t.GetBuiltInScript(category, name)
	if err != nil {
		return nil, err
	}

	result, err := t.RunScript(ctx, serverID, script)
	if err != nil {
		return nil, err
	}

	result.ScriptName = fmt.Sprintf("%s/%s", category, name)
	return result, nil
}

// GetBuiltInScript returns a built-in script by category and name.
func (t *ToolboxService) GetBuiltInScript(category, name string) (string, error) {
	scripts := t.getBuiltInScripts()
	key := fmt.Sprintf("%s/%s", category, name)
	script, ok := scripts[key]
	if !ok {
		return "", fmt.Errorf("built-in script not found: %s", key)
	}
	return script, nil
}

// ListBuiltInScripts lists all available built-in scripts.
func (t *ToolboxService) ListBuiltInScripts() map[string]string {
	return t.getBuiltInScripts()
}

// ========== Custom Script CRUD ==========

// CreateScript creates a new custom script.
func (t *ToolboxService) CreateScript(ctx context.Context, script *ToolboxScript) error {
	if script.ID == "" {
		script.ID = uuid.New().String()
	}
	script.IsBuiltIn = false
	return t.db.WithContext(ctx).Create(script).Error
}

// ListScripts lists all custom scripts.
func (t *ToolboxService) ListScripts(ctx context.Context, tenantID, category string) ([]ToolboxScript, error) {
	var scripts []ToolboxScript
	query := t.db.WithContext(ctx).Where("is_built_in = ?", false)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if err := query.Order("created_at DESC").Find(&scripts).Error; err != nil {
		return nil, err
	}
	return scripts, nil
}

// GetScript gets a script by ID.
func (t *ToolboxService) GetScript(ctx context.Context, id string) (*ToolboxScript, error) {
	var script ToolboxScript
	if err := t.db.WithContext(ctx).First(&script, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &script, nil
}

// UpdateScript updates a custom script.
func (t *ToolboxService) UpdateScript(ctx context.Context, script *ToolboxScript) error {
	return t.db.WithContext(ctx).Save(script).Error
}

// DeleteScript deletes a custom script.
func (t *ToolboxService) DeleteScript(ctx context.Context, id string) error {
	return t.db.WithContext(ctx).Delete(&ToolboxScript{}, "id = ?", id).Error
}

// ========== Helpers ==========

func (t *ToolboxService) getExecutor(ctx context.Context, serverID string) (*sshClientExecutor, error) {
	b := &Bridge{DB: t.db}
	return b.getRemoteExecutor(ctx, serverID)
}

// validateScript checks for dangerous commands.
func (t *ToolboxService) validateScript(script string) error {
	dangerousPatterns := []string{
		`rm\s+-rf\s+/`,
		`mkfs\.`,
		`dd\s+if=`,
		`>\s*/dev/sd`,
		`chmod\s+777\s+/`,
		`shutdown`,
		`reboot`,
		`init\s+0`,
		`:(){ :|:& };:`, // fork bomb
	}

	for _, pattern := range dangerousPatterns {
		matched, _ := regexp.MatchString(pattern, script)
		if matched {
			return fmt.Errorf("script contains dangerous command pattern: %s", pattern)
		}
	}
	return nil
}

// getBuiltInScripts returns the built-in script library.
func (t *ToolboxService) getBuiltInScripts() map[string]string {
	return map[string]string{
		// Network diagnostics
		"network/ping":      "ping -c 4 8.8.8.8 2>&1",
		"network/dns":       "nslookup google.com 2>&1 || dig google.com 2>&1",
		"network/ports":     "ss -tlnp 2>&1",
		"network/connections": "ss -tnp 2>&1 | head -50",
		"network/firewall":  "iptables -L -n 2>&1 | head -30 || ufw status 2>&1",
		"network/trace":     "traceroute -m 10 8.8.8.8 2>&1 || tracepath 8.8.8.8 2>&1",

		// Disk management
		"disk/usage":        "df -h 2>&1",
		"disk/inodes":       "df -i 2>&1",
		"disk/large-files":  "find / -type f -size +100M 2>/dev/null | head -20",
		"disk/duplicates":   "find /var/log -type f -name '*.gz' -size +50M 2>/dev/null | head -10",
		"disk/io":           "iostat -x 1 2 2>&1 || cat /proc/diskstats 2>&1 | head -10",

		// Security
		"security/users":        "last -10 2>&1",
		"security/ssh-auth":     "grep -E 'Failed|Accepted' /var/log/auth.log 2>&1 | tail -20 || journalctl -u sshd --no-pager -n 20 2>&1",
		"security/open-ports":   "ss -tlnp 2>&1",
		"security/sudo-users":   "getent group sudo 2>&1 || getent group wheel 2>&1",
		"security/cron-jobs":    "crontab -l 2>&1; ls -la /etc/cron.d/ 2>&1",

		// System
		"system/info":       "uname -a && cat /etc/os-release | grep PRETTY_NAME",
		"system/uptime":     "uptime",
		"system/memory":     "free -h",
		"system/cpu":        "top -bn1 | head -5",
		"system/load":       "cat /proc/loadavg",
		"system/processes":  "ps aux --sort=-%mem | head -15",
		"system/docker":     "docker ps 2>&1 && docker system df 2>&1",
		"system/services":   "systemctl list-units --type=service --state=running 2>&1 | head -20",

		// Log management
		"logs/journal":      "journalctl --no-pager -n 50 2>&1",
		"logs/nginx-error":  "tail -50 /var/log/nginx/error.log 2>&1",
		"logs/nginx-access": "tail -20 /var/log/nginx/access.log 2>&1",
		"logs/mysql":        "tail -30 /var/log/mysql/error.log 2>&1 || journalctl -u mysql --no-pager -n 30 2>&1",
		"logs/system":       "tail -50 /var/log/syslog 2>&1 || journalctl --no-pager -n 50 2>&1",

		// One-click fix
		"fix/clear-cache":   "sync && echo 3 > /proc/sys/vm/drop_caches 2>&1 && echo 'Cache cleared'",
		"fix/clear-logs":    "find /var/log -name '*.gz' -mtime +7 -delete 2>&1; find /tmp -type f -mtime +3 -delete 2>&1; echo 'Old logs cleaned'",
		"fix/restart-nginx": "systemctl restart nginx 2>&1 && systemctl status nginx --no-pager",
		"fix/restart-docker": "systemctl restart docker 2>&1 && systemctl status docker --no-pager",
		"fix/disk-cleanup":  "docker system prune -f 2>&1; apt-get clean 2>&1; yum clean all 2>&1; echo 'Disk cleanup done'",
	}
}
