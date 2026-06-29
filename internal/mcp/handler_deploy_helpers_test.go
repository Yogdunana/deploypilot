package mcp

import (
	"os"
	"testing"
)

func TestExtractRegistry_DockerHub(t *testing.T) {
	tests := []struct {
		image    string
		expected string
	}{
		{"nginx", "docker.io"},
		{"nginx:latest", "docker.io"},
		{"library/nginx", "docker.io"},
		{"library/nginx:latest", "docker.io"},
	}

	for _, tt := range tests {
		result := extractRegistry(tt.image)
		if result != tt.expected {
			t.Errorf("extractRegistry(%q) = %q, want %q", tt.image, result, tt.expected)
		}
	}
}

func TestExtractRegistry_CustomRegistry(t *testing.T) {
	tests := []struct {
		image    string
		expected string
	}{
		{"registry.example.com/nginx", "registry.example.com"},
		{"registry.example.com:5000/nginx", "registry.example.com:5000"},
		{"gcr.io/google-containers/nginx", "gcr.io"},
		{"ghcr.io/user/repo", "ghcr.io"},
		{"my-registry.com:8080/project/image:latest", "my-registry.com:8080"},
	}

	for _, tt := range tests {
		result := extractRegistry(tt.image)
		if result != tt.expected {
			t.Errorf("extractRegistry(%q) = %q, want %q", tt.image, result, tt.expected)
		}
	}
}

func TestValidateImageRegistry_NoWhitelist(t *testing.T) {
	// Clear the environment
	os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	tests := []struct {
		image string
	}{
		{"nginx"},
		{"nginx:latest"},
		{"registry.example.com/image"},
		{"gcr.io/google-containers/nginx"},
	}

	for _, tt := range tests {
		err := validateImageRegistry(tt.image)
		if err != nil {
			t.Errorf("validateImageRegistry(%q) unexpected error: %v", tt.image, err)
		}
	}
}

func TestValidateImageRegistry_WithWhitelist(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "gcr.io,ghcr.io,docker.io")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	tests := []struct {
		image    string
		wantErr  bool
	}{
		{"nginx", false}, // docker.io is whitelisted
		{"gcr.io/google-containers/nginx", false},
		{"ghcr.io/user/repo", false},
		{"registry.example.com/image", true},
		{"quay.io/coreos/nginx", true},
	}

	for _, tt := range tests {
		err := validateImageRegistry(tt.image)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateImageRegistry(%q) error = %v, wantErr %v", tt.image, err, tt.wantErr)
		}
	}
}

func TestValidateImageRegistry_SubdomainMatching(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "gcr.io")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	tests := []struct {
		image   string
		wantErr bool
	}{
		{"gcr.io/project/image", false},
		{"asia.gcr.io/project/image", false}, // subdomain of gcr.io
		{"gcr.io.uk/image", true},             // different TLD
	}

	for _, tt := range tests {
		err := validateImageRegistry(tt.image)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateImageRegistry(%q) error = %v, wantErr %v", tt.image, err, tt.wantErr)
		}
	}
}

func TestValidateVolumePath_ValidPaths(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/app:/data:/opt:/tmp")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	validPaths := []string{
		"/app",
		"/app/data",
		"/data",
		"/data/containers",
		"/opt",
		"/tmp",
	}

	for _, path := range validPaths {
		err := validateVolumePath(path)
		if err != nil {
			t.Errorf("validateVolumePath(%q) unexpected error: %v", path, err)
		}
	}
}

func TestValidateVolumePath_RelativePath(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/app:/data")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	relativePaths := []string{
		"app/data",
		"./app/data",
		"../data",
	}

	for _, path := range relativePaths {
		err := validateVolumePath(path)
		if err == nil {
			t.Errorf("validateVolumePath(%q) expected error for relative path", path)
		}
	}
}

func TestValidateVolumePath_BlockedPaths(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/app:/data:/opt:/tmp")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	blockedPaths := []string{
		"/etc",
		"/etc/passwd",
		"/root",
		"/root/.ssh",
		"/proc",
		"/proc/self",
		"/sys",
		"/dev",
		"/bin",
		"/sbin",
		"/usr",
		"/usr/bin",
	}

	for _, path := range blockedPaths {
		err := validateVolumePath(path)
		if err == nil {
			t.Errorf("validateVolumePath(%q) expected error for blocked path", path)
		}
	}
}

func TestValidateVolumePath_PathTraversal(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/app:/data")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	traversalPaths := []string{
		"/app/../etc",
		"/app/../../root",
		"/data/../../etc/passwd",
		"/app/../../proc/self",
	}

	for _, path := range traversalPaths {
		err := validateVolumePath(path)
		if err == nil {
			t.Errorf("validateVolumePath(%q) expected error for path traversal", path)
		}
	}
}

func TestValidateVolumePath_OutsideAllowedRoots(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/app:/data")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	outsidePaths := []string{
		"/var",
		"/home",
		"/srv",
		"/opt",
	}

	for _, path := range outsidePaths {
		err := validateVolumePath(path)
		if err == nil {
			t.Errorf("validateVolumePath(%q) expected error for path outside allowed roots", path)
		}
	}
}

func TestValidateVolumePath_DefaultAllowedRoots(t *testing.T) {
	// Use default allowed roots
	os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	defaultRoots := []string{"/app", "/data", "/opt", "/tmp"}

	for _, path := range defaultRoots {
		err := validateVolumePath(path)
		if err != nil {
			t.Errorf("validateVolumePath(%q) with default roots unexpected error: %v", path, err)
		}
	}
}

func TestValidateVolumePath_NonBlockedPathNoEnvSet(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	// With default allowed roots (/app, /data, /opt, /tmp), only those paths are allowed
	// Other paths like /var and /home are NOT in the default allowed list and will be blocked
	allowed := []string{"/app", "/data", "/opt", "/tmp"}

	for _, path := range allowed {
		err := validateVolumePath(path)
		if err != nil {
			t.Errorf("validateVolumePath(%q) unexpected error: %v", path, err)
		}
	}

	// These are outside the default allowed roots
	outsidePaths := []string{"/var", "/home", "/srv"}
	for _, path := range outsidePaths {
		err := validateVolumePath(path)
		if err == nil {
			t.Errorf("validateVolumePath(%q) expected error for path outside default roots", path)
		}
	}
}

func TestGetAllowedVolumeRoots_DefaultRoots(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	roots := getAllowedVolumeRoots()
	if len(roots) != 4 {
		t.Errorf("expected 4 default roots, got %d", len(roots))
	}

	expected := []string{"/app", "/data", "/opt", "/tmp"}
	for i, r := range roots {
		if r != expected[i] {
			t.Errorf("root[%d] = %q, want %q", i, r, expected[i])
		}
	}
}

func TestGetAllowedVolumeRoots_CustomRoots(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/custom:/another")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	roots := getAllowedVolumeRoots()
	if len(roots) != 2 {
		t.Errorf("expected 2 custom roots, got %d", len(roots))
	}
	if roots[0] != "/custom" || roots[1] != "/another" {
		t.Errorf("unexpected roots: %v", roots)
	}
}

func TestGetAllowedRegistries_NoEnv(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	registries := getAllowedRegistries()
	if registries != nil {
		t.Errorf("expected nil registries, got %v", registries)
	}
}

func TestGetAllowedRegistries_WithEnv(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "gcr.io,ghcr.io")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	registries := getAllowedRegistries()
	if len(registries) != 2 {
		t.Errorf("expected 2 registries, got %d", len(registries))
	}
	if registries[0] != "gcr.io" || registries[1] != "ghcr.io" {
		t.Errorf("unexpected registries: %v", registries)
	}
}
