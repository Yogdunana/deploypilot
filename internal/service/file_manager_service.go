package service

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/sandbox"
	"github.com/Yogdunana/deploypilot/internal/util"
	"gorm.io/gorm"
)

// FileManagerService provides remote file management capabilities via SSH.
// All operations go through the sandbox for safety validation.
type FileManagerService struct {
	db      *gorm.DB
	sb      *sandbox.Sandbox
	logger  *slog.Logger
}

// NewFileManagerService creates a new FileManagerService.
func NewFileManagerService(db *gorm.DB, sb *sandbox.Sandbox) *FileManagerService {
	return &FileManagerService{
		db:     db,
		sb:     sb,
		logger: slog.Default(),
	}
}

// FileEntry represents a file or directory entry.
type FileEntry struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsDir        bool      `json:"is_dir"`
	Size         int64     `json:"size"`
	Permissions  string    `json:"permissions"`
	ModTime      time.Time `json:"mod_time"`
	Owner        string    `json:"owner,omitempty"`
	Group        string    `json:"group,omitempty"`
}

// FileContent represents file content for read/write operations.
type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

// DiskUsage represents disk usage information.
type DiskUsage struct {
	Path      string `json:"path"`
	Total     int64  `json:"total_bytes"`
	Used      int64  `json:"used_bytes"`
	Available int64  `json:"available_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

// Blocked paths that should never be accessible.
var blockedPaths = []string{
	"/proc", "/sys", "/dev", "/boot",
}

// blockedPathRegex matches paths that should be blocked.
var blockedPathRegex = regexp.MustCompile(`^/(proc|sys|dev|boot)(/|$)`)

// isPathBlocked checks if a path is in the blocked list.
func isPathBlocked(path string) bool {
	cleaned := filepath.Clean(path)
	for _, bp := range blockedPaths {
		if cleaned == bp || strings.HasPrefix(cleaned, bp+"/") {
			return true
		}
	}
	return blockedPathRegex.MatchString(cleaned)
}

// ListFiles lists files and directories at the given remote path.
func (f *FileManagerService) ListFiles(ctx context.Context, serverID, remotePath string) ([]FileEntry, error) {
	if isPathBlocked(remotePath) {
		return nil, fmt.Errorf("access denied: path %s is in a restricted system directory", remotePath)
	}

	cmd := fmt.Sprintf("ls -la --time-style=long-iso %s 2>/dev/null || ls -la %s 2>/dev/null", util.ShellQuote(remotePath), util.ShellQuote(remotePath))
	if err := f.sb.Validate(cmd); err != nil {
		return nil, fmt.Errorf("sandbox blocked: %w", err)
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	output, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return parseLsOutput(output, remotePath)
}

// ReadFile reads the content of a remote file.
func (f *FileManagerService) ReadFile(ctx context.Context, serverID, remotePath string, maxBytes int64) (*FileContent, error) {
	if isPathBlocked(remotePath) {
		return nil, fmt.Errorf("access denied: path %s is in a restricted system directory", remotePath)
	}

	limit := ""
	if maxBytes > 0 {
		limit = fmt.Sprintf(" | head -c %d", maxBytes)
	}

	cmd := fmt.Sprintf("cat %s%s", util.ShellQuote(remotePath), limit)
	if err := f.sb.Validate(cmd); err != nil {
		return nil, fmt.Errorf("sandbox blocked: %w", err)
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	content, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Get file size
	sizeCmd := fmt.Sprintf("stat -c %%s %s 2>/dev/null || wc -c < %s", util.ShellQuote(remotePath), util.ShellQuote(remotePath))
	sizeOutput, _ := exec.RunCommand(ctx, sizeCmd)
	size, _ := strconv.ParseInt(strings.TrimSpace(sizeOutput), 10, 64)

	return &FileContent{
		Path:    remotePath,
		Content: content,
		Size:    size,
	}, nil
}

// WriteFile writes content to a remote file.
func (f *FileManagerService) WriteFile(ctx context.Context, serverID, remotePath, content string) error {
	if isPathBlocked(remotePath) {
		return fmt.Errorf("access denied: path %s is in a restricted system directory", remotePath)
	}

	// Use base64 encoding to safely transfer content via SSH
	encoded := base64Encode(content)
	cmd := fmt.Sprintf("echo '%s' | base64 -d > %s", encoded, util.ShellQuote(remotePath))
	if err := f.sb.Validate(cmd); err != nil {
		return fmt.Errorf("sandbox blocked: %w", err)
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	_, err = exec.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// DeleteFile deletes a remote file or directory.
func (f *FileManagerService) DeleteFile(ctx context.Context, serverID, remotePath string) error {
	if isPathBlocked(remotePath) {
		return fmt.Errorf("access denied: path %s is in a restricted system directory", remotePath)
	}

	cmd := fmt.Sprintf("rm -rf %s", util.ShellQuote(remotePath))
	if err := f.sb.Validate(cmd); err != nil {
		return fmt.Errorf("sandbox blocked: %w", err)
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	_, err = exec.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

// CreateDirectory creates a remote directory.
func (f *FileManagerService) CreateDirectory(ctx context.Context, serverID, remotePath string) error {
	if isPathBlocked(remotePath) {
		return fmt.Errorf("access denied: path %s is in a restricted system directory", remotePath)
	}

	cmd := fmt.Sprintf("mkdir -p %s", util.ShellQuote(remotePath))
	if err := f.sb.Validate(cmd); err != nil {
		return fmt.Errorf("sandbox blocked: %w", err)
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	_, err = exec.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

// MoveFile moves/renames a remote file.
func (f *FileManagerService) MoveFile(ctx context.Context, serverID, srcPath, dstPath string) error {
	if isPathBlocked(srcPath) || isPathBlocked(dstPath) {
		return fmt.Errorf("access denied: path is in a restricted system directory")
	}

	cmd := fmt.Sprintf("mv %s %s", util.ShellQuote(srcPath), util.ShellQuote(dstPath))
	if err := f.sb.Validate(cmd); err != nil {
		return fmt.Errorf("sandbox blocked: %w", err)
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = exec.Close() }()

	_, err = exec.RunCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}
	return nil
}

// GetDiskUsage returns disk usage information for a remote path.
func (f *FileManagerService) GetDiskUsage(ctx context.Context, serverID, remotePath string) (*DiskUsage, error) {
	cmd := fmt.Sprintf("df -B1 --output=size,used,avail,pcent %s 2>/dev/null | tail -1", util.ShellQuote(remotePath))
	if err := f.sb.Validate(cmd); err != nil {
		return nil, fmt.Errorf("sandbox blocked: %w", err)
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	output, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	return parseDfOutput(output, remotePath)
}

// GetFileInfo returns detailed information about a single remote file.
func (f *FileManagerService) GetFileInfo(ctx context.Context, serverID, remotePath string) (*FileEntry, error) {
	if isPathBlocked(remotePath) {
		return nil, fmt.Errorf("access denied: path %s is in a restricted system directory", remotePath)
	}

	cmd := fmt.Sprintf("stat -c '%%A %%U %%G %%s %%Y' %s 2>/dev/null", util.ShellQuote(remotePath))
	if err := f.sb.Validate(cmd); err != nil {
		return nil, fmt.Errorf("sandbox blocked: %w", err)
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	output, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return parseStatOutput(output, remotePath)
}

// SearchFiles searches for files matching a pattern on a remote server.
func (f *FileManagerService) SearchFiles(ctx context.Context, serverID, searchPath, pattern string, maxResults int) ([]FileEntry, error) {
	if isPathBlocked(searchPath) {
		return nil, fmt.Errorf("access denied: path %s is in a restricted system directory", searchPath)
	}

	limit := ""
	if maxResults > 0 {
		limit = fmt.Sprintf(" | head -n %d", maxResults)
	}

	cmd := fmt.Sprintf("find %s -name '*%s*' -type f%s 2>/dev/null",
		util.ShellQuote(searchPath), pattern, limit)
	if err := f.sb.Validate(cmd); err != nil {
		return nil, fmt.Errorf("sandbox blocked: %w", err)
	}

	exec, err := f.getExecutor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = exec.Close() }()

	output, err := exec.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to search files: %w", err)
	}

	var entries []FileEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entries = append(entries, FileEntry{
			Name: filepath.Base(line),
			Path: line,
			IsDir: false,
		})
	}
	return entries, nil
}

// getExecutor creates an SSH executor for the given server.
func (f *FileManagerService) getExecutor(ctx context.Context, serverID string) (*sshClientExecutor, error) {
	// Reuse Bridge's SSH executor creation logic
	b := &Bridge{DB: f.db}
	return b.getRemoteExecutor(ctx, serverID)
}

// base64Encode encodes a string to base64.
func base64Encode(s string) string {
	return encodeToString([]byte(s))
}

// encodeToString is a simple base64 encoder (avoids importing encoding/base64 in service).
func encodeToString(data []byte) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder
	result.Grow(len(data) * 4 / 3)

	for i := 0; i < len(data); i += 3 {
		n := len(data) - i
		if n > 3 {
			n = 3
		}
		var val uint32
		for j := 0; j < n; j++ {
			val |= uint32(data[i+j]) << uint((2-j)*8)
		}
		for j := 0; j < n+1; j++ {
			result.WriteByte(charset[(val>>(uint((3-j)*6)))&0x3F])
		}
	}
	// Padding
	for result.Len()%4 != 0 {
		result.WriteByte('=')
	}
	return result.String()
}

// parseLsOutput parses the output of `ls -la` into FileEntry slices.
func parseLsOutput(output, basePath string) ([]FileEntry, error) {
	var entries []FileEntry
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "total") {
			continue
		}

		// Skip the first line if it's the "total N" line
		if strings.HasPrefix(line, "total ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		perms := fields[0]
		owner := fields[2]
		group := fields[3]
		isDir := strings.HasPrefix(perms, "d")

		sizeStr := fields[4]
		size, _ := strconv.ParseInt(sizeStr, 10, 64)

		// Date fields: fields[5], fields[6], fields[7]
		dateStr := strings.Join(fields[5:8], " ")
		modTime, _ := time.Parse("2006-01-02 15:04", dateStr)
		if modTime.IsZero() {
			// Try alternate format: "Jan  5 12:34"
			modTime, _ = time.Parse("Jan 02 15:04", strings.Join(fields[5:8], " "))
		}
		if modTime.IsZero() {
			// Try: "Jan  5  2024"
			modTime, _ = time.Parse("Jan 02 2006", strings.Join(fields[5:8], " "))
		}

		name := strings.Join(fields[8:], " ")
		// Remove trailing -> symlink target for display
		if idx := strings.Index(name, " -> "); idx > 0 {
			name = name[:idx]
		}

		fullPath := filepath.Join(basePath, name)
		entries = append(entries, FileEntry{
			Name:        name,
			Path:        fullPath,
			IsDir:       isDir,
			Size:        size,
			Permissions: perms,
			ModTime:     modTime,
			Owner:       owner,
			Group:       group,
		})
	}

	// Sort: directories first, then by name
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

// parseDfOutput parses the output of `df -B1`.
func parseDfOutput(output, path string) (*DiskUsage, error) {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) < 4 {
		return nil, fmt.Errorf("failed to parse df output")
	}

	total, _ := strconv.ParseInt(fields[0], 10, 64)
	used, _ := strconv.ParseInt(fields[1], 10, 64)
	avail, _ := strconv.ParseInt(fields[2], 10, 64)

	pctStr := strings.TrimRight(fields[3], "%")
	pct, _ := strconv.ParseFloat(pctStr, 64)

	return &DiskUsage{
		Path:         path,
		Total:        total,
		Used:         used,
		Available:    avail,
		UsagePercent: pct,
	}, nil
}

// parseStatOutput parses the output of `stat -c`.
func parseStatOutput(output, path string) (*FileEntry, error) {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) < 5 {
		return nil, fmt.Errorf("failed to parse stat output")
	}

	perms := fields[0]
	owner := fields[1]
	group := fields[2]
	size, _ := strconv.ParseInt(fields[3], 10, 64)
	ts, _ := strconv.ParseInt(fields[4], 10, 64)

	return &FileEntry{
		Name:        filepath.Base(path),
		Path:        path,
		IsDir:       strings.HasPrefix(perms, "d"),
		Size:        size,
		Permissions: perms,
		ModTime:     time.Unix(ts, 0),
		Owner:       owner,
		Group:       group,
	}, nil
}
