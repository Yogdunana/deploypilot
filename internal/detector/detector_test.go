package detector

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// mockExecutor is a deterministic, in-memory CommandExecutor used to drive
// the detector without spawning real processes.
//
// Lookup order is deterministic:
//  1. If cmd == exactKey → return the registered response.
//  2. Otherwise, scan the registered substring keys in *insertion order*
//     (using `ordered` slice) and return the first match.
//  3. If nothing matches, return "" with no error.
type mockExecutor struct {
	mu       sync.Mutex
	exacts   map[string]string
	errs     map[string]error
	ordered  []string
	contains map[string]string
	calls    []string
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		exacts:   map[string]string{},
		errs:     map[string]error{},
		contains: map[string]string{},
	}
}

// setExact registers a response for an exact command string.
func (m *mockExecutor) setExact(cmd, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exacts[cmd] = value
}

// setContains registers a response keyed by a substring. Earlier
// registrations win (LIFO off – first-set wins).
func (m *mockExecutor) setContains(substr, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.contains[substr]; !exists {
		m.ordered = append(m.ordered, substr)
	}
	m.contains[substr] = value
}

func (m *mockExecutor) setErr(substr string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errs[substr] = err
}

func (m *mockExecutor) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockExecutor) getCall(i int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if i < len(m.calls) {
		return m.calls[i]
	}
	return ""
}

func (m *mockExecutor) RunCommand(_ context.Context, cmd string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, cmd)
	if err, ok := m.errs[cmd]; ok {
		return "", err
	}
	if v, ok := m.exacts[cmd]; ok {
		return v, nil
	}
	for _, k := range m.ordered {
		if strings.Contains(cmd, k) {
			return m.contains[k], nil
		}
	}
	return "", nil
}

// Compile-time check that mockExecutor satisfies deployer.CommandExecutor.
var _ deployer.CommandExecutor = (*mockExecutor)(nil)

// ---------- extractVersion & extractVersionFromNginxOutput ----------

// TestExtractVersion_SemverFull covers the most common case: a SemVer
// three-part version is picked up before any other pattern.
func TestExtractVersion_SemverFull(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"mysqld  Ver 8.0.32 for Linux on x86_64", "8.0.32"},
		{"postgres (PostgreSQL) 15.4 (Ubuntu 15.4-1.pgdg22.04+1)", "15.4"},
		{"redis-server 7.2.4", "7.2.4"},
		{"Apache/2.4.57 (Ubuntu)", "2.4.57"},
		// The SemVer pattern is greedy – it picks the first three-part
		// version in the input.
		{"v1.20 nginx/1.24.0", "1.24.0"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := extractVersion(tc.in); got != tc.want {
				t.Errorf("extractVersion(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestExtractVersion_PrefixedV exercises the v-prefix fallback when no
// full SemVer is present.
func TestExtractVersion_PrefixedV(t *testing.T) {
	if got := extractVersion("tool v3.10 extra"); got != "3.10" {
		t.Errorf("extractVersion with v-prefix = %q, want 3.10", got)
	}
}

// TestExtractVersion_ShortFallback exercises the X.Y fallback used when
// only major.minor is available.
func TestExtractVersion_ShortFallback(t *testing.T) {
	if got := extractVersion("running version 1.2 of something"); got != "1.2" {
		t.Errorf("extractVersion short = %q, want 1.2", got)
	}
}

// TestExtractVersion_NoMatch returns empty when no pattern matches.
func TestExtractVersion_NoMatch(t *testing.T) {
	if got := extractVersion("nothing here at all"); got != "" {
		t.Errorf("extractVersion with no match = %q, want \"\"", got)
	}
}

// TestExtractVersionFromNginxOutput covers both nginx and openresty forms.
func TestExtractVersionFromNginxOutput(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"nginx version: nginx/1.24.0", "1.24.0"},
		{"nginx version: openresty/1.25.3.1", "1.25.3.1"},
		{"garbled output", ""}, // falls through, no version
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := extractVersionFromNginxOutput(tc.in); got != tc.want {
				t.Errorf("extractVersionFromNginxOutput(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// ---------- detector end-to-end: MySQL / MariaDB / Postgres / Redis / MongoDB / Nginx / Apache / OpenResty ----------

func TestDetector_DetectAll_NoComponents(t *testing.T) {
	mock := newMockExecutor()
	d := New(mock)
	comps, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	if len(comps) != 0 {
		t.Errorf("expected no components, got %d: %+v", len(comps), comps)
	}
	// 4 database detectors + 3 web-server detectors = 7 ps probes
	// (each detector runs a "ps ... | grep ... | head -1" probe).
	if got, want := mock.callCount(), 7; got < want {
		t.Errorf("expected at least %d probe calls, got %d", want, got)
	}
}

func TestDetector_DetectMySQL_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.setContains("mysqld|mariadbd", "mysql 8.0.32")
	mock.setContains("ss -tlnp", "LISTEN 0 80 0.0.0.0:3306")
	mock.setContains("which mysqld", "/usr/sbin/mysqld")
	d := New(mock)
	comps, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	c := comps[0]
	if c.Type != ComponentMySQL {
		t.Errorf("Type = %q, want %q", c.Type, ComponentMySQL)
	}
	if c.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", c.Status, StatusRunning)
	}
	if c.Version != "8.0.32" {
		t.Errorf("Version = %q, want 8.0.32", c.Version)
	}
	if c.Port != 3306 {
		t.Errorf("Port = %d, want 3306", c.Port)
	}
	if c.BinaryPath != "/usr/sbin/mysqld" {
		t.Errorf("BinaryPath = %q, want /usr/sbin/mysqld", c.BinaryPath)
	}
}

func TestDetector_DetectMySQL_Stopped(t *testing.T) {
	// ps is empty (no process), but `which` still finds the binary.
	mock := newMockExecutor()
	// Default empty output for "ps ... | grep mysqld" – mock returns "".
	mock.setContains("which mysqld", "/usr/sbin/mysqld")
	d := New(mock)
	comps, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases: %v", err)
	}
	if len(comps) != 1 || comps[0].Status != StatusStopped {
		t.Fatalf("expected one stopped component, got %+v", comps)
	}
	if comps[0].BinaryPath != "/usr/sbin/mysqld" {
		t.Errorf("BinaryPath = %q, want /usr/sbin/mysqld", comps[0].BinaryPath)
	}
}

func TestDetector_DetectMySQL_MariaDB(t *testing.T) {
	// MariaDB is detected when the process line contains "mariadbd".
	mock := newMockExecutor()
	mock.setContains("mysqld|mariadbd", "/usr/sbin/mariadbd --version 10.11.4-MariaDB")
	mock.setContains("ss -tlnp", "LISTEN 0 80 0.0.0.0:3306")
	mock.setContains("which mysqld", "/usr/sbin/mariadbd")
	d := New(mock)
	comps, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	if comps[0].Name != "MariaDB" {
		t.Errorf("Name = %q, want MariaDB", comps[0].Name)
	}
	if comps[0].Version != "10.11.4" {
		// 10.11.4-MariaDB → full semver regex captures 10.11.4
		t.Errorf("Version = %q, want 10.11.4", comps[0].Version)
	}
}

func TestDetector_DetectMySQL_NotFound(t *testing.T) {
	// Both ps and which return empty → status NotFound, component omitted.
	mock := newMockExecutor()
	d := New(mock)
	comps, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases: %v", err)
	}
	// NotFound components are filtered out of the result list.
	if len(comps) != 0 {
		t.Errorf("expected no components, got %d: %+v", len(comps), comps)
	}
}

func TestDetector_DetectMySQL_PsErrorThenStopped(t *testing.T) {
	// ps returns an error, but `which` finds the binary → StatusStopped.
	mock := newMockExecutor()
	mock.setErr("mysqld|mariadbd", errors.New("ps failed"))
	mock.setContains("which mysqld", "/usr/sbin/mysqld")
	d := New(mock)
	comps, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases: %v", err)
	}
	if len(comps) != 1 || comps[0].Status != StatusStopped {
		t.Fatalf("expected one stopped component, got %+v", comps)
	}
}

func TestDetector_DetectPostgres_Running(t *testing.T) {
	mock := newMockExecutor()
	// The ps probe (`ps aux | grep postgres | ...`) must match a more
	// specific substring than `data_directory`, otherwise the
	// `psql -U postgres ...` data_directory probe would return the ps
	// output by accident.
	mock.setContains("ps aux | grep postgres", "postgres 15.4")
	mock.setContains("ss -tlnp", "LISTEN 0 128 0.0.0.0:5432")
	mock.setContains("which postgres", "/usr/lib/postgresql/15/bin/postgres")
	mock.setContains("data_directory", "/var/lib/postgresql/15/main")
	d := New(mock)
	comps, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	c := comps[0]
	if c.Type != ComponentPostgres {
		t.Errorf("Type = %q, want %q", c.Type, ComponentPostgres)
	}
	if c.Port != 5432 {
		t.Errorf("Port = %d, want 5432", c.Port)
	}
	if c.InstallPath != "/var/lib/postgresql/15/main" {
		t.Errorf("InstallPath = %q, want data dir", c.InstallPath)
	}
}

func TestDetector_DetectRedis_Responding(t *testing.T) {
	mock := newMockExecutor()
	mock.setContains("ps aux | grep redis", "redis-server 7.2.4")
	mock.setContains("ss -tlnp", "LISTEN 0 128 0.0.0.0:6379")
	mock.setContains("which redis-server", "/usr/bin/redis-server")
	mock.setContains("redis-cli ping", "PONG")
	d := New(mock)
	comps, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	c := comps[0]
	if c.Type != ComponentRedis {
		t.Errorf("Type = %q, want redis", c.Type)
	}
	if c.Port != 6379 {
		t.Errorf("Port = %d, want 6379", c.Port)
	}
	if c.Details != "responding to PING" {
		t.Errorf("Details = %q, want responding to PING", c.Details)
	}
	if c.Version != "7.2.4" {
		t.Errorf("Version = %q, want 7.2.4", c.Version)
	}
}

func TestDetector_DetectRedis_DefaultPort(t *testing.T) {
	// Port probe returns nothing → default port 6379 must be applied.
	mock := newMockExecutor()
	mock.setContains("ps aux | grep redis", "redis-server 6.2.0")
	// No ss response → empty port output → fallback to 6379.
	d := New(mock)
	comps, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases: %v", err)
	}
	if len(comps) != 1 || comps[0].Port != 6379 {
		t.Errorf("expected default port 6379, got %+v", comps)
	}
}

func TestDetector_DetectMongoDB_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.setContains("ps aux | grep mongod", "mongod 7.0.0")
	mock.setContains("ss -tlnp", "LISTEN 0 128 0.0.0.0:27017")
	mock.setContains("which mongod", "/usr/bin/mongod")
	d := New(mock)
	comps, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	c := comps[0]
	if c.Type != ComponentMongoDB {
		t.Errorf("Type = %q, want mongodb", c.Type)
	}
	if c.Port != 27017 {
		t.Errorf("Port = %d, want 27017", c.Port)
	}
}

func TestDetector_DetectNginx_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.setContains("ps aux | grep nginx", "nginx: master process /usr/sbin/nginx")
	mock.setContains("nginx -v", "nginx version: nginx/1.24.0")
	mock.setContains("ss -tlnp", "LISTEN 0 128 0.0.0.0:80")
	mock.setContains("which nginx", "/usr/sbin/nginx")
	// The detector shells out to `nginx -t 2>&1 | grep 'configuration
	// file' | sed ...`. The mock's `setContains("nginx -t", ...)` key would
	// also match the longer composite command, so we register a more
	// specific key for the full conf-path probe and a separate one for the
	// simple "nginx -v" probe.
	mock.setContains("configuration file", "/etc/nginx/nginx.conf")
	d := New(mock)
	comps, err := d.DetectWebServers(context.Background())
	if err != nil {
		t.Fatalf("DetectWebServers: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	c := comps[0]
	if c.Type != ComponentNginx {
		t.Errorf("Type = %q, want nginx", c.Type)
	}
	if c.Version != "1.24.0" {
		t.Errorf("Version = %q, want 1.24.0", c.Version)
	}
	if c.Port != 80 {
		t.Errorf("Port = %d, want 80", c.Port)
	}
	if c.InstallPath != "/etc/nginx/nginx.conf" {
		t.Errorf("InstallPath = %q, want config path", c.InstallPath)
	}
}

func TestDetector_DetectApache_Running(t *testing.T) {
	mock := newMockExecutor()
	// Match only the Apache httpd|apache2 ps probe (no other probe
	// contains "httpd|apache2").
	mock.setContains("ps aux | grep -E 'httpd|apache2'", "/usr/sbin/apache2 -k start")
	mock.setContains("httpd -v", "Server version: Apache/2.4.57 (Ubuntu)")
	// Apache's port probe is `ss -tlnp ... | grep -E 'httpd|apache2' | grep -oP ...`.
	// The mock must return a port (not a full `ss` line) – we register the
	// answer for the specific Apache port probe.
	mock.setContains("grep -E 'httpd|apache2'", "8080")
	// Apache's binary probe (`which httpd || which apache2`) and a
	// fall-through path output the binary path.
	mock.setContains("which httpd", "/usr/sbin/apache2")
	d := New(mock)
	comps, err := d.DetectWebServers(context.Background())
	if err != nil {
		t.Fatalf("DetectWebServers: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d: %+v", len(comps), comps)
	}
	c := comps[0]
	if c.Type != ComponentApache {
		t.Errorf("Type = %q, want apache", c.Type)
	}
	if c.Port != 8080 {
		t.Errorf("Port = %d, want 8080", c.Port)
	}
	if c.Version != "2.4.57" {
		t.Errorf("Version = %q, want 2.4.57", c.Version)
	}
}

func TestDetector_DetectOpenResty_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.setContains("ps aux | grep openresty", "nginx: master process /usr/local/openresty/nginx/sbin/nginx")
	mock.setContains("nginx -v", "nginx version: openresty/1.25.3.1")
	mock.setContains("ss -tlnp", "LISTEN 0 128 0.0.0.0:80")
	mock.setContains("which openresty", "/usr/local/openresty/nginx/sbin/nginx")
	d := New(mock)
	comps, err := d.DetectWebServers(context.Background())
	if err != nil {
		t.Fatalf("DetectWebServers: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	c := comps[0]
	if c.Type != ComponentOpenResty {
		t.Errorf("Type = %q, want openresty", c.Type)
	}
	if c.Version != "1.25.3.1" {
		t.Errorf("Version = %q, want 1.25.3.1", c.Version)
	}
}

func TestDetector_DetectOpenResty_StoppedButBinaryReportsOpenResty(t *testing.T) {
	// No openresty in ps, but `which openresty` returns the OpenResty
	// nginx binary and `nginx -v` reports openresty → still detected as
	// stopped.
	mock := newMockExecutor()
	// Default: ps probe returns empty (no key registered).
	mock.setContains("which openresty", "/usr/local/openresty/nginx/sbin/nginx")
	mock.setContains("nginx -v", "nginx version: openresty/1.21.4.1")
	d := New(mock)
	comps, err := d.DetectWebServers(context.Background())
	if err != nil {
		t.Fatalf("DetectWebServers: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	if comps[0].Status != StatusStopped {
		t.Errorf("Status = %q, want stopped", comps[0].Status)
	}
	if comps[0].Version != "1.21.4.1" {
		t.Errorf("Version = %q, want 1.21.4.1", comps[0].Version)
	}
	if comps[0].BinaryPath != "/usr/local/openresty/nginx/sbin/nginx" {
		t.Errorf("BinaryPath = %q", comps[0].BinaryPath)
	}
}

func TestDetector_DetectOpenResty_StoppedButPlainNginxBinary(t *testing.T) {
	// `which openresty || which nginx` returns a path, but `nginx -v`
	// does NOT contain "openresty" → StatusNotFound (the nginx binary
	// is not OpenResty).
	mock := newMockExecutor()
	mock.setContains("which openresty", "/usr/sbin/nginx")
	mock.setContains("nginx -v", "nginx version: nginx/1.24.0") // plain nginx
	d := New(mock)
	comps, err := d.DetectWebServers(context.Background())
	if err != nil {
		t.Fatalf("DetectWebServers: %v", err)
	}
	if len(comps) != 0 {
		t.Errorf("expected no components (plain nginx is not openresty), got %+v", comps)
	}
}

func TestDetector_DetectOpenResty_NotFound(t *testing.T) {
	// No openresty, no nginx binary, no version probe match → NotFound.
	mock := newMockExecutor()
	d := New(mock)
	comps, err := d.DetectWebServers(context.Background())
	if err != nil {
		t.Fatalf("DetectWebServers: %v", err)
	}
	if len(comps) != 0 {
		t.Errorf("expected no components, got %d: %+v", len(comps), comps)
	}
}

func TestDetector_DetectAll_Mixed(t *testing.T) {
	// Multiple components found at once.
	mock := newMockExecutor()
	mock.setContains("ps aux | grep redis", "redis-server 7.2.4")
	mock.setContains("ps aux | grep nginx", "nginx: worker process")
	mock.setContains("nginx -v", "nginx version: nginx/1.24.0")
	// All other components absent.
	d := New(mock)
	comps, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("expected 2 components, got %d: %+v", len(comps), comps)
	}
	gotTypes := map[ComponentType]bool{}
	for _, c := range comps {
		gotTypes[c.Type] = true
	}
	if !gotTypes[ComponentRedis] || !gotTypes[ComponentNginx] {
		t.Errorf("expected redis+nginx, got %v", gotTypes)
	}
}

func TestDetector_ContextDeadline(t *testing.T) {
	// An already-cancelled context should not crash the detector.
	mock := newMockExecutor()
	d := New(mock)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := d.DetectAll(ctx)
	if err != nil {
		// The current implementation does not propagate the cancellation
		// through to the mock executor; it should still return cleanly.
		t.Logf("DetectAll returned (acceptable): %v", err)
	}
}

func TestDetector_New_UsesDefaultTimeout(t *testing.T) {
	d := New(newMockExecutor())
	if d.timeout != 15*time.Second {
		t.Errorf("default timeout = %v, want 15s", d.timeout)
	}
}

// ---------- summary rendering ----------

func TestGetDetectionSummary_Empty(t *testing.T) {
	got := GetDetectionSummary(nil)
	if got != "No system components detected" {
		t.Errorf("summary for nil = %q", got)
	}
}

func TestGetDetectionSummary_Multiple(t *testing.T) {
	comps := []DetectedComponent{
		{Type: ComponentRedis, Name: "Redis", Status: StatusRunning, Version: "7.2.4", Port: 6379},
		{Type: ComponentNginx, Name: "Nginx", Status: StatusStopped, Port: 80},
		{Type: ComponentMySQL, Name: "MySQL", Status: StatusInstalled},
	}
	got := GetDetectionSummary(comps)
	for _, want := range []string{"Detected 3 component", "Redis", "Nginx", "MySQL", "v7.2.4", ":6379", ":80"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary missing %q in:\n%s", want, got)
		}
	}
}
