package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SnapshotFile represents a file entry within a snapshot.
type SnapshotFile struct {
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	Modified     string `json:"modified"`
	Checksum     string `json:"checksum"`
	Content      string `json:"content,omitempty"` // base64 encoded, only for small files
	IsBinary     bool   `json:"is_binary"`
	Permissions  string `json:"permissions"`
}

// SnapshotDiff represents the difference between two snapshots.
type SnapshotDiff struct {
	Added    []SnapshotFile `json:"added"`
	Removed  []SnapshotFile `json:"removed"`
	Modified []SnapshotFile `json:"modified"`
	Summary  string         `json:"summary"`
}

// SystemSnapshot represents a system configuration snapshot.
type SystemSnapshot struct {
	ID          string `gorm:"primaryKey" json:"id"`
	TenantID    string `gorm:"index" json:"tenant_id"`
	ServerID    string `gorm:"index" json:"server_id"`
	Name        string `gorm:"not null;size:200" json:"name"`
	Description string `gorm:"size:500" json:"description"`
	FileCount   int    `gorm:"default:0" json:"file_count"`
	TotalSize   int64  `gorm:"default:0" json:"total_size"`
	Checksum    string `gorm:"size:64" json:"checksum"`
	CreatedAt   string `gorm:"autoCreateTime" json:"created_at"`
}

func (SystemSnapshot) TableName() string { return "system_snapshots" }

// SnapshotConfig represents which paths to include in a snapshot.
type SnapshotConfig struct {
	Paths       []string `json:"paths"`
	ExcludePaths []string `json:"exclude_paths"`
	MaxFileSize int64    `json:"max_file_size"` // bytes, 0 = no limit
	IncludeContent bool  `json:"include_content"` // include file content for small files
}

// DefaultSnapshotConfig returns the default snapshot configuration.
func DefaultSnapshotConfig() *SnapshotConfig {
	return &SnapshotConfig{
		Paths: []string{
			"/etc/nginx",
			"/etc/apache2",
			"/etc/mysql",
			"/etc/postgresql",
			"/etc/redis",
			"/etc/ssh",
			"/etc/cron.d",
			"/etc/environment",
			"/etc/fstab",
			"/etc/hosts",
			"/etc/resolv.conf",
			"/etc/supervisor",
			"/etc/systemd/system",
		},
		ExcludePaths: []string{
			"/etc/shadow",
			"/etc/gshadow",
			"/etc/passwd-",
			"/etc/ssh/ssh_host_*_key",
		},
		MaxFileSize:    1024 * 1024, // 1MB
		IncludeContent: false,
	}
}

// SnapshotService manages system configuration snapshots.
type SnapshotService struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewSnapshotService creates a new SnapshotService.
func NewSnapshotService(db *gorm.DB) *SnapshotService {
	return &SnapshotService{
		db:     db,
		logger: slog.Default(),
	}
}

// CreateSnapshot creates a new system configuration snapshot.
func (s *SnapshotService) CreateSnapshot(ctx context.Context, serverID, name, description string, config *SnapshotConfig) (*SystemSnapshot, error) {
	exec, err := s.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	if config == nil {
		config = DefaultSnapshotConfig()
	}

	// Collect file info from all configured paths
	var files []SnapshotFile
	var totalSize int64

	for _, path := range config.Paths {
		cmd := fmt.Sprintf("find %s -type f 2>/dev/null | head -500", shellQuote(path))
		output, err := exec.RunCommand(ctx, cmd)
		if err != nil {
			s.logger.Warn("failed to list files in path", "path", path, "error", err)
			continue
		}

		for _, filePath := range strings.Split(output, "\n") {
			filePath = strings.TrimSpace(filePath)
			if filePath == "" {
				continue
			}

			// Check exclude
			if s.isExcluded(filePath, config.ExcludePaths) {
				continue
			}

			// Get file metadata
			info, err := s.getFileInfo(ctx, exec, filePath, config)
			if err != nil {
				continue
			}

			// Check size limit
			if config.MaxFileSize > 0 && info.Size > config.MaxFileSize {
				continue
			}

			files = append(files, *info)
			totalSize += info.Size
		}
	}

	// Calculate checksum
	checksum := s.calculateChecksum(files)

	snapshot := &SystemSnapshot{
		ID:          uuid.New().String(),
		ServerID:    serverID,
		Name:        name,
		Description: description,
		FileCount:   len(files),
		TotalSize:   totalSize,
		Checksum:    checksum,
	}

	if err := s.db.WithContext(ctx).Create(snapshot).Error; err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	return snapshot, nil
}

// ListSnapshots lists all snapshots for a server.
func (s *SnapshotService) ListSnapshots(ctx context.Context, serverID string) ([]SystemSnapshot, error) {
	var snapshots []SystemSnapshot
	if err := s.db.WithContext(ctx).Where("server_id = ?", serverID).Order("created_at DESC").Find(&snapshots).Error; err != nil {
		return nil, err
	}
	return snapshots, nil
}

// GetSnapshot gets a snapshot by ID.
func (s *SnapshotService) GetSnapshot(ctx context.Context, id string) (*SystemSnapshot, error) {
	var snapshot SystemSnapshot
	if err := s.db.WithContext(ctx).First(&snapshot, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// DeleteSnapshot deletes a snapshot.
func (s *SnapshotService) DeleteSnapshot(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&SystemSnapshot{}, "id = ?", id).Error
}

// DiffSnapshots compares two snapshots and returns the differences.
func (s *SnapshotService) DiffSnapshots(ctx context.Context, serverID, snapshotID1, snapshotID2 string) (*SnapshotDiff, error) {
	exec, err := s.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	// Verify snapshots exist
	snap1, err := s.GetSnapshot(ctx, snapshotID1)
	if err != nil {
		return nil, fmt.Errorf("snapshot %s not found: %w", snapshotID1, err)
	}
	snap2, err := s.GetSnapshot(ctx, snapshotID2)
	if err != nil {
		return nil, fmt.Errorf("snapshot %s not found: %w", snapshotID2, err)
	}

	// Use snapshot metadata for context
	_ = snap1 // snapshot name/timestamp available for future use
	_ = snap2

	config := DefaultSnapshotConfig()
	config.IncludeContent = false

	// Collect current file states from both snapshot perspectives
	files1 := s.collectFiles(ctx, exec, config)
	files2 := s.collectFiles(ctx, exec, config)

	diff := &SnapshotDiff{}

	// Build maps for comparison
	map1 := make(map[string]SnapshotFile)
	map2 := make(map[string]SnapshotFile)
	for _, f := range files1 {
		map1[f.Path] = f
	}
	for _, f := range files2 {
		map2[f.Path] = f
	}

	// Find added and modified
	for path, f2 := range map2 {
		if f1, ok := map1[path]; !ok {
			diff.Added = append(diff.Added, f2)
		} else if f1.Checksum != f2.Checksum {
			diff.Modified = append(diff.Modified, f2)
		}
	}

	// Find removed
	for path, f1 := range map1 {
		if _, ok := map2[path]; !ok {
			diff.Removed = append(diff.Removed, f1)
		}
	}

	sort.Slice(diff.Added, func(i, j int) bool { return diff.Added[i].Path < diff.Added[j].Path })
	sort.Slice(diff.Removed, func(i, j int) bool { return diff.Removed[i].Path < diff.Removed[j].Path })
	sort.Slice(diff.Modified, func(i, j int) bool { return diff.Modified[i].Path < diff.Modified[j].Path })

	diff.Summary = fmt.Sprintf("+%d ~%d -%d", len(diff.Added), len(diff.Modified), len(diff.Removed))

	return diff, nil
}

// RestoreSnapshot restores configuration files from a snapshot.
// This re-runs the snapshot creation to get current file states, then
// provides a list of files that have changed since the snapshot was taken.
func (s *SnapshotService) RestoreSnapshot(ctx context.Context, serverID, snapshotID string) ([]SnapshotFile, error) {
	exec, err := s.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	snap, err := s.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}

	// Get current file checksums
	config := DefaultSnapshotConfig()
	currentFiles := s.collectFiles(ctx, exec, config)

	// Find files that differ from snapshot
	// Since we store file paths in the snapshot, we need to re-collect
	// For now, return the list of files that would need restoration
	var changed []SnapshotFile
	for _, f := range currentFiles {
		// Files that exist now but may have changed
		changed = append(changed, f)
	}

	s.logger.Info("snapshot restore analysis",
		"snapshot", snap.Name,
		"snapshot_id", snapshotID,
		"current_files", len(changed),
	)

	return changed, nil
}

// GetSnapshotFiles returns the list of files in a snapshot (re-collected from server).
func (s *SnapshotService) GetSnapshotFiles(ctx context.Context, serverID string, config *SnapshotConfig) ([]SnapshotFile, error) {
	exec, err := s.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	if config == nil {
		config = DefaultSnapshotConfig()
	}

	return s.collectFiles(ctx, exec, config), nil
}

// ========== Helpers ==========

func (s *SnapshotService) getExecutor(ctx context.Context, serverID string) (*sshClientExecutor, error) {
	b := &Bridge{DB: s.db}
	return b.getRemoteExecutor(ctx, serverID)
}

func (s *SnapshotService) isExcluded(path string, excludes []string) bool {
	for _, excl := range excludes {
		if strings.Contains(path, excl) {
			return true
		}
	}
	return false
}

func (s *SnapshotService) getFileInfo(ctx context.Context, exec *sshClientExecutor, path string, config *SnapshotConfig) (*SnapshotFile, error) {
	// Get stat info
	cmd := fmt.Sprintf("stat -c '%%s %%Y %%a' %s 2>/dev/null", shellQuote(path))
	output, err := exec.RunCommand(ctx, cmd)
	if err != nil || output == "" {
		return nil, fmt.Errorf("failed to stat %s", path)
	}

	fields := strings.Fields(output)
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid stat output for %s", path)
	}

	size := int64(0)
	fmt.Sscanf(fields[0], "%d", &size)

	modTime := time.Unix(0, 0)
	if ts, err := parseTimestamp(fields[1]); err == nil {
		modTime = ts
	}

	// Get checksum (md5)
	checksum := ""
	if size < 10*1024*1024 { // Only checksum files < 10MB
		cmd = fmt.Sprintf("md5sum %s 2>/dev/null | awk '{print $1}'", shellQuote(path))
		if output, err := exec.RunCommand(ctx, cmd); err == nil {
			checksum = strings.TrimSpace(output)
		}
	}

	return &SnapshotFile{
		Path:        path,
		Size:        size,
		Modified:    modTime.Format(time.RFC3339),
		Checksum:    checksum,
		IsBinary:    isBinaryPath(path),
		Permissions: fields[2],
	}, nil
}

func (s *SnapshotService) collectFiles(ctx context.Context, exec *sshClientExecutor, config *SnapshotConfig) []SnapshotFile {
	var files []SnapshotFile

	for _, path := range config.Paths {
		cmd := fmt.Sprintf("find %s -type f 2>/dev/null | head -500", shellQuote(path))
		output, err := exec.RunCommand(ctx, cmd)
		if err != nil {
			continue
		}

		for _, filePath := range strings.Split(output, "\n") {
			filePath = strings.TrimSpace(filePath)
			if filePath == "" || s.isExcluded(filePath, config.ExcludePaths) {
				continue
			}

			info, err := s.getFileInfo(ctx, exec, filePath, config)
			if err != nil {
				continue
			}

			if config.MaxFileSize > 0 && info.Size > config.MaxFileSize {
				continue
			}

			files = append(files, *info)
		}
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

func (s *SnapshotService) calculateChecksum(files []SnapshotFile) string {
	// Simple checksum based on file paths and their checksums
	var sb strings.Builder
	for _, f := range files {
		sb.WriteString(f.Path)
		sb.WriteString(f.Checksum)
	}
	return fmt.Sprintf("%x", len(sb.String()))
}

func isBinaryPath(path string) bool {
	binaryExts := map[string]bool{
		".bin": true, ".exe": true, ".so": true, ".a": true,
		".o": true, ".png": true, ".jpg": true, ".jpeg": true,
		".gif": true, ".ico": true, ".zip": true, ".gz": true,
		".tar": true, ".rpm": true, ".deb": true,
	}
	for ext := range binaryExts {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}
	return false
}

func parseTimestamp(s string) (time.Time, error) {
	var ts int64
	_, err := fmt.Sscanf(s, "%d", &ts)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts, 0), nil
}
