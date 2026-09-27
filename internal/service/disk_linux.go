//go:build linux

package service

import (
	"fmt"
	"syscall"
)

// requireFreeDiskSpace verifies the filesystem holding path has at least minBytes
// free. Named apart from preflight's checkDiskSpace, which takes different args.
func requireFreeDiskSpace(path string, minBytes uint64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return fmt.Errorf("failed to stat filesystem: %w", err)
	}
	free := stat.Bavail * uint64(stat.Bsize)
	if free < minBytes {
		return fmt.Errorf("insufficient disk space: %d bytes free, need at least %d", free, minBytes)
	}
	return nil
}
