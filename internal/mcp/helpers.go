package mcp

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func validateVolumePath(hostPath string) error {
	if strings.Contains(hostPath, "..") {
		return fmt.Errorf("volume path contains path traversal: %s", hostPath)
	}
	if !filepath.IsAbs(hostPath) {
		return fmt.Errorf("volume path must be absolute: %s", hostPath)
	}
	cleaned := filepath.Clean(hostPath)
	allowedRoots := getAllowedVolumeRoots()
	if len(allowedRoots) > 0 {
		allowed := false
		for _, root := range allowedRoots {
			if strings.HasPrefix(cleaned, root) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("volume path not under allowed root directories: %s", hostPath)
		}
	}
	return nil
}

// getAllowedVolumeRoots returns the list of allowed volume root directories.
// Configured via DEPLOYPILOT_ALLOWED_VOLUME_ROOTS (colon-separated).
func getAllowedVolumeRoots() []string {
	if roots := os.Getenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS"); roots != "" {
		return strings.Split(roots, ":")
	}
	return nil
}

// validateImageRegistry checks if a Docker image is from an allowed registry.
func validateImageRegistry(image string) error {
	registry := extractRegistry(image)
	allowed := getAllowedRegistries()
	if len(allowed) == 0 {
		if strings.HasPrefix(registry, "http://") {
			slog.Warn("deploying image from non-HTTPS registry", "registry", registry, "image", image)
		}
		return nil
	}
	for _, a := range allowed {
		if registry == a || strings.HasSuffix(registry, "."+a) {
			return nil
		}
	}
	return fmt.Errorf("image registry not in whitelist: %s (allowed: %v)", registry, allowed)
}

// extractRegistry extracts the registry from a Docker image name.
func extractRegistry(image string) string {
	parts := strings.SplitN(image, "/", 2)
	if len(parts) < 2 {
		return "docker.io"
	}
	if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
		return parts[0]
	}
	return "docker.io"
}

// getAllowedRegistries returns the list of allowed image registries.
// Configured via DEPLOYPILOT_ALLOWED_REGISTRIES (comma-separated).
func getAllowedRegistries() []string {
	if registries := os.Getenv("DEPLOYPILOT_ALLOWED_REGISTRIES"); registries != "" {
		return strings.Split(registries, ",")
	}
	return nil
}
