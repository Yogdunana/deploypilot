package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/Yogdunana/deploypilot/internal/version"
)

// GitHubRelease represents a GitHub API release response.
type GitHubRelease struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Body    string  `json:"body"`
	Draft   bool    `json:"draft"`
	Prerelease bool `json:"prerelease"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
	PublishedAt time.Time `json:"published_at"`
}

// UpgradeProgress represents the progress of an upgrade operation.
type UpgradeProgress struct {
	Step      string `json:"step"`
	Message   string `json:"message"`
	Percent   int    `json:"percent"`
	Status    string `json:"status"` // "running", "success", "error"
	Timestamp int64  `json:"timestamp"`
}

// UpgradeResult represents the result of an upgrade operation.
type UpgradeResult struct {
	Success      bool   `json:"success"`
	OldVersion   string `json:"old_version"`
	NewVersion   string `json:"new_version"`
	Message      string `json:"message"`
	RollbackPath string `json:"rollback_path,omitempty"`
}

// UpgradeService handles system upgrade operations.
type UpgradeService struct {
	installDir    string
	mu            sync.Mutex
	inProgress    bool
	progress      UpgradeProgress
	progressChans []chan UpgradeProgress
}

// NewUpgradeService creates a new UpgradeService.
func NewUpgradeService(installDir string) *UpgradeService {
	if installDir == "" {
		installDir = "/opt/deploypilot"
	}
	return &UpgradeService{installDir: installDir}
}

// CheckUpdate checks GitHub Releases API for the latest version.
func (b *Bridge) CheckSystemUpdate(ctx context.Context) (interface{}, error) {
	current := version.Version
	if current == "dev" {
		current = "0.0.0-dev"
	}

	release, err := fetchLatestRelease(ctx)
	if err != nil {
		slog.Warn("failed to check for updates", "error", err)
		// Return current info without update check
		return map[string]interface{}{
			"current_version":  current,
			"latest_version":   current,
			"update_available": false,
			"message":          fmt.Sprintf("failed to check updates: %v", err),
		}, nil
	}

	latest := release.TagName
	hasUpdate := compareVersions(latest, current) > 0

	result := map[string]interface{}{
		"current_version":  current,
		"latest_version":   latest,
		"update_available": hasUpdate,
		"release_notes":    release.Body,
		"published_at":     release.PublishedAt.Format(time.RFC3339),
	}

	if hasUpdate {
		result["message"] = fmt.Sprintf("a new version %s is available", latest)
	} else {
		result["message"] = "you are running the latest version of DeployPilot"
	}

	return result, nil
}

// fetchLatestRelease fetches the latest release from GitHub API.
func fetchLatestRelease(ctx context.Context) (*GitHubRelease, error) {
	url := "https://api.github.com/repos/Yogdunana/deploypilot/releases/latest"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DeployPilot/"+version.Version)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &release, nil
}

// compareVersions compares two semantic version strings.
// Returns 1 if a > b, -1 if a < b, 0 if equal.
func compareVersions(a, b string) int {
	// Strip 'v' prefix
	if len(a) > 0 && a[0] == 'v' {
		a = a[1:]
	}
	if len(b) > 0 && b[0] == 'v' {
		b = b[1:]
	}

	// Simple semver comparison (major.minor.patch)
	var majA, minA, patA, majB, minB, patB int
	fmt.Sscanf(a, "%d.%d.%d", &majA, &minA, &patA)
	fmt.Sscanf(b, "%d.%d.%d", &majB, &minB, &patB)

	if majA != majB {
		return majA - majB
	}
	if minA != minB {
		return minA - minB
	}
	return patA - patB
}

// --- Upgrade execution ---

// SetInstallDir sets the installation directory (for testing).
func (us *UpgradeService) SetInstallDir(dir string) {
	us.installDir = dir
}

// IsInProgress returns whether an upgrade is currently running.
func (us *UpgradeService) IsInProgress() bool {
	us.mu.Lock()
	defer us.mu.Unlock()
	return us.inProgress
}

// GetProgress returns the current upgrade progress.
func (us *UpgradeService) GetProgress() UpgradeProgress {
	us.mu.Lock()
	defer us.mu.Unlock()
	return us.progress
}

// SubscribeProgress subscribes to upgrade progress updates.
func (us *UpgradeService) SubscribeProgress() chan UpgradeProgress {
	ch := make(chan UpgradeProgress, 10)
	us.mu.Lock()
	us.progressChans = append(us.progressChans, ch)
	us.mu.Unlock()
	return ch
}

// UnsubscribeProgress unsubscribes from upgrade progress updates.
func (us *UpgradeService) UnsubscribeProgress(ch chan UpgradeProgress) {
	us.mu.Lock()
	defer us.mu.Unlock()
	for i, c := range us.progressChans {
		if c == ch {
			us.progressChans = append(us.progressChans[:i], us.progressChans[i+1:]...)
			close(ch)
			break
		}
	}
}

func (us *UpgradeService) emitProgress(step, message string, percent int, status string) {
	us.mu.Lock()
	us.progress = UpgradeProgress{
		Step:      step,
		Message:   message,
		Percent:   percent,
		Status:    status,
		Timestamp: time.Now().Unix(),
	}
	chans := make([]chan UpgradeProgress, len(us.progressChans))
	copy(chans, us.progressChans)
	us.mu.Unlock()

	for _, ch := range chans {
		select {
		case ch <- us.progress:
		default:
			// Channel full, skip
		}
	}
}

// PerformUpgrade executes the full upgrade process.
func (us *UpgradeService) PerformUpgrade(ctx context.Context, targetVersion string) (*UpgradeResult, error) {
	us.mu.Lock()
	if us.inProgress {
		us.mu.Unlock()
		return nil, fmt.Errorf("an upgrade is already in progress")
	}
	us.inProgress = true
	us.mu.Unlock()

	defer func() {
		us.mu.Lock()
		us.inProgress = false
		us.mu.Unlock()
	}()

	oldVersion := version.Version

	// Step 1: Fetch release info
	us.emitProgress("check", "fetching release information...", 5, "running")
	release, err := fetchLatestRelease(ctx)
	if err != nil {
		us.emitProgress("check", fmt.Sprintf("failed to fetch release: %v", err), 5, "error")
		return nil, fmt.Errorf("fetch release: %w", err)
	}

	ver := targetVersion
	if ver == "" || ver == "latest" {
		ver = release.TagName
	}

	us.emitProgress("check", fmt.Sprintf("target version: %s", ver), 10, "running")

	// Step 2: Pre-flight checks
	us.emitProgress("preflight", "running pre-flight checks...", 15, "running")
	if err := us.preflightChecks(ver); err != nil {
		us.emitProgress("preflight", fmt.Sprintf("pre-flight check failed: %v", err), 15, "error")
		return nil, fmt.Errorf("preflight: %w", err)
	}

	// Step 3: Backup current binaries
	us.emitProgress("backup", "backing up current binaries...", 25, "running")
	backupDir, err := us.backupBinaries()
	if err != nil {
		us.emitProgress("backup", fmt.Sprintf("backup failed: %v", err), 25, "error")
		return nil, fmt.Errorf("backup: %w", err)
	}
	us.emitProgress("backup", fmt.Sprintf("backed up to %s", backupDir), 35, "running")

	// Step 4: Download new binaries
	us.emitProgress("download", "downloading new binaries...", 40, "running")
	if err := us.downloadBinaries(ctx, ver); err != nil {
		us.emitProgress("download", fmt.Sprintf("download failed: %v", err), 40, "error")
		// Auto-rollback
		_ = us.restoreBinaries(backupDir)
		return nil, fmt.Errorf("download: %w (auto-rollback attempted)", err)
	}
	us.emitProgress("download", "binaries downloaded successfully", 65, "running")

	// Step 5: Verify new binaries
	us.emitProgress("verify", "verifying new binaries...", 70, "running")
	if err := us.verifyBinaries(); err != nil {
		us.emitProgress("verify", fmt.Sprintf("verification failed: %v", err), 70, "error")
		_ = us.restoreBinaries(backupDir)
		return nil, fmt.Errorf("verify: %w (auto-rollback attempted)", err)
	}

	// Step 6: Replace binaries
	us.emitProgress("replace", "replacing binaries...", 80, "running")
	if err := us.replaceBinaries(); err != nil {
		us.emitProgress("replace", fmt.Sprintf("replace failed: %v", err), 80, "error")
		_ = us.restoreBinaries(backupDir)
		return nil, fmt.Errorf("replace: %w (auto-rollback attempted)", err)
	}

	// Step 7: Restart services
	us.emitProgress("restart", "restarting services...", 90, "running")
	if err := us.restartServices(); err != nil {
		us.emitProgress("restart", fmt.Sprintf("restart failed: %v", err), 90, "error")
		_ = us.restoreBinaries(backupDir)
		return nil, fmt.Errorf("restart: %w (auto-rollback attempted)", err)
	}

	us.emitProgress("complete", fmt.Sprintf("successfully upgraded to %s", ver), 100, "success")

	return &UpgradeResult{
		Success:      true,
		OldVersion:   oldVersion,
		NewVersion:   ver,
		Message:      fmt.Sprintf("successfully upgraded from %s to %s", oldVersion, ver),
		RollbackPath: backupDir,
	}, nil
}

func (us *UpgradeService) preflightChecks(targetVersion string) error {
	// Check if already on target version
	if compareVersions(version.Version, targetVersion) >= 0 {
		return fmt.Errorf("current version %s is already at or newer than %s", version.Version, targetVersion)
	}

	// Check install directory exists
	if _, err := os.Stat(us.installDir); os.IsNotExist(err) {
		return fmt.Errorf("install directory %s does not exist", us.installDir)
	}

	// Check bin directory
	binDir := filepath.Join(us.installDir, "bin")
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		return fmt.Errorf("bin directory %s does not exist", binDir)
	}

	// Check systemd services exist
	for _, svc := range []string{"deploypilot.service", "deploypilot-mcp.service"} {
		if _, err := os.Stat(filepath.Join("/etc/systemd/system", svc)); os.IsNotExist(err) {
			slog.Warn("systemd service not found, skipping restart check", "service", svc)
		}
	}

	// Check disk space (need at least 100MB free)
	var stat syscall.Statfs_t
	_ = syscall.Statfs(us.installDir, &stat)
	// 100MB in bytes
	freeSpace := stat.Bavail * uint64(stat.Bsize)
	if freeSpace < 100*1024*1024 {
		return fmt.Errorf("insufficient disk space: %d bytes free, need at least 100MB", freeSpace)
	}

	return nil
}

func (us *UpgradeService) backupBinaries() (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(us.installDir, "backups", "upgrade-"+timestamp)

	binDir := filepath.Join(us.installDir, "bin")
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}

	// Backup all binaries
	for _, binary := range []string{"api-server", "mcp-server", "deploypilot"} {
		src := filepath.Join(binDir, binary)
		dst := filepath.Join(backupDir, binary)

		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue // Skip missing binaries
		}

		if err := copyFile(src, dst); err != nil {
			return "", fmt.Errorf("backup %s: %w", binary, err)
		}
	}

	return backupDir, nil
}

func (us *UpgradeService) downloadBinaries(ctx context.Context, ver string) error {
	arch := runtime.GOARCH
	osName := runtime.GOOS

	// Map GOARCH to release binary naming
	archStr := arch
	if arch == "amd64" {
		archStr = "amd64"
	} else if arch == "arm64" {
		archStr = "arm64"
	}

	tmpDir := filepath.Join(us.installDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0750); err != nil {
		return fmt.Errorf("create tmp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	binaries := []string{"api-server", "mcp-server", "deploypilot"}
	for i, binary := range binaries {
		filename := fmt.Sprintf("%s-%s-%s", binary, osName, archStr)
		downloadURL := fmt.Sprintf(
			"https://github.com/Yogdunana/deploypilot/releases/download/%s/%s",
			ver, filename,
		)

		us.emitProgress("download", fmt.Sprintf("downloading %s (%d/%d)...", binary, i+1, len(binaries)), 40+i*8, "running")

		destPath := filepath.Join(tmpDir, binary)
		if err := downloadFile(ctx, downloadURL, destPath); err != nil {
			return fmt.Errorf("download %s: %w", binary, err)
		}

		// Make executable
		if err := os.Chmod(destPath, 0755); err != nil {
			return fmt.Errorf("chmod %s: %w", binary, err)
		}
	}

	// Move from tmp to bin (atomic-ish)
	binDir := filepath.Join(us.installDir, "bin")
	for _, binary := range binaries {
		src := filepath.Join(tmpDir, binary)
		dst := filepath.Join(binDir, binary)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("move %s: %w", binary, err)
		}
	}

	return nil
}

func (us *UpgradeService) verifyBinaries() error {
	binDir := filepath.Join(us.installDir, "bin")
	for _, binary := range []string{"api-server", "mcp-server", "deploypilot"} {
		path := filepath.Join(binDir, binary)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("binary %s not found after download", binary)
		}

		// Check if binary is executable
		cmd := exec.Command(path, "--version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s --version failed: %w (output: %s)", binary, err, string(output))
		}
		slog.Info("verified binary", "binary", binary, "version", string(output))
	}
	return nil
}

func (us *UpgradeService) replaceBinaries() error {
	// Binaries are already in place from downloadBinaries
	// This step exists for future atomic replacement logic
	return nil
}

func (us *UpgradeService) restartServices() error {
	// Restart systemd services
	services := []string{"deploypilot-mcp.service", "deploypilot.service"}
	for _, svc := range services {
		svcPath := filepath.Join("/etc/systemd/system", svc)
		if _, err := os.Stat(svcPath); os.IsNotExist(err) {
			slog.Info("service not found, skipping restart", "service", svc)
			continue
		}

		slog.Info("restarting service", "service", svc)
		cmd := exec.Command("systemctl", "restart", svc)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("restart %s: %w (output: %s)", svc, err, string(output))
		}
		slog.Info("service restarted", "service", svc)
	}

	return nil
}

func (us *UpgradeService) restoreBinaries(backupDir string) error {
	slog.Warn("attempting rollback from backup", "backup_dir", backupDir)

	binDir := filepath.Join(us.installDir, "bin")
	for _, binary := range []string{"api-server", "mcp-server", "deploypilot"} {
		src := filepath.Join(backupDir, binary)
		dst := filepath.Join(binDir, binary)

		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}

		if err := copyFile(src, dst); err != nil {
			slog.Error("failed to restore binary", "binary", binary, "error", err)
			continue
		}
		slog.Info("restored binary", "binary", binary)
	}

	// Restart services with old binaries
	_ = us.restartServices()

	return nil
}

// --- Helper functions ---

func downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "DeployPilot/"+version.Version)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0755)
}
