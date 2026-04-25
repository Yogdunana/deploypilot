package builder

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/model"
)

// dockerRunner abstracts docker CLI execution for testability.
type dockerRunner interface {
	Run(ctx context.Context, name string, args ...string) error
	RunWithStdin(ctx context.Context, name string, stdin io.Reader, args ...string) error
}

// realDockerRunner executes actual docker commands.
type realDockerRunner struct{}

func (r *realDockerRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func (r *realDockerRunner) RunWithStdin(ctx context.Context, name string, stdin io.Reader, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// registryCredentials holds resolved registry authentication info.
type registryCredentials struct {
	url      string
	username string
	password string
}

// resolveRegistryCredentials determines the registry URL and credentials
// from the build config, either by looking up a stored registry by ID or
// by using the directly provided overrides.
func resolveRegistryCredentials(config *BuildConfig) (*registryCredentials, error) {
	if config.RegistryID != "" {
		reg, err := model.GetRegistry(config.RegistryID)
		if err != nil {
			return nil, fmt.Errorf("failed to load registry %s: %w", config.RegistryID, err)
		}
		return &registryCredentials{
			url:      reg.URL,
			username: reg.Username,
			password: reg.Password,
		}, nil
	}

	if config.RegistryURL != "" {
		return &registryCredentials{
			url:      config.RegistryURL,
			username: config.RegistryUser,
			password: config.RegistryPass,
		}, nil
	}

	return nil, fmt.Errorf("no registry configured: set registry_id or registry_url")
}

// buildRemoteTag constructs the full remote image reference.
// For Docker Hub (docker.io): user/image:tag
// For GHCR (ghcr.io): ghcr.io/owner/image:tag
// For custom registries: registry-url/image:tag
func buildRemoteTag(regURL, appName, tag string) string {
	host := strings.TrimPrefix(regURL, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")
	host = strings.TrimSuffix(host, "/v1/")
	host = strings.TrimSuffix(host, "/v2/")

	switch {
	case host == "docker.io" || host == "registry-1.docker.io" || host == "registry.hub.docker.com":
		// Docker Hub: username/imagename:tag
		return fmt.Sprintf("%s/%s:%s", appName, appName, tag)
	case strings.Contains(host, "ghcr.io"):
		// GHCR: ghcr.io/owner/image:tag
		return fmt.Sprintf("%s/%s:%s", host, appName, tag)
	default:
		// Custom registry
		return fmt.Sprintf("%s/%s:%s", host, appName, tag)
	}
}

// PushImage pushes a Docker image to a configured registry.
// It returns the remote image reference on success, or an error.
func (b *Builder) PushImage(ctx context.Context, config *BuildConfig, localImage string) (string, error) {
	creds, err := resolveRegistryCredentials(config)
	if err != nil {
		return "", err
	}

	runner := b.dockerRunner
	if runner == nil {
		runner = &realDockerRunner{}
	}

	// Determine the tag to use for the remote image
	tag := config.ImageTag
	if tag == "" {
		tag = "latest"
	}

	remoteTag := buildRemoteTag(creds.url, config.AppName, tag)

	// Step 1: Tag the local image with the remote reference
	if err := runner.Run(ctx, "docker", "tag", localImage, remoteTag); err != nil {
		return "", fmt.Errorf("docker tag failed: %w", err)
	}

	// Step 2: Login to the registry (pipe password via stdin for security)
	loginHost := strings.TrimPrefix(creds.url, "https://")
	loginHost = strings.TrimPrefix(loginHost, "http://")
	loginHost = strings.TrimSuffix(loginHost, "/")

	if creds.username != "" {
		stdin := strings.NewReader(creds.password)
		if err := runner.RunWithStdin(ctx, "docker", stdin, "login", "-u", creds.username, "--password-stdin", loginHost); err != nil {
			return "", fmt.Errorf("docker login failed: %w", err)
		}
	}

	// Step 3: Push the image
	if err := runner.Run(ctx, "docker", "push", remoteTag); err != nil {
		return "", fmt.Errorf("docker push failed: %w", err)
	}

	return remoteTag, nil
}
