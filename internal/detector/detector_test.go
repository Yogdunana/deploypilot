package detector

import (
	"context"
	"strings"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// mockExecutor is a programmable CommandExecutor used by the tests below.
// It returns canned output for the requested shell command and records
// every command it received so tests can assert on call patterns.
type mockExecutor struct {
	// responses maps a command substring -> (output, error).
	// The first matching entry wins.
	responses map[string]struct {
		out string
		err error
	}
	// defaultOut is returned when no key matches.
	defaultOut string
	// calls is appended for every RunCommand invocation.
	calls []string
}

func (m *mockExecutor) RunCommand(_ context.Context, cmd string) (string, error) {
	m.calls = append(m.calls, cmd)
	for needle, r := range m.responses {
		if strings.Contains(cmd, needle) {
			return r.out, r.err
		}
	}
	return m.defaultOut, nil
}

func newMock() *mockExecutor {
	return &mockExecutor{
		responses: map[string]struct {
			out string
			err error
		}{},
	}
}

func TestExtractVersion_FullSemver(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"mysqld-style", "mysql 8.0.34 (Linux)", "8.0.34"},
		{"with-prefix-v", "Server version: v1.21.0", "1.21.0"},
		{"short", "Server version 1.21", "1.21"},
		{"empty", "", ""},
		{"no-version", "running process", ""},
		{"postfix-letter", "redis 7.0.5rc1 (server)", "7.0.5rc1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractVersion(tc.in); got != tc.want {
				t.Errorf("extractVersion(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestExtractVersionFromNginxOutput(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"nginx", "nginx version: nginx/1.24.0", "1.24.0"},
		{"openresty", "nginx version: openresty/1.25.3.1", "1.25.3.1"},
		{"plain", "some other text", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractVersionFromNginxOutput(tc.in); got != tc.want {
				t.Errorf("extractVersionFromNginxOutput(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestGetDetectionSummary_Empty(t *testing.T) {
	got := GetDetectionSummary(nil)
	if !strings.Contains(got, "No system components") {
		t.Errorf("expected 'No system components' message, got %q", got)
	}
}

func TestGetDetectionSummary_Multiple(t *testing.T) {
	components := []DetectedComponent{
		{Type: ComponentMySQL, Name: "MySQL", Status: StatusRunning, Version: "8.0.34", Port: 3306},
		{Type: ComponentNginx, Name: "Nginx", Status: StatusStopped, Port: 80},
		{Type: ComponentRedis, Name: "Redis", Status: StatusInstalled, Port: 6379},
		{Type: ComponentMongoDB, Name: "MongoDB", Status: StatusNotFound},
	}
	got := GetDetectionSummary(components)
	if !strings.Contains(got, "Detected 4 component") {
		t.Errorf("expected 'Detected 4 component' in summary, got %q", got)
	}
	if !strings.Contains(got, "MySQL") || !strings.Contains(got, "v8.0.34") {
		t.Errorf("expected MySQL version in summary, got %q", got)
	}
	if !strings.Contains(got, "running") {
		t.Errorf("expected status text in summary, got %q", got)
	}
	// All components are listed, but the not_found entry uses a different icon.
	if !strings.Contains(got, "MongoDB") {
		t.Errorf("expected MongoDB entry in summary, got %q", got)
	}
	if !strings.Contains(got, "not_found") {
		t.Errorf("expected 'not_found' status text, got %q", got)
	}
}

func TestDetectMySQL_NotFound(t *testing.T) {
	m := newMock()
	d := New(m)
	ctx := context.Background()

	got, err := d.detectMySQL(ctx)
	if err != nil {
		t.Fatalf("detectMySQL returned error: %v", err)
	}
	if got.Status != StatusNotFound {
		t.Errorf("expected StatusNotFound, got %v", got.Status)
	}
	if got.Type != ComponentMySQL {
		t.Errorf("expected ComponentMySQL, got %v", got.Type)
	}
}

func TestDetectMySQL_StoppedBinaryExists(t *testing.T) {
	m := newMock()
	m.responses["ps aux"] = struct {
		out string
		err error
	}{out: "", err: nil}
	m.responses["which mysqld"] = struct {
		out string
		err error
	}{out: "/usr/sbin/mysqld", err: nil}
	d := New(m)
	ctx := context.Background()

	got, err := d.detectMySQL(ctx)
	if err != nil {
		t.Fatalf("detectMySQL returned error: %v", err)
	}
	if got.Status != StatusStopped {
		t.Errorf("expected StatusStopped, got %v", got.Status)
	}
	if got.BinaryPath != "/usr/sbin/mysqld" {
		t.Errorf("BinaryPath = %q, want /usr/sbin/mysqld", got.BinaryPath)
	}
}

func TestDetectMySQL_Running(t *testing.T) {
	m := newMock()
	m.responses["ps aux"] = struct {
		out string
		err error
	}{out: "mysql 8.0.34 --user=mysql", err: nil}
	m.responses["ss -tlnp"] = struct {
		out string
		err error
	}{out: "3306", err: nil}
	m.responses["which mysqld"] = struct {
		out string
		err error
	}{out: "/usr/sbin/mysqld", err: nil}
	d := New(m)
	ctx := context.Background()

	got, err := d.detectMySQL(ctx)
	if err != nil {
		t.Fatalf("detectMySQL returned error: %v", err)
	}
	if got.Status != StatusRunning {
		t.Errorf("expected StatusRunning, got %v", got.Status)
	}
	if got.Port != 3306 {
		t.Errorf("Port = %d, want 3306", got.Port)
	}
	if got.Version != "8.0.34" {
		t.Errorf("Version = %q, want 8.0.34", got.Version)
	}
	if got.Name != "MySQL/MariaDB" {
		t.Errorf("Name = %q, want MySQL/MariaDB", got.Name)
	}
}

func TestDetectMySQL_RunningMariaDB(t *testing.T) {
	m := newMock()
	m.responses["ps aux"] = struct {
		out string
		err error
	}{out: "mariadbd 10.11.0 --user=mysql", err: nil}
	m.responses["ss -tlnp"] = struct {
		out string
		err error
	}{out: "", err: nil}
	m.responses["which mysqld"] = struct {
		out string
		err error
	}{out: "/usr/sbin/mariadbd", err: nil}
	d := New(m)
	ctx := context.Background()

	got, _ := d.detectMySQL(ctx)
	if got.Name != "MariaDB" {
		t.Errorf("expected Name=MariaDB, got %q", got.Name)
	}
	if got.Port != 3306 {
		t.Errorf("expected default port 3306, got %d", got.Port)
	}
}

func TestDetectRedis_RunningWithPing(t *testing.T) {
	m := newMock()
	m.responses["ps aux"] = struct {
		out string
		err error
	}{out: "redis-server 7.2.0 *:6379", err: nil}
	m.responses["ss -tlnp"] = struct {
		out string
		err error
	}{out: "6379", err: nil}
	m.responses["which redis-server"] = struct {
		out string
		err error
	}{out: "/usr/bin/redis-server", err: nil}
	m.responses["redis-cli ping"] = struct {
		out string
		err error
	}{out: "PONG", err: nil}
	d := New(m)

	got, err := d.detectRedis(context.Background())
	if err != nil {
		t.Fatalf("detectRedis returned error: %v", err)
	}
	if got.Status != StatusRunning {
		t.Errorf("Status = %v, want StatusRunning", got.Status)
	}
	if got.Version != "7.2.0" {
		t.Errorf("Version = %q, want 7.2.0", got.Version)
	}
	if got.Details != "responding to PING" {
		t.Errorf("Details = %q, want 'responding to PING'", got.Details)
	}
}

func TestDetectNginx_Running(t *testing.T) {
	m := newMock()
	m.responses["ps aux"] = struct {
		out string
		err error
	}{out: "nginx: master process", err: nil}
	m.responses["nginx -v"] = struct {
		out string
		err error
	}{out: "nginx version: nginx/1.24.0", err: nil}
	m.responses["ss -tlnp"] = struct {
		out string
		err error
	}{out: "80", err: nil}
	m.responses["which nginx"] = struct {
		out string
		err error
	}{out: "/usr/sbin/nginx", err: nil}
	m.responses["nginx -t"] = struct {
		out string
		err error
	}{out: "/etc/nginx/nginx.conf", err: nil}
	d := New(m)

	got, err := d.detectNginx(context.Background())
	if err != nil {
		t.Fatalf("detectNginx returned error: %v", err)
	}
	if got.Status != StatusRunning {
		t.Errorf("Status = %v, want StatusRunning", got.Status)
	}
	if got.Version != "1.24.0" {
		t.Errorf("Version = %q, want 1.24.0", got.Version)
	}
	if got.InstallPath != "/etc/nginx/nginx.conf" {
		t.Errorf("InstallPath = %q, want /etc/nginx/nginx.conf", got.InstallPath)
	}
}

func TestDetectOpenResty_Running(t *testing.T) {
	m := newMock()
	m.responses["ps aux"] = struct {
		out string
		err error
	}{out: "openresty: master", err: nil}
	m.responses["nginx -v"] = struct {
		out string
		err error
	}{out: "nginx version: openresty/1.25.3.1", err: nil}
	m.responses["ss -tlnp"] = struct {
		out string
		err error
	}{out: "80", err: nil}
	m.responses["which openresty"] = struct {
		out string
		err error
	}{out: "/usr/local/openresty/bin/openresty", err: nil}
	d := New(m)

	got, _ := d.detectOpenResty(context.Background())
	if got.Status != StatusRunning {
		t.Errorf("Status = %v, want StatusRunning", got.Status)
	}
	if got.Version != "1.25.3.1" {
		t.Errorf("Version = %q, want 1.25.3.1", got.Version)
	}
}

func TestDetectAll_FiltersNotFoundAndContinues(t *testing.T) {
	// Only PostgreSQL is "running" — other components report not_found.
	m := newMock()
	m.responses["ps aux"] = struct {
		out string
		err error
	}{out: "", err: nil}
	m.responses["which mysqld"] = struct {
		out string
		err error
	}{out: "", err: nil}
	m.responses["which redis-server"] = struct {
		out string
		err error
	}{out: "", err: nil}
	m.responses["which mongod"] = struct {
		out string
		err error
	}{out: "", err: nil}
	m.responses["which httpd"] = struct {
		out string
		err error
	}{out: "", err: nil}
	m.responses["which openresty"] = struct {
		out string
		err error
	}{out: "", err: nil}
	m.responses["which nginx"] = struct {
		out string
		err error
	}{out: "", err: nil}
	// Override the postgres branch.
	m.responses["ps aux | grep postgres"] = struct {
		out string
		err error
	}{out: "postgres (PostgreSQL) 15.4", err: nil}
	m.responses["ss -tlnp | grep postgres"] = struct {
		out string
		err error
	}{out: "5432", err: nil}
	m.responses["which postgres"] = struct {
		out string
		err error
	}{out: "/usr/lib/postgresql/15/bin/postgres", err: nil}
	// Differentiate ps aux lines per detector.
	originalRunCommand := m.RunCommand
	_ = originalRunCommand // keep compiler happy

	// Switch to a per-line dispatcher so each component gets a distinct ps output.
	m = newMock()
	m.responses["ps aux | grep postgres"] = struct {
		out string
		err error
	}{out: "postgres 15.4 --user=postgres", err: nil}
	m.responses["ss -tlnp | grep postgres"] = struct {
		out string
		err error
	}{out: "5432", err: nil}
	m.responses["which postgres"] = struct {
		out string
		err error
	}{out: "/usr/lib/postgresql/15/bin/postgres", err: nil}
	m.responses["psql -U postgres"] = struct {
		out string
		err error
	}{out: "/var/lib/postgresql/15/main", err: nil}

	d := New(m)
	got, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 component, got %d: %+v", len(got), got)
	}
	if got[0].Type != ComponentPostgres {
		t.Errorf("expected ComponentPostgres, got %v", got[0].Type)
	}
	if got[0].Status != StatusRunning {
		t.Errorf("expected StatusRunning, got %v", got[0].Status)
	}
}

func TestDetectDatabases_OnlyDatabases(t *testing.T) {
	m := newMock()
	// Everything absent.
	d := New(m)
	got, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 components, got %d: %+v", len(got), got)
	}
}

func TestDetectWebServers_OnlyWebServers(t *testing.T) {
	m := newMock()
	d := New(m)
	got, err := d.DetectWebServers(context.Background())
	if err != nil {
		t.Fatalf("DetectWebServers returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 components, got %d: %+v", len(got), got)
	}
}

// Ensure Detector satisfies the expected interface.
var _ deployer.CommandExecutor = (*mockExecutor)(nil)
