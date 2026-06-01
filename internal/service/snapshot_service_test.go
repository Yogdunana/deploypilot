package service

import (
	"context"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/database"
)

type snapshotMockExecutor struct {
	responses map[string]string
}

func (m *snapshotMockExecutor) RunCommand(_ context.Context, cmd string) (string, error) {
	if out, ok := m.responses[cmd]; ok {
		return out, nil
	}
	return "", nil
}

func (m *snapshotMockExecutor) Close() error {
	return nil
}

func TestDefaultSnapshotConfig(t *testing.T) {
	cfg := DefaultSnapshotConfig()

	if cfg == nil {
		t.Fatal("DefaultSnapshotConfig returned nil")
	}

	if len(cfg.Paths) == 0 {
		t.Error("expected non-empty paths")
	}

	if len(cfg.ExcludePaths) == 0 {
		t.Error("expected non-empty exclude paths")
	}

	if cfg.MaxFileSize != 1024*1024 {
		t.Errorf("expected MaxFileSize=1MB, got %d", cfg.MaxFileSize)
	}

	if cfg.IncludeContent {
		t.Error("expected IncludeContent=false by default")
	}

	expectedPaths := []string{"/etc/nginx", "/etc/ssh", "/etc/systemd/system"}
	for _, p := range expectedPaths {
		found := false
		for _, ep := range cfg.Paths {
			if ep == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected path %s in default config", p)
		}
	}

	sensitivePaths := []string{"/etc/shadow", "/etc/gshadow"}
	for _, p := range sensitivePaths {
		found := false
		for _, ep := range cfg.ExcludePaths {
			if ep == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected sensitive path %s to be excluded by default", p)
		}
	}
}

func TestSnapshotService_CreateAndList(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	svc := NewSnapshotService(db)
	exec := &snapshotMockExecutor{
		responses: map[string]string{
			"find '/etc/nginx' -type f 2>/dev/null | head -500": "/etc/nginx/nginx.conf\n/etc/nginx/sites-enabled/default",
			"stat -c '%s %Y %a' '/etc/nginx/nginx.conf' 2>/dev/null": "1024 1704067200 644",
			"md5sum '/etc/nginx/nginx.conf' 2>/dev/null | awk '{print $1}'": "abc123def456",
			"stat -c '%s %Y %a' '/etc/nginx/sites-enabled/default' 2>/dev/null": "512 1704067200 644",
			"md5sum '/etc/nginx/sites-enabled/default' 2>/dev/null | awk '{print $1}'": "def789ghi012",
		},
	}

	ctx := context.Background()
	_ = exec

	svc.db = db

	snap := &SystemSnapshot{
		ID:          "snap-001",
		ServerID:    "server-001",
		Name:        "test-snapshot",
		Description: "Test snapshot",
		FileCount:   2,
		TotalSize:   1536,
		Checksum:    "test123",
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	if err := db.Create(snap).Error; err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	snaps, err := svc.ListSnapshots(ctx, "server-001")
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}

	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}

	if snaps[0].Name != "test-snapshot" {
		t.Errorf("expected name 'test-snapshot', got %q", snaps[0].Name)
	}
}

func TestSnapshotService_GetAndDelete(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	svc := NewSnapshotService(db)

	snap := &SystemSnapshot{
		ID:          "snap-del-001",
		ServerID:    "server-001",
		Name:        "delete-test",
		Description: "Snapshot to delete",
		FileCount:   1,
		TotalSize:   100,
		Checksum:    "del123",
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	if err := db.Create(snap).Error; err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	ctx := context.Background()

	retrieved, err := svc.GetSnapshot(ctx, "snap-del-001")
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}
	if retrieved.Name != "delete-test" {
		t.Errorf("expected name 'delete-test', got %q", retrieved.Name)
	}

	err = svc.DeleteSnapshot(ctx, "snap-del-001")
	if err != nil {
		t.Fatalf("DeleteSnapshot failed: %v", err)
	}

	_, err = svc.GetSnapshot(ctx, "snap-del-001")
	if err == nil {
		t.Error("expected error when getting deleted snapshot")
	}
}

func TestSnapshotFile_IsBinary(t *testing.T) {
	binaryPaths := []string{
		"/path/to/binary.bin",
		"/path/to/app.exe",
		"/path/to/library.so",
		"/path/to/image.png",
		"/path/to/archive.zip",
		"/path/to/archive.tar.gz",
	}

	for _, p := range binaryPaths {
		if !isBinaryPath(p) {
			t.Errorf("expected %q to be detected as binary", p)
		}
	}

	textPaths := []string{
		"/path/to/config.conf",
		"/path/to/script.sh",
		"/path/to/data.json",
		"/path/to/readme.md",
	}

	for _, p := range textPaths {
		if isBinaryPath(p) {
			t.Errorf("expected %q to be detected as text", p)
		}
	}
}

func TestParseTimestamp(t *testing.T) {
	ts := int64(1704067200)
	expected := time.Unix(ts, 0)

	result, err := parseTimestamp("1704067200")
	if err != nil {
		t.Fatalf("parseTimestamp failed: %v", err)
	}

	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestParseTimestamp_Invalid(t *testing.T) {
	_, err := parseTimestamp("not-a-number")
	if err == nil {
		t.Error("expected error for invalid timestamp")
	}

	_, err = parseTimestamp("")
	if err == nil {
		t.Error("expected error for empty timestamp")
	}
}

func TestSnapshotService_IsExcluded(t *testing.T) {
	svc := &SnapshotService{}
	excludes := []string{"/etc/shadow", "/etc/gshadow", "/etc/ssh/ssh_host_"}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/etc/shadow", true},
		{"/etc/gshadow", true},
		{"/etc/ssh/ssh_host_rsa_key", true},
		{"/etc/ssh/ssh_host_ed25519_key", true},
		{"/etc/passwd", false},
		{"/etc/nginx/nginx.conf", false},
		{"/etc/ssh/sshd_config", false},
	}

	for _, tt := range tests {
		result := svc.isExcluded(tt.path, excludes)
		if result != tt.expected {
			t.Errorf("isExcluded(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

func TestSnapshotService_CalculateChecksum(t *testing.T) {
	svc := &SnapshotService{}

	files := []SnapshotFile{
		{Path: "/etc/nginx/nginx.conf", Checksum: "abc123"},
		{Path: "/etc/nginx/sites-enabled/default", Checksum: "def456"},
	}

	checksum := svc.calculateChecksum(files)
	if checksum == "" {
		t.Error("expected non-empty checksum")
	}

	checksum2 := svc.calculateChecksum(files)
	if checksum != checksum2 {
		t.Error("expected deterministic checksum")
	}

	differentFiles := []SnapshotFile{
		{Path: "/etc/nginx/nginx.conf", Checksum: "different"},
	}
	checksum3 := svc.calculateChecksum(differentFiles)
	if checksum == checksum3 {
		t.Error("expected different checksum for different files")
	}
}

func TestSnapshotDiff_Structure(t *testing.T) {
	diff := &SnapshotDiff{
		Added: []SnapshotFile{
			{Path: "/etc/new/file.conf", Size: 100},
		},
		Removed: []SnapshotFile{
			{Path: "/etc/old/file.conf", Size: 50},
		},
		Modified: []SnapshotFile{
			{Path: "/etc/modified/file.conf", Size: 200},
		},
		Summary: "+1 ~1 -1",
	}

	if len(diff.Added) != 1 {
		t.Errorf("expected 1 added file, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 1 {
		t.Errorf("expected 1 removed file, got %d", len(diff.Removed))
	}
	if len(diff.Modified) != 1 {
		t.Errorf("expected 1 modified file, got %d", len(diff.Modified))
	}
	if diff.Summary != "+1 ~1 -1" {
		t.Errorf("expected summary '+1 ~1 -1', got %q", diff.Summary)
	}
}

func TestSystemSnapshot_TableName(t *testing.T) {
	snap := SystemSnapshot{}
	if snap.TableName() != "system_snapshots" {
		t.Errorf("expected table name 'system_snapshots', got %q", snap.TableName())
	}
}

func TestSnapshotService_CreateSnapshot_DefaultConfig(t *testing.T) {
	db, err := database.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	_ = NewSnapshotService(db)
	ctx := context.Background()
	_ = ctx

	cfg := DefaultSnapshotConfig()

	if len(cfg.Paths) == 0 {
		t.Error("DefaultSnapshotConfig should return valid paths")
	}

	if cfg.MaxFileSize != 1024*1024 {
		t.Errorf("expected MaxFileSize=1MB, got %d", cfg.MaxFileSize)
	}

	if cfg.IncludeContent {
		t.Error("IncludeContent should be false by default")
	}

	for _, path := range cfg.Paths {
		if path == "" {
			t.Error("found empty path in default config")
		}
	}

	for _, path := range cfg.ExcludePaths {
		if path == "" {
			t.Error("found empty path in exclude config")
		}
	}

	_ = ctx
}

func TestSnapshotFile_Structure(t *testing.T) {
	file := SnapshotFile{
		Path:        "/etc/nginx/nginx.conf",
		Size:        1024,
		Modified:    "2024-01-01T00:00:00Z",
		Checksum:    "abc123def456",
		Content:     "",
		IsBinary:    false,
		Permissions: "644",
	}

	if file.Path != "/etc/nginx/nginx.conf" {
		t.Errorf("expected path '/etc/nginx/nginx.conf', got %q", file.Path)
	}
	if file.Size != 1024 {
		t.Errorf("expected size 1024, got %d", file.Size)
	}
	if file.IsBinary {
		t.Error("expected IsBinary=false for .conf file")
	}
	if file.Permissions != "644" {
		t.Errorf("expected permissions '644', got %q", file.Permissions)
	}
}

func TestSnapshotConfig_Structure(t *testing.T) {
	cfg := &SnapshotConfig{
		Paths:          []string{"/etc/nginx", "/etc/ssh"},
		ExcludePaths:   []string{"/etc/shadow"},
		MaxFileSize:    1024 * 1024,
		IncludeContent: false,
	}

	if len(cfg.Paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(cfg.Paths))
	}
	if len(cfg.ExcludePaths) != 1 {
		t.Errorf("expected 1 exclude path, got %d", len(cfg.ExcludePaths))
	}
	if cfg.MaxFileSize != 1024*1024 {
		t.Errorf("expected MaxFileSize=1MB, got %d", cfg.MaxFileSize)
	}
}
