//go:build !linux

package service

import "log/slog"

// requireFreeDiskSpace is a no-op off Linux. statfs has no portable equivalent,
// and the upgrade always targets a Linux host, so skipping the check beats failing it.
func requireFreeDiskSpace(path string, minBytes uint64) error {
	slog.Debug("disk space check skipped: no statfs on this platform", "path", path)
	return nil
}
