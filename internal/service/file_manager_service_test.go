package service

import (
	"testing"
)

func TestIsPathBlocked_DirectBlockedPaths(t *testing.T) {
	blocked := []string{
		"/proc",
		"/sys",
		"/dev",
		"/boot",
		"/etc",
		"/root",
		"/var/log",
		"/bin",
		"/sbin",
		"/usr",
	}

	for _, path := range blocked {
		if !isPathBlocked(path) {
			t.Errorf("isPathBlocked(%q) = false, want true", path)
		}
	}
}

func TestIsPathBlocked_BlockedPrefixes(t *testing.T) {
	blockedPrefixes := []string{
		"/proc/self",
		"/sys/kernel",
		"/dev/null",
		"/boot/grub",
		"/etc/passwd",
		"/root/.ssh",
		"/var/log/syslog",
		"/bin/bash",
		"/sbin/init",
		"/usr/bin",
		"/usr/local/bin",
	}

	for _, path := range blockedPrefixes {
		if !isPathBlocked(path) {
			t.Errorf("isPathBlocked(%q) = false, want true", path)
		}
	}
}

func TestIsPathBlocked_AllowedPaths(t *testing.T) {
	allowed := []string{
		"/",
		"/app",
		"/data",
		"/opt",
		"/tmp",
		"/home",
		"/var",
		"/srv",
		"/app/data",
		"/data/app",
		"/opt/services",
		"/tmp/deploy",
	}

	for _, path := range allowed {
		if isPathBlocked(path) {
			t.Errorf("isPathBlocked(%q) = true, want false", path)
		}
	}
}

func TestIsPathBlocked_NonAbsolutePaths(t *testing.T) {
	nonAbsolute := []string{
		"proc",
		"etc/passwd",
		"./app",
		"../etc",
		"tmp/file",
	}

	for _, path := range nonAbsolute {
		if isPathBlocked(path) {
			t.Errorf("isPathBlocked(%q) = true, want false (non-absolute paths should not be blocked)", path)
		}
	}
}

func TestIsPathBlocked_CleanPathNormalization(t *testing.T) {
	// Paths with extra slashes or dots should be normalized
	tests := []struct {
		path     string
		expected bool
	}{
		{"/app//data", false},
		{"/app/./data", false},
		{"/app/../data", false},
		{"/app/../../etc", true},
		{"/app/../../root", true},
		{"/etc/./passwd", true},
		{"/proc/../proc/sys", true},
	}

	for _, tt := range tests {
		result := isPathBlocked(tt.path)
		if result != tt.expected {
			t.Errorf("isPathBlocked(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

func TestIsPathBlocked_EmptyPath(t *testing.T) {
	// Empty path should not be blocked (will fail later in actual operations)
	if isPathBlocked("") {
		t.Error("isPathBlocked(\"\") = true, want false")
	}
}

func TestParseLsOutput_BasicParsing(t *testing.T) {
	// Use standard ls -la format with just filenames (as when ls is run on a directory)
	output := `total 4
-rw-r--r-- 1 owner group 123 Jan 15 14:30 file.txt`

	entries, err := parseLsOutput(output, "/data")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Name != "file.txt" {
		t.Errorf("expected Name='file.txt', got %q", entry.Name)
	}
	if entry.Path != "/data/file.txt" {
		t.Errorf("expected Path='/data/file.txt', got %q", entry.Path)
	}
	if entry.IsDir {
		t.Error("expected IsDir=false")
	}
	if entry.Permissions != "-rw-r--r--" {
		t.Errorf("expected Permissions='-rw-r--r--', got %q", entry.Permissions)
	}
	if entry.Owner != "owner" {
		t.Errorf("expected Owner='owner', got %q", entry.Owner)
	}
	if entry.Group != "group" {
		t.Errorf("expected Group='group', got %q", entry.Group)
	}
	if entry.Size != 123 {
		t.Errorf("expected Size=123, got %d", entry.Size)
	}
}

func TestParseLsOutput_DirectoryEntry(t *testing.T) {
	output := `total 4
drwxr-xr-x 2 owner group 4096 Jan 15 14:30 mydir`

	entries, err := parseLsOutput(output, "/app")

	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if !entry.IsDir {
		t.Error("expected IsDir=true")
	}
	if entry.Name != "mydir" {
		t.Errorf("expected Name='mydir', got %q", entry.Name)
	}
	if entry.Path != "/app/mydir" {
		t.Errorf("expected Path='/app/mydir', got %q", entry.Path)
	}
}

func TestParseLsOutput_EmptyOutput(t *testing.T) {
	entries, err := parseLsOutput("", "/app")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseLsOutput_OnlyTotalLine(t *testing.T) {
	output := "total 128"
	entries, err := parseLsOutput(output, "/app")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseLsOutput_SortingDirectoriesFirst(t *testing.T) {
	output := `total 8
-rw-r--r-- 1 owner group 100 Jan 15 14:30 zebra.txt
drwxr-xr-x 2 owner group 4096 Jan 15 14:30 alpha
drwxr-xr-x 2 owner group 4096 Jan 15 14:30 beta
-rw-r--r-- 1 owner group 200 Jan 15 14:30 gamma.txt`

	entries, err := parseLsOutput(output, "/app")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}

	if len(entries) != 4 {
		t.Errorf("expected 4 entries, got %d", len(entries))
	}

	// Directories should come first, sorted by name, then files sorted by name
	if !entries[0].IsDir || entries[0].Name != "alpha" {
		t.Errorf("first entry should be alpha dir, got %+v", entries[0])
	}
	if !entries[1].IsDir || entries[1].Name != "beta" {
		t.Errorf("second entry should be beta dir, got %+v", entries[1])
	}
	if entries[2].IsDir || entries[2].Name != "gamma.txt" {
		t.Errorf("third entry should be gamma.txt file, got %+v", entries[2])
	}
	if entries[3].IsDir || entries[3].Name != "zebra.txt" {
		t.Errorf("fourth entry should be zebra.txt file, got %+v", entries[3])
	}
}

func TestParseLsOutput_SymlinkWithArrow(t *testing.T) {
	output := `total 4
lrwxrwxrwx 1 owner group 12 Jan 15 14:30 link.txt -> /data/file.txt`

	entries, err := parseLsOutput(output, "/app")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	// Name should not include "->"
	if entries[0].Name != "link.txt" {
		t.Errorf("expected name 'link.txt', got %q", entries[0].Name)
	}

	// Path should be the full path
	if entries[0].Path != "/app/link.txt" {
		t.Errorf("expected path '/app/link.txt', got %q", entries[0].Path)
	}
}

func TestParseLsOutput_PermissionsAndOwnership(t *testing.T) {
	output := `total 4
-rwxr-x--- 1 www-data www-data 123 Jan 15 14:30 script.sh`

	entries, err := parseLsOutput(output, "/app")
	if err != nil {
		t.Fatalf("parseLsOutput failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Permissions != "-rwxr-x---" {
		t.Errorf("expected Permissions='-rwxr-x---', got %q", entry.Permissions)
	}
	if entry.Owner != "www-data" {
		t.Errorf("expected Owner='www-data', got %q", entry.Owner)
	}
	if entry.Group != "www-data" {
		t.Errorf("expected Group='www-data', got %q", entry.Group)
	}
	if entry.Size != 123 {
		t.Errorf("expected Size=123, got %d", entry.Size)
	}
}

func TestParseDfOutput_ValidOutput(t *testing.T) {
	output := "500000000 250000000 250000000 50%"

	usage, err := parseDfOutput(output, "/")
	if err != nil {
		t.Fatalf("parseDfOutput failed: %v", err)
	}

	if usage.Total != 500000000 {
		t.Errorf("expected Total=500000000, got %d", usage.Total)
	}
	if usage.Used != 250000000 {
		t.Errorf("expected Used=250000000, got %d", usage.Used)
	}
	if usage.Available != 250000000 {
		t.Errorf("expected Available=250000000, got %d", usage.Available)
	}
	if usage.UsagePercent != 50.0 {
		t.Errorf("expected UsagePercent=50.0, got %f", usage.UsagePercent)
	}
	if usage.Path != "/" {
		t.Errorf("expected Path='/', got %q", usage.Path)
	}
}

func TestParseDfOutput_InvalidOutput(t *testing.T) {
	tests := []string{
		"invalid",
		"only two fields",
		"",
		"1 2 3", // missing percent
	}

	for _, output := range tests {
		_, err := parseDfOutput(output, "/")
		if err == nil {
			t.Errorf("parseDfOutput(%q) should fail, got nil", output)
		}
	}
}

func TestParseDfOutput_ZeroAvailable(t *testing.T) {
	output := "1000000 1000000 0 100%"

	usage, err := parseDfOutput(output, "/data")
	if err != nil {
		t.Fatalf("parseDfOutput failed: %v", err)
	}

	if usage.Available != 0 {
		t.Errorf("expected Available=0, got %d", usage.Available)
	}
	if usage.UsagePercent != 100.0 {
		t.Errorf("expected UsagePercent=100.0, got %f", usage.UsagePercent)
	}
}

func TestParseStatOutput_ValidOutput(t *testing.T) {
	output := "-rw-r--r-- www-data www-data 4096 1620000000"

	entry, err := parseStatOutput(output, "/app/config.yaml")
	if err != nil {
		t.Fatalf("parseStatOutput failed: %v", err)
	}

	if entry.Permissions != "-rw-r--r--" {
		t.Errorf("expected Permissions='-rw-r--r--', got %q", entry.Permissions)
	}
	if entry.Owner != "www-data" {
		t.Errorf("expected Owner='www-data', got %q", entry.Owner)
	}
	if entry.Group != "www-data" {
		t.Errorf("expected Group='www-data', got %q", entry.Group)
	}
	if entry.Size != 4096 {
		t.Errorf("expected Size=4096, got %d", entry.Size)
	}
	if entry.ModTime.Unix() != 1620000000 {
		t.Errorf("expected ModTime.Unix()=1620000000, got %d", entry.ModTime.Unix())
	}
	if entry.IsDir {
		t.Error("expected IsDir=false for regular file")
	}
}

func TestParseStatOutput_Directory(t *testing.T) {
	output := "drwxr-xr-x root root 4096 1620000000"

	entry, err := parseStatOutput(output, "/app")

	if err != nil {
		t.Fatalf("parseStatOutput failed: %v", err)
	}
	if !entry.IsDir {
		t.Error("expected IsDir=true")
	}
}

func TestParseStatOutput_InvalidOutput(t *testing.T) {
	tests := []string{
		"invalid",
		"only four fields",
		"",
		"a b", // missing fields
	}

	for _, output := range tests {
		_, err := parseStatOutput(output, "/app")
		if err == nil {
			t.Errorf("parseStatOutput(%q) should fail, got nil", output)
		}
	}
}

func TestBase64Encode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "aGVsbG8="},
		{"", ""},
		{"test data", "dGVzdCBkYXRh"},
	}

	for _, tt := range tests {
		result := base64Encode(tt.input)
		if result != tt.expected {
			t.Errorf("base64Encode(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestEncodeToString(t *testing.T) {
	data := []byte("hello")
	result := encodeToString(data)
	if result != "aGVsbG8=" {
		t.Errorf("encodeToString() = %q, want %q", result, "aGVsbG8=")
	}
}
