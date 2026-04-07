package deployer

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// CommandExecutor abstracts SSH command execution for testability.
type CommandExecutor interface {
	RunCommand(ctx context.Context, cmd string) (string, error)
}

// DeployConfig holds configuration for a container deployment.
type DeployConfig struct {
	Image         string            `json:"image"`
	ContainerName string            `json:"container_name"`
	Ports         string            `json:"ports,omitempty"`          // e.g. "8080:80"
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	RestartPolicy string            `json:"restart_policy,omitempty"` // unless-stopped, always, no
	Network       string            `json:"network,omitempty"`
	Volumes       string            `json:"volumes,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	ResourceLimits                  // optional
}

// ResourceLimits holds CPU/memory constraints.
type ResourceLimits struct {
	CPU    string `json:"cpu,omitempty"`    // e.g. "2"
	Memory string `json:"memory,omitempty"` // e.g. "4GB"
}

// ContainerStatus represents the status of a deployed container.
type ContainerStatus struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Status    string            `json:"status"` // running, exited, restarting
	Ports     string            `json:"ports"`
	CreatedAt time.Time         `json:"created_at"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// HealthCheckConfig holds health check parameters.
type HealthCheckConfig struct {
	Type     string        `json:"type"`     // http, tcp, none
	Target   string        `json:"target"`   // e.g. "http://localhost:8080/health"
	Interval time.Duration `json:"interval"`
	Timeout  time.Duration `json:"timeout"`
	Retries  int           `json:"retries"`
}

// DockerDeployer handles Docker container deployments via SSH.
type DockerDeployer struct {
	executor CommandExecutor
}

// New creates a new DockerDeployer with the given command executor.
func New(executor CommandExecutor) *DockerDeployer {
	return &DockerDeployer{executor: executor}
}

// Deploy pulls an image and runs a container on the remote server.
func (d *DockerDeployer) Deploy(ctx context.Context, cfg DeployConfig) (*ContainerStatus, error) {
	if cfg.Image == "" {
		return nil, fmt.Errorf("image is required")
	}
	if cfg.ContainerName == "" {
		return nil, fmt.Errorf("container_name is required")
	}

	// Step 1: Pull image
	pullCmd := fmt.Sprintf("docker pull %s", cfg.Image)
	if _, err := d.executor.RunCommand(ctx, pullCmd); err != nil {
		return nil, fmt.Errorf("failed to pull image %s: %w", cfg.Image, err)
	}

	// Step 2: Remove existing container with same name (if any)
	_, _ = d.executor.RunCommand(ctx, fmt.Sprintf("docker rm -f %s 2>/dev/null || true", cfg.ContainerName))

	// Step 3: Build docker run command
	runCmd := d.buildRunCommand(cfg)

	// Step 4: Run container
	output, err := d.executor.RunCommand(ctx, runCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run container: %w", err)
	}

	// Extract container ID from output
	containerID := strings.TrimSpace(output)
	if containerID == "" {
		containerID = cfg.ContainerName
	}

	// Step 5: Get container status
	status, err := d.GetContainerStatus(ctx, cfg.ContainerName)
	if err != nil {
		// Return minimal status if inspection fails
		return &ContainerStatus{
			ID:     containerID,
			Name:   cfg.ContainerName,
			Image:  cfg.Image,
			Status: "created",
		}, nil
	}

	return status, nil
}

// GetContainerStatus returns the status of a container by name.
func (d *DockerDeployer) GetContainerStatus(ctx context.Context, name string) (*ContainerStatus, error) {
	cmd := fmt.Sprintf("docker inspect --format '{{.Id}}|{{.Name}}|{{.Config.Image}}|{{.State.Status}}|{{.Created}}' %s 2>/dev/null", name)
	output, err := d.executor.RunCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", name, err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("container %s not found", name)
	}

	parts := strings.Split(output, "|")
	if len(parts) < 5 {
		return nil, fmt.Errorf("unexpected inspect output format: %s", output)
	}

	createdAt, err := time.Parse(time.RFC3339, parts[4])
	if err != nil {
		createdAt = time.Now()
	}

	return &ContainerStatus{
		ID:        parts[0],
		Name:      strings.TrimPrefix(parts[1], "/"),
		Image:     parts[2],
		Status:    parts[3],
		CreatedAt: createdAt,
	}, nil
}

// Stop stops a running container.
func (d *DockerDeployer) Stop(ctx context.Context, name string) error {
	_, err := d.executor.RunCommand(ctx, fmt.Sprintf("docker stop %s", name))
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", name, err)
	}
	return nil
}

// Remove removes a container.
func (d *DockerDeployer) Remove(ctx context.Context, name string) error {
	_, err := d.executor.RunCommand(ctx, fmt.Sprintf("docker rm -f %s", name))
	if err != nil {
		return fmt.Errorf("failed to remove container %s: %w", name, err)
	}
	return nil
}

// GetContainerLogs returns logs for a container.
func (d *DockerDeployer) GetContainerLogs(ctx context.Context, name string, tail int) (string, error) {
	cmd := fmt.Sprintf("docker logs --tail %d %s 2>&1", tail, name)
	output, err := d.executor.RunCommand(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get logs for %s: %w", name, err)
	}
	return output, nil
}

// HealthCheck performs a health check on a deployed container.
func (d *DockerDeployer) HealthCheck(ctx context.Context, cfg HealthCheckConfig) error {
	if cfg.Type == "none" || cfg.Type == "" {
		return nil
	}

	if cfg.Retries == 0 {
		cfg.Retries = 3
	}
	if cfg.Interval == 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 3 * time.Second
	}

	var lastErr error
	for i := 0; i < cfg.Retries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		checkCmd := d.buildHealthCheckCommand(cfg)
		_, lastErr = d.executor.RunCommand(ctx, checkCmd)
		if lastErr == nil {
			return nil
		}

		if i < cfg.Retries-1 {
			time.Sleep(cfg.Interval)
		}
	}

	return fmt.Errorf("health check failed after %d retries: %w", cfg.Retries, lastErr)
}

// buildRunCommand constructs a docker run command from config.
func (d *DockerDeployer) buildRunCommand(cfg DeployConfig) string {
	var args []string

	args = append(args, "docker run -d")

	// Container name
	args = append(args, fmt.Sprintf("--name %s", cfg.ContainerName))

	// Restart policy
	if cfg.RestartPolicy == "" {
		cfg.RestartPolicy = "unless-stopped"
	}
	args = append(args, fmt.Sprintf("--restart %s", cfg.RestartPolicy))

	// Ports
	if cfg.Ports != "" {
		args = append(args, fmt.Sprintf("-p %s", cfg.Ports))
	}

	// Volumes
	if cfg.Volumes != "" {
		args = append(args, fmt.Sprintf("-v %s", cfg.Volumes))
	}

	// Environment variables
	for k, v := range cfg.EnvVars {
		args = append(args, fmt.Sprintf("-e %s=%s", k, v))
	}

	// Labels
	for k, v := range cfg.Labels {
		args = append(args, fmt.Sprintf("-l %s=%s", k, v))
	}

	// Resource limits
	if cfg.CPU != "" {
		args = append(args, fmt.Sprintf("--cpus %s", cfg.CPU))
	}
	if cfg.Memory != "" {
		args = append(args, fmt.Sprintf("--memory %s", cfg.Memory))
	}

	// Network
	if cfg.Network != "" {
		args = append(args, fmt.Sprintf("--network %s", cfg.Network))
	}

	// Image
	args = append(args, cfg.Image)

	return strings.Join(args, " ")
}

// buildHealthCheckCommand constructs a health check command.
func (d *DockerDeployer) buildHealthCheckCommand(cfg HealthCheckConfig) string {
	switch cfg.Type {
	case "http":
		return fmt.Sprintf("curl -sf --max-time %d %s", int(cfg.Timeout.Seconds()), cfg.Target)
	case "tcp":
		return fmt.Sprintf("timeout %d bash -c 'echo > /dev/tcp/%s' 2>/dev/null", int(cfg.Timeout.Seconds()), strings.TrimPrefix(cfg.Target, "tcp://"))
	default:
		return "echo ok"
	}
}
