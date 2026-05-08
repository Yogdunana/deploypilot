package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/util"
)

// getRemoteExecutor creates an SSH executor for the given server.
// It looks up the server record, finds its credential, decrypts the password/key,
// and returns an SSH client that satisfies deployer.CommandExecutor.

// ComposeDeploy deploys an app using docker-compose.
func (b *Bridge) ComposeDeploy(ctx context.Context, appID string) (string, error) {
	var app model.App
	if err := b.DB.First(&app, "id = ?", appID).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	executor, err := b.getRemoteExecutor(ctx, app.ServerID)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	workDir := fmt.Sprintf("/opt/deploypilot/apps/%s", app.Name)
	projectName := app.ComposeProjectName
	if projectName == "" {
		projectName = app.Name
	}

	// Prepend project name to compose content if not present
	composeContent := app.ComposeContent
	if !strings.Contains(composeContent, "name:") && !strings.Contains(composeContent, "version:") {
		// Wrap with project name
		composeContent = fmt.Sprintf("name: %s\n\n%s", projectName, composeContent)
	}

	// Parse env vars from JSON string
	var envVars map[string]string
	if app.EnvVars != "" {
		if err := json.Unmarshal([]byte(app.EnvVars), &envVars); err != nil {
			slog.Warn("failed to unmarshal app env vars", "app_id", app.ID, "error", err)
		}
	}

	deployer := deployer.NewComposeDeployer(executor)
	out, err := deployer.ComposeUp(ctx, workDir, composeContent, envVars)
	if err != nil {
		return out, err
	}

	// Save deployment record
	if b.DB != nil {
		snapshotJSON, _ := json.Marshal(map[string]interface{}{
			"compose_content":     composeContent,
			"compose_project_name": projectName,
			"env_vars":            envVars,
		})
		record := &model.DeploymentRecord{
			ID:             generateID(),
			TenantID:       app.TenantID,
			ServerID:       app.ServerID,
			AppName:        app.Name,
			AppID:          app.ID,
			DeployType:     "compose_up",
			ConfigSnapshot: string(snapshotJSON),
			Status:         "success",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := b.DB.Create(record).Error; err != nil {
			slog.Error("failed to save compose deployment record", "error", err)
		}
	}

	return out, nil
}

// ComposeStop stops a compose deployment.
func (b *Bridge) ComposeStop(ctx context.Context, appID string) (string, error) {
	var app model.App
	if err := b.DB.First(&app, "id = ?", appID).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	executor, err := b.getRemoteExecutor(ctx, app.ServerID)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	workDir := fmt.Sprintf("/opt/deploypilot/apps/%s", app.Name)
	deployer := deployer.NewComposeDeployer(executor)
	return deployer.ComposeDown(ctx, workDir)
}

// ComposePs lists compose containers.
func (b *Bridge) ComposePs(ctx context.Context, appID string) (string, error) {
	var app model.App
	if err := b.DB.First(&app, "id = ?", appID).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	executor, err := b.getRemoteExecutor(ctx, app.ServerID)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	workDir := fmt.Sprintf("/opt/deploypilot/apps/%s", app.Name)
	deployer := deployer.NewComposeDeployer(executor)
	return deployer.ComposePs(ctx, workDir)
}

// ComposeLogs gets compose service logs.
func (b *Bridge) ComposeLogs(ctx context.Context, appID, service, tail string) (string, error) {
	var app model.App
	if err := b.DB.First(&app, "id = ?", appID).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	executor, err := b.getRemoteExecutor(ctx, app.ServerID)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	workDir := fmt.Sprintf("/opt/deploypilot/apps/%s", app.Name)
	deployer := deployer.NewComposeDeployer(executor)
	return deployer.ComposeLogs(ctx, workDir, service, tail)
}

// ComposeRestart restarts compose services.
func (b *Bridge) ComposeRestart(ctx context.Context, appID, service string) (string, error) {
	var app model.App
	if err := b.DB.First(&app, "id = ?", appID).Error; err != nil {
		return "", fmt.Errorf("app not found: %w", err)
	}

	executor, err := b.getRemoteExecutor(ctx, app.ServerID)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	workDir := fmt.Sprintf("/opt/deploypilot/apps/%s", app.Name)
	deployer := deployer.NewComposeDeployer(executor)
	return deployer.ComposeRestart(ctx, workDir, service)
}

// ---------- Phase 3.5: Preflight Visualization ----------

// RunPreflightFull runs all preflight checks without short-circuiting and returns a full report.
func (b *Bridge) ExecCommand(ctx context.Context, serverID, command string, timeout int) (string, error) {
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	if serverID == "" {
		// Local execution
		return b.Executor.RunCommand(execCtx, command)
	}

	// Remote execution via SSH
	remoteExec, err := b.getRemoteExecutor(ctx, serverID)
	if err != nil {
		return "", fmt.Errorf("failed to get remote executor for server %s: %w", serverID, err)
	}
	defer func() {
		if cerr := remoteExec.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	return remoteExec.RunCommand(execCtx, command)
}

// ---------- Phase 3.3: ListImages ----------

func (b *Bridge) ListImages(ctx context.Context, serverID, filter string) (string, error) {
	dockerCmd := `docker images --format "{{.Repository}}:{{.Tag}}\t{{.Size}}\t{{.CreatedSince}}"`
	if filter != "" {
		dockerCmd += " | grep " + util.ShellQuote(filter)
	}

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if serverID == "" {
		return b.Executor.RunCommand(execCtx, dockerCmd)
	}

	remoteExec, err := b.getRemoteExecutor(ctx, serverID)
	if err != nil {
		return "", fmt.Errorf("failed to get remote executor for server %s: %w", serverID, err)
	}
	defer func() {
		if cerr := remoteExec.Close(); cerr != nil {
			slog.Warn("failed to close remote executor", "error", cerr)
		}
	}()

	return remoteExec.RunCommand(execCtx, dockerCmd)
}

// ---------- Phase 3.3: PortForward ----------

func (b *Bridge) PortForward(ctx context.Context, action, serverID string, localPort, remotePort int, remoteHost string) (string, error) {
	switch action {
	case "list":
		b.portForwardMu.RLock()
		defer b.portForwardMu.RUnlock()
		if len(b.portForwards) == 0 {
			return "No active port forwards.", nil
		}
		var lines []string
		for key, pf := range b.portForwards {
			lines = append(lines, fmt.Sprintf("  %s -> server=%s remote=%s:%d (key=%s)", pf.Command, pf.ServerID, pf.RemoteHost, pf.RemotePort, key))
		}
		return fmt.Sprintf("Active port forwards (%d):\n%s", len(b.portForwards), strings.Join(lines, "\n")), nil

	case "create":
		if serverID == "" {
			return "", fmt.Errorf("server_id is required for create action")
		}
		if localPort <= 0 || remotePort <= 0 {
			return "", fmt.Errorf("local_port and remote_port must be positive integers")
		}
		if remoteHost == "" {
			remoteHost = "127.0.0.1"
		}

		key := fmt.Sprintf("%s:%d", serverID, localPort)
		b.portForwardMu.Lock()
		defer b.portForwardMu.Unlock()

		if _, exists := b.portForwards[key]; exists {
			return "", fmt.Errorf("port forward already exists for %s (local port %d is already in use)", key, localPort)
		}

		// Get server info for SSH connection
		row := make(map[string]interface{})
		if err := b.DB.Table("servers").Where("id = ?", serverID).Take(&row).Error; err != nil {
			return "", fmt.Errorf("server not found: %w", err)
		}

		host := toString(row["host"])
		port := toInt(row["port"])
		username := toString(row["username"])
		if username == "" {
			username = os.Getenv("DEPLOYPILOT_SSH_DEFAULT_USER")
		}
		if username == "" {
			return "", fmt.Errorf("SSH username not configured for server %s (configure DEPLOYPILOT_SSH_DEFAULT_USER or set server username)", serverID)
		}

		sshCmd := fmt.Sprintf("ssh -N -L %d:%s:%d -p %d %s@%s",
			localPort, remoteHost, remotePort, port, util.ShellQuote(username), util.ShellQuote(host))

		// Execute SSH tunnel in background
		execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		// We run the SSH command and let it run in the background
		// Since CommandExecutor.RunCommand is blocking, we launch it in a goroutine
		go func() {
			remoteExec, err := b.getRemoteExecutor(ctx, serverID)
			if err != nil {
				slog.Error("failed to create SSH tunnel", "error", err)
				return
			}
			defer func() {
				if cerr := remoteExec.Close(); cerr != nil {
					slog.Warn("failed to close remote executor", "error", cerr)
				}
			}()

			tunnelCmd := fmt.Sprintf("ssh -f -N -L %d:%s:%d -p %d %s@%s",
				localPort, remoteHost, remotePort, port, util.ShellQuote(username), util.ShellQuote(host))
			if _, err := remoteExec.RunCommand(execCtx, tunnelCmd); err != nil {
				slog.Error("SSH tunnel command failed", "error", err)
			}
			cancel()
		}()

		b.portForwards[key] = &portForwardEntry{
			ServerID:   serverID,
			LocalPort:  localPort,
			RemotePort: remotePort,
			RemoteHost: remoteHost,
			Command:    sshCmd,
		}

		return fmt.Sprintf("Port forward created: localhost:%d -> %s:%d:%d (server: %s)", localPort, remoteHost, remotePort, port, serverID), nil

	case "delete":
		if serverID == "" {
			return "", fmt.Errorf("server_id is required for delete action")
		}
		if localPort <= 0 {
			return "", fmt.Errorf("local_port must be a positive integer")
		}

		key := fmt.Sprintf("%s:%d", serverID, localPort)
		b.portForwardMu.Lock()
		defer b.portForwardMu.Unlock()

		entry, exists := b.portForwards[key]
		if !exists {
			return "", fmt.Errorf("port forward not found for %s", key)
		}

		// Kill the SSH tunnel process
		killCmd := fmt.Sprintf("pkill -f 'ssh.*-L %d:%s:%d'", localPort, util.ShellQuote(entry.RemoteHost), entry.RemotePort)
		if _, err := b.Executor.RunCommand(ctx, killCmd); err != nil {
			slog.Warn("failed to kill SSH tunnel process", "error", err)
		}

		delete(b.portForwards, key)
		return fmt.Sprintf("Port forward deleted: %s", key), nil

	default:
		return "", fmt.Errorf("invalid action: %s (must be 'create', 'delete', or 'list')", action)
	}
}
