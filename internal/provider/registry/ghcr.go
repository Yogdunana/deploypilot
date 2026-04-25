package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const (
	// defaultGHCRURL is the default GitHub Container Registry URL.
	defaultGHCRURL = "https://ghcr.io/v2/"
)

// GHCRProvider implements RegistryProvider for GitHub Container Registry.
type GHCRProvider struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewGHCRProvider creates a new GHCR registry provider.
func NewGHCRProvider(url, username, password string) *GHCRProvider {
	if url == "" {
		url = defaultGHCRURL
	}
	// GHCR accepts any non-empty username; the password is the GitHub token.
	if username == "" {
		username = "OWNER"
	}
	return &GHCRProvider{
		baseURL:  url,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL allows overriding the API base URL (for testing).
func (g *GHCRProvider) SetBaseURL(url string) {
	g.baseURL = url
}

// Name returns the provider name.
func (g *GHCRProvider) Name() string { return "ghcr" }

// Login authenticates with GHCR using docker CLI.
func (g *GHCRProvider) Login(ctx context.Context) error {
	registry := strings.TrimSuffix(g.baseURL, "/v2/")
	registry = strings.TrimSuffix(registry, "/v2")

	cmd := exec.CommandContext(ctx, "docker", "login", "-u", g.username, "-p", g.password, registry)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker login failed: %w: %s", err, string(output))
	}
	return nil
}

// Push pushes an image to GHCR using docker CLI.
func (g *GHCRProvider) Push(ctx context.Context, localImage, remoteTag string) error {
	cmd := exec.CommandContext(ctx, "docker", "push", remoteTag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker push failed: %w: %s", err, string(output))
	}
	return nil
}

// Pull pulls an image from GHCR using docker CLI.
func (g *GHCRProvider) Pull(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, "docker", "pull", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker pull failed: %w: %s", err, string(output))
	}
	return nil
}

// Tag creates a tag for an image using docker CLI.
func (g *GHCRProvider) Tag(ctx context.Context, sourceImage, targetTag string) error {
	cmd := exec.CommandContext(ctx, "docker", "tag", sourceImage, targetTag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker tag failed: %w: %s", err, string(output))
	}
	return nil
}

// ListTags lists tags for a repository using the Docker Registry HTTP API v2.
// GHCR uses the GitHub token directly as a Bearer token.
func (g *GHCRProvider) ListTags(ctx context.Context, repository string) ([]string, error) {
	url := fmt.Sprintf("%s%s/tags/list", g.baseURL, repository)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// GHCR accepts the GitHub token directly as a Bearer token.
	req.Header.Set("Authorization", "Bearer "+g.password)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list tags (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse tags response: %w", err)
	}

	return result.Tags, nil
}

// Ping checks if GHCR is accessible by hitting the /v2/ endpoint.
func (g *GHCRProvider) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to ping registry: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Docker Registry API v2 returns 200 or 401 for /v2/
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("registry ping failed with HTTP %d", resp.StatusCode)
	}

	return nil
}
