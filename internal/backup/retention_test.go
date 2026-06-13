package backup

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// fakeBackupFile writes a tiny placeholder file with the given name and
// mtime into dir, then returns its full path. The content is unimportant
// for retention — the policy only looks at filename and ModTime.
func fakeBackupFile(t *testing.T, dir, name string, mtime time.Time) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
	return path
}

// listBackups returns backup filenames currently in dir, sorted.
func listBackups(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// newRetentionService constructs a Service that only has the
// configuration fields ApplyRetention reads. All other fields are left
// zero-valued because ApplyRetention does not touch them.
func newRetentionService(cfg Config) *Service {
	return &Service{config: cfg}
}

// TestApplyRetention_DisabledWhenNoThresholds covers the early-return
// branch: with both RetentionCount and RetentionDays <= 0, no files
// should be touched even if the directory is full.
func TestApplyRetention_DisabledWhenNoThresholds(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	fakeBackupFile(t, dir, "db-20240101.bak", now.Add(-48*time.Hour))
	fakeBackupFile(t, dir, "db-20240102.bak", now.Add(-24*time.Hour))

	svc := newRetentionService(Config{BackupDir: dir, RetentionCount: 0, RetentionDays: 0})
	svc.ApplyRetention(context.Background())

	if got := listBackups(t, dir); len(got) != 2 {
		t.Errorf("expected 2 files untouched, got %v", got)
	}
}

// TestApplyRetention_ByCountOnly verifies that the count-based policy
// prunes oldest files first, keeping exactly RetentionCount of them.
func TestApplyRetention_ByCountOnly(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// Create 5 files, all "fresh" so days-based retention never triggers.
	// i=0 is the newest, i=4 is the oldest.
	for i := 0; i < 5; i++ {
		fakeBackupFile(t, dir,
			"db-2024010"+string(rune('0'+i))+".bak",
			now.Add(-time.Duration(i)*time.Hour))
	}

	svc := newRetentionService(Config{BackupDir: dir, RetentionCount: 2, RetentionDays: 0})
	svc.ApplyRetention(context.Background())

	got := listBackups(t, dir)
	if len(got) != 2 {
		t.Fatalf("expected 2 files kept, got %d (%v)", len(got), got)
	}
	// Newest two should remain: db-20240100.bak and db-20240101.bak.
	want := []string{"db-20240100.bak", "db-20240101.bak"}
	sort.Strings(got)
	for i, w := range want {
		if got[i] != w {
			t.Errorf("kept file[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// TestApplyRetention_ByDaysOnly verifies the age-based policy removes
// any file older than RetentionDays.
func TestApplyRetention_ByDaysOnly(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	fakeBackupFile(t, dir, "db-old.bak", now.AddDate(0, 0, -10))   // 10 days old
	fakeBackupFile(t, dir, "db-fresh.bak", now.AddDate(0, 0, -1))  // 1 day old
	fakeBackupFile(t, dir, "db-borderline.bak", now.AddDate(0, 0, -3)) // 3 days old

	svc := newRetentionService(Config{BackupDir: dir, RetentionCount: 0, RetentionDays: 5})
	svc.ApplyRetention(context.Background())

	got := listBackups(t, dir)
	if len(got) != 2 {
		t.Fatalf("expected 2 files kept, got %d (%v)", len(got), got)
	}
	// Only the 10-day-old file should be removed.
	for _, name := range got {
		if name == "db-old.bak" {
			t.Errorf("db-old.bak should have been removed (10 days > 5)")
		}
	}
}

// TestApplyRetention_ByBothPolicies applies both policies together: the
// count policy should never keep more than RetentionCount, and the days
// policy should additionally drop any file older than RetentionDays.
func TestApplyRetention_ByBothPolicies(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// 3 fresh files, 2 stale ones.
	for i := 0; i < 3; i++ {
		fakeBackupFile(t, dir,
			"db-fresh-"+string(rune('a'+i))+".bak",
			now.Add(-time.Duration(i+1)*time.Hour))
	}
	fakeBackupFile(t, dir, "db-stale-1.bak", now.AddDate(0, 0, -10))
	fakeBackupFile(t, dir, "db-stale-2.bak", now.AddDate(0, 0, -20))

	svc := newRetentionService(Config{BackupDir: dir, RetentionCount: 5, RetentionDays: 5})
	svc.ApplyRetention(context.Background())

	got := listBackups(t, dir)
	// Expect exactly 3 fresh files: both stale ones removed by days policy,
	// count policy permits up to 5.
	if len(got) != 3 {
		t.Fatalf("expected 3 files kept, got %d (%v)", len(got), got)
	}
	for _, name := range got {
		if len(name) >= 9 && name[:9] == "db-stale-" {
			t.Errorf("stale file %q should have been removed", name)
		}
	}
}

// TestApplyRetention_IgnoresUnrelatedFiles ensures the file-name
// filter is correctly applied: files outside the db-*.bak / *.meta.json
// patterns must be left alone, even if they are very old.
func TestApplyRetention_IgnoresUnrelatedFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	fakeBackupFile(t, dir, "db-keep.bak", now.Add(-time.Hour))
	fakeBackupFile(t, dir, "db-stale.bak", now.AddDate(0, 0, -30))
	fakeBackupFile(t, dir, "unrelated.log", now.AddDate(0, 0, -365))
	fakeBackupFile(t, dir, "notes.txt", now.AddDate(0, 0, -365))
	fakeBackupFile(t, dir, "db-stale.meta.json", now.AddDate(0, 0, -30))

	svc := newRetentionService(Config{BackupDir: dir, RetentionCount: 0, RetentionDays: 7})
	svc.ApplyRetention(context.Background())

	got := listBackups(t, dir)
	sort.Strings(got)
	want := []string{"db-keep.bak", "notes.txt", "unrelated.log"}
	if len(got) != len(want) {
		t.Fatalf("expected %d files kept, got %d (%v)", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("kept file[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// TestApplyRetention_EmptyDirectory is a smoke test: calling the
// function against a directory with no backup files must not error
// and must not panic.
func TestApplyRetention_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	svc := newRetentionService(Config{BackupDir: dir, RetentionCount: 3, RetentionDays: 7})
	svc.ApplyRetention(context.Background())

	if got := listBackups(t, dir); len(got) != 0 {
		t.Errorf("expected empty directory, got %v", got)
	}
}

// TestApplyRetention_MissingDirectory makes sure the function logs and
// returns cleanly when the configured directory does not exist, instead
// of panicking.
func TestApplyRetention_MissingDirectory(t *testing.T) {
	svc := newRetentionService(Config{
		BackupDir:      filepath.Join(t.TempDir(), "does-not-exist"),
		RetentionCount: 1,
		RetentionDays:  1,
	})
	// Should not panic.
	svc.ApplyRetention(context.Background())
}

// TestApplyRetention_MetaJsonRetained confirms that .meta.json sidecar
// files are subject to the same retention policy as the .bak files.
func TestApplyRetention_MetaJsonRetained(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// One .bak (fresh) and one .meta.json (stale).
	fakeBackupFile(t, dir, "db-only.bak", now.Add(-time.Hour))
	fakeBackupFile(t, dir, "db-old.meta.json", now.AddDate(0, 0, -30))

	svc := newRetentionService(Config{BackupDir: dir, RetentionCount: 0, RetentionDays: 7})
	svc.ApplyRetention(context.Background())

	got := listBackups(t, dir)
	if len(got) != 1 {
		t.Fatalf("expected 1 file kept, got %d (%v)", len(got), got)
	}
	if got[0] != "db-only.bak" {
		t.Errorf("kept file = %q, want db-only.bak", got[0])
	}
}
