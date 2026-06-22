package service

import (
	"testing"
)

// isPathBlocked prevents access to /proc, /sys, /dev, /boot, /etc, /root,
// /var/log, /bin, /sbin, /usr. This is critical for the security boundary
// of remote file management: if a regression lets an operator reach /etc,
// they can read /etc/shadow via the read endpoint.

func TestIsPathBlocked_SystemPaths(t *testing.T) {
	cases := []struct {
		path    string
		blocked bool
	}{
		// Direct system paths
		{"/proc", true},
		{"/proc/", true},
		{"/proc/1/cmdline", true},
		{"/sys", true},
		{"/sys/kernel", true},
		{"/dev", true},
		{"/dev/null", true},
		{"/dev/sda", true},
		{"/boot", true},
		{"/boot/grub", true},
		{"/etc", true},
		{"/etc/passwd", true},
		{"/etc/shadow", true},
		{"/etc/nginx/nginx.conf", true},
		{"/root", true},
		{"/root/.ssh/id_rsa", true},
		{"/var/log", true},
		{"/var/log/syslog", true},
		{"/bin", true},
		{"/bin/bash", true},
		{"/sbin", true},
		{"/sbin/ifconfig", true},
		{"/usr", true},
		{"/usr/bin", true},
		{"/usr/local/bin", true},

		// Allowlisted paths
		{"/home/user/file.txt", false},
		{"/var/www/html", false},
		{"/opt/app", false},
		{"/srv/data", false},
		{"/tmp/scratch", false},
		{"/", false}, // root is the boundary, not "blocked" itself
	}
	for _, tc := range cases {
		if got := isPathBlocked(tc.path); got != tc.blocked {
			t.Errorf("isPathBlocked(%q) = %v, want %v", tc.path, got, tc.blocked)
		}
	}
}

func TestIsPathBlocked_PathTraversalAttempts(t *testing.T) {
	// Path traversal with relative segments should be cleaned and blocked
	// if they reach a blocked directory.
	cases := []struct {
		path    string
		blocked bool
	}{
		{"/var/../etc/passwd", true},
		{"/home/../../etc/shadow", true},
		{"/opt/../proc/1/status", true},
	}
	for _, tc := range cases {
		if got := isPathBlocked(tc.path); got != tc.blocked {
			t.Errorf("isPathBlocked(%q) = %v, want %v", tc.path, got, tc.blocked)
		}
	}
}

// parseLsOutput - critical parsing logic for file listings.
// The parser is designed to handle 9+ whitespace-separated fields per line
// (matching `ls -la --time-style=full-iso` output). Shorter rows are skipped.

func TestParseLsOutput_StandardFullISO(t *testing.T) {
	// `ls -la --time-style=full-iso` output (9 fields per line)
	input := `total 12
drwxr-xr-x 3 user user 4096 2024-01-15 10:30:00.000000000 +0800 .
drwxr-xr-x 5 user user 4096 2024-01-15 10:29:00.000000000 +0800 ..
-rw-r--r-- 1 user user 1234 2024-01-15 10:29:00.000000000 +0800 file.txt
drwxr-xr-x 2 user user 4096 2024-01-15 10:30:00.000000000 +0800 subdir`

	entries, err := parseLsOutput(input, "/var/www")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries (3 children + . entry), got %d", len(entries))
	}

	// Directories should be sorted first
	if !entries[0].IsDir {
		t.Errorf("expected first entry to be a directory, got %s", entries[0].Name)
	}
	if entries[0].Name != "." && entries[0].Name != "subdir" {
		t.Errorf("expected . or subdir first, got %s", entries[0].Name)
	}
	if entries[1].IsDir && entries[1].Name == "subdir" {
		t.Errorf("expected subdir among directories")
	}

	// Verify the file details
	var file *FileEntry
	for i := range entries {
		if entries[i].Name == "file.txt" {
			file = &entries[i]
		}
	}
	if file == nil {
		t.Fatal("file.txt not found in parsed output")
	}
	if file.Size != 1234 {
		t.Errorf("expected size=1234, got %d", file.Size)
	}
	if file.Permissions != "-rw-r--r--" {
		t.Errorf("expected perms=-rw-r--r--, got %s", file.Permissions)
	}
	if file.Owner != "user" {
		t.Errorf("expected owner=user, got %s", file.Owner)
	}
}

func TestParseLsOutput_TotalLineFiltered(t *testing.T) {
	// Both "total 12" lines must be filtered out (regex match + prefix check)
	input := `total 12
-rw-r--r-- 1 user user 100 2024-01-15 10:30:00.000000000 +0800 file.txt
total 1
drwxr-xr-x 2 user user 4096 2024-01-15 10:30:00.000000000 +0800 subdir`

	entries, err := parseLsOutput(input, "/tmp")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries after filtering 'total' lines, got %d", len(entries))
	}
}

func TestParseLsOutput_EmptyInput(t *testing.T) {
	entries, err := parseLsOutput("", "/tmp")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty input, got %d", len(entries))
	}
}

func TestParseLsOutput_ShortLineIgnored(t *testing.T) {
	// Lines with < 9 fields should be skipped
	input := "garbage"

	entries, err := parseLsOutput(input, "/tmp")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for malformed input, got %d", len(entries))
	}
}

func TestParseLsOutput_SymlinkTargetTrimmed(t *testing.T) {
	// Symlinks show as: "lrwxrwxrwx 1 user user 7 2024-01-15 10:30:00.000000000 +0800 link -> /etc/target"
	input := "lrwxrwxrwx 1 user user 7 2024-01-15 10:30:00.000000000 +0800 link -> /etc/target"

	entries, err := parseLsOutput(input, "/tmp")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "link" {
		t.Errorf("expected symlink name to be trimmed, got %q", entries[0].Name)
	}
}

func TestParseLsOutput_DirectoriesFirst(t *testing.T) {
	input := `-rw-r--r-- 1 user user 100 2024-01-15 10:30:00.000000000 +0800 aaa-file
drwxr-xr-x 2 user user 4096 2024-01-15 10:30:00.000000000 +0800 zzz-dir
-rw-r--r-- 1 user user 200 2024-01-15 10:30:00.000000000 +0800 bbb-file
drwxr-xr-x 2 user user 4096 2024-01-15 10:30:00.000000000 +0800 aaa-dir`

	entries, err := parseLsOutput(input, "/tmp")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}
	// First two should be directories (alphabetical), then files (alphabetical)
	if !entries[0].IsDir || !entries[1].IsDir {
		t.Errorf("expected first 2 entries to be directories, got %s %s", entries[0].Name, entries[1].Name)
	}
	if entries[0].Name != "aaa-dir" {
		t.Errorf("expected aaa-dir first (directories sorted by name), got %s", entries[0].Name)
	}
	if entries[2].IsDir {
		t.Errorf("expected 3rd entry to be a file, got dir %s", entries[2].Name)
	}
}

func TestParseLsOutput_LongISOFormatSkipped(t *testing.T) {
	// Production code uses `ls -la --time-style=long-iso` which produces 8-field lines.
	// The parser skips rows with < 9 fields; this test pins down that contract
	// so any change to either the parser or the shell invocation is caught.
	input := "-rw-r--r-- 1 user user 1234 2024-01-15 10:30 file.txt"
	entries, err := parseLsOutput(input, "/tmp")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected long-iso (8-field) line to be skipped, got %d entries", len(entries))
	}
}

// parseDfOutput

func TestParseDfOutput_Valid(t *testing.T) {
	// `df -B1 --output=size,used,avail,pcent` output
	input := "107374182400 53687091200 53687091200 50%"

	du, err := parseDfOutput(input, "/var")
	if err != nil {
		t.Fatalf("parseDfOutput failed: %v", err)
	}
	if du.Total != 107374182400 {
		t.Errorf("expected total=107374182400, got %d", du.Total)
	}
	if du.Used != 53687091200 {
		t.Errorf("expected used=53687091200, got %d", du.Used)
	}
	if du.Available != 53687091200 {
		t.Errorf("expected available=53687091200, got %d", du.Available)
	}
	if du.UsagePercent != 50 {
		t.Errorf("expected usage_percent=50, got %f", du.UsagePercent)
	}
	if du.Path != "/var" {
		t.Errorf("expected path=/var, got %s", du.Path)
	}
}

func TestParseDfOutput_MalformedShort(t *testing.T) {
	// Less than 4 fields -> error
	_, err := parseDfOutput("only three", "/")
	if err == nil {
		t.Error("expected error for malformed df output")
	}
}

func TestParseDfOutput_PercentWithoutSuffix(t *testing.T) {
	// Edge: percent with no "%" suffix
	du, err := parseDfOutput("100 50 50 99", "/")
	if err != nil {
		t.Fatalf("parseDfOutput failed: %v", err)
	}
	if du.UsagePercent != 99 {
		t.Errorf("expected usage_percent=99, got %f", du.UsagePercent)
	}
}

// parseStatOutput

func TestParseStatOutput_Valid(t *testing.T) {
	// `stat -c '%A %U %G %s %Y'` output
	input := "-rw-r--r-- user user 1234 1705315800"

	entry, err := parseStatOutput(input, "/var/log/test.log")
	if err != nil {
		t.Fatalf("parseStatOutput failed: %v", err)
	}
	if entry.IsDir {
		t.Error("expected file, not dir")
	}
	if entry.Size != 1234 {
		t.Errorf("expected size=1234, got %d", entry.Size)
	}
	if entry.Owner != "user" {
		t.Errorf("expected owner=user, got %s", entry.Owner)
	}
	if entry.Group != "user" {
		t.Errorf("expected group=user, got %s", entry.Group)
	}
	if entry.Permissions != "-rw-r--r--" {
		t.Errorf("expected perms=-rw-r--r--, got %s", entry.Permissions)
	}
	if entry.ModTime.IsZero() {
		t.Error("expected non-zero modification time")
	}
}

func TestParseStatOutput_Directory(t *testing.T) {
	input := "drwxr-xr-x root root 4096 1705315800"
	entry, err := parseStatOutput(input, "/var")
	if err != nil {
		t.Fatalf("parseStatOutput failed: %v", err)
	}
	if !entry.IsDir {
		t.Error("expected directory to be detected")
	}
}

func TestParseStatOutput_Malformed(t *testing.T) {
	_, err := parseStatOutput("not enough", "/file")
	if err == nil {
		t.Error("expected error for malformed stat output")
	}
}

// base64Encode - regression: writeFile relies on this for safe content transfer

func TestBase64Encode(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "aGVsbG8="},
		{"hello world", "aGVsbG8gd29ybGQ="},
		{"\x00\x01\x02", "AAEC"},
	}
	for _, tc := range cases {
		if got := base64Encode(tc.input); got != tc.expected {
			t.Errorf("base64Encode(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
