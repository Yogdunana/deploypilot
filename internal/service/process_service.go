package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProcessInfo represents a running process on a remote server.
type ProcessInfo struct {
	PID         int     `json:"pid"`
	User        string  `json:"user"`
	CPU         float64 `json:"cpu"`
	Memory      float64 `json:"memory"` // MB
	VirtualMem  float64 `json:"virtual_memory"` // MB
	State       string  `json:"state"`  // R, S, D, Z, T
	Start       string  `json:"start"`
	Time        string  `json:"time"`
	Command     string  `json:"command"`
}

// ProcessStats represents aggregate process statistics.
type ProcessStats struct {
	Total      int     `json:"total"`
	Running    int     `json:"running"`
	Sleeping   int     `json:"sleeping"`
	Stopped    int     `json:"stopped"`
	Zombie     int     `json:"zombie"`
	TotalCPU   float64 `json:"total_cpu"`
	TotalMem   float64 `json:"total_mem"` // MB
	Processes  []ProcessInfo `json:"processes,omitempty"`
}

// ProcessRule represents a monitoring rule for auto-restart.
type ProcessRule struct {
	ID             string `gorm:"primaryKey" json:"id"`
	TenantID       string `gorm:"index" json:"tenant_id"`
	ServerID       string `gorm:"index" json:"server_id"`
	Name           string `gorm:"not null;size:100" json:"name"`
	ProcessPattern string `gorm:"not null;size:500" json:"process_pattern"` // grep pattern
	RestartCommand string `gorm:"size:1000" json:"restart_command"`
	AutoRestart    bool   `gorm:"default:false" json:"auto_restart"`
	MaxRestarts    int    `gorm:"default:5" json:"max_restarts"`
	RestartCount   int    `gorm:"default:0" json:"restart_count"`
	Enabled        bool   `gorm:"default:true" json:"enabled"`
	CreatedAt      string `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      string `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ProcessRule) TableName() string { return "process_rules" }

// ProcessService manages remote server processes via SSH.
type ProcessService struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewProcessService creates a new ProcessService.
func NewProcessService(db *gorm.DB) *ProcessService {
	return &ProcessService{
		db:     db,
		logger: slog.Default(),
	}
}

// ListProcesses lists processes on a remote server.
func (p *ProcessService) ListProcesses(ctx context.Context, serverID string, filter string) (*ProcessStats, error) {
	exec, err := p.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	cmd := "ps aux --no-headers 2>/dev/null || ps aux 2>/dev/null"
	if filter != "" {
		cmd = fmt.Sprintf("ps aux --no-headers 2>/dev/null | grep -E %s | grep -v grep || ps aux 2>/dev/null | grep -E %s | grep -v grep",
			util.ShellQuote(filter), util.ShellQuote(filter))
	}

	output, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	processes, stats := parsePsAux(output)
	stats.Processes = processes

	return &stats, nil
}

// GetProcess gets details of a specific process by PID.
func (p *ProcessService) GetProcess(ctx context.Context, serverID string, pid int) (*ProcessInfo, error) {
	exec, err := p.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	cmd := fmt.Sprintf("ps -p %d -o pid=,user=,%%cpu=,%%mem=,vsz=,rss=,stat=,start=,time=,args= --no-headers 2>/dev/null", pid)
	output, err := exec.RunCommand(ctx, cmd)
	if err != nil || output == "" {
		return nil, fmt.Errorf("process %d not found", pid)
	}

	processes, _ := parsePsAux(output)
	if len(processes) == 0 {
		return nil, fmt.Errorf("process %d not found", pid)
	}
	return &processes[0], nil
}

// KillProcess sends a signal to a process.
func (p *ProcessService) KillProcess(ctx context.Context, serverID string, pid int, signal string) error {
	exec, err := p.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	if signal == "" {
		signal = "TERM"
	}

	// Validate signal
	validSignals := map[string]bool{
		"TERM": true, "KILL": true, "HUP": true, "INT": true,
		"USR1": true, "USR2": true, "STOP": true, "CONT": true,
	}
	if !validSignals[signal] {
		return fmt.Errorf("invalid signal: %s", signal)
	}

	cmd := fmt.Sprintf("sudo kill -%s %d", signal, pid)
	_, err = exec.RunCommand(ctx, cmd)
	return err
}

// SearchProcesses searches for processes matching a pattern.
func (p *ProcessService) SearchProcesses(ctx context.Context, serverID, pattern string) ([]ProcessInfo, error) {
	exec, err := p.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	cmd := fmt.Sprintf("ps aux --no-headers 2>/dev/null | grep -E %s | grep -v grep || echo ''", util.ShellQuote(pattern))
	output, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to search processes: %w", err)
	}

	if output == "" || strings.TrimSpace(output) == "" {
		return []ProcessInfo{}, nil
	}

	processes, _ := parsePsAux(output)
	return processes, nil
}

// GetProcessTree returns the process tree (parent-child relationships).
func (p *ProcessService) GetProcessTree(ctx context.Context, serverID string) ([]ProcessInfo, error) {
	exec, err := p.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	cmd := "ps auxf --no-headers 2>/dev/null || ps auxf 2>/dev/null"
	output, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get process tree: %w", err)
	}

	processes, _ := parsePsAux(output)
	return processes, nil
}

// SystemResources returns system resource usage (CPU, memory, disk, uptime).
func (p *ProcessService) SystemResources(ctx context.Context, serverID string) (map[string]interface{}, error) {
	exec, err := p.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	result := make(map[string]interface{})

	// Uptime
	if output, err := exec.RunCommand(ctx, "uptime -p 2>/dev/null || uptime 2>/dev/null"); err == nil {
		result["uptime"] = strings.TrimSpace(output)
	}

	// CPU usage (1-second sample)
	if output, err := exec.RunCommand(ctx, "top -bn1 | head -5 2>/dev/null"); err == nil {
		result["top_summary"] = strings.TrimSpace(output)
	}

	// Memory info
	if output, err := exec.RunCommand(ctx, "free -m 2>/dev/null"); err == nil {
		result["memory"] = strings.TrimSpace(output)
	}

	// Disk usage
	if output, err := exec.RunCommand(ctx, "df -h --total 2>/dev/null | tail -1"); err == nil {
		result["disk"] = strings.TrimSpace(output)
	}

	// Load average
	if output, err := exec.RunCommand(ctx, "cat /proc/loadavg 2>/dev/null"); err == nil {
		result["load_average"] = strings.TrimSpace(output)
	}

	return result, nil
}

// ========== Process Rules (CRUD) ==========

// CreateRule creates a new process monitoring rule.
func (p *ProcessService) CreateRule(ctx context.Context, rule *ProcessRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	return p.db.WithContext(ctx).Create(rule).Error
}

// ListRules lists all process monitoring rules.
func (p *ProcessService) ListRules(ctx context.Context, tenantID, serverID string) ([]ProcessRule, error) {
	var rules []ProcessRule
	query := p.db.WithContext(ctx).Where("enabled = ?", true)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if serverID != "" {
		query = query.Where("server_id = ?", serverID)
	}
	if err := query.Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// GetRule gets a process monitoring rule by ID.
func (p *ProcessService) GetRule(ctx context.Context, id string) (*ProcessRule, error) {
	var rule ProcessRule
	if err := p.db.WithContext(ctx).First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateRule updates a process monitoring rule.
func (p *ProcessService) UpdateRule(ctx context.Context, rule *ProcessRule) error {
	return p.db.WithContext(ctx).Save(rule).Error
}

// DeleteRule deletes a process monitoring rule.
func (p *ProcessService) DeleteRule(ctx context.Context, id string) error {
	return p.db.WithContext(ctx).Delete(&ProcessRule{}, "id = ?", id).Error
}

// ========== Helpers ==========

func (p *ProcessService) getExecutor(ctx context.Context, serverID string) (*sshClientExecutor, error) {
	b := &Bridge{DB: p.db}
	return b.getRemoteExecutor(ctx, serverID)
}

// parsePsAux parses the output of `ps aux` into ProcessInfo structs.
func parsePsAux(output string) ([]ProcessInfo, ProcessStats) {
	var processes []ProcessInfo
	stats := ProcessStats{}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		info := parsePsAuxLine(line)
		if info == nil {
			continue
		}

		processes = append(processes, *info)
		stats.Total++
		stats.TotalCPU += info.CPU
		stats.TotalMem += info.Memory

		switch info.State {
		case "R":
			stats.Running++
		case "S":
			stats.Sleeping++
		case "D":
			stats.Stopped++
		case "Z":
			stats.Zombie++
		case "T":
			stats.Stopped++
		}
	}

	return processes, stats
}

// psAuxRegex matches a line from `ps aux` output.
// Format: USER PID %CPU %MEM VSZ RSS STAT START TIME COMMAND
var psAuxRegex = regexp.MustCompile(`^(\S+)\s+(\d+)\s+([\d.]+)\s+([\d.]+)\s+(\d+)\s+(\d+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.+)$`)

func parsePsAuxLine(line string) *ProcessInfo {
	matches := psAuxRegex.FindStringSubmatch(line)
	if len(matches) < 11 {
		return nil
	}

	cpu, _ := strconv.ParseFloat(matches[3], 64)
	vsz, _ := strconv.ParseFloat(matches[5], 64)
	rss, _ := strconv.ParseFloat(matches[6], 64)
	pid, _ := strconv.Atoi(matches[2])

	return &ProcessInfo{
		PID:        pid,
		User:       matches[1],
		CPU:        cpu,
		Memory:     rss / 1024, // Convert KB to MB
		VirtualMem: vsz / 1024, // Convert KB to MB
		State:      matches[7],
		Start:      matches[8],
		Time:       matches[9],
		Command:    matches[10],
	}
}
