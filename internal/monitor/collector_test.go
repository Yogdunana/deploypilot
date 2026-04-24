package monitor

import (
	"context"
	"testing"
)

// mockCollectorExecutor implements deployer.CommandExecutor for testing.
type mockCollectorExecutor struct {
	responses map[string]string
	errs      map[string]error
}

func (m *mockCollectorExecutor) RunCommand(_ context.Context, cmd string) (string, error) {
	if err, ok := m.errs[cmd]; ok {
		return "", err
	}
	for pattern, output := range m.responses {
		for i := 0; i <= len(cmd)-len(pattern); i++ {
			if cmd[i:i+len(pattern)] == pattern {
				return output, nil
			}
		}
	}
	return "", nil
}

func TestCollectSystemMetrics(t *testing.T) {
	exec := &mockCollectorExecutor{
		responses: map[string]string{
			"cat /proc/stat":  "cpu  100 10 50 840 0 0 0 0 0 0",
			"free -m":         "Mem:   8000  4000  3000  100  1000  3500",
			"df -h /":         "/dev/sda1  100G  45G   50G  48% /",
		},
	}
	c := NewCollector(exec)

	metrics, err := c.CollectSystemMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(metrics) == 0 {
		t.Fatal("expected metrics to be collected")
	}

	// Check CPU metric
	foundCPU := false
	foundMem := false
	foundDisk := false
	for _, m := range metrics {
		if m.Name == "cpu_usage" && m.Type == MetricCPU {
			foundCPU = true
			if m.Unit != "percent" {
				t.Errorf("expected unit 'percent', got %q", m.Unit)
			}
		}
		if m.Name == "memory_usage_percent" && m.Type == MetricMemory {
			foundMem = true
		}
		if m.Name == "disk_usage_percent" && m.Type == MetricDisk {
			foundDisk = true
			if m.Value != 48 {
				t.Errorf("expected disk usage 48%%, got %.2f", m.Value)
			}
		}
	}

	if !foundCPU {
		t.Error("expected cpu_usage metric")
	}
	if !foundMem {
		t.Error("expected memory_usage_percent metric")
	}
	if !foundDisk {
		t.Error("expected disk_usage_percent metric")
	}
}

func TestCollectContainerMetrics(t *testing.T) {
	exec := &mockCollectorExecutor{
		responses: map[string]string{
			"docker stats": "12.34%|50MiB / 1GiB|5.00%|1.2kB / 0B|0B / 0B",
		},
	}
	c := NewCollector(exec)

	metrics, err := c.CollectContainerMetrics(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(metrics) == 0 {
		t.Fatal("expected container metrics")
	}

	foundCPU := false
	foundMemPerc := false
	for _, m := range metrics {
		if m.Name == "container_cpu" {
			foundCPU = true
			if m.Value != 12.34 {
				t.Errorf("expected CPU 12.34%%, got %.2f", m.Value)
			}
		}
		if m.Name == "container_memory_percent" {
			foundMemPerc = true
			if m.Value != 5.0 {
				t.Errorf("expected memory percent 5.0%%, got %.2f", m.Value)
			}
		}
		if m.Labels != nil && m.Labels["container"] != "my-app" {
			t.Errorf("expected container label 'my-app', got %q", m.Labels["container"])
		}
	}

	if !foundCPU {
		t.Error("expected container_cpu metric")
	}
	if !foundMemPerc {
		t.Error("expected container_memory_percent metric")
	}
}

func TestCollectContainerMetrics_Empty(t *testing.T) {
	exec := &mockCollectorExecutor{
		responses: map[string]string{},
	}
	c := NewCollector(exec)

	_, err := c.CollectContainerMetrics(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing container")
	}
}

func TestCollectContainerHealth(t *testing.T) {
	exec := &mockCollectorExecutor{
		responses: map[string]string{
			"State.Health.Status": "healthy",
		},
	}
	c := NewCollector(exec)

	metric, err := c.CollectContainerHealth(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metric.Value != 1 {
		t.Errorf("expected healthy value 1, got %.1f", metric.Value)
	}
	if metric.Labels["health"] != "healthy" {
		t.Errorf("expected health label 'healthy', got %q", metric.Labels["health"])
	}
}

func TestParseCPUFromProcStat(t *testing.T) {
	tests := []struct {
		line string
		want float64
	}{
		{"cpu  100 10 50 840 0 0 0 0 0 0", 16.0},
		{"cpu  0 0 0 1000 0 0 0 0 0 0", 0.0},
		{"cpu  1000 0 0 0 0 0 0 0 0 0", 100.0},
	}
	for _, tt := range tests {
		got := parseCPUFromProcStat(tt.line)
		if got < tt.want-1 || got > tt.want+1 {
			t.Errorf("parseCPUFromProcStat(%q) = %.2f, want ~%.2f", tt.line, got, tt.want)
		}
	}
}

func TestParseFreeMemory(t *testing.T) {
	perc, used, total := parseFreeMemory("Mem:   8000  4000  3000  100  1000  3500")
	if total != 8000 {
		t.Errorf("expected total 8000, got %d", total)
	}
	if used != 4000 {
		t.Errorf("expected used 4000, got %d", used)
	}
	if perc < 49 || perc > 51 {
		t.Errorf("expected percent ~50, got %.2f", perc)
	}
}

func TestParseDfOutput(t *testing.T) {
	perc, used, avail := parseDfOutput("/dev/sda1  100G  45G   50G  48% /")
	if perc != 48 {
		t.Errorf("expected percent 48, got %.2f", perc)
	}
	if used != 45 {
		t.Errorf("expected used 45GB, got %.2f", used)
	}
	if avail != 50 {
		t.Errorf("expected avail 50GB, got %.2f", avail)
	}
}

func TestParsePercentValue(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"12.34%", 12.34},
		{"95%", 95.0},
		{"0%", 0.0},
		{"invalid", 0.0},
	}
	for _, tt := range tests {
		got := parsePercentValue(tt.input)
		if got != tt.want {
			t.Errorf("parsePercentValue(%q) = %.2f, want %.2f", tt.input, got, tt.want)
		}
	}
}

func TestParseSizeToGB(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"50G", 50},
		{"200M", 200.0 / 1024},
		{"1T", 1024},
		{"100K", 100.0 / (1024 * 1024)},
		{"", 0},
	}
	for _, tt := range tests {
		got := parseSizeToGB(tt.input)
		if got < tt.want-0.01 || got > tt.want+0.01 {
			t.Errorf("parseSizeToGB(%q) = %.4f, want %.4f", tt.input, got, tt.want)
		}
	}
}

func TestParseMemoryValue(t *testing.T) {
	tests := []struct {
		input   string
		wantVal float64
		wantU   string
	}{
		{"50MiB / 1GiB", 50, "mib"},
		{"1GiB / 2GiB", 1024, "mib"},
		{"128KiB / 1GiB", 0.125, "mib"},
		{"100B / 1GiB", 100.0 / (1024 * 1024), "mib"},
		{"", 0, "bytes"},
	}
	for _, tt := range tests {
		gotVal, gotU := parseMemoryValue(tt.input)
		if gotVal < tt.wantVal-0.01 || gotVal > tt.wantVal+0.01 {
			t.Errorf("parseMemoryValue(%q) value = %.4f, want ~%.4f", tt.input, gotVal, tt.wantVal)
		}
		if gotU != tt.wantU {
			t.Errorf("parseMemoryValue(%q) unit = %q, want %q", tt.input, gotU, tt.wantU)
		}
	}
}

func TestCollectSystemMetrics_Failure(t *testing.T) {
	exec := &mockCollectorExecutor{
		responses: map[string]string{},
	}
	c := NewCollector(exec)

	_, err := c.CollectSystemMetrics(context.Background())
	if err == nil {
		t.Fatal("expected error when all commands fail")
	}
}

func TestCollectContainerHealth_Nil(t *testing.T) {
	exec := &mockCollectorExecutor{
		responses: map[string]string{
			"State.Health.Status": "<nil>",
		},
	}
	c := NewCollector(exec)

	metric, err := c.CollectContainerHealth(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metric.Labels["health"] != "none" {
		t.Errorf("expected health 'none' for nil, got %q", metric.Labels["health"])
	}
	if metric.Value != -1 {
		t.Errorf("expected value -1 for none, got %.1f", metric.Value)
	}
}

func TestCollectContainerHealth_Unhealthy(t *testing.T) {
	exec := &mockCollectorExecutor{
		responses: map[string]string{
			"State.Health.Status": "unhealthy",
		},
	}
	c := NewCollector(exec)

	metric, err := c.CollectContainerHealth(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metric.Value != 0 {
		t.Errorf("expected value 0 for unhealthy, got %.1f", metric.Value)
	}
}

func TestCollectContainerHealth_Starting(t *testing.T) {
	exec := &mockCollectorExecutor{
		responses: map[string]string{
			"State.Health.Status": "starting",
		},
	}
	c := NewCollector(exec)

	metric, err := c.CollectContainerHealth(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metric.Value != 0.5 {
		t.Errorf("expected value 0.5 for starting, got %.1f", metric.Value)
	}
}

func TestParseCPUFromProcStat_EdgeCases(t *testing.T) {
	// Too few fields
	got := parseCPUFromProcStat("cpu")
	if got != 0 {
		t.Errorf("expected 0 for too few fields, got %.2f", got)
	}

	// Empty
	got = parseCPUFromProcStat("")
	if got != 0 {
		t.Errorf("expected 0 for empty, got %.2f", got)
	}
}

func TestParseFreeMemory_ZeroTotal(t *testing.T) {
	perc, _, total := parseFreeMemory("Mem:   0  0  0  0  0  0")
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if perc != 0 {
		t.Errorf("expected percent 0, got %.2f", perc)
	}
}

func TestParseFreeMemory_TooFewFields(t *testing.T) {
	perc, used, total := parseFreeMemory("Mem:")
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if perc != 0 {
		t.Errorf("expected percent 0, got %.2f", perc)
	}
	if used != 0 {
		t.Errorf("expected used 0, got %d", used)
	}
}

func TestParseDfOutput_TooFewFields(t *testing.T) {
	perc, used, avail := parseDfOutput("short")
	if perc != 0 {
		t.Errorf("expected percent 0, got %.2f", perc)
	}
	if used != 0 {
		t.Errorf("expected used 0, got %.2f", used)
	}
	if avail != 0 {
		t.Errorf("expected avail 0, got %.2f", avail)
	}
}
