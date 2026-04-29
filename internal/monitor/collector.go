package monitor

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// MetricType defines the type of metric.
type MetricType string

const (
	MetricCPU       MetricType = "cpu"
	MetricMemory    MetricType = "memory"
	MetricDisk      MetricType = "disk"
	MetricContainer MetricType = "container"
	MetricNetwork   MetricType = "network"
	MetricDiskIO    MetricType = "disk_io"
)

// Metric represents a single metric data point.
type Metric struct {
	Type      MetricType         `json:"type"`
	Name      string             `json:"name"`
	Value     float64            `json:"value"`
	Unit      string             `json:"unit"` // percent, bytes, count
	Timestamp time.Time          `json:"timestamp"`
	Labels    map[string]string  `json:"labels,omitempty"`
}

// Collector gathers system and container metrics.
type Collector struct {
	executor deployer.CommandExecutor
}

// NewCollector creates a new Collector with the given command executor.
func NewCollector(executor deployer.CommandExecutor) *Collector {
	return &Collector{executor: executor}
}

// CollectSystemMetrics gathers CPU, memory, and disk usage metrics.
func (c *Collector) CollectSystemMetrics(ctx context.Context) ([]Metric, error) {
	var metrics []Metric
	now := time.Now()

	// CPU usage from /proc/stat (first line: aggregate)
	cpuOut, err := c.executor.RunCommand(ctx, "cat /proc/stat 2>/dev/null | head -1")
	if err == nil && cpuOut != "" {
		cpuPerc := parseCPUFromProcStat(cpuOut)
		metrics = append(metrics, Metric{
			Type:      MetricCPU,
			Name:      "cpu_usage",
			Value:     cpuPerc,
			Unit:      "percent",
			Timestamp: now,
		})
	}

	// Memory usage from free
	memOut, err := c.executor.RunCommand(ctx, "free -m 2>/dev/null | grep Mem")
	if err == nil && memOut != "" {
		memPerc, memUsed, memTotal := parseFreeMemory(memOut)
		metrics = append(metrics, Metric{
			Type:      MetricMemory,
			Name:      "memory_usage_percent",
			Value:     memPerc,
			Unit:      "percent",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricMemory,
			Name:      "memory_used_mb",
			Value:     float64(memUsed),
			Unit:      "megabytes",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricMemory,
			Name:      "memory_total_mb",
			Value:     float64(memTotal),
			Unit:      "megabytes",
			Timestamp: now,
		})
	}

	// Disk usage from df
	diskOut, err := c.executor.RunCommand(ctx, "df -h / 2>/dev/null | tail -1")
	if err == nil && diskOut != "" {
		diskPerc, diskUsed, diskAvail := parseDfOutput(diskOut)
		metrics = append(metrics, Metric{
			Type:      MetricDisk,
			Name:      "disk_usage_percent",
			Value:     diskPerc,
			Unit:      "percent",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricDisk,
			Name:      "disk_used_gb",
			Value:     diskUsed,
			Unit:      "gigabytes",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricDisk,
			Name:      "disk_available_gb",
			Value:     diskAvail,
			Unit:      "gigabytes",
			Timestamp: now,
		})
	}

	// Network I/O metrics
	networkMetrics := collectNetworkMetrics(ctx, c.executor)
	metrics = append(metrics, networkMetrics...)

	// Disk I/O metrics
	diskIOMetrics := collectDiskIOMetrics(ctx, c.executor)
	metrics = append(metrics, diskIOMetrics...)

	if len(metrics) == 0 {
		return nil, fmt.Errorf("failed to collect any system metrics")
	}

	return metrics, nil
}

// CollectContainerMetrics gathers container resource usage.
func (c *Collector) CollectContainerMetrics(ctx context.Context, containerName string) ([]Metric, error) {
	cmd := fmt.Sprintf(
		"docker stats --no-stream --format '{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}|{{.NetIO}}|{{.BlockIO}}' %s 2>/dev/null",
		containerName,
	)
	output, err := c.executor.RunCommand(ctx, cmd)
	if err != nil || output == "" {
		return nil, fmt.Errorf("failed to collect metrics for container %s: %w", containerName, err)
	}

	output = strings.TrimSpace(output)
	parts := strings.Split(output, "|")
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected docker stats output: %s", output)
	}

	now := time.Now()
	var metrics []Metric

	// CPU percentage
	cpuPerc := parsePercentValue(strings.TrimSpace(parts[0]))
	metrics = append(metrics, Metric{
		Type:      MetricCPU,
		Name:      "container_cpu",
		Value:     cpuPerc,
		Unit:      "percent",
		Timestamp: now,
		Labels:    map[string]string{"container": containerName},
	})

	// Memory usage (e.g. "50MiB / 1GiB")
	memStr := strings.TrimSpace(parts[1])
	memUsed, memUnit := parseMemoryValue(memStr)
	metrics = append(metrics, Metric{
		Type:      MetricMemory,
		Name:      "container_memory_used",
		Value:     memUsed,
		Unit:      memUnit,
		Timestamp: now,
		Labels:    map[string]string{"container": containerName},
	})

	// Memory percentage
	memPerc := parsePercentValue(strings.TrimSpace(parts[2]))
	metrics = append(metrics, Metric{
		Type:      MetricMemory,
		Name:      "container_memory_percent",
		Value:     memPerc,
		Unit:      "percent",
		Timestamp: now,
		Labels:    map[string]string{"container": containerName},
	})

	return metrics, nil
}

// CollectContainerHealth checks container health status.
func (c *Collector) CollectContainerHealth(ctx context.Context, containerName string) (*Metric, error) {
	cmd := fmt.Sprintf(
		"docker inspect --format '{{.State.Health.Status}}' %s 2>/dev/null",
		containerName,
	)
	output, err := c.executor.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get health for container %s: %w", containerName, err)
	}

	output = strings.TrimSpace(output)
	if output == "<nil>" || output == "" {
		output = "none"
	}

	// Convert health status to numeric: healthy=1, unhealthy=0, starting=0.5, none=-1
	var value float64
	switch output {
	case "healthy":
		value = 1
	case "unhealthy":
		value = 0
	case "starting":
		value = 0.5
	default:
		value = -1
	}

	return &Metric{
		Type:      MetricContainer,
		Name:      "container_health",
		Value:     value,
		Unit:      "status",
		Timestamp: time.Now(),
		Labels:    map[string]string{"container": containerName, "health": output},
	}, nil
}

// parseCPUFromProcStat extracts CPU usage percentage from /proc/stat first line.
// Format: cpu  user nice system idle iowait irq softirq steal guest guest_nice
func parseCPUFromProcStat(line string) float64 {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0
	}

	var values []float64
	for _, f := range fields[1:] {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			continue
		}
		values = append(values, v)
	}

	if len(values) < 4 {
		return 0
	}

	// idle = values[3], total = sum of all
	var total float64
	for _, v := range values {
		total += v
	}
	if total == 0 {
		return 0
	}

	idle := values[3]
	return (1 - idle/total) * 100
}

// parseFreeMemory parses the output of "free -m | grep Mem".
// Expected format: Mem:   total  used  free  shared  buff/cache  available
func parseFreeMemory(line string) (percent float64, used, total int) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return 0, 0, 0
	}
	total, _ = strconv.Atoi(fields[1])
	used, _ = strconv.Atoi(fields[2])
	if total == 0 {
		return 0, used, total
	}
	percent = float64(used) / float64(total) * 100
	return percent, used, total
}

// parseDfOutput parses the output of "df -h /".
// Expected format: /dev/sda1  size  used  avail  use%  mount
func parseDfOutput(line string) (percent float64, usedGB, availGB float64) {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0, 0, 0
	}

	pStr := strings.TrimSuffix(fields[4], "%")
	percent, _ = strconv.ParseFloat(pStr, 64)

	usedGB = parseSizeToGB(fields[2])
	availGB = parseSizeToGB(fields[3])

	return percent, usedGB, availGB
}

// parseSizeToGB converts a human-readable size string (e.g., "50G", "200M") to GB.
func parseSizeToGB(s string) float64 {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0
	}
	suffix := strings.ToUpper(s[len(s)-1:])
	numStr := s[:len(s)-1]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	switch suffix {
	case "T":
		return num * 1024
	case "G":
		return num
	case "M":
		return num / 1024
	case "K":
		return num / (1024 * 1024)
	default:
		return num
	}
}

// parsePercentValue parses a percentage string like "12.34%" to a float64.
func parsePercentValue(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// parseMemoryValue parses a memory string like "50MiB / 1GiB" into (value, unit).
func parseMemoryValue(s string) (float64, string) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 0 {
		return 0, "bytes"
	}
	memPart := strings.TrimSpace(parts[0])
	if len(memPart) == 0 {
		return 0, "bytes"
	}

	// Extract numeric part and unit
	var numStr string
	var unit string
	for i, c := range memPart {
		if c >= '0' && c <= '9' || c == '.' {
			numStr += string(c)
		} else {
			unit = memPart[i:]
			break
		}
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, "bytes"
	}

	// Normalize to MiB
	switch strings.ToUpper(unit) {
	case "GIB", "GB":
		return num * 1024, "mib"
	case "MIB", "MB":
		return num, "mib"
	case "KIB", "KB":
		return num / 1024, "mib"
	case "B":
		return num / (1024 * 1024), "mib"
	default:
		return num, "mib"
	}
}

// collectNetworkMetrics reads /proc/net/dev and returns per-interface network I/O metrics.
// Skips the loopback interface ("lo").
func collectNetworkMetrics(ctx context.Context, executor deployer.CommandExecutor) []Metric {
	var metrics []Metric
	now := time.Now()

	out, err := executor.RunCommand(ctx, "cat /proc/net/dev 2>/dev/null")
	if err != nil || out == "" {
		return metrics
	}

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ":") {
			continue
		}

		// Split interface name and data
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}

		fields := strings.Fields(parts[1])
		// Expected fields (at least 16):
		// receive: bytes, packets, errs, drop, fifo, frame, compressed, multicast
		// transmit: bytes, packets, errs, drop, fifo, colls, carrier, compressed
		if len(fields) < 10 {
			continue
		}

		recvBytes, _ := strconv.ParseFloat(fields[0], 64)
		recvPackets, _ := strconv.ParseFloat(fields[1], 64)
		recvErrors, _ := strconv.ParseFloat(fields[2], 64)
		txBytes, _ := strconv.ParseFloat(fields[8], 64)
		txPackets, _ := strconv.ParseFloat(fields[9], 64)
		txErrors, _ := strconv.ParseFloat(fields[10], 64)

		metrics = append(metrics, Metric{
			Type:      MetricNetwork,
			Name:      fmt.Sprintf("%s_rx_bytes", iface),
			Value:     recvBytes,
			Unit:      "bytes",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricNetwork,
			Name:      fmt.Sprintf("%s_rx_packets", iface),
			Value:     recvPackets,
			Unit:      "count",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricNetwork,
			Name:      fmt.Sprintf("%s_rx_errors", iface),
			Value:     recvErrors,
			Unit:      "count",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricNetwork,
			Name:      fmt.Sprintf("%s_tx_bytes", iface),
			Value:     txBytes,
			Unit:      "bytes",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricNetwork,
			Name:      fmt.Sprintf("%s_tx_packets", iface),
			Value:     txPackets,
			Unit:      "count",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricNetwork,
			Name:      fmt.Sprintf("%s_tx_errors", iface),
			Value:     txErrors,
			Unit:      "count",
			Timestamp: now,
		})
	}

	return metrics
}

// collectDiskIOMetrics reads /proc/diskstats and returns per-device disk I/O metrics.
// Focuses on main disks (sd[a-z], vd[a-z], nvme[0-9]+) and skips partitions.
func collectDiskIOMetrics(ctx context.Context, executor deployer.CommandExecutor) []Metric {
	var metrics []Metric
	now := time.Now()

	out, err := executor.RunCommand(ctx, "cat /proc/diskstats 2>/dev/null")
	if err != nil || out == "" {
		return metrics
	}

	// Pattern to match main disk devices, not partitions
	// Matches: sda, vda, xvda, nvme0n1, etc. Skips: sda1, vda2, etc.
	mainDiskRe := regexp.MustCompile(`^(sd[a-z]|vd[a-z]|xvd[a-z]|nvme\d+n\d+|mmcblk\d+)$`)

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		// /proc/diskstats format: major minor name reads_completed reads_merged sectors_read ...
		// At minimum: major, minor, name, and a few stat fields
		if len(fields) < 7 {
			continue
		}

		devName := fields[2]
		if !mainDiskRe.MatchString(devName) {
			continue
		}

		// Field indices (0-based): 0=major, 1=minor, 2=name
		// Stats start at index 3:
		// 3=reads completed, 4=reads merged, 5=sectors read,
		// 6=ms reading, 7=writes completed, 8=writes merged,
		// 9=sectors written, 10=ms writing
		readsCompleted, _ := strconv.ParseFloat(fields[3], 64)
		sectorsRead, _ := strconv.ParseFloat(fields[5], 64)
		writesCompleted, _ := strconv.ParseFloat(fields[7], 64)
		sectorsWritten, _ := strconv.ParseFloat(fields[9], 64)

		metrics = append(metrics, Metric{
			Type:      MetricDiskIO,
			Name:      fmt.Sprintf("%s_reads_completed", devName),
			Value:     readsCompleted,
			Unit:      "count",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricDiskIO,
			Name:      fmt.Sprintf("%s_sectors_read", devName),
			Value:     sectorsRead,
			Unit:      "sectors",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricDiskIO,
			Name:      fmt.Sprintf("%s_writes_completed", devName),
			Value:     writesCompleted,
			Unit:      "count",
			Timestamp: now,
		})
		metrics = append(metrics, Metric{
			Type:      MetricDiskIO,
			Name:      fmt.Sprintf("%s_sectors_written", devName),
			Value:     sectorsWritten,
			Unit:      "sectors",
			Timestamp: now,
		})
	}

	return metrics
}
