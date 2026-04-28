package deployer

import (
	"context"
	"fmt"
	"strings"
)

// ComposeDeployer handles docker-compose deployments.
type ComposeDeployer struct {
	executor CommandExecutor
}

// NewComposeDeployer creates a new ComposeDeployer.
func NewComposeDeployer(executor CommandExecutor) *ComposeDeployer {
	return &ComposeDeployer{executor: executor}
}

// ComposeUp writes a compose file and runs docker-compose up -d.
func (d *ComposeDeployer) ComposeUp(ctx context.Context, workDir, composeContent string, envVars map[string]string) (string, error) {
	// Write compose file
	writeCmd := fmt.Sprintf("mkdir -p %s && cat > %s/docker-compose.yml << 'DEPLOYPILOT_COMPOSE_EOF'\n%s\nDEPLOYPILOT_COMPOSE_EOF", workDir, workDir, composeContent)
	if out, err := d.executor.RunCommand(ctx, writeCmd); err != nil {
		return "", fmt.Errorf("failed to write compose file: %w, output: %s", err, out)
	}

	// Build env args
	var envArgs []string
	for k, v := range envVars {
		envArgs = append(envArgs, fmt.Sprintf("%s=%s", k, v))
	}
	envCmd := ""
	if len(envArgs) > 0 {
		envCmd = fmt.Sprintf("export %s && ", strings.Join(envArgs, " "))
	}

	// Run docker-compose up -d
	cmd := fmt.Sprintf("cd %s && %sdocker-compose up -d --remove-orphans 2>&1", workDir, envCmd)
	out, err := d.executor.RunCommand(ctx, cmd)
	if err != nil {
		return out, fmt.Errorf("docker-compose up failed: %w", err)
	}
	return out, nil
}

// ComposeDown stops and removes containers defined in compose file.
func (d *ComposeDeployer) ComposeDown(ctx context.Context, workDir string) (string, error) {
	cmd := fmt.Sprintf("cd %s && docker-compose down --remove-orphans 2>&1", workDir)
	out, err := d.executor.RunCommand(ctx, cmd)
	if err != nil {
		return out, fmt.Errorf("docker-compose down failed: %w", err)
	}
	return out, nil
}

// ComposePs lists containers managed by docker-compose.
func (d *ComposeDeployer) ComposePs(ctx context.Context, workDir string) (string, error) {
	cmd := fmt.Sprintf("cd %s && docker-compose ps --format 'table {{.Name}}\t{{.State}}\t{{.Ports}}' 2>&1", workDir)
	out, err := d.executor.RunCommand(ctx, cmd)
	if err != nil {
		return out, fmt.Errorf("docker-compose ps failed: %w", err)
	}
	return out, nil
}

// ComposeLogs fetches logs from compose services.
func (d *ComposeDeployer) ComposeLogs(ctx context.Context, workDir, service, tail string) (string, error) {
	cmd := fmt.Sprintf("cd %s && docker-compose logs", workDir)
	if service != "" {
		cmd += " " + service
	}
	if tail != "" {
		cmd += " --tail " + tail
	}
	cmd += " 2>&1"
	out, err := d.executor.RunCommand(ctx, cmd)
	if err != nil {
		return out, fmt.Errorf("docker-compose logs failed: %w", err)
	}
	return out, nil
}

// ComposeRestart restarts compose services.
func (d *ComposeDeployer) ComposeRestart(ctx context.Context, workDir, service string) (string, error) {
	cmd := fmt.Sprintf("cd %s && docker-compose restart", workDir)
	if service != "" {
		cmd += " " + service
	}
	cmd += " 2>&1"
	out, err := d.executor.RunCommand(ctx, cmd)
	if err != nil {
		return out, fmt.Errorf("docker-compose restart failed: %w", err)
	}
	return out, nil
}
