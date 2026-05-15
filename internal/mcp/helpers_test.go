package mcp

import (
	"os"
	"testing"
)

func TestValidateVolumePath_ValidAbsolutePath(t *testing.T) {
	tests := []string{
		"/app/data",
		"/data/volumes",
		"/opt/app",
		"/tmp/uploads",
	}

	for _, path := range tests {
		err := validateVolumePath(path)
		if err != nil {
			t.Errorf("validateVolumePath(%q) = %v, want nil", path, err)
		}
	}
}

func TestValidateVolumePath_PathTraversalAttempt(t *testing.T) {
	tests := []string{
		"../etc/passwd",
		"/app/../../../etc/passwd",
		"/data/../../root/.ssh",
		"/tmp/../../../etc/shadow",
		"/app/..%2F..%2Fetc/passwd",
	}

	for _, path := range tests {
		err := validateVolumePath(path)
		if err == nil {
			t.Errorf("validateVolumePath(%q) = nil, want error for path traversal", path)
		}
		if path != "..%2F..%2Fetc/passwd" {
			if err == nil || err.Error() == "" {
				t.Errorf("validateVolumePath(%q) should return error for path traversal", path)
			}
		}
	}
}

func TestValidateVolumePath_RelativePathRejected(t *testing.T) {
	tests := []string{
		"relative/path",
		"./local/data",
		"~/data",
		"data/volumes",
	}

	for _, path := range tests {
		err := validateVolumePath(path)
		if err == nil {
			t.Errorf("validateVolumePath(%q) = nil, want error for relative path", path)
		}
	}
}

func TestValidateVolumePath_CustomAllowedRoots(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/custom/root:/another/root")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	err := validateVolumePath("/custom/root/data")
	if err != nil {
		t.Errorf("validateVolumePath(/custom/root/data) = %v, want nil with custom roots", err)
	}

	err = validateVolumePath("/another/root/volumes")
	if err != nil {
		t.Errorf("validateVolumePath(/another/root/volumes) = %v, want nil with custom roots", err)
	}

	err = validateVolumePath("/app/data")
	if err == nil {
		t.Errorf("validateVolumePath(/app/data) = nil, want error when /app is not in custom roots")
	}
}

func TestValidateVolumePath_NilEnvResetsToDefaults(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	defaultRoots := []string{"/app", "/data", "/opt", "/tmp"}
	for _, root := range defaultRoots {
		err := validateVolumePath(root + "/subdir")
		if err != nil {
			t.Errorf("validateVolumePath(%s/subdir) = %v, want nil for default root", root, err)
		}
	}

	err := validateVolumePath("/var/data")
	if err == nil {
		t.Errorf("validateVolumePath(/var/data) = nil, want error for non-default root")
	}
}

func TestExtractRegistry_DockerHub(t *testing.T) {
	tests := []struct {
		image   string
		want    string
	}{
		{"nginx:latest", "docker.io"},
		{"ubuntu:20.04", "docker.io"},
		{"golang", "docker.io"},
		{"redis", "docker.io"},
	}

	for _, tt := range tests {
		got := extractRegistry(tt.image)
		if got != tt.want {
			t.Errorf("extractRegistry(%q) = %q, want %q", tt.image, got, tt.want)
		}
	}
}

func TestExtractRegistry_CustomRegistry(t *testing.T) {
	tests := []struct {
		image string
		want  string
	}{
		{"gcr.io/google-containers/pause:latest", "gcr.io"},
		{"ghcr.io/user/repo:latest", "ghcr.io"},
		{"my-registry.example.com:5000/app:latest", "my-registry.example.com:5000"},
		{"registry.internal/project/image:tag", "registry.internal"},
	}

	for _, tt := range tests {
		got := extractRegistry(tt.image)
		if got != tt.want {
			t.Errorf("extractRegistry(%q) = %q, want %q", tt.image, got, tt.want)
		}
	}
}

func TestExtractRegistry_DockerHubOfficial(t *testing.T) {
	tests := []string{
		"library/nginx",
		"library/postgres",
		"library/redis:alpine",
	}

	for _, image := range tests {
		got := extractRegistry(image)
		if got != "docker.io" {
			t.Errorf("extractRegistry(%q) = %q, want docker.io", image, got)
		}
	}
}

func TestValidateImageRegistry_NoWhitelist(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	tests := []struct {
		image string
		want  bool
	}{
		{"nginx:latest", true},
		{"gcr.io/google-containers/pause:latest", true},
		{"ghcr.io/user/repo", true},
		{"docker.io/library/nginx", true},
	}

	for _, tt := range tests {
		err := validateImageRegistry(tt.image)
		if (err == nil) != tt.want {
			t.Errorf("validateImageRegistry(%q) error = %v, want success=%v", tt.image, err, tt.want)
		}
	}
}

func TestValidateImageRegistry_WithWhitelist(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "gcr.io,ghcr.io,registry.example.com")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	tests := []struct {
		image string
		want  bool
	}{
		{"gcr.io/google-containers/pause:latest", true},
		{"ghcr.io/user/repo:latest", true},
		{"registry.example.com/app:latest", true},
		{"nginx:latest", false},
		{"docker.io/library/nginx", false},
		{"quay.io/coreos/pause:latest", false},
	}

	for _, tt := range tests {
		err := validateImageRegistry(tt.image)
		if (err == nil) != tt.want {
			t.Errorf("validateImageRegistry(%q) = %v, want success=%v", tt.image, err, tt.want)
		}
	}
}

func TestValidateImageRegistry_SubdomainMatch(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "gcr.io")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	tests := []struct {
		image string
		want  bool
	}{
		{"gcr.io/project/image", true},
		{"asia.gcr.io/project/image", true},
		{"us.gcr.io/project/image", true},
		{"eu.gcr.io/project/image", true},
		{"mirror.gcr.io/project/image", true},
	}

	for _, tt := range tests {
		err := validateImageRegistry(tt.image)
		if (err == nil) != tt.want {
			t.Errorf("validateImageRegistry(%q) = %v, want success=%v", tt.image, err, tt.want)
		}
	}
}

func TestValidateImageRegistry_NonHTTPSWarning(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	err := validateImageRegistry("http://insecure-registry.com/image:latest")
	if err != nil {
		t.Errorf("validateImageRegistry with no whitelist should allow HTTP, got error: %v", err)
	}
}

func TestGetAllowedVolumeRoots_Custom(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/custom:/another")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	roots := getAllowedVolumeRoots()
	if len(roots) != 2 {
		t.Fatalf("getAllowedVolumeRoots() returned %d roots, want 2", len(roots))
	}
	if roots[0] != "/custom" || roots[1] != "/another" {
		t.Errorf("getAllowedVolumeRoots() = %v, want [/custom /another]", roots)
	}
}

func TestGetAllowedVolumeRoots_Default(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	roots := getAllowedVolumeRoots()
	expected := []string{"/app", "/data", "/opt", "/tmp"}

	if len(roots) != len(expected) {
		t.Fatalf("getAllowedVolumeRoots() returned %d roots, want %d", len(roots), len(expected))
	}

	for i, root := range roots {
		if root != expected[i] {
			t.Errorf("getAllowedVolumeRoots()[%d] = %q, want %q", i, root, expected[i])
		}
	}
}

func TestGetAllowedRegistries_Custom(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "gcr.io,ghcr.io,docker.io")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	registries := getAllowedRegistries()
	if len(registries) != 3 {
		t.Fatalf("getAllowedRegistries() returned %d registries, want 3", len(registries))
	}
	if registries[0] != "gcr.io" || registries[1] != "ghcr.io" || registries[2] != "docker.io" {
		t.Errorf("getAllowedRegistries() = %v, want [gcr.io ghcr.io docker.io]", registries)
	}
}

func TestGetAllowedRegistries_Empty(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	registries := getAllowedRegistries()
	if registries != nil {
		t.Errorf("getAllowedRegistries() = %v, want nil when env not set", registries)
	}
}

func TestValidateVolumePath_ComplexTraversal(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"/app/./data", false},
		{"/app//data", false},
		{"/app/data/../secret", true},
	}

	for _, tt := range tests {
		err := validateVolumePath(tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateVolumePath(%q) = %v, want error=%v", tt.path, err, tt.wantErr)
		}
	}
}

func TestValidateImageRegistry_EmptyRegistry(t *testing.T) {
	err := validateImageRegistry("repo/image")
	if err != nil {
		t.Errorf("validateImageRegistry(%q) = %v, want nil for Docker Hub default", "repo/image", err)
	}
}

func TestValidateImageRegistry_PortInRegistry(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "registry.example.com:5000")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	err := validateImageRegistry("registry.example.com:5000/myapp:latest")
	if err != nil {
		t.Errorf("validateImageRegistry(%q) = %v, want nil for exact match with port", "registry.example.com:5000/myapp:latest", err)
	}
}

func TestValidateImageRegistry_RegistryWithPortNotWhitelisted(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "registry.example.com")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	err := validateImageRegistry("registry.example.com:5000/myapp:latest")
	if err == nil {
		t.Errorf("validateImageRegistry(%q) = nil, want error for port mismatch", "registry.example.com:5000/myapp:latest")
	}
}
