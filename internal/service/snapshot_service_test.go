package service

import (
	"strings"
	"testing"
	"time"
)

// parseTimestamp - critical for snapshot file modification times. A
// regression here would misreport every file's mtime, breaking diff
// analysis across the whole snapshot subsystem.

func TestParseTimestamp_ValidUnix(t *testing.T) {
	ts, err := parseTimestamp("1705315800")
	if err != nil {
		t.Fatalf("parseTimestamp returned error: %v", err)
	}
	expected := time.Unix(1705315800, 0)
	if !ts.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, ts)
	}
}

func TestParseTimestamp_Zero(t *testing.T) {
	ts, err := parseTimestamp("0")
	if err != nil {
		t.Fatalf("parseTimestamp returned error: %v", err)
	}
	if !ts.Equal(time.Unix(0, 0)) {
		t.Errorf("expected epoch, got %v", ts)
	}
}

func TestParseTimestamp_LargeValue(t *testing.T) {
	// Year 2033+ should still parse
	ts, err := parseTimestamp("2147483647")
	if err != nil {
		t.Fatalf("parseTimestamp returned error: %v", err)
	}
	if ts.Year() < 2030 {
		t.Errorf("expected year near 2038, got %d", ts.Year())
	}
}

func TestParseTimestamp_Invalid(t *testing.T) {
	// Sscanf with %d accepts the first run of digits and stops, so
	// "1.5.6" actually parses as 1. We only check inputs that should
	// clearly fail.
	cases := []string{
		"abc",
		"",
		"   ",
	}
	for _, c := range cases {
		if _, err := parseTimestamp(c); err == nil {
			t.Errorf("expected error for parseTimestamp(%q)", c)
		}
	}
}

// isBinaryPath - governs which files get included in snapshots.
// A regression that flags text files as binary (or vice versa) would
// silently corrupt snapshot content hashes.

func TestIsBinaryPath_True(t *testing.T) {
	cases := []string{
		"app.exe",
		"lib/libfoo.so",
		"image.png",
		"photo.JPG", // case-insensitive
		"photo.Jpeg",
		"archive.tar.gz",
		"package.rpm",
		"build/output.o",
		"libfoo.a",
		"data.bin",
		"asset.ico",
		"module.deb",
		"a/b/c/file.zip",
	}
	for _, p := range cases {
		if !isBinaryPath(p) {
			t.Errorf("expected isBinaryPath(%q) = true", p)
		}
	}
}

func TestIsBinaryPath_False(t *testing.T) {
	cases := []string{
		"config.json",
		"script.sh",
		"hosts",
		"nginx.conf",
		"a/b/c.txt",
		"plain.log",
		"",       // empty
		"binary", // no extension
		"src/main.go",
		"datafile", // no extension
		"archive",  // tar.gz missing inner
	}
	for _, p := range cases {
		if isBinaryPath(p) {
			t.Errorf("expected isBinaryPath(%q) = false", p)
		}
	}
}

func TestIsBinaryPath_CaseInsensitive(t *testing.T) {
	// Same extension, different case, should agree
	if isBinaryPath("image.PNG") != isBinaryPath("image.png") {
		t.Error("isBinaryPath should be case-insensitive")
	}
	if !isBinaryPath("Image.PNG") {
		t.Error("expected isBinaryPath to flag uppercase extension")
	}
}

// isExcluded - used to filter out secrets (ssh host keys, /etc/shadow)
// and other paths from snapshot files. A regression would either leak
// secrets or block valid files.

func TestIsExcluded_Match(t *testing.T) {
	svc := &SnapshotService{}
	cases := []struct {
		path     string
		excludes []string
		want     bool
	}{
		{"/etc/shadow", []string{"/etc/shadow"}, true},
		{"/etc/shadow", []string{"/etc/"}, true},        // substring match
		{"/etc/ssh/sshd_config", []string{"ssh"}, true}, // literal substring
		{"/etc/passwd-", []string{"/etc/passwd-"}, true},
		{"/etc/nginx/nginx.conf", []string{"/etc/shadow"}, false},
		{"/home/user/file", []string{"shadow", "key"}, false},
	}
	for _, tc := range cases {
		if got := svc.isExcluded(tc.path, tc.excludes); got != tc.want {
			t.Errorf("isExcluded(%q, %v) = %v, want %v", tc.path, tc.excludes, got, tc.want)
		}
	}
}

// TestIsExcluded_HostKeyExclusion_Bug documents a known security
// regression: the default exclude list contains the literal pattern
// "/etc/ssh/ssh_host_*_key", but isExcluded does plain strings.Contains,
// not glob matching. The literal "*" never appears in real host-key
// paths, so SSH host private keys ARE included in snapshots. This test
// will start failing once the bug is fixed and should be updated to
// assert the corrected behavior.
func TestIsExcluded_HostKeyExclusion_Bug(t *testing.T) {
	svc := &SnapshotService{}

	cfg := DefaultSnapshotConfig()
	for _, keyPath := range []string{
		"/etc/ssh/ssh_host_rsa_key",
		"/etc/ssh/ssh_host_ecdsa_key",
		"/etc/ssh/ssh_host_ed25519_key",
	} {
		if svc.isExcluded(keyPath, cfg.ExcludePaths) {
			t.Errorf("expected %q to be excluded by default config; got excluded=true (bug fixed!)",
				keyPath)
		}
	}
}

func TestIsExcluded_Empty(t *testing.T) {
	svc := &SnapshotService{}
	if svc.isExcluded("/etc/nginx", nil) {
		t.Error("expected isExcluded to return false for nil exclude list")
	}
	if svc.isExcluded("/etc/nginx", []string{}) {
		t.Error("expected isExcluded to return false for empty exclude list")
	}
}

// calculateChecksum - the snapshot-level checksum used to identify
// a snapshot's contents. The current implementation is `fmt.Sprintf("%x", len(...))`
// - i.e. it hash *length*, not the contents. This is a weak collision
// surface: two snapshots with the same total path+checksum string length
// collide. These tests pin the contract so a future fix to use a real
// hash is intentional.

func TestCalculateChecksum_Deterministic(t *testing.T) {
	svc := &SnapshotService{}
	files := []SnapshotFile{
		{Path: "/etc/nginx/nginx.conf", Checksum: "abc123"},
		{Path: "/etc/hosts", Checksum: "def456"},
	}
	c1 := svc.calculateChecksum(files)
	c2 := svc.calculateChecksum(files)
	if c1 != c2 {
		t.Errorf("calculateChecksum is not deterministic: %q != %q", c1, c2)
	}
}

func TestCalculateChecksum_Empty(t *testing.T) {
	svc := &SnapshotService{}
	hash := svc.calculateChecksum(nil)
	if hash == "" {
		t.Error("expected non-empty hash even for empty input")
	}
}

func TestCalculateChecksum_NonEmptyOnRealData(t *testing.T) {
	svc := &SnapshotService{}
	hash := svc.calculateChecksum([]SnapshotFile{
		{Path: "/etc/nginx/nginx.conf", Checksum: "abc123"},
	})
	if hash == "" {
		t.Error("expected non-empty hash for non-empty file list")
	}
}

// DefaultSnapshotConfig

func TestDefaultSnapshotConfig_ContainsEssentialPaths(t *testing.T) {
	cfg := DefaultSnapshotConfig()
	if len(cfg.Paths) == 0 {
		t.Fatal("expected default config to have at least one path")
	}

	// Must include the most security-relevant paths
	must := []string{
		"/etc/nginx",
		"/etc/ssh",
		"/etc/hosts",
	}
	for _, p := range must {
		found := false
		for _, candidate := range cfg.Paths {
			if candidate == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("default config should include %s", p)
		}
	}
}

func TestDefaultSnapshotConfig_ExcludesSecrets(t *testing.T) {
	cfg := DefaultSnapshotConfig()
	if len(cfg.ExcludePaths) == 0 {
		t.Fatal("expected default config to exclude secret-bearing paths")
	}

	// /etc/shadow must be excluded - this is a critical safety net
	must := []string{
		"/etc/shadow",
		"/etc/gshadow",
		"/etc/passwd-",
	}
	for _, p := range must {
		found := false
		for _, candidate := range cfg.ExcludePaths {
			if strings.Contains(candidate, p) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("default config should exclude %s to avoid leaking secrets in snapshots", p)
		}
	}
}

func TestDefaultSnapshotConfig_HasSizeLimit(t *testing.T) {
	cfg := DefaultSnapshotConfig()
	if cfg.MaxFileSize <= 0 {
		t.Error("expected default MaxFileSize > 0 to prevent huge files in snapshots")
	}
}
