package registry

import (
	"testing"
)

// ========== Factory Function Tests ==========

func TestNewRegistryProviderDockerHub(t *testing.T) {
	p, err := NewRegistryProvider("docker_hub", "https://registry-1.docker.io/v2/", "user", "pass")
	if err != nil {
		t.Fatalf("NewRegistryProvider() error = %v", err)
	}
	if p.Name() != "docker_hub" {
		t.Errorf("Name() = %q, want %q", p.Name(), "docker_hub")
	}
	if _, ok := p.(*DockerHubProvider); !ok {
		t.Errorf("expected *DockerHubProvider, got %T", p)
	}
}

func TestNewRegistryProviderGHCR(t *testing.T) {
	p, err := NewRegistryProvider("ghcr", "https://ghcr.io/v2/", "OWNER", "token")
	if err != nil {
		t.Fatalf("NewRegistryProvider() error = %v", err)
	}
	if p.Name() != "ghcr" {
		t.Errorf("Name() = %q, want %q", p.Name(), "ghcr")
	}
	if _, ok := p.(*GHCRProvider); !ok {
		t.Errorf("expected *GHCRProvider, got %T", p)
	}
}

func TestNewRegistryProviderUnsupported(t *testing.T) {
	_, err := NewRegistryProvider("harbor", "https://harbor.example.com/v2/", "user", "pass")
	if err == nil {
		t.Error("NewRegistryProvider() should return error for unsupported provider")
	}
}

func TestNewRegistryProviderUnsupportedACR(t *testing.T) {
	_, err := NewRegistryProvider("acr", "https://myacr.azurecr.io/v2/", "user", "pass")
	if err == nil {
		t.Error("NewRegistryProvider() should return error for unsupported provider acr")
	}
}

func TestNewRegistryProviderDockerHubEmptyURL(t *testing.T) {
	p, err := NewRegistryProvider("docker_hub", "", "user", "pass")
	if err != nil {
		t.Fatalf("NewRegistryProvider() error = %v", err)
	}
	if p.Name() != "docker_hub" {
		t.Errorf("Name() = %q, want %q", p.Name(), "docker_hub")
	}
	// Verify default URL is used
	dp := p.(*DockerHubProvider)
	if dp.baseURL != defaultDockerHubURL {
		t.Errorf("baseURL = %q, want default %q", dp.baseURL, defaultDockerHubURL)
	}
}

func TestNewRegistryProviderGHCREmptyURL(t *testing.T) {
	p, err := NewRegistryProvider("ghcr", "", "OWNER", "token")
	if err != nil {
		t.Fatalf("NewRegistryProvider() error = %v", err)
	}
	if p.Name() != "ghcr" {
		t.Errorf("Name() = %q, want %q", p.Name(), "ghcr")
	}
	// Verify default URL is used
	gp := p.(*GHCRProvider)
	if gp.baseURL != defaultGHCRURL {
		t.Errorf("baseURL = %q, want default %q", gp.baseURL, defaultGHCRURL)
	}
}

func TestNewRegistryProviderGHCRDefaultUsername(t *testing.T) {
	p, err := NewRegistryProvider("ghcr", "", "", "token")
	if err != nil {
		t.Fatalf("NewRegistryProvider() error = %v", err)
	}
	gp := p.(*GHCRProvider)
	if gp.username != "OWNER" {
		t.Errorf("username = %q, want default %q", gp.username, "OWNER")
	}
}

// ========== Interface Compliance Tests ==========

func TestDockerHubImplementsRegistryProvider(t *testing.T) {
	var _ RegistryProvider = (*DockerHubProvider)(nil)
}

func TestGHCRImplementsRegistryProvider(t *testing.T) {
	var _ RegistryProvider = (*GHCRProvider)(nil)
}
