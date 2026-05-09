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
	// defaultDockerHubURL is the default Docker Hub registry URL.
	defaultDockerHubURL = "https://registry-1.docker.io/v2/"
	// dockerHubAuthURL is the Docker Hub authentication realm.
	dockerHubAuthURL = "https://auth.docker.io/token"
)

// DockerHubProvider implements RegistryProvider for Docker Hub.
type DockerHubProvider struct {
	baseURL    string
	authURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewDockerHubProvider creates a new Docker Hub registry provider.
func NewDockerHubProvider(url, username, password string) *DockerHubProvider {
	if url == "" {
		url = defaultDockerHubURL
	}
	return &DockerHubProvider{
		baseURL:  url,
		authURL:  dockerHubAuthURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL allows overriding the API base URL (for testing).
func (d *DockerHubProvider) SetBaseURL(url string) {
	d.baseURL = url
}

// Name returns the provider name.
func (d *DockerHubProvider) Name() string { return "docker_hub" }

// Login authenticates with Docker Hub using docker CLI.
func (d *DockerHubProvider) Login(ctx context.Context) error {
	registry := strings.TrimSuffix(d.baseURL, "/v2/")
	registry = strings.TrimSuffix(registry, "/v2")

	cmd := exec.CommandContext(ctx, "docker", "login", "-u", d.username, "--password-stdin", registry)
	cmd.Stdin = strings.NewReader(d.password)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker login failed: %w: %s", err, string(output))
	}
	return nil
}

// Push pushes an image to Docker Hub using docker CLI.
func (d *DockerHubProvider) Push(ctx context.Context, localImage, remoteTag string) error {
	cmd := exec.CommandContext(ctx, "docker", "push", remoteTag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker push failed: %w: %s", err, string(output))
	}
	return nil
}

// Pull pulls an image from Docker Hub using docker CLI.
func (d *DockerHubProvider) Pull(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, "docker", "pull", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker pull failed: %w: %s", err, string(output))
	}
	return nil
}

// Tag creates a tag for an image using docker CLI.
func (d *DockerHubProvider) Tag(ctx context.Context, sourceImage, targetTag string) error {
	cmd := exec.CommandContext(ctx, "docker", "tag", sourceImage, targetTag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker tag failed: %w: %s", err, string(output))
	}
	return nil
}

// ListTags lists tags for a repository using the Docker Registry HTTP API v2.
func (d *DockerHubProvider) ListTags(ctx context.Context, repository string) ([]string, error) {
	url := fmt.Sprintf("%s%s/tags/list", d.baseURL, repository)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	token, err := d.getAuthToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := d.httpClient.Do(req)
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

// Ping checks if Docker Hub is accessible by hitting the /v2/ endpoint.
func (d *DockerHubProvider) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.baseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
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

// getAuthToken obtains a Bearer token from the auth endpoint using Basic Auth.
func (d *DockerHubProvider) getAuthToken(ctx context.Context) (string, error) {
	tokenURL := fmt.Sprintf("%s?service=registry.docker.io&scope=repository:%s:pull",
		d.authURL, "library/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.SetBasicAuth(d.username, d.password)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request auth token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get auth token (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp authTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenResp.Token, nil
}

// tagsResponse represents the Docker Registry API v2 tags/list response.
type tagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// authTokenResponse represents the Docker auth token response.
type authTokenResponse struct {
	Token string `json:"token"`
}
