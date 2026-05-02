package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Monitor represents an uptime monitoring target.
type Monitor struct {
	ID           string     `json:"id" gorm:"primaryKey"`
	TenantID     string     `json:"tenant_id" gorm:"column:tenant_id"`
	Name         string     `json:"name" gorm:"column:name;not null"`
	Type         string     `json:"type" gorm:"column:type;not null"`
	Target       string     `json:"target" gorm:"column:target;not null"`
	Interval     int        `json:"interval" gorm:"column:interval;default:60"`
	Timeout      int        `json:"timeout" gorm:"column:timeout;default:10"`
	Retries      int        `json:"retries" gorm:"column:retries;default:3"`
	Status       string     `json:"status" gorm:"column:status;default:unknown"`
	Enabled      bool       `json:"enabled" gorm:"column:enabled;default:1"`
	LastCheck    *time.Time `json:"last_check" gorm:"column:last_check"`
	LastStatus   string     `json:"last_status" gorm:"column:last_status"`
	Uptime       float64    `json:"uptime" gorm:"column:uptime;default:100"`
	TotalChecks  int        `json:"total_checks" gorm:"column:total_checks;default:0"`
	UpChecks     int        `json:"up_checks" gorm:"column:up_checks;default:0"`
	AvgLatency   float64    `json:"avg_latency" gorm:"column:avg_latency;default:0"`
	CreatedAt    time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"column:updated_at"`
}

func (Monitor) TableName() string {
	return "monitors"
}

// Heartbeat represents a heartbeat monitor.
type Heartbeat struct {
	ID        string     `json:"id" gorm:"primaryKey"`
	TenantID  string     `json:"tenant_id" gorm:"column:tenant_id"`
	Name      string     `json:"name" gorm:"column:name;not null"`
	Token     string     `json:"token" gorm:"column:token;not null;unique"`
	Interval  int        `json:"interval" gorm:"column:interval;default:60"`
	Timeout   int        `json:"timeout" gorm:"column:timeout;default:120"`
	Status    string     `json:"status" gorm:"column:status;default:unknown"`
	LastBeat  *time.Time `json:"last_beat" gorm:"column:last_beat"`
	Enabled   bool       `json:"enabled" gorm:"column:enabled;default:1"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"column:updated_at"`
}

func (Heartbeat) TableName() string {
	return "heartbeats"
}

// MonitorCheckResult represents a single check result.
type MonitorCheckResult struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	MonitorID  string    `json:"monitor_id" gorm:"column:monitor_id;not null"`
	Status     string    `json:"status" gorm:"column:status"`
	StatusCode int       `json:"status_code" gorm:"column:status_code;default:0"`
	Latency    float64   `json:"latency" gorm:"column:latency;default:0"`
	Message    string    `json:"message" gorm:"column:message"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at"`
}

func (MonitorCheckResult) TableName() string {
	return "monitor_check_results"
}

// MonitorService provides uptime monitoring and heartbeat detection logic.
type MonitorService struct {
	db *gorm.DB
}

// NewMonitorService creates a new MonitorService.
func NewMonitorService(db *gorm.DB) *MonitorService {
	return &MonitorService{db: db}
}

// ========== Monitor CRUD ==========

// CreateMonitor creates a new uptime monitor.
func (s *MonitorService) CreateMonitor(ctx context.Context, mon *Monitor) error {
	return s.db.WithContext(ctx).Create(mon).Error
}

// ListMonitors lists all monitors, optionally filtered by tenantID.
func (s *MonitorService) ListMonitors(ctx context.Context, tenantID string) (interface{}, error) {
	var monitors []Monitor
	query := s.db.WithContext(ctx).Order("created_at DESC")
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Find(&monitors).Error; err != nil {
		return nil, err
	}
	return monitors, nil
}

// GetMonitor gets a monitor by ID.
func (s *MonitorService) GetMonitor(ctx context.Context, id string) (*Monitor, error) {
	var mon Monitor
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&mon).Error; err != nil {
		return nil, err
	}
	return &mon, nil
}

// UpdateMonitor updates a monitor.
func (s *MonitorService) UpdateMonitor(ctx context.Context, mon *Monitor) error {
	return s.db.WithContext(ctx).Save(mon).Error
}

// DeleteMonitor deletes a monitor by ID.
func (s *MonitorService) DeleteMonitor(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&Monitor{}).Error
}

// ========== Monitor Checks ==========

// CheckMonitor triggers an immediate check on a specific monitor.
func (s *MonitorService) CheckMonitor(ctx context.Context, id string) (interface{}, error) {
	mon, err := s.GetMonitor(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("monitor not found: %w", err)
	}
	if !mon.Enabled {
		return nil, fmt.Errorf("monitor is disabled")
	}

	result := s.performCheck(ctx, mon)
	if err := s.db.WithContext(ctx).Create(result).Error; err != nil {
		return nil, err
	}
	s.updateMonitorStats(ctx, mon, result)
	return result, nil
}

// CheckAllMonitors checks all enabled monitors.
func (s *MonitorService) CheckAllMonitors(ctx context.Context) (interface{}, error) {
	var monitors []Monitor
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&monitors).Error; err != nil {
		return nil, err
	}

	var results []MonitorCheckResult
	for i := range monitors {
		result := s.performCheck(ctx, &monitors[i])
		if err := s.db.WithContext(ctx).Create(result).Error; err != nil {
			continue
		}
		s.updateMonitorStats(ctx, &monitors[i], result)
		results = append(results, *result)
	}
	return results, nil
}

// GetMonitorResults gets recent check results for a monitor.
func (s *MonitorService) GetMonitorResults(ctx context.Context, id string, limit int) (interface{}, error) {
	var results []MonitorCheckResult
	query := s.db.WithContext(ctx).Where("monitor_id = ?", id).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// GetMonitorSLA gets SLA metrics for a monitor over a time period.
func (s *MonitorService) GetMonitorSLA(ctx context.Context, id string, days int) (interface{}, error) {
	mon, err := s.GetMonitor(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("monitor not found: %w", err)
	}

	since := time.Now().AddDate(0, 0, -days)
	var totalChecks int64
	var upChecks int64

	s.db.WithContext(ctx).Model(&MonitorCheckResult{}).
		Where("monitor_id = ? AND created_at >= ?", id, since).
		Count(&totalChecks)

	s.db.WithContext(ctx).Model(&MonitorCheckResult{}).
		Where("monitor_id = ? AND created_at >= ? AND status = ?", id, since, "up").
		Count(&upChecks)

	var avgLatency float64
	s.db.WithContext(ctx).Model(&MonitorCheckResult{}).
		Where("monitor_id = ? AND created_at >= ?", id, since).
		Select("COALESCE(AVG(latency), 0)").
		Scan(&avgLatency)

	uptime := float64(100)
	if totalChecks > 0 {
		uptime = float64(upChecks) / float64(totalChecks) * 100
	}

	return map[string]interface{}{
		"monitor_id":   id,
		"monitor_name": mon.Name,
		"period_days":  days,
		"total_checks": totalChecks,
		"up_checks":    upChecks,
		"uptime":       fmt.Sprintf("%.2f", uptime),
		"avg_latency":  fmt.Sprintf("%.2f", avgLatency),
	}, nil
}

// ========== Heartbeat ==========

// CreateHeartbeat creates a new heartbeat monitor.
func (s *MonitorService) CreateHeartbeat(ctx context.Context, hb *Heartbeat) error {
	if hb.Token == "" {
		hb.Token = generateHeartbeatToken()
	}
	return s.db.WithContext(ctx).Create(hb).Error
}

// ListHeartbeats lists all heartbeats, optionally filtered by tenantID.
func (s *MonitorService) ListHeartbeats(ctx context.Context, tenantID string) (interface{}, error) {
	var heartbeats []Heartbeat
	query := s.db.WithContext(ctx).Order("created_at DESC")
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Find(&heartbeats).Error; err != nil {
		return nil, err
	}
	return heartbeats, nil
}

// DeleteHeartbeat deletes a heartbeat by ID.
func (s *MonitorService) DeleteHeartbeat(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&Heartbeat{}).Error
}

// PingHeartbeat receives a heartbeat ping, updating LastBeat.
func (s *MonitorService) PingHeartbeat(ctx context.Context, token string) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&Heartbeat{}).
		Where("token = ? AND enabled = ?", token, true).
		Updates(map[string]interface{}{
			"last_beat": now,
			"status":    "up",
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("heartbeat not found or disabled")
	}
	return nil
}

// CheckHeartbeats checks all heartbeats for timeouts and returns timed-out ones.
func (s *MonitorService) CheckHeartbeats(ctx context.Context) (interface{}, error) {
	var heartbeats []Heartbeat
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&heartbeats).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	var timedOut []Heartbeat
	for i := range heartbeats {
		hb := &heartbeats[i]
		if hb.LastBeat == nil {
			if time.Since(hb.CreatedAt) > time.Duration(hb.Timeout)*time.Second {
				hb.Status = "down"
				s.db.WithContext(ctx).Save(hb)
				timedOut = append(timedOut, *hb)
			}
			continue
		}
		elapsed := now.Sub(*hb.LastBeat)
		if elapsed > time.Duration(hb.Timeout)*time.Second {
			hb.Status = "down"
			s.db.WithContext(ctx).Save(hb)
			timedOut = append(timedOut, *hb)
		}
	}
	return timedOut, nil
}

// ========== Status Page ==========

// GetStatusPage returns public status page data.
func (s *MonitorService) GetStatusPage(ctx context.Context, tenantID string) (interface{}, error) {
	var monitors []Monitor
	query := s.db.WithContext(ctx).Where("enabled = ?", true)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Find(&monitors).Error; err != nil {
		return nil, err
	}

	allUp := true
	anyUp := false
	for _, m := range monitors {
		if m.Status == "up" {
			anyUp = true
		} else {
			allUp = false
		}
	}

	overallStatus := "unknown"
	if len(monitors) > 0 {
		if allUp {
			overallStatus = "operational"
		} else if anyUp {
			overallStatus = "partial_outage"
		} else {
			overallStatus = "major_outage"
		}
	}

	return map[string]interface{}{
		"overall_status": overallStatus,
		"monitors":       monitors,
		"total":          len(monitors),
		"up":             countUpMonitors(monitors),
		"down":           len(monitors) - countUpMonitors(monitors),
	}, nil
}

// ========== Prometheus Metrics ==========

// GetPrometheusMetrics exports metrics in Prometheus exposition format.
func (s *MonitorService) GetPrometheusMetrics(ctx context.Context) (string, error) {
	var monitors []Monitor
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&monitors).Error; err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, m := range monitors {
		labels := fmt.Sprintf(`name="%s",type="%s",target="%s"`, m.Name, m.Type, m.Target)
		sb.WriteString(fmt.Sprintf(`deploypilot_monitor_up{%s} %d`, labels, boolToInt(m.Status == "up")))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`deploypilot_monitor_uptime_percent{%s} %.2f`, labels, m.Uptime))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`deploypilot_monitor_avg_latency_ms{%s} %.2f`, labels, m.AvgLatency))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`deploypilot_monitor_total_checks{%s} %d`, labels, m.TotalChecks))
		sb.WriteString("\n")
	}

	var heartbeats []Heartbeat
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&heartbeats).Error; err != nil {
		return "", err
	}
	for _, hb := range heartbeats {
		labels := fmt.Sprintf(`name="%s"`, hb.Name)
		sb.WriteString(fmt.Sprintf(`deploypilot_heartbeat_up{%s} %d`, labels, boolToInt(hb.Status == "up")))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// ========== Internal helpers ==========

// performCheck executes a check based on the monitor type.
func (s *MonitorService) performCheck(ctx context.Context, mon *Monitor) *MonitorCheckResult {
	result := &MonitorCheckResult{
		ID:        generateID(),
		MonitorID: mon.ID,
		CreatedAt: time.Now(),
	}

	timeout := time.Duration(mon.Timeout) * time.Second
	switch mon.Type {
	case "http":
		result.Status, result.Latency, result.StatusCode, result.Message = checkHTTP(mon.Target, timeout)
	case "tcp":
		result.Status, result.Latency, result.StatusCode, result.Message = checkTCP(mon.Target, timeout)
	case "ping":
		result.Status, result.Latency, result.StatusCode, result.Message = checkPing(mon.Target, timeout)
	default:
		result.Status = "down"
		result.Message = fmt.Sprintf("unsupported monitor type: %s", mon.Type)
	}

	return result
}

// updateMonitorStats updates the monitor's statistics after a check.
func (s *MonitorService) updateMonitorStats(ctx context.Context, mon *Monitor, result *MonitorCheckResult) {
	mon.TotalChecks++
	if result.Status == "up" {
		mon.UpChecks++
	}
	mon.Status = result.Status
	mon.LastStatus = result.Status
	now := time.Now()
	mon.LastCheck = &now

	if mon.TotalChecks > 0 {
		mon.Uptime = float64(mon.UpChecks) / float64(mon.TotalChecks) * 100
	}

	totalLatency := mon.AvgLatency * float64(mon.TotalChecks-1)
	mon.AvgLatency = (totalLatency + result.Latency) / float64(mon.TotalChecks)

	s.db.WithContext(ctx).Save(mon)
}

// checkHTTP performs an HTTP check against the target URL.
func checkHTTP(target string, timeout time.Duration) (status string, latency float64, statusCode int, message string) {
	if !strings.HasPrefix(target, "http") {
		target = "http://" + target
	}

	client := &http.Client{Timeout: timeout}
	start := time.Now()
	resp, err := client.Get(target)
	latency = float64(time.Since(start).Milliseconds())

	if err != nil {
		return "down", latency, 0, err.Error()
	}
	defer resp.Body.Close()

	statusCode = resp.StatusCode
	if statusCode >= 200 && statusCode < 400 {
		return "up", latency, statusCode, ""
	}
	return "down", latency, statusCode, fmt.Sprintf("HTTP %d", statusCode)
}

// checkTCP performs a TCP connection check against host:port.
func checkTCP(target string, timeout time.Duration) (status string, latency float64, statusCode int, message string) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", target, timeout)
	latency = float64(time.Since(start).Milliseconds())

	if err != nil {
		return "down", latency, 0, err.Error()
	}
	conn.Close()
	return "up", latency, 0, ""
}

// checkPing performs an ICMP ping check against the target IP/hostname.
func checkPing(target string, timeout time.Duration) (status string, latency float64, statusCode int, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", fmt.Sprintf("%.0f", timeout.Seconds()), target)
	start := time.Now()
	output, err := cmd.CombinedOutput()
	latency = float64(time.Since(start).Milliseconds())

	if err != nil {
		return "down", latency, 0, string(output)
	}
	return "up", latency, 0, ""
}

// generateHeartbeatToken generates a 32-character hex token for heartbeat identification.
func generateHeartbeatToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// generateID generates a unique ID using crypto/rand.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// boolToInt converts a bool to 0 or 1 for Prometheus output.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// countUpMonitors counts monitors with status "up".
func countUpMonitors(monitors []Monitor) int {
	count := 0
	for _, m := range monitors {
		if m.Status == "up" {
			count++
		}
	}
	return count
}
