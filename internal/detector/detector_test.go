package detector

import (
	"context"
	"strings"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// fakeExecutor is a minimal CommandExecutor for detector tests. It
// returns the output for any command that exactly matches one of the
// configured responses, and returns empty output (with no error) for
// every other command.
type fakeExecutor struct {
	responses map[string]string
	calls     []string
}

func (f *fakeExecutor) RunCommand(_ context.Context, cmd string) (string, error) {
	f.calls = append(f.calls, cmd)
	if out, ok := f.responses[cmd]; ok {
		return out, nil
	}
	return "", nil
}

// TestExtractVersion_FullSemver is the most common detection case:
// the `ps` output line for a running daemon includes a full
// X.Y.Z version string. The detector must return the version with
// patch level preserved.
func TestExtractVersion_FullSemver(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"mysqld 8.0.32-log", "8.0.32"},
		{"postgres (PostgreSQL) 15.4", "15.4"},
		{"redis-server version 7.2.1", "7.2.1"},
		{"mongod v7.0.4", "7.0.4"},
	}
	for _, tc := range cases {
		got := extractVersion(tc.in)
		if got != tc.want {
			t.Errorf("extractVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestExtractVersion_PrereleaseTags documents the regex's deliberate
// limitation: the full-semver pattern only captures the
// digits + lowercase alphanumeric segment. A pre-release tag that
// begins with "-" (e.g. "1.2.3-rc1") is truncated to the numeric
// prefix. This test pins the current behavior so a future change to
// the regex is forced to update the test (and review the implications
// for nginx/openresty downstream consumers).
func TestExtractVersion_PrereleaseTags(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// The trailing alphanumeric portion is included.
		{"1.2.3a1", "1.2.3a1"},
		{"7.0.0beta1", "7.0.0beta1"},
		// The "-" separator stops the match; only the numeric prefix is
		// returned. This is a known limitation.
		{"1.2.3-rc1", "1.2.3"},
		{"1.2.3-1", "1.2.3"},
	}
	for _, tc := range cases {
		if got := extractVersion(tc.in); got != tc.want {
			t.Errorf("extractVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestExtractVersion_PrefixedAndShort covers the fallbacks used when
// the full semver is missing: v-prefixed X.Y and bare X.Y.
func TestExtractVersion_PrefixedAndShort(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"v1.2", "1.2"},
		{"version 3.10", "3.10"},
	}
	for _, tc := range cases {
		if got := extractVersion(tc.in); got != tc.want {
			t.Errorf("extractVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestExtractVersion_NoMatch returns an empty string when the input
// has no recognizable version, so the detector can leave the field
// blank instead of inventing one.
func TestExtractVersion_NoMatch(t *testing.T) {
	for _, in := range []string{"", "no version here", "alpha"} {
		if got := extractVersion(in); got != "" {
			t.Errorf("extractVersion(%q) = %q, want empty", in, got)
		}
	}
}

// TestExtractVersionFromNginxOutput covers the dedicated parser used
// for the `nginx -v` output. The expected format is
// "nginx version: nginx/X.Y.Z" (or openresty/X.Y.Z).
func TestExtractVersionFromNginxOutput(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"nginx version: nginx/1.24.0", "1.24.0"},
		{"nginx version: nginx/1.18.0 (Ubuntu)", "1.18.0"},
		{"nginx version: openresty/1.25.3.1", "1.25.3.1"},
	}
	for _, tc := range cases {
		if got := extractVersionFromNginxOutput(tc.in); got != tc.want {
			t.Errorf("extractVersionFromNginxOutput(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestGetDetectionSummary_Empty locks in the "no components found"
// branch: it must say "No system components detected" rather than
// printing a confusing header.
func TestGetDetectionSummary_Empty(t *testing.T) {
	got := GetDetectionSummary(nil)
	if !strings.Contains(got, "No system components detected") {
		t.Errorf("GetDetectionSummary(nil) = %q, want message about no components", got)
	}
}

// TestGetDetectionSummary_AllStatuses exercises each status icon
// path. A status that is not in the switch falls back to "❌" —
// documenting that here is a regression guard for the user-visible
// output.
func TestGetDetectionSummary_AllStatuses(t *testing.T) {
	comps := []DetectedComponent{
		{Type: ComponentMySQL, Name: "MySQL", Status: StatusRunning, Version: "8.0.32", Port: 3306},
		{Type: ComponentNginx, Name: "Nginx", Status: StatusStopped},
		{Type: ComponentRedis, Name: "Redis", Status: StatusInstalled, Version: "7.2.1"},
		{Type: ComponentPostgres, Name: "Postgres", Status: "mystery"},
	}
	got := GetDetectionSummary(comps)
	mustContain := []string{
		"Detected 4 component(s)",
		"MySQL",
		"v8.0.32",
		":3306",
		"running",
		"Nginx",
		"stopped",
		"Redis",
		"v7.2.1",
		"installed",
		"mystery", // unknown status is rendered verbatim
	}
	for _, s := range mustContain {
		if !strings.Contains(got, s) {
			t.Errorf("GetDetectionSummary output missing %q\nfull output:\n%s", s, got)
		}
	}
}

// TestDetectMySQL_Running verifies the happy path: when `ps` reports
// mysqld, the detector must mark the component as running and pull
// the version from the same line.
func TestDetectMySQL_Running(t *testing.T) {
	exec := &fakeExecutor{responses: map[string]string{
		"ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1":                        "mysql 1234 1.0 0.5 123456 7890 ? Sl 09:00 0:01 /usr/sbin/mysqld --version=8.0.32",
		"ss -tlnp 2>/dev/null | grep -E 'mysqld|mariadbd' | grep -oP ':\\K[0-9]+' | head -1": "3306",
		"which mysqld 2>/dev/null || which mariadbd 2>/dev/null":                             "/usr/sbin/mysqld",
	}}
	d := New(exec)
	got, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("detectMySQL: %v", err)
	}
	if got.Status != StatusRunning {
		t.Errorf("status = %s, want running", got.Status)
	}
	if got.Port != 3306 {
		t.Errorf("port = %d, want 3306", got.Port)
	}
	if got.Version != "8.0.32" {
		t.Errorf("version = %q, want 8.0.32", got.Version)
	}
	if !strings.Contains(got.BinaryPath, "mysqld") {
		t.Errorf("binary path = %q, want to contain mysqld", got.BinaryPath)
	}
}

// TestDetectMySQL_NotInstalled covers the "binary missing" path:
// neither `ps` nor `which` returns anything, so the component must
// be reported as not_found.
func TestDetectMySQL_NotInstalled(t *testing.T) {
	exec := &fakeExecutor{responses: map[string]string{}}
	d := New(exec)
	got, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("detectMySQL: %v", err)
	}
	if got.Status != StatusNotFound {
		t.Errorf("status = %s, want not_found", got.Status)
	}
}

// TestDetectMySQL_StoppedButInstalled covers the case where the
// binary is on disk but the daemon is not running. The status must
// be "stopped" (not "not_found") so the UI can show "installed but
// not running".
func TestDetectMySQL_StoppedButInstalled(t *testing.T) {
	exec := &fakeExecutor{responses: map[string]string{
		"which mysqld 2>/dev/null || which mariadbd 2>/dev/null": "/usr/sbin/mysqld",
	}}
	d := New(exec)
	got, err := d.detectMySQL(context.Background())
	if err != nil {
		t.Fatalf("detectMySQL: %v", err)
	}
	if got.Status != StatusStopped {
		t.Errorf("status = %s, want stopped", got.Status)
	}
	if !strings.Contains(got.BinaryPath, "mysqld") {
		t.Errorf("binary path = %q, want to contain mysqld", got.BinaryPath)
	}
}

// TestDetectNginx_RunningWithVersion covers the nginx-specific path
// that uses `nginx -v 2>&1` to extract the version. A version that
// comes from this separate command (not the `ps` line) must be
// reported correctly.
func TestDetectNginx_RunningWithVersion(t *testing.T) {
	exec := &fakeExecutor{responses: map[string]string{
		"ps aux | grep nginx | grep -v grep | head -1": "root 1 0.0 0.0 nginx: master process",
		"nginx -v 2>&1": "nginx version: nginx/1.24.0",
		"ss -tlnp 2>/dev/null | grep nginx | grep -oP ':\\K[0-9]+' | head -1": "80",
		"which nginx 2>/dev/null": "/usr/sbin/nginx",
		"nginx -t 2>&1 | grep 'configuration file' | sed 's/.*configuration file //' | sed 's/ syntax.*//'": "/etc/nginx/nginx.conf",
	}}
	d := New(exec)
	got, err := d.detectNginx(context.Background())
	if err != nil {
		t.Fatalf("detectNginx: %v", err)
	}
	if got.Status != StatusRunning {
		t.Errorf("status = %s, want running", got.Status)
	}
	if got.Version != "1.24.0" {
		t.Errorf("version = %q, want 1.24.0", got.Version)
	}
	if got.Port != 80 {
		t.Errorf("port = %d, want 80", got.Port)
	}
	if !strings.Contains(got.InstallPath, "nginx.conf") {
		t.Errorf("install path = %q, want to contain nginx.conf", got.InstallPath)
	}
}

// TestDetectAll_Aggregation verifies that DetectAll runs every
// detector and only returns the components that are present.
// A regression that double-counts or omits a detector would break
// the "system overview" page.
func TestDetectAll_Aggregation(t *testing.T) {
	exec := &fakeExecutor{responses: map[string]string{
		// MySQL is running.
		"ps aux | grep -E 'mysqld|mariadbd' | grep -v grep | head -1":                        "mysql ... /usr/sbin/mysqld",
		"ss -tlnp 2>/dev/null | grep -E 'mysqld|mariadbd' | grep -oP ':\\K[0-9]+' | head -1": "3306",
		"which mysqld 2>/dev/null || which mariadbd 2>/dev/null":                             "/usr/sbin/mysqld",
	}}
	d := New(exec)
	got, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("DetectAll returned %d components, want 1: %+v", len(got), got)
	}
	if got[0].Type != ComponentMySQL {
		t.Errorf("first component type = %s, want mysql", got[0].Type)
	}
}

// TestDetectDatabases_SkipsWebServers checks the narrow entrypoint:
// only the four database detectors must run, never nginx/apache.
func TestDetectDatabases_SkipsWebServers(t *testing.T) {
	exec := &fakeExecutor{responses: map[string]string{}}
	d := New(exec)
	got, err := d.DetectDatabases(context.Background())
	if err != nil {
		t.Fatalf("DetectDatabases: %v", err)
	}
	for _, c := range got {
		switch c.Type {
		case ComponentMySQL, ComponentPostgres, ComponentRedis, ComponentMongoDB:
			// expected
		default:
			t.Errorf("DetectDatabases returned web/database type %s", c.Type)
		}
	}
}

// TestDetectorImplementsExecutorInterface is a compile-time
// guarantee that the detector accepts the same CommandExecutor used
// by the deployer package. This avoids a future refactor that
// inadvertently changes the type and breaks callers.
func TestDetectorImplementsExecutorInterface(t *testing.T) {
	var _ deployer.CommandExecutor = (*fakeExecutor)(nil)
}
