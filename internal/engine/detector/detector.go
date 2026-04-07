package detector

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DetectionLevel represents the depth of environment detection.
type DetectionLevel int

const (
	LevelOS      DetectionLevel = 1 // OS info
	LevelDocker  DetectionLevel = 2 // + Docker availability
	LevelPort    DetectionLevel = 3 // + Port availability
	LevelService DetectionLevel = 4 // + Service health
)

// EnvReport holds the complete environment detection result.
type EnvReport struct {
	OS        *OSInfo              `json:"os"`
	Docker    *DockerInfo          `json:"docker,omitempty"`
	Ports     map[int]*PortInfo    `json:"ports,omitempty"`
	Services  map[string]*SvcInfo  `json:"services,omitempty"`
	Level     DetectionLevel       `json:"level"`
	Timestamp time.Time            `json:"timestamp"`
	Errors    []string             `json:"errors,omitempty"`
}

// OSInfo holds operating system information.
type OSInfo struct {
	GOOS   string `json:"goos"`
	Arch   string `json:"arch"`
	Kernel string `json:"kernel,omitempty"`
}

// DockerInfo holds Docker availability information.
type DockerInfo struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Running   bool   `json:"running"`
}

// PortInfo holds port availability information.
type PortInfo struct {
	Port     int       `json:"port"`
	Open     bool      `json:"open"`
	Protocol string    `json:"protocol"`
	Service  string    `json:"service,omitempty"`
}

// SvcInfo holds service health information.
type SvcInfo struct {
	Name     string `json:"name"`
	Healthy  bool   `json:"healthy"`
	Response string `json:"response,omitempty"`
	Latency  string `json:"latency,omitempty"`
	Error    string `json:"error,omitempty"`
}

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	Run(name string, args ...string) (string, error)
}

// Detector performs environment detection.
type Detector struct {
	runner CommandRunner
}

// New creates a new Detector with the given command runner.
func New(runner CommandRunner) *Detector {
	return &Detector{runner: runner}
}

// Detect performs environment detection up to the specified level.
func (d *Detector) Detect(ctx context.Context, level DetectionLevel, ports []int, services []string) *EnvReport {
	report := &EnvReport{
		Ports:    make(map[int]*PortInfo),
		Services: make(map[string]*SvcInfo),
		Level:    level,
		Timestamp: time.Now(),
	}

	// Level 1: OS detection
	report.OS = d.detectOS()
	if level >= LevelDocker {
		// Level 2: Docker detection
		report.Docker = d.detectDocker()
	}
	if level >= LevelPort {
		// Level 3: Port detection
		for _, port := range ports {
			report.Ports[port] = d.detectPort(port)
		}
	}
	if level >= LevelService {
		// Level 4: Service detection
		var wg sync.WaitGroup
		var mu sync.Mutex
		for _, svc := range services {
			wg.Add(1)
			go func(s string) {
				defer wg.Done()
				info := d.detectService(ctx, s)
				mu.Lock()
				report.Services[s] = info
				mu.Unlock()
			}(svc)
		}
		wg.Wait()
	}

	return report
}

func (d *Detector) detectOS() *OSInfo {
	return &OSInfo{
		GOOS: runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

func (d *Detector) detectDocker() *DockerInfo {
	info := &DockerInfo{}

	// Check if docker is installed
	out, err := d.runner.Run("docker", "--version")
	if err != nil {
		info.Installed = false
		return info
	}
	info.Installed = true
	info.Version = strings.TrimSpace(out)

	// Check if docker daemon is running
	_, err = d.runner.Run("docker", "info")
	info.Running = (err == nil)

	return info
}

func (d *Detector) detectPort(port int) *PortInfo {
	info := &PortInfo{
		Port:     port,
		Protocol: "tcp",
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err == nil {
		info.Open = true
		conn.Close()
	} else {
		info.Open = false
	}

	return info
}

func (d *Detector) detectService(ctx context.Context, target string) *SvcInfo {
	info := &SvcInfo{Name: target}

	start := time.Now()

	// Try TCP connection for tcp:// targets
	if strings.HasPrefix(target, "tcp://") {
		addr := strings.TrimPrefix(target, "tcp://")
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		info.Latency = time.Since(start).String()
		if err == nil {
			info.Healthy = true
			conn.Close()
		} else {
			info.Error = err.Error()
		}
		return info
	}

	// For other targets, just mark as unknown
	info.Error = "unsupported protocol"
	return info
}

// Summary returns a human-readable summary of the detection report.
func (r *EnvReport) Summary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Environment Detection (Level %d)\n", r.Level))
	sb.WriteString(fmt.Sprintf("OS: %s/%s\n", r.OS.GOOS, r.OS.Arch))

	if r.Docker != nil {
		status := "not installed"
		if r.Docker.Installed {
			status = r.Docker.Version
			if !r.Docker.Running {
				status += " (daemon not running)"
			}
		}
		sb.WriteString(fmt.Sprintf("Docker: %s\n", status))
	}

	for port, info := range r.Ports {
		state := "closed"
		if info.Open {
			state = "open"
		}
		sb.WriteString(fmt.Sprintf("Port %d: %s\n", port, state))
	}

	for name, svc := range r.Services {
		state := "unhealthy"
		if svc.Healthy {
			state = "healthy"
		}
		sb.WriteString(fmt.Sprintf("Service %s: %s\n", name, state))
	}

	return sb.String()
}
