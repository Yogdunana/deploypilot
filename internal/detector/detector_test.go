package detector

import (
	"context"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

type mockExecutor struct {
	responses map[string]string
	errors    map[string]error
}

func (m *mockExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	if err, ok := m.errors[cmd]; ok {
		return "", err
	}
	if resp, ok := m.responses[cmd]; ok {
		return resp, nil
	}
	return "", nil
}

func newMockExecutor(responses map[string]string) *mockExecutor {
	return &mockExecutor{responses: responses, errors: make(map[string]error)}
}

func TestNewDetector(t *testing.T) {
	mock := newMockExecutor(map[string]string{})
	d := New(mock)
	if d == nil {
		t.Fatal("expected non-nil Detector")
	}
	if d.executor != mock {
		t.Error("expected executor to be set")
	}
}

func TestDetectAll_NoComponents(t *testing.T) {
	mock := newMockExecutor(map[string]string{})
	d := New(mock)

	components, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(components) != 0 {
		t.Errorf("expected 0 components, got %d", len(components))
	}
}

func TestDetectMySQL_Running(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1": "root 1234 0.0 0.1 123456 7890 ? Ssl 00:00 0:00 mysqld --version=8.0.36",
		"ss -tlnp 2>/dev/null | grep -E 'mysqld|mariadbd' | grep -oP ':\\K[0-9]+' | head -1": "3306",
		"which mysqld 2>/dev/null || which mariadbd 2>/dev/null": "/usr/bin/mysqld",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Type != ComponentMySQL {
		t.Errorf("expected type mysql, got %s", comp.Type)
	}
	if comp.Status != StatusRunning {
		t.Errorf("expected status running, got %s", comp.Status)
	}
	if comp.Port != 3306 {
		t.Errorf("expected port 3306, got %d", comp.Port)
	}
	if comp.Version != "8.0.36" {
		t.Errorf("expected version 8.0.36, got %s", comp.Version)
	}
}

func TestDetectMySQL_Stopped(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1": "",
		"which mysqld 2>/dev/null || which mariadbd 2>/dev/null":     "/usr/bin/mysqld",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Status != StatusStopped {
		t.Errorf("expected status stopped, got %s", comp.Status)
	}
}

func TestDetectMySQL_NotFound(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1": "",
		"which mysqld 2>/dev/null || which mariadbd 2>/dev/null":     "",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Status != StatusNotFound {
		t.Errorf("expected status not_found, got %s", comp.Status)
	}
}

func TestDetectMySQL_MariaDB(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1": "root 1234 0.0 0.1 123456 7890 ? Ssl 00:00 0:00 mariadbd --version=10.6.15",
		"ss -tlnp 2>/dev/null | grep -E 'mysqld|mariadbd' | grep -oP ':\\K[0-9]+' | head -1": "3306",
		"which mysqld 2>/dev/null || which mariadbd 2>/dev/null": "/usr/bin/mariadbd",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Name != "MariaDB" {
		t.Errorf("expected name MariaDB, got %s", comp.Name)
	}
}

func TestDetectPostgreSQL_Running(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep postgres | grep -v grep | head -1": "postgres 1234 0.0 0.1 123456 7890 ? Ss 00:00 0:00 postgres -D /var/lib/postgresql/16/main",
		"ss -tlnp 2>/dev/null | grep postgres | grep -oP ':\\K[0-9]+' | head -1": "5432",
		"which postgres 2>/dev/null": "/usr/bin/postgres",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectPostgreSQL(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Type != ComponentPostgres {
		t.Errorf("expected type postgresql, got %s", comp.Type)
	}
	if comp.Status != StatusRunning {
		t.Errorf("expected status running, got %s", comp.Status)
	}
	if comp.Port != 5432 {
		t.Errorf("expected port 5432, got %d", comp.Port)
	}
}

func TestDetectRedis_Running(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep redis | grep -v grep | head -1": "redis 1234 0.0 0.1 123456 7890 ? Ssl 00:00 0:00 redis-server 127.0.0.1:6379",
		"ss -tlnp 2>/dev/null | grep redis | grep -oP ':\\K[0-9]+' | head -1": "6379",
		"which redis-server 2>/dev/null": "/usr/bin/redis-server",
		"redis-cli ping 2>/dev/null":     "PONG",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectRedis(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Type != ComponentRedis {
		t.Errorf("expected type redis, got %s", comp.Type)
	}
	if comp.Status != StatusRunning {
		t.Errorf("expected status running, got %s", comp.Status)
	}
	if comp.Details != "responding to PING" {
		t.Errorf("expected details 'responding to PING', got %s", comp.Details)
	}
}

func TestDetectMongoDB_Running(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep mongod | grep -v grep | head -1": "mongodb 1234 0.0 0.5 123456 7890 ? Ssl 00:00 0:00 mongod --dbpath=/data/db",
		"ss -tlnp 2>/dev/null | grep mongod | grep -oP ':\\K[0-9]+' | head -1": "27017",
		"which mongod 2>/dev/null": "/usr/bin/mongod",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectMongoDB(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Type != ComponentMongoDB {
		t.Errorf("expected type mongodb, got %s", comp.Type)
	}
	if comp.Status != StatusRunning {
		t.Errorf("expected status running, got %s", comp.Status)
	}
}

func TestDetectNginx_Running(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep nginx | grep -v grep | head -1": "root 1234 0.0 0.1 123456 7890 ? Ss 00:00 0:00 nginx: master process /usr/sbin/nginx",
		"nginx -v 2>&1":                                   "nginx version: nginx/1.24.0",
		"ss -tlnp 2>/dev/null | grep nginx | grep -oP ':\\K[0-9]+' | head -1": "80",
		"which nginx 2>/dev/null": "/usr/sbin/nginx",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectNginx(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Type != ComponentNginx {
		t.Errorf("expected type nginx, got %s", comp.Type)
	}
	if comp.Version != "1.24.0" {
		t.Errorf("expected version 1.24.0, got %s", comp.Version)
	}
}

func TestDetectApache_Running(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep -E 'httpd|apache2' | grep -v grep | head -1": "www-data 1234 0.0 0.1 123456 7890 ? Ss 00:00 0:00 /usr/sbin/apache2 -k start",
		"httpd -v 2>/dev/null || apache2 -v 2>/dev/null":            "Server version: Apache/2.4.58 (Ubuntu)",
		"ss -tlnp 2>/dev/null | grep -E 'httpd|apache2' | grep -oP ':\\K[0-9]+' | head -1": "80",
		"which httpd 2>/dev/null || which apache2 2>/dev/null": "/usr/sbin/apache2",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectApache(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Type != ComponentApache {
		t.Errorf("expected type apache, got %s", comp.Type)
	}
	if comp.Name != "Apache HTTPD" {
		t.Errorf("expected name Apache HTTPD, got %s", comp.Name)
	}
}

func TestDetectOpenResty_Running(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep openresty | grep -v grep | head -1": "root 1234 0.0 0.1 123456 7890 ? Ss 00:00 0:00 openresty",
		"nginx -v 2>&1":                                     "nginx version: openresty/1.25.3.1",
		"ss -tlnp 2>/dev/null | grep openresty | grep -oP ':\\K[0-9]+' | head -1": "80",
		"which openresty 2>/dev/null || which nginx 2>/dev/null": "/usr/bin/openresty",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectOpenResty(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Type != ComponentOpenResty {
		t.Errorf("expected type openresty, got %s", comp.Type)
	}
	if comp.Version != "1.25.3.1" {
		t.Errorf("expected version 1.25.3.1, got %s", comp.Version)
	}
}

func TestDetectOpenResty_NotFoundAsNginx(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep openresty | grep -v grep | head -1": "",
		"which openresty 2>/dev/null || which nginx 2>/dev/null": "/usr/bin/nginx",
		"nginx -v 2>&1":                                           "nginx version: nginx/1.24.0",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	comp, err := d.detectOpenResty(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.Status != StatusNotFound {
		t.Errorf("expected status not_found (nginx is not openresty), got %s", comp.Status)
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"mysqld --version=8.0.36", "8.0.36"},
		{"v1.2.3-alpha", "1.2.3"},
		{"postgres 16.1", "16.1"},
		{"redis-server v7.2.4", "7.2.4"},
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
		{"some other output v1.0", "1.0"},
	}

	for _, tt := range tests {
		result := extractVersionFromNginxOutput(tt.input)
		if result != tt.expected {
			t.Errorf("extractVersionFromNginxOutput(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetDetectionSummary(t *testing.T) {
	components := []DetectedComponent{
		{Type: ComponentMySQL, Name: "MySQL", Status: StatusRunning, Version: "8.0.36", Port: 3306},
		{Type: ComponentRedis, Name: "Redis", Status: StatusStopped},
	}

	summary := GetDetectionSummary(components)
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if len(summary) < 10 {
		t.Errorf("expected longer summary, got %q", summary)
	}
}

func TestGetDetectionSummary_Empty(t *testing.T) {
	summary := GetDetectionSummary([]DetectedComponent{})
	if summary != "No system components detected" {
		t.Errorf("expected 'No system components detected', got %q", summary)
	}
}

func TestDetectDatabases(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1": "",
		"which mysqld 2>/dev/null || which mariadbd 2>/dev/null":     "/usr/bin/mysqld",
		"ps aux | grep postgres | grep -v grep | head -1":             "",
		"which postgres 2>/dev/null":                                  "",
		"ps aux | grep redis | grep -v grep | head -1":                "",
		"which redis-server 2>/dev/null":                              "",
		"ps aux | grep mongod | grep -v grep | head -1":               "",
		"which mongod 2>/dev/null":                                    "",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	components, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(components) != 1 {
		t.Errorf("expected 1 component (MySQL stopped), got %d", len(components))
	}
}

func TestDetectWebServers(t *testing.T) {
	responses := map[string]string{
		"ps aux | grep nginx | grep -v grep | head -1":                "",
		"which nginx 2>/dev/null":                                     "",
		"ps aux | grep -E 'httpd|apache2' | grep -v grep | head -1":   "",
		"which httpd 2>/dev/null || which apache2 2>/dev/null":        "",
		"ps aux | grep openresty | grep -v grep | head -1":            "",
		"which openresty 2>/dev/null || which nginx 2>/dev/null":      "",
	}
	mock := newMockExecutor(responses)
	d := New(mock)

	components, err := d.DetectWebServers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(components) != 0 {
		t.Errorf("expected 0 components, got %d", len(components))
	}
}

var _ deployer.CommandExecutor = (*mockExecutor)(nil)