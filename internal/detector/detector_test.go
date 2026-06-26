package detector

import (
	"context"
	"errors"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

type mockExecutor struct {
	commands map[string]string
	errors   map[string]error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		commands: make(map[string]string),
		errors:   make(map[string]error),
	}
}

func (m *mockExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	if err, ok := m.errors[cmd]; ok {
		return "", err
	}
	if out, ok := m.commands[cmd]; ok {
		return out, nil
	}
	return "", errors.New("mock: unknown command")
}

func TestNewDetector(t *testing.T) {
	mock := newMockExecutor()
	d := New(mock)
	if d == nil {
		t.Fatal("New() returned nil")
	}
	if d.executor != mock {
		t.Error("executor not set correctly")
	}
}

func TestDetectAll(t *testing.T) {
	mock := newMockExecutor()
	d := New(mock)

	components, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}
	if len(components) != 0 {
		t.Errorf("expected 0 components with empty mock, got %d", len(components))
	}
}

func TestDetectDatabases(t *testing.T) {
	mock := newMockExecutor()
	d := New(mock)

	components, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases() error = %v", err)
	}
	if len(components) != 0 {
		t.Errorf("expected 0 components with empty mock, got %d", len(components))
	}
}

func TestDetectWebServers(t *testing.T) {
	mock := newMockExecutor()
	d := New(mock)

	components, err := d.DetectWebServers(context.Background())
	if err != nil {
		t.Fatalf("DetectWebServers() error = %v", err)
	}
	if len(components) != 0 {
		t.Errorf("expected 0 components with empty mock, got %d", len(components))
	}
}

func TestDetectMySQL_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1"] = "mysql 1234 0.0 0.0 1234 5678 ? S 10:00 0:00 mysqld --version=8.0.33"
	mock.commands["ss -tlnp 2>/dev/null | grep -E 'mysqld|mariadbd' | grep -oP ':\\K[0-9]+' | head -1"] = "3306"
	mock.commands["which mysqld 2>/dev/null || which mariadbd 2>/dev/null"] = "/usr/bin/mysqld"

	d := New(mock)
	comp, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("detectMySQL() error = %v", err)
	}
	if comp.Type != ComponentMySQL {
		t.Errorf("Type = %q, want %q", comp.Type, ComponentMySQL)
	}
	if comp.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", comp.Status, StatusRunning)
	}
	if comp.Version != "8.0.33" {
		t.Errorf("Version = %q, want %q", comp.Version, "8.0.33")
	}
	if comp.Port != 3306 {
		t.Errorf("Port = %d, want 3306", comp.Port)
	}
}

func TestDetectMySQL_Stopped(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1"] = ""
	mock.commands["which mysqld 2>/dev/null || which mariadbd 2>/dev/null"] = "/usr/bin/mysqld"

	d := New(mock)
	comp, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("detectMySQL() error = %v", err)
	}
	if comp.Status != StatusStopped {
		t.Errorf("Status = %q, want %q", comp.Status, StatusStopped)
	}
	if comp.BinaryPath != "/usr/bin/mysqld" {
		t.Errorf("BinaryPath = %q, want %q", comp.BinaryPath, "/usr/bin/mysqld")
	}
}

func TestDetectMySQL_NotFound(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1"] = ""
	mock.commands["which mysqld 2>/dev/null || which mariadbd 2>/dev/null"] = ""

	d := New(mock)
	comp, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("detectMySQL() error = %v", err)
	}
	if comp.Status != StatusNotFound {
		t.Errorf("Status = %q, want %q", comp.Status, StatusNotFound)
	}
}

func TestDetectMySQL_MariaDB(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1"] = "mysql 1234 0.0 0.0 1234 5678 ? S 10:00 0:00 mariadbd --version=10.6.12"
	mock.commands["which mysqld 2>/dev/null || which mariadbd 2>/dev/null"] = "/usr/bin/mariadbd"

	d := New(mock)
	comp, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("detectMySQL() error = %v", err)
	}
	if comp.Name != "MariaDB" {
		t.Errorf("Name = %q, want %q", comp.Name, "MariaDB")
	}
}

func TestDetectPostgreSQL_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep postgres | grep -v grep | head -1"] = "postgres 5678 0.0 0.0 1234 5678 ? S 10:00 0:00 postgres --version=15.2"
	mock.commands["ss -tlnp 2>/dev/null | grep postgres | grep -oP ':\\K[0-9]+' | head -1"] = "5432"
	mock.commands["which postgres 2>/dev/null"] = "/usr/bin/postgres"
	mock.commands["psql -U postgres -c \"SHOW data_directory;\" 2>/dev/null | sed -n '3p' | tr -d ' '"] = "/var/lib/postgresql/15/main"

	d := New(mock)
	comp, err := d.detectPostgreSQL(context.Background())
	if err != nil {
		t.Fatalf("detectPostgreSQL() error = %v", err)
	}
	if comp.Type != ComponentPostgres {
		t.Errorf("Type = %q, want %q", comp.Type, ComponentPostgres)
	}
	if comp.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", comp.Status, StatusRunning)
	}
	if comp.InstallPath != "/var/lib/postgresql/15/main" {
		t.Errorf("InstallPath = %q, want %q", comp.InstallPath, "/var/lib/postgresql/15/main")
	}
}

func TestDetectRedis_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep redis | grep -v grep | head -1"] = "redis 7890 0.0 0.0 1234 5678 ? S 10:00 0:00 redis-server --version=7.0.12"
	mock.commands["ss -tlnp 2>/dev/null | grep redis | grep -oP ':\\K[0-9]+' | head -1"] = "6379"
	mock.commands["which redis-server 2>/dev/null"] = "/usr/bin/redis-server"
	mock.commands["redis-cli ping 2>/dev/null"] = "PONG"

	d := New(mock)
	comp, err := d.detectRedis(context.Background())
	if err != nil {
		t.Fatalf("detectRedis() error = %v", err)
	}
	if comp.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", comp.Status, StatusRunning)
	}
	if comp.Details != "responding to PING" {
		t.Errorf("Details = %q, want %q", comp.Details, "responding to PING")
	}
}

func TestDetectMongoDB_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep mongod | grep -v grep | head -1"] = "mongod 2345 0.0 0.0 1234 5678 ? S 10:00 0:00 mongod --version=6.0.5"
	mock.commands["ss -tlnp 2>/dev/null | grep mongod | grep -oP ':\\K[0-9]+' | head -1"] = "27017"
	mock.commands["which mongod 2>/dev/null"] = "/usr/bin/mongod"

	d := New(mock)
	comp, err := d.detectMongoDB(context.Background())
	if err != nil {
		t.Fatalf("detectMongoDB() error = %v", err)
	}
	if comp.Type != ComponentMongoDB {
		t.Errorf("Type = %q, want %q", comp.Type, ComponentMongoDB)
	}
	if comp.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", comp.Status, StatusRunning)
	}
}

func TestDetectNginx_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep nginx | grep -v grep | head -1"] = "root 3456 0.0 0.0 1234 5678 ? S 10:00 0:00 nginx"
	mock.commands["nginx -v 2>&1"] = "nginx version: nginx/1.24.0"
	mock.commands["ss -tlnp 2>/dev/null | grep nginx | grep -oP ':\\K[0-9]+' | head -1"] = "80"
	mock.commands["which nginx 2>/dev/null"] = "/usr/sbin/nginx"
	mock.commands["nginx -t 2>&1 | grep 'configuration file' | sed 's/.*configuration file //' | sed 's/ syntax.*//'"] = "/etc/nginx/nginx.conf"

	d := New(mock)
	comp, err := d.detectNginx(context.Background())
	if err != nil {
		t.Fatalf("detectNginx() error = %v", err)
	}
	if comp.Type != ComponentNginx {
		t.Errorf("Type = %q, want %q", comp.Type, ComponentNginx)
	}
	if comp.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", comp.Status, StatusRunning)
	}
	if comp.Version != "1.24.0" {
		t.Errorf("Version = %q, want %q", comp.Version, "1.24.0")
	}
	if comp.InstallPath != "/etc/nginx/nginx.conf" {
		t.Errorf("InstallPath = %q, want %q", comp.InstallPath, "/etc/nginx/nginx.conf")
	}
}

func TestDetectApache_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep -E 'httpd|apache2' | grep -v grep | head -1"] = "root 4567 0.0 0.0 1234 5678 ? S 10:00 0:00 apache2 -v=2.4.57"
	mock.commands["httpd -v 2>/dev/null || apache2 -v 2>/dev/null"] = "Server version: Apache/2.4.57"
	mock.commands["ss -tlnp 2>/dev/null | grep -E 'httpd|apache2' | grep -oP ':\\K[0-9]+' | head -1"] = "80"
	mock.commands["which httpd 2>/dev/null || which apache2 2>/dev/null"] = "/usr/sbin/apache2"

	d := New(mock)
	comp, err := d.detectApache(context.Background())
	if err != nil {
		t.Fatalf("detectApache() error = %v", err)
	}
	if comp.Type != ComponentApache {
		t.Errorf("Type = %q, want %q", comp.Type, ComponentApache)
	}
	if comp.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", comp.Status, StatusRunning)
	}
}

func TestDetectOpenResty_Running(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep openresty | grep -v grep | head -1"] = "root 5678 0.0 0.0 1234 5678 ? S 10:00 0:00 openresty"
	mock.commands["nginx -v 2>&1"] = "nginx version: openresty/1.25.3.1"
	mock.commands["ss -tlnp 2>/dev/null | grep openresty | grep -oP ':\\K[0-9]+' | head -1"] = "80"
	mock.commands["which openresty 2>/dev/null || which nginx 2>/dev/null"] = "/usr/local/openresty/bin/openresty"

	d := New(mock)
	comp, err := d.detectOpenResty(context.Background())
	if err != nil {
		t.Fatalf("detectOpenResty() error = %v", err)
	}
	if comp.Type != ComponentOpenResty {
		t.Errorf("Type = %q, want %q", comp.Type, ComponentOpenResty)
	}
	if comp.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", comp.Status, StatusRunning)
	}
	if comp.Version != "1.25.3.1" {
		t.Errorf("Version = %q, want %q", comp.Version, "1.25.3.1")
	}
}

func TestDetectOpenResty_Stopped(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep openresty | grep -v grep | head -1"] = ""
	mock.commands["which openresty 2>/dev/null || which nginx 2>/dev/null"] = "/usr/local/openresty/bin/openresty"
	mock.commands["nginx -v 2>&1"] = "nginx version: openresty/1.25.3.1"

	d := New(mock)
	comp, err := d.detectOpenResty(context.Background())
	if err != nil {
		t.Fatalf("detectOpenResty() error = %v", err)
	}
	if comp.Status != StatusStopped {
		t.Errorf("Status = %q, want %q", comp.Status, StatusStopped)
	}
}

func TestDetectOpenResty_NotFound(t *testing.T) {
	mock := newMockExecutor()
	mock.commands["ps aux | grep openresty | grep -v grep | head -1"] = ""
	mock.commands["which openresty 2>/dev/null || which nginx 2>/dev/null"] = "/usr/sbin/nginx"
	mock.commands["nginx -v 2>&1"] = "nginx version: nginx/1.24.0"

	d := New(mock)
	comp, err := d.detectOpenResty(context.Background())
	if err != nil {
		t.Fatalf("detectOpenResty() error = %v", err)
	}
	if comp.Status != StatusNotFound {
		t.Errorf("Status = %q, want %q", comp.Status, StatusNotFound)
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"mysqld --version=8.0.33", "8.0.33"},
		{"v1.2.3", "1.2.3"},
		{"v1.2", "1.2"},
		{"nginx/1.24.0", "1.24.0"},
		{"Server version: Apache/2.4.57", "2.4.57"},
		{"redis-server 7.0.12", "7.0.12"},
		{"no version here", ""},
	}

	for _, tt := range tests {
		result := extractVersion(tt.input)
		if result != tt.expected {
			t.Errorf("extractVersion(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractVersionFromNginxOutput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"nginx version: nginx/1.24.0", "1.24.0"},
		{"nginx version: openresty/1.25.3.1", "1.25.3.1"},
		{"some other output", ""},
	}

	for _, tt := range tests {
		result := extractVersionFromNginxOutput(tt.input)
		if result != tt.expected {
			t.Errorf("extractVersionFromNginxOutput(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetDetectionSummary(t *testing.T) {
	tests := []struct {
		components []DetectedComponent
		contains   string
	}{
		{[]DetectedComponent{}, "No system components detected"},
		{[]DetectedComponent{{Type: ComponentMySQL, Name: "MySQL", Status: StatusRunning, Version: "8.0.33", Port: 3306}}, "MySQL"},
		{[]DetectedComponent{{Type: ComponentNginx, Name: "Nginx", Status: StatusStopped}}, "Nginx"},
	}

	for _, tt := range tests {
		result := GetDetectionSummary(tt.components)
		if !contains(result, tt.contains) {
			t.Errorf("GetDetectionSummary() = %q, should contain %q", result, tt.contains)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

var _ deployer.CommandExecutor = (*mockExecutor)(nil)