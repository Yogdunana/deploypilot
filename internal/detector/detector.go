package detector

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// ComponentType represents the type of detected system component.
type ComponentType string

const (
	ComponentMySQL    ComponentType = "mysql"
	ComponentPostgres ComponentType = "postgresql"
	ComponentRedis    ComponentType = "redis"
	ComponentMongoDB  ComponentType = "mongodb"
	ComponentNginx    ComponentType = "nginx"
	ComponentApache   ComponentType = "apache"
	ComponentOpenResty ComponentType = "openresty"
)

// ComponentStatus represents the detection status of a component.
type ComponentStatus string

const (
	StatusInstalled ComponentStatus = "installed"
	StatusRunning   ComponentStatus = "running"
	StatusStopped   ComponentStatus = "stopped"
	StatusNotFound  ComponentStatus = "not_found"
)

// DetectedComponent represents a detected system component.
type DetectedComponent struct {
	Type     ComponentType   `json:"type"`
	Name     string          `json:"name"`
	Status   ComponentStatus `json:"status"`
	Version  string          `json:"version,omitempty"`
	Port     int             `json:"port,omitempty"`
	InstallPath string       `json:"install_path,omitempty"`
	BinaryPath   string      `json:"binary_path,omitempty"`
	Details  string          `json:"details,omitempty"`
}

// Detector detects system components (databases, web servers) on a remote host.
type Detector struct {
	executor deployer.CommandExecutor
	timeout  time.Duration
}

// New creates a new Detector.
func New(executor deployer.CommandExecutor) *Detector {
	return &Detector{
		executor: executor,
		timeout:  15 * time.Second,
	}
}

// DetectAll detects all supported system components.
func (d *Detector) DetectAll(ctx context.Context) ([]DetectedComponent, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	var results []DetectedComponent

	// Detect databases
	dbDetectors := []func(context.Context) (DetectedComponent, error){
		d.detectMySQL,
		d.detectPostgreSQL,
		d.detectRedis,
		d.detectMongoDB,
	}

	// Detect web servers
	webDetectors := []func(context.Context) (DetectedComponent, error){
		d.detectNginx,
		d.detectApache,
		d.detectOpenResty,
	}

	for _, detector := range dbDetectors {
		comp, err := detector(ctx)
		if err != nil {
			slog.Debug("detection failed", "error", err)
			continue
		}
		if comp.Status != StatusNotFound {
			results = append(results, comp)
		}
	}

	for _, detector := range webDetectors {
		comp, err := detector(ctx)
		if err != nil {
			slog.Debug("detection failed", "error", err)
			continue
		}
		if comp.Status != StatusNotFound {
			results = append(results, comp)
		}
	}

	return results, nil
}

// DetectDatabases detects only database components.
func (d *Detector) DetectDatabases(ctx context.Context) ([]DetectedComponent, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	var results []DetectedComponent
	detectors := []func(context.Context) (DetectedComponent, error){
		d.detectMySQL,
		d.detectPostgreSQL,
		d.detectRedis,
		d.detectMongoDB,
	}

	for _, detector := range detectors {
		comp, err := detector(ctx)
		if err != nil {
			continue
		}
		if comp.Status != StatusNotFound {
			results = append(results, comp)
		}
	}

	return results, nil
}

// DetectWebServers detects only web server components.
func (d *Detector) DetectWebServers(ctx context.Context) ([]DetectedComponent, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	var results []DetectedComponent
	detectors := []func(context.Context) (DetectedComponent, error){
		d.detectNginx,
		d.detectApache,
		d.detectOpenResty,
	}

	for _, detector := range detectors {
		comp, err := detector(ctx)
		if err != nil {
			continue
		}
		if comp.Status != StatusNotFound {
			results = append(results, comp)
		}
	}

	return results, nil
}

// detectMySQL detects MySQL/MariaDB instances.
func (d *Detector) detectMySQL(ctx context.Context) (DetectedComponent, error) {
	comp := DetectedComponent{Type: ComponentMySQL, Name: "MySQL/MariaDB"}

	// Check if mysqld or mariadbd is running
	output, err := d.executor.RunCommand(ctx, "ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1")
	if err != nil || output == "" {
		// Check if binary exists
		binOutput, binErr := d.executor.RunCommand(ctx, "which mysqld 2>/dev/null || which mariadbd 2>/dev/null")
		if binErr != nil || binOutput == "" {
			comp.Status = StatusNotFound
			return comp, nil
		}
		comp.Status = StatusStopped
		comp.BinaryPath = strings.TrimSpace(binOutput)
		return comp, nil
	}

	comp.Status = StatusRunning

	// Extract version
	if version := extractVersion(output); version != "" {
		comp.Version = version
	}

	// Check listening port
	portOutput, _ := d.executor.RunCommand(ctx, "ss -tlnp 2>/dev/null | grep -E 'mysqld|mariadbd' | grep -oP ':\\K[0-9]+' | head -1")
	if portOutput != "" {
		comp.Port, _ = strconv.Atoi(strings.TrimSpace(portOutput))
	}
	if comp.Port == 0 {
		comp.Port = 3306 // default
	}

	// Check binary path
	binOutput, _ := d.executor.RunCommand(ctx, "which mysqld 2>/dev/null || which mariadbd 2>/dev/null")
	if binOutput != "" {
		comp.BinaryPath = strings.TrimSpace(binOutput)
	}

	// Check if it's MariaDB
	if strings.Contains(output, "mariadbd") || strings.Contains(output, "MariaDB") {
		comp.Name = "MariaDB"
	}

	return comp, nil
}

// detectPostgreSQL detects PostgreSQL instances.
func (d *Detector) detectPostgreSQL(ctx context.Context) (DetectedComponent, error) {
	comp := DetectedComponent{Type: ComponentPostgres, Name: "PostgreSQL"}

	output, err := d.executor.RunCommand(ctx, "ps aux | grep postgres | grep -v grep | head -1")
	if err != nil || output == "" {
		binOutput, binErr := d.executor.RunCommand(ctx, "which postgres 2>/dev/null")
		if binErr != nil || binOutput == "" {
			comp.Status = StatusNotFound
			return comp, nil
		}
		comp.Status = StatusStopped
		comp.BinaryPath = strings.TrimSpace(binOutput)
		return comp, nil
	}

	comp.Status = StatusRunning

	if version := extractVersion(output); version != "" {
		comp.Version = version
	}

	// Check listening port
	portOutput, _ := d.executor.RunCommand(ctx, "ss -tlnp 2>/dev/null | grep postgres | grep -oP ':\\K[0-9]+' | head -1")
	if portOutput != "" {
		comp.Port, _ = strconv.Atoi(strings.TrimSpace(portOutput))
	}
	if comp.Port == 0 {
		comp.Port = 5432 // default
	}

	binOutput, _ := d.executor.RunCommand(ctx, "which postgres 2>/dev/null")
	if binOutput != "" {
		comp.BinaryPath = strings.TrimSpace(binOutput)
	}

	// Get data directory
	dataDir, _ := d.executor.RunCommand(ctx, "psql -U postgres -c \"SHOW data_directory;\" 2>/dev/null | sed -n '3p' | tr -d ' '")
	if dataDir != "" {
		comp.InstallPath = strings.TrimSpace(dataDir)
	}

	return comp, nil
}

// detectRedis detects Redis instances.
func (d *Detector) detectRedis(ctx context.Context) (DetectedComponent, error) {
	comp := DetectedComponent{Type: ComponentRedis, Name: "Redis"}

	output, err := d.executor.RunCommand(ctx, "ps aux | grep redis | grep -v grep | head -1")
	if err != nil || output == "" {
		binOutput, binErr := d.executor.RunCommand(ctx, "which redis-server 2>/dev/null")
		if binErr != nil || binOutput == "" {
			comp.Status = StatusNotFound
			return comp, nil
		}
		comp.Status = StatusStopped
		comp.BinaryPath = strings.TrimSpace(binOutput)
		return comp, nil
	}

	comp.Status = StatusRunning

	if version := extractVersion(output); version != "" {
		comp.Version = version
	}

	// Check listening port
	portOutput, _ := d.executor.RunCommand(ctx, "ss -tlnp 2>/dev/null | grep redis | grep -oP ':\\K[0-9]+' | head -1")
	if portOutput != "" {
		comp.Port, _ = strconv.Atoi(strings.TrimSpace(portOutput))
	}
	if comp.Port == 0 {
		comp.Port = 6379 // default
	}

	binOutput, _ := d.executor.RunCommand(ctx, "which redis-server 2>/dev/null")
	if binOutput != "" {
		comp.BinaryPath = strings.TrimSpace(binOutput)
	}

	// Get Redis info
	infoOutput, infoErr := d.executor.RunCommand(ctx, "redis-cli ping 2>/dev/null")
	if infoErr == nil && strings.TrimSpace(infoOutput) == "PONG" {
		comp.Details = "responding to PING"
	}

	return comp, nil
}

// detectMongoDB detects MongoDB instances.
func (d *Detector) detectMongoDB(ctx context.Context) (DetectedComponent, error) {
	comp := DetectedComponent{Type: ComponentMongoDB, Name: "MongoDB"}

	output, err := d.executor.RunCommand(ctx, "ps aux | grep mongod | grep -v grep | head -1")
	if err != nil || output == "" {
		binOutput, binErr := d.executor.RunCommand(ctx, "which mongod 2>/dev/null")
		if binErr != nil || binOutput == "" {
			comp.Status = StatusNotFound
			return comp, nil
		}
		comp.Status = StatusStopped
		comp.BinaryPath = strings.TrimSpace(binOutput)
		return comp, nil
	}

	comp.Status = StatusRunning

	if version := extractVersion(output); version != "" {
		comp.Version = version
	}

	// Check listening port
	portOutput, _ := d.executor.RunCommand(ctx, "ss -tlnp 2>/dev/null | grep mongod | grep -oP ':\\K[0-9]+' | head -1")
	if portOutput != "" {
		comp.Port, _ = strconv.Atoi(strings.TrimSpace(portOutput))
	}
	if comp.Port == 0 {
		comp.Port = 27017 // default
	}

	binOutput, _ := d.executor.RunCommand(ctx, "which mongod 2>/dev/null")
	if binOutput != "" {
		comp.BinaryPath = strings.TrimSpace(binOutput)
	}

	return comp, nil
}

// detectNginx detects Nginx instances.
func (d *Detector) detectNginx(ctx context.Context) (DetectedComponent, error) {
	comp := DetectedComponent{Type: ComponentNginx, Name: "Nginx"}

	output, err := d.executor.RunCommand(ctx, "ps aux | grep nginx | grep -v grep | head -1")
	if err != nil || output == "" {
		binOutput, binErr := d.executor.RunCommand(ctx, "which nginx 2>/dev/null")
		if binErr != nil || binOutput == "" {
			comp.Status = StatusNotFound
			return comp, nil
		}
		comp.Status = StatusStopped
		comp.BinaryPath = strings.TrimSpace(binOutput)
		return comp, nil
	}

	comp.Status = StatusRunning

	// Get version from binary
	verOutput, _ := d.executor.RunCommand(ctx, "nginx -v 2>&1")
	if verOutput != "" {
		comp.Version = extractVersionFromNginxOutput(verOutput)
	}

	// Check listening port
	portOutput, _ := d.executor.RunCommand(ctx, "ss -tlnp 2>/dev/null | grep nginx | grep -oP ':\\K[0-9]+' | head -1")
	if portOutput != "" {
		comp.Port, _ = strconv.Atoi(strings.TrimSpace(portOutput))
	}
	if comp.Port == 0 {
		comp.Port = 80 // default
	}

	binOutput, _ := d.executor.RunCommand(ctx, "which nginx 2>/dev/null")
	if binOutput != "" {
		comp.BinaryPath = strings.TrimSpace(binOutput)
	}

	// Get config path
	confOutput, _ := d.executor.RunCommand(ctx, "nginx -t 2>&1 | grep 'configuration file' | sed 's/.*configuration file //' | sed 's/ syntax.*//'")
	if confOutput != "" {
		comp.InstallPath = strings.TrimSpace(confOutput)
	}

	return comp, nil
}

// detectApache detects Apache httpd instances.
func (d *Detector) detectApache(ctx context.Context) (DetectedComponent, error) {
	comp := DetectedComponent{Type: ComponentApache, Name: "Apache HTTPD"}

	output, err := d.executor.RunCommand(ctx, "ps aux | grep -E 'httpd|apache2' | grep -v grep | head -1")
	if err != nil || output == "" {
		binOutput, binErr := d.executor.RunCommand(ctx, "which httpd 2>/dev/null || which apache2 2>/dev/null")
		if binErr != nil || binOutput == "" {
			comp.Status = StatusNotFound
			return comp, nil
		}
		comp.Status = StatusStopped
		comp.BinaryPath = strings.TrimSpace(binOutput)
		return comp, nil
	}

	comp.Status = StatusRunning

	// Get version
	verOutput, _ := d.executor.RunCommand(ctx, "httpd -v 2>/dev/null || apache2 -v 2>/dev/null")
	if verOutput != "" {
		comp.Version = extractVersion(verOutput)
	}

	// Check listening port
	portOutput, _ := d.executor.RunCommand(ctx, "ss -tlnp 2>/dev/null | grep -E 'httpd|apache2' | grep -oP ':\\K[0-9]+' | head -1")
	if portOutput != "" {
		comp.Port, _ = strconv.Atoi(strings.TrimSpace(portOutput))
	}
	if comp.Port == 0 {
		comp.Port = 80 // default
	}

	binOutput, _ := d.executor.RunCommand(ctx, "which httpd 2>/dev/null || which apache2 2>/dev/null")
	if binOutput != "" {
		comp.BinaryPath = strings.TrimSpace(binOutput)
	}

	return comp, nil
}

// detectOpenResty detects OpenResty instances.
func (d *Detector) detectOpenResty(ctx context.Context) (DetectedComponent, error) {
	comp := DetectedComponent{Type: ComponentOpenResty, Name: "OpenResty"}

	output, err := d.executor.RunCommand(ctx, "ps aux | grep openresty | grep -v grep | head -1")
	if err != nil || output == "" {
		binOutput, binErr := d.executor.RunCommand(ctx, "which openresty 2>/dev/null || which nginx 2>/dev/null")
		if binErr != nil || binOutput == "" {
			comp.Status = StatusNotFound
			return comp, nil
		}
		// Check if the nginx binary is actually OpenResty
		verOutput, verErr := d.executor.RunCommand(ctx, "nginx -v 2>&1")
		if verErr == nil && strings.Contains(verOutput, "openresty") {
			comp.Status = StatusStopped
			comp.BinaryPath = strings.TrimSpace(binOutput)
			comp.Version = extractVersionFromNginxOutput(verOutput)
			return comp, nil
		}
		comp.Status = StatusNotFound
		return comp, nil
	}

	comp.Status = StatusRunning

	verOutput, _ := d.executor.RunCommand(ctx, "nginx -v 2>&1")
	if verOutput != "" && strings.Contains(verOutput, "openresty") {
		comp.Version = extractVersionFromNginxOutput(verOutput)
	}

	portOutput, _ := d.executor.RunCommand(ctx, "ss -tlnp 2>/dev/null | grep openresty | grep -oP ':\\K[0-9]+' | head -1")
	if portOutput != "" {
		comp.Port, _ = strconv.Atoi(strings.TrimSpace(portOutput))
	}
	if comp.Port == 0 {
		comp.Port = 80 // default
	}

	binOutput, _ := d.executor.RunCommand(ctx, "which openresty 2>/dev/null || which nginx 2>/dev/null")
	if binOutput != "" {
		comp.BinaryPath = strings.TrimSpace(binOutput)
	}

	return comp, nil
}

// extractVersion extracts a version string from command output using regex.
func extractVersion(output string) string {
	patterns := []string{
		`(\d+\.\d+\.\d+[a-z0-9]*)`,  // e.g. 8.0.35, 15.2
		`v(\d+\.\d+)`,                 // e.g. v7.2
		`(\d+\.\d+)`,                  // e.g. 7.2
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(output)
		if len(matches) > 1 {
			return matches[1]
		}
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// extractVersionFromNginxOutput extracts version from nginx -v output.
// nginx -v output format: "nginx version: nginx/1.24.0" or "nginx version: openresty/1.25.3.1"
func extractVersionFromNginxOutput(output string) string {
	// Try to extract from "nginx/X.Y.Z" or "openresty/X.Y.Z"
	re := regexp.MustCompile(`(?:nginx|openresty)/([\d.]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return extractVersion(output)
}

// GetDetectionSummary returns a human-readable summary of detected components.
func GetDetectionSummary(components []DetectedComponent) string {
	if len(components) == 0 {
		return "No system components detected"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Detected %d component(s):\n", len(components)))
	for _, c := range components {
		statusIcon := "❌"
		switch c.Status {
		case StatusRunning:
			statusIcon = "✅"
		case StatusStopped:
			statusIcon = "⏸️"
		case StatusInstalled:
			statusIcon = "📦"
		}
		line := fmt.Sprintf("  %s %s", statusIcon, c.Name)
		if c.Version != "" {
			line += fmt.Sprintf(" (v%s)", c.Version)
		}
		if c.Port > 0 {
			line += fmt.Sprintf(" [:%d]", c.Port)
		}
		line += fmt.Sprintf(" — %s", c.Status)
		sb.WriteString(line + "\n")
	}
	return sb.String()
}
