package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MonitorType defines the type of monitoring check.
type MonitorType string

const (
	MonitorHTTP MonitorType = "http"
	MonitorTCP  MonitorType = "tcp"
	MonitorPing MonitorType = "ping"
)

// MonitorStatus represents the status of a monitor check.
type MonitorStatus string

const (
	StatusUp      MonitorStatus = "up"
	StatusDown    MonitorStatus = "down"
	StatusDegraded MonitorStatus = "degraded"
)

// Monitor represents an uptime monitoring target.
type Monitor struct {
	ID          string `gorm:"primaryKey" json:"id"`
	TenantID    string `gorm:"index" json:"tenant_id"`
	Name        string `gorm:"not null;size:200" json:"name"`
	Type        string `gorm:"not null;size:20;index" json:"type"` // http, tcp, ping
	Target      string `gorm:"not null;size:500" json:"target"`   // URL, host:port, or IP
	Interval    int    `gorm:"default:60" json:"interval"`         // check interval in seconds
	Timeout     int    `gorm:"default:10" json:"timeout"`          // request timeout in seconds
	Retries     int    `gorm:"default:3" json:"retries"`           // consecutive failures before alert
	Status      string `gorm:"size:20;default:'unknown'" json:"status"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	LastCheck   string `gorm:"" json:"last_check"`
	LastStatus  string `gorm:"size:20" json:"last_status"`
	Uptime      float64 `gorm:"default:100" json:"uptime"`        // uptime percentage
	TotalChecks int    `gorm:"default:0" json:"total_checks"`
	UpChecks    int    `gorm:"default:0" json:"up_checks"`
	AvgLatency  float64 `gorm:"default:0" json:"avg_latency"`    // ms
	CreatedAt   string `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   string `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Monitor) TableName() string { return "monitors" }

// MonitorCheckResult represents the result of a single check.
type MonitorCheckResult struct {
	ID          string  `json:"id"`
	MonitorID   string  `json:"monitor_id"`
	Status      string  `json:"status"`      // up, down
	StatusCode  int     `json:"status_code"` // HTTP status code
	Latency     float64 `json:"latency"`     // ms
	Message     string  `json:"message"`
	CreatedAt   string  `json:"created_at"`
}

func (MonitorCheckResult) TableName() string { return "monitor_check_results" }

// Heartbeat represents an application heartbeat.
type Heartbeat struct {
	ID        string `gorm:"primaryKey" json:"id"`
	TenantID  string `gorm:"index" json:"tenant_id"`
	Name      string `gorm:"not null;size:200" json:"name"`
	Token     string `gorm:"not null;size:100;uniqueIndex" json:"token"` // unique token for heartbeat
	Interval  int    `gorm:"default:60" json:"interval"`                 // expected interval in seconds
	Timeout   int    `gorm:"default:120" json:"timeout"`                // alert after this many seconds
	Status    string `gorm:"size:20;default:'unknown'" json:"status"`
	LastBeat  string `gorm:"" json:"last_beat"`
	Enabled   bool   `gorm:"default:true" json:"enabled"`
	CreatedAt string `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt string `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Heartbeat) TableName() string { return "heartbeats" }

// MonitorService provides uptime monitoring, heartbeat detection, and metrics.
type MonitorService struct {
	db     *gorm.DB
	logger *slog.Logger
	client *http.Client
}

// NewMonitorService creates a new MonitorService.
func NewMonitorService(db *gorm.DB) *MonitorService {
	return &MonitorService{
		db:     db,
		logger: slog.Default(),
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // monitoring needs to check self-signed certs
				DialContext: (&net.Dialer{Timeout: 15 * time.Second}).DialContext,
			},
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// ========== Monitor CRUD ==========

// CreateMonitor creates a new monitor.
func (m *MonitorService) CreateMonitor(ctx context.Context, mon *Monitor) error {
	if mon.ID == "" {
		mon.ID = uuid.New().String()
	}
	return m.db.WithContext(ctx).Create(mon).Error
}

// ListMonitors lists all monitors.
func (m *MonitorService) ListMonitors(ctx context.Context, tenantID string) ([]Monitor, error) {
	var monitors []Monitor
	query := m.db.WithContext(ctx)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Order("created_at DESC").Find(&monitors).Error; err != nil {
		return nil, err
	}
	return monitors, nil
}

// GetMonitor gets a monitor by ID.
func (m *MonitorService) GetMonitor(ctx context.Context, id string) (*Monitor, error) {
	var mon Monitor
	if err := m.db.WithContext(ctx).First(&mon, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &mon, nil
}

// UpdateMonitor updates a monitor.
func (m *MonitorService) UpdateMonitor(ctx context.Context, mon *Monitor) error {
	return m.db.WithContext(ctx).Save(mon).Error
}

// DeleteMonitor deletes a monitor.
func (m *MonitorService) DeleteMonitor(ctx context.Context, id string) error {
	return m.db.WithContext(ctx).Delete(&Monitor{}, "id = ?", id).Error
}

// ========== Monitor Checks ==========

// CheckMonitor performs a single check on a monitor target.
func (m *MonitorService) CheckMonitor(ctx context.Context, monitorID string) (*MonitorCheckResult, error) {
	mon, err := m.GetMonitor(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	var result MonitorCheckResult
	result.ID = uuid.New().String()
	result.MonitorID = monitorID
	result.CreatedAt = time.Now().Format(time.RFC3339)

	start := time.Now()

	switch MonitorType(mon.Type) {
	case MonitorHTTP:
		result = m.checkHTTP(ctx, *mon, result)
	case MonitorTCP:
		result = m.checkTCP(*mon, result)
	case MonitorPing:
		result = m.checkPing(*mon, result)
	default:
		result.Status = "down"
		result.Message = fmt.Sprintf("unknown monitor type: %s", mon.Type)
	}

	latency := time.Since(start).Milliseconds()
	result.Latency = float64(latency)

	// Save result
	if err := m.db.WithContext(ctx).Create(&result).Error; err != nil {
		m.logger.Warn("failed to save check result", "error", err)
	}

	// Update monitor stats
	m.updateMonitorStats(ctx, mon, &result)

	return &result, nil
}

// CheckAllMonitors checks all enabled monitors.
func (m *MonitorService) CheckAllMonitors(ctx context.Context) ([]MonitorCheckResult, error) {
	var monitors []Monitor
	if err := m.db.WithContext(ctx).Where("enabled = ?", true).Find(&monitors).Error; err != nil {
		return nil, err
	}

	var results []MonitorCheckResult
	for _, mon := range monitors {
		result, err := m.CheckMonitor(ctx, mon.ID)
		if err != nil {
			m.logger.Warn("monitor check failed", "monitor_id", mon.ID, "error", err)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// GetMonitorResults gets recent check results for a monitor.
func (m *MonitorService) GetMonitorResults(ctx context.Context, monitorID string, limit int) ([]MonitorCheckResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var results []MonitorCheckResult
	if err := m.db.WithContext(ctx).Where("monitor_id = ?", monitorID).
		Order("created_at DESC").Limit(limit).Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// GetMonitorSLA returns the SLA (uptime percentage) for a monitor.
func (m *MonitorService) GetMonitorSLA(ctx context.Context, monitorID string, days int) (map[string]interface{}, error) {
	if days <= 0 {
		days = 30
	}

	since := time.Now().AddDate(0, 0, -days).Format(time.RFC3339)

	var totalChecks int64
	var upChecks int64
	m.db.WithContext(ctx).Model(&MonitorCheckResult{}).
		Where("monitor_id = ? AND created_at >= ?", monitorID, since).
		Count(&totalChecks)
	m.db.WithContext(ctx).Model(&MonitorCheckResult{}).
		Where("monitor_id = ? AND status = ? AND created_at >= ?", monitorID, "up", since).
		Count(&upChecks)

	uptime := float64(100)
	if totalChecks > 0 {
		uptime = float64(upChecks) / float64(totalChecks) * 100
	}

	// Average latency
	var avgLatency float64
	m.db.WithContext(ctx).Model(&MonitorCheckResult{}).
		Where("monitor_id = ? AND created_at >= ?", monitorID, since).
		Select("COALESCE(AVG(latency), 0)").Scan(&avgLatency)

	return map[string]interface{}{
		"monitor_id":   monitorID,
		"period_days":  days,
		"total_checks": totalChecks,
		"up_checks":    upChecks,
		"down_checks":  totalChecks - upChecks,
		"uptime_pct":   fmt.Sprintf("%.4f", uptime),
		"avg_latency":  fmt.Sprintf("%.1f", avgLatency),
	}, nil
}

// GetStatusPage returns a summary for the public status page.
func (m *MonitorService) GetStatusPage(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	var monitors []Monitor
	query := m.db.WithContext(ctx).Where("enabled = ?", true)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Find(&monitors).Error; err != nil {
		return nil, err
	}

	var overallUp, overallDown int
	var services []map[string]interface{}

	for _, mon := range monitors {
		status := mon.Status
		if status == "" {
			status = "unknown"
		}
		if status == "up" {
			overallUp++
		} else {
			overallDown++
		}

		services = append(services, map[string]interface{}{
			"name":       mon.Name,
			"type":       mon.Type,
			"target":     mon.Target,
			"status":     status,
			"uptime":     mon.Uptime,
			"last_check": mon.LastCheck,
			"avg_latency": mon.AvgLatency,
		})
	}

	overallStatus := "operational"
	if overallDown > 0 {
		overallStatus = "degraded"
	}
	if overallUp == 0 && overallDown > 0 {
		overallStatus = "down"
	}

	return map[string]interface{}{
		"overall_status": overallStatus,
		"total":          len(monitors),
		"up":             overallUp,
		"down":           overallDown,
		"services":       services,
	}, nil
}

// ========== Heartbeat ==========

// CreateHeartbeat creates a new heartbeat monitor.
func (m *MonitorService) CreateHeartbeat(ctx context.Context, hb *Heartbeat) error {
	if hb.ID == "" {
		hb.ID = uuid.New().String()
	}
	if hb.Token == "" {
		hb.Token = uuid.New().String()
	}
	return m.db.WithContext(ctx).Create(hb).Error
}

// ListHeartbeats lists all heartbeats.
func (m *MonitorService) ListHeartbeats(ctx context.Context, tenantID string) ([]Heartbeat, error) {
	var heartbeats []Heartbeat
	query := m.db.WithContext(ctx)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Order("created_at DESC").Find(&heartbeats).Error; err != nil {
		return nil, err
	}
	return heartbeats, nil
}

// GetHeartbeat gets a heartbeat by ID.
func (m *MonitorService) GetHeartbeat(ctx context.Context, id string) (*Heartbeat, error) {
	var hb Heartbeat
	if err := m.db.WithContext(ctx).First(&hb, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &hb, nil
}

// GetHeartbeatByToken finds a heartbeat by its unique token.
func (m *MonitorService) GetHeartbeatByToken(ctx context.Context, token string) (*Heartbeat, error) {
	var hb Heartbeat
	if err := m.db.WithContext(ctx).First(&hb, "token = ?", token).Error; err != nil {
		return nil, err
	}
	return &hb, nil
}

// PingHeartbeat records a heartbeat ping.
func (m *MonitorService) PingHeartbeat(ctx context.Context, token string) error {
	hb, err := m.GetHeartbeatByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("heartbeat not found: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	return m.db.WithContext(ctx).Model(&Heartbeat{}).Where("id = ?", hb.ID).Updates(map[string]interface{}{
		"last_beat": now,
		"status":    "up",
	}).Error
}

// DeleteHeartbeat deletes a heartbeat monitor.
func (m *MonitorService) DeleteHeartbeat(ctx context.Context, id string) error {
	return m.db.WithContext(ctx).Delete(&Heartbeat{}, "id = ?", id).Error
}

// CheckHeartbeats checks all heartbeats for timeouts.
func (m *MonitorService) CheckHeartbeats(ctx context.Context) ([]Heartbeat, error) {
	var heartbeats []Heartbeat
	if err := m.db.WithContext(ctx).Where("enabled = ?", true).Find(&heartbeats).Error; err != nil {
		return nil, err
	}

	var timedOut []Heartbeat
	now := time.Now()

	for _, hb := range heartbeats {
		if hb.LastBeat == "" {
			continue
		}

		lastBeat, err := time.Parse(time.RFC3339, hb.LastBeat)
		if err != nil {
			continue
		}

		timeout := time.Duration(hb.Timeout) * time.Second
		if timeout == 0 {
			timeout = 120 * time.Second
		}

		if now.Sub(lastBeat) > timeout {
			m.db.WithContext(ctx).Model(&Heartbeat{}).Where("id = ?", hb.ID).Update("status", "down")
			hb.Status = "down"
			timedOut = append(timedOut, hb)
		}
	}

	return timedOut, nil
}

// ========== Prometheus Metrics ==========

// GetPrometheusMetrics returns metrics in Prometheus exposition format.
func (m *MonitorService) GetPrometheusMetrics(ctx context.Context) (string, error) {
	var monitors []Monitor
	if err := m.db.WithContext(ctx).Where("enabled = ?", true).Find(&monitors).Error; err != nil {
		return "", err
	}

	var sb strings.Builder

	for _, mon := range monitors {
		labels := fmt.Sprintf(`name="%s",type="%s",target="%s"`, mon.Name, mon.Type, mon.Target)
		sb.WriteString(fmt.Sprintf(`deploypilot_monitor_up{%s} %d`, labels, boolToInt(mon.Status == "up")))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`deploypilot_monitor_latency_ms{%s} %.2f`, labels, mon.AvgLatency))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`deploypilot_monitor_uptime_pct{%s} %.4f`, labels, mon.Uptime))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`deploypilot_monitor_total_checks{%s} %d`, labels, mon.TotalChecks))
		sb.WriteString("\n")
	}

	// Heartbeat metrics
	var heartbeats []Heartbeat
	if err := m.db.WithContext(ctx).Where("enabled = ?", true).Find(&heartbeats).Error; err == nil {
		for _, hb := range heartbeats {
			labels := fmt.Sprintf(`name="%s"`, hb.Name)
			sb.WriteString(fmt.Sprintf(`deploypilot_heartbeat_up{%s} %d`, labels, boolToInt(hb.Status == "up")))
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// ========== Internal check methods ==========

func (m *MonitorService) checkHTTP(ctx context.Context, mon Monitor, result MonitorCheckResult) MonitorCheckResult {
	req, err := http.NewRequestWithContext(ctx, "GET", mon.Target, nil)
	if err != nil {
		result.Status = "down"
		result.Message = fmt.Sprintf("invalid URL: %v", err)
		return result
	}

	req.Header.Set("User-Agent", "DeployPilot-Monitor/1.0")

	timeout := time.Duration(mon.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	client := *m.client
	client.Timeout = timeout

	resp, err := client.Do(req)
	if err != nil {
		result.Status = "down"
		result.Message = fmt.Sprintf("connection failed: %v", err)
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	result.StatusCode = resp.StatusCode

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "up"
		result.Message = fmt.Sprintf("HTTP %d OK", resp.StatusCode)
	} else if resp.StatusCode >= 500 {
		result.Status = "down"
		result.Message = fmt.Sprintf("HTTP %d Server Error", resp.StatusCode)
	} else {
		result.Status = "degraded"
		result.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

func (m *MonitorService) checkTCP(mon Monitor, result MonitorCheckResult) MonitorCheckResult {
	timeout := time.Duration(mon.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	conn, err := net.DialTimeout("tcp", mon.Target, timeout)
	if err != nil {
		result.Status = "down"
		result.Message = fmt.Sprintf("connection refused: %v", err)
		return result
	}
	defer func() { _ = conn.Close() }()

	result.Status = "up"
	result.Message = fmt.Sprintf("TCP connection established to %s", mon.Target)
	return result
}

func (m *MonitorService) checkPing(mon Monitor, result MonitorCheckResult) MonitorCheckResult {
	// Use exec to run ping on the server where the monitor is configured
	// For now, do a simple TCP connection check as a fallback
	timeout := time.Duration(mon.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// Try ICMP-like check via TCP connection to common ports
	target := mon.Target
	if !strings.Contains(target, ":") {
		target = target + ":80"
	}

	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		result.Status = "down"
		result.Message = fmt.Sprintf("host unreachable: %v", err)
		return result
	}
	defer conn.Close()

	result.Status = "up"
	result.Message = fmt.Sprintf("host %s is reachable", mon.Target)
	return result
}

func (m *MonitorService) updateMonitorStats(ctx context.Context, mon *Monitor, result *MonitorCheckResult) {
	mon.TotalChecks++
	if result.Status == "up" {
		mon.UpChecks++
		mon.Status = "up"
	} else {
		mon.Status = "down"
	}

	if mon.TotalChecks > 0 {
		mon.Uptime = float64(mon.UpChecks) / float64(mon.TotalChecks) * 100
	}

	// Update average latency
	mon.AvgLatency = (mon.AvgLatency*float64(mon.TotalChecks-1) + result.Latency) / float64(mon.TotalChecks)

	mon.LastCheck = time.Now().Format(time.RFC3339)
	mon.LastStatus = result.Status

	m.db.WithContext(ctx).Save(mon)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
