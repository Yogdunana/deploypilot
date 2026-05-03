package mcp

import (
	"os"
	"testing"
)

func TestValidateVolumePath_PathTraversal(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"path with double dots", "/data/../../../etc/passwd", true},
		{"path with double dots relative", "/opt/../../etc/shadow", true},
		{"clean path", "/opt/deploypilot/data", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVolumePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVolumePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateVolumePath_NotAbsolute(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"relative path", "data/volumes"},
		{"home relative", "~/data"},
		{"current dir", "./data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVolumePath(tt.path)
			if err == nil {
				t.Errorf("validateVolumePath(%q) expected error for non-absolute path", tt.path)
			}
		})
	}
}

func TestValidateVolumePath_AllowedRoots(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/opt:/data")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"path under allowed root /opt", "/opt/app/data", false},
		{"path under allowed root /data", "/data/volumes", false},
		{"path outside allowed roots", "/var/log", true},
		{"path outside allowed roots /home", "/home/user", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVolumePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVolumePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateVolumePath_CleanPath(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/opt:/data")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"path with extra slashes", "/opt///app///data", false},
		{"path with trailing slash", "/opt/app/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVolumePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVolumePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestExtractRegistry(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{"dockerhub official", "nginx:latest", "docker.io"},
		{"dockerhub with registry", "docker.io/nginx:latest", "docker.io"},
		{"ghcr.io", "ghcr.io/user/image:tag", "ghcr.io"},
		{"custom registry with port", "registry.example.com:5000/image:tag", "registry.example.com:5000"},
		{"gcr.io", "gcr.io/project/image", "gcr.io"},
		{"quay.io", "quay.io/coreos/etcd", "quay.io"},
		{"dockerhub with org", "library/nginx", "docker.io"},
		{"simple image no tag", "nginx", "docker.io"},
		{"image with digest", "nginx@sha256:abc123", "docker.io"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRegistry(tt.image)
			if result != tt.expected {
				t.Errorf("extractRegistry(%q) = %q, want %q", tt.image, result, tt.expected)
			}
		})
	}
}

func TestValidateImageRegistry_NoWhitelist(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")
	tests := []struct {
		name    string
		image   string
		wantErr bool
	}{
		{"dockerhub image", "nginx:latest", false},
		{"ghcr image", "ghcr.io/user/image:tag", false},
		{"custom registry", "my-registry.com/image:tag", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImageRegistry(tt.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateImageRegistry(%q) error = %v, wantErr %v", tt.image, err, tt.wantErr)
			}
		})
	}
}

func TestValidateImageRegistry_WithWhitelist(t *testing.T) {
	os.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "docker.io,ghcr.io")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	tests := []struct {
		name    string
		image   string
		wantErr bool
	}{
		{"dockerhub allowed", "nginx:latest", false},
		{"ghcr.io allowed", "ghcr.io/user/image:tag", false},
		{"docker.io explicit", "docker.io/library/nginx", false},
		{"private registry not allowed", "my-registry.com/image:tag", true},
		{"quay not allowed", "quay.io/coreos/etcd", true},
		{"subdomain of allowed", "docker.io", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImageRegistry(tt.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateImageRegistry(%q) error = %v, wantErr %v", tt.image, err, tt.wantErr)
			}
		})
	}
}

func TestGetAllowedRegistries(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")
	if got := getAllowedRegistries(); got != nil {
		t.Errorf("getAllowedRegistries() = %v, want nil", got)
	}

	os.Setenv("DEPLOYPILOT_ALLOWED_REGISTRIES", "docker.io,ghcr.io,registry.example.com")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_REGISTRIES")

	got := getAllowedRegistries()
	expected := []string{"docker.io", "ghcr.io", "registry.example.com"}
	if len(got) != len(expected) {
		t.Fatalf("getAllowedRegistries() = %v, want %v", got, expected)
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Errorf("getAllowedRegistries()[%d] = %q, want %q", i, got[i], expected[i])
		}
	}
}

func TestGetAllowedVolumeRoots(t *testing.T) {
	os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")
	if got := getAllowedVolumeRoots(); got != nil {
		t.Errorf("getAllowedVolumeRoots() = %v, want nil", got)
	}

	os.Setenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS", "/opt:/data:/var/data")
	defer os.Unsetenv("DEPLOYPILOT_ALLOWED_VOLUME_ROOTS")

	got := getAllowedVolumeRoots()
	expected := []string{"/opt", "/data", "/var/data"}
	if len(got) != len(expected) {
		t.Fatalf("getAllowedVolumeRoots() = %v, want %v", got, expected)
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Errorf("getAllowedVolumeRoots()[%d] = %q, want %q", i, got[i], expected[i])
		}
	}
}
