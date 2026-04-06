package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Deployer abstracts deployment operations for the MCP server.
type Deployer interface {
	Deploy(ctx context.Context, cfg DeployConfig) (*ContainerStatus, error)
	GetContainerStatus(ctx context.Context, name string) (*ContainerStatus, error)
	ListApps(ctx context.Context) ([]ContainerStatus, error)
	ListServers(ctx context.Context) ([]ServerInfo, error)
	CreateApp(ctx context.Context, cfg CreateAppConfig) (string, error)
	DeleteApp(ctx context.Context, appID string) error
	Stop(ctx context.Context, name string) error
	Remove(ctx context.Context, name string) error
	GetContainerLogs(ctx context.Context, name string, tail int) (string, error)
	Rollback(ctx context.Context, containerName, previousImage string) (*ContainerStatus, error)
	Backup(ctx context.Context, appID string) (string, error)
	Restore(ctx context.Context, backupID string) (*ContainerStatus, error)
	DetectEnv(ctx context.Context, level int, ports []int, services []string) (interface{}, error)
	HealthCheck(ctx context.Context, target, healthType string) (interface{}, error)
}

// DeployConfig mirrors deployer.DeployConfig to avoid circular imports.
type DeployConfig struct {
	Image         string            `json:"image"`
	ContainerName string            `json:"container_name"`
	Ports         string            `json:"ports,omitempty"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	RestartPolicy string            `json:"restart_policy,omitempty"`
	Network       string            `json:"network,omitempty"`
	Volumes       string            `json:"volumes,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	CPU           string            `json:"cpu,omitempty"`
	Memory        string            `json:"memory,omitempty"`
}

// ContainerStatus mirrors deployer.ContainerStatus.
type ContainerStatus struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Status    string            `json:"status"`
	Ports     string            `json:"ports,omitempty"`
	CreatedAt string            `json:"created_at,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ServerInfo represents a registered server.
type ServerInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Status string `json:"status"`
}

// CreateAppConfig holds parameters for creating a new application.
type CreateAppConfig struct {
	Name       string `json:"name"`
	RepoURL    string `json:"repo_url"`
	Branch     string `json:"branch,omitempty"`
	Domain     string `json:"domain,omitempty"`
	TechStack  string `json:"tech_stack,omitempty"`
	DeployMode string `json:"deploy_mode,omitempty"`
	ServerID   string `json:"server_id,omitempty"`
}

// NewServer creates a new MCP server with deploy tools registered.
func NewServer(deployer Deployer) *server.MCPServer {
	s := server.NewMCPServer(
		"DeployPilot",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	// Register deploy_app tool
	deployTool := mcp.NewTool("deploy_app",
		mcp.WithDescription("Deploy a Docker container on a remote server"),
		mcp.WithString("image",
			mcp.Required(),
			mcp.Description("Docker image to deploy (e.g. nginx:latest)"),
		),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name for the container"),
		),
		mcp.WithString("ports",
			mcp.Description("Port mapping (e.g. 8080:80)"),
		),
		mcp.WithString("env_vars",
			mcp.Description("Environment variables as JSON object (e.g. {\"KEY\":\"value\"})"),
		),
		mcp.WithString("restart_policy",
			mcp.Description("Restart policy: unless-stopped, always, no"),
		),
		mcp.WithString("network",
			mcp.Description("Docker network to attach"),
		),
		mcp.WithString("volumes",
			mcp.Description("Volume mappings (e.g. /host/path:/container/path)"),
		),
		mcp.WithString("labels",
			mcp.Description("Labels as JSON object (e.g. {\"app\":\"myapp\"})"),
		),
		mcp.WithString("cpu",
			mcp.Description("CPU limit (e.g. 2)"),
		),
		mcp.WithString("memory",
			mcp.Description("Memory limit (e.g. 4GB)"),
		),
	)

	s.AddTool(deployTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeployApp(ctx, deployer, request)
	})

	// Register get_deploy_status tool
	statusTool := mcp.NewTool("get_deploy_status",
		mcp.WithDescription("Get the status of a deployed container"),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name of the container to check"),
		),
	)

	s.AddTool(statusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetDeployStatus(ctx, deployer, request)
	})

	// Register list_apps tool
	listAppsTool := mcp.NewTool("list_apps",
		mcp.WithDescription("List all deployed applications"),
	)

	s.AddTool(listAppsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListApps(ctx, deployer, request)
	})

	// Register list_servers tool
	listServersTool := mcp.NewTool("list_servers",
		mcp.WithDescription("List all registered servers"),
	)

	s.AddTool(listServersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListServers(ctx, deployer, request)
	})

	// Register create_app tool
	createAppTool := mcp.NewTool("create_app",
		mcp.WithDescription("Register a new application for deployment"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Application name"),
		),
		mcp.WithString("repo_url",
			mcp.Required(),
			mcp.Description("Git repository URL"),
		),
		mcp.WithString("branch",
			mcp.Description("Git branch (default: main)"),
		),
		mcp.WithString("domain",
			mcp.Description("Custom domain for the app"),
		),
		mcp.WithString("tech_stack",
			mcp.Description("Technology stack: docker, node, python, go, java"),
		),
		mcp.WithString("deploy_mode",
			mcp.Description("Deploy mode: api, direct, cicd"),
		),
		mcp.WithString("server_id",
			mcp.Description("Target server ID"),
		),
	)

	s.AddTool(createAppTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateApp(ctx, deployer, request)
	})

	// Register delete_app tool
	deleteAppTool := mcp.NewTool("delete_app",
		mcp.WithDescription("Delete an application and stop its container"),
		mcp.WithString("app_id",
			mcp.Required(),
			mcp.Description("ID of the application to delete"),
		),
	)

	s.AddTool(deleteAppTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteApp(ctx, deployer, request)
	})

	// Register rollback tool
	rollbackTool := mcp.NewTool("rollback",
		mcp.WithDescription("Rollback a container to a previous image version"),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name of the container to rollback"),
		),
		mcp.WithString("previous_image",
			mcp.Required(),
			mcp.Description("Previous Docker image to rollback to"),
		),
	)

	s.AddTool(rollbackTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRollback(ctx, deployer, request)
	})

	// Register backup tool
	backupTool := mcp.NewTool("backup",
		mcp.WithDescription("Create a backup of an application"),
		mcp.WithString("app_id",
			mcp.Required(),
			mcp.Description("ID of the application to backup"),
		),
	)

	s.AddTool(backupTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBackup(ctx, deployer, request)
	})

	// Register restore tool
	restoreTool := mcp.NewTool("restore",
		mcp.WithDescription("Restore an application from a backup"),
		mcp.WithString("backup_id",
			mcp.Required(),
			mcp.Description("ID of the backup to restore from"),
		),
	)

	s.AddTool(restoreTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRestore(ctx, deployer, request)
	})

	// Register get_app_logs tool
	getLogsTool := mcp.NewTool("get_app_logs",
		mcp.WithDescription("Get logs from a deployed container"),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name of the container"),
		),
		mcp.WithString("tail",
			mcp.Description("Number of lines to retrieve (default: 100)"),
		),
	)

	s.AddTool(getLogsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAppLogs(ctx, deployer, request)
	})

	// Register detect_env tool
	detectEnvTool := mcp.NewTool("detect_env",
		mcp.WithDescription("Detect server environment (OS, Docker, ports, services)"),
		mcp.WithString("level",
			mcp.Description("Detection level: 1=OS, 2=+Docker, 3=+Ports, 4=+Services (default: 2)"),
		),
		mcp.WithString("ports",
			mcp.Description("Comma-separated port list to check (e.g. 8080,3000)"),
		),
		mcp.WithString("services",
			mcp.Description("Comma-separated service URLs to check (e.g. tcp://localhost:3306)"),
		),
	)

	s.AddTool(detectEnvTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDetectEnv(ctx, deployer, request)
	})

	// Register health_check tool
	healthCheckTool := mcp.NewTool("health_check",
		mcp.WithDescription("Check health of a deployed service"),
		mcp.WithString("target",
			mcp.Required(),
			mcp.Description("Health check target URL (e.g. http://localhost:8080/health or tcp://localhost:3306)"),
		),
		mcp.WithString("type",
			mcp.Description("Health check type: http or tcp (default: http)"),
		),
	)

	s.AddTool(healthCheckTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleHealthCheck(ctx, deployer, request)
	})

	return s
}

func handleDeployApp(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	image, err := request.RequireString("image")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cfg := DeployConfig{
		Image:         image,
		ContainerName: containerName,
	}

	if v := request.GetString("ports", ""); v != "" {
		cfg.Ports = v
	}
	if v := request.GetString("restart_policy", ""); v != "" {
		cfg.RestartPolicy = v
	}
	if v := request.GetString("network", ""); v != "" {
		cfg.Network = v
	}
	if v := request.GetString("volumes", ""); v != "" {
		cfg.Volumes = v
	}
	if v := request.GetString("cpu", ""); v != "" {
		cfg.CPU = v
	}
	if v := request.GetString("memory", ""); v != "" {
		cfg.Memory = v
	}

	// Parse env vars JSON
	if envStr := request.GetString("env_vars", ""); envStr != "" {
		var envVars map[string]string
		if err := json.Unmarshal([]byte(envStr), &envVars); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid env_vars JSON: %v", err)), nil
		}
		cfg.EnvVars = envVars
	}

	// Parse labels JSON
	if labelsStr := request.GetString("labels", ""); labelsStr != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsStr), &labels); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid labels JSON: %v", err)), nil
		}
		cfg.Labels = labels
	}

	status, err := deployer.Deploy(ctx, cfg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("deploy failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Container %s deployed successfully", containerName),
		"container": map[string]string{
			"id":     status.ID,
			"name":   status.Name,
			"image":  status.Image,
			"status": status.Status,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListApps(ctx context.Context, deployer Deployer, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	apps, err := deployer.ListApps(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list apps: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"total":  len(apps),
		"apps":   apps,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListServers(ctx context.Context, deployer Deployer, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	servers, err := deployer.ListServers(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list servers: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"total":   len(servers),
		"servers": servers,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleCreateApp(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	repoURL, err := request.RequireString("repo_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cfg := CreateAppConfig{
		Name:    name,
		RepoURL: repoURL,
		Branch:  request.GetString("branch", "main"),
		Domain:  request.GetString("domain", ""),
		TechStack: request.GetString("tech_stack", "docker"),
		DeployMode: request.GetString("deploy_mode", "api"),
		ServerID: request.GetString("server_id", ""),
	}

	appID, err := deployer.CreateApp(ctx, cfg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create app: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Application %s created successfully", name),
		"app": map[string]string{
			"id":       appID,
			"name":     name,
			"repo_url": repoURL,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDeleteApp(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := deployer.DeleteApp(ctx, appID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete app: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Application %s deleted successfully", appID),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleRollback(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	previousImage, err := request.RequireString("previous_image")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status, err := deployer.Rollback(ctx, containerName, previousImage)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("rollback failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Container %s rolled back to %s", containerName, previousImage),
		"container": map[string]string{
			"id":     status.ID,
			"name":   status.Name,
			"image":  status.Image,
			"status": status.Status,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleBackup(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	backupID, err := deployer.Backup(ctx, appID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("backup failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Backup created for app %s", appID),
		"backup": map[string]string{
			"id": backupID,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleRestore(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	backupID, err := request.RequireString("backup_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status, err := deployer.Restore(ctx, backupID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("restore failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Application restored from backup %s", backupID),
		"container": map[string]string{
			"id":     status.ID,
			"name":   status.Name,
			"image":  status.Image,
			"status": status.Status,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetAppLogs(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tail := 100
	if t := request.GetString("tail", ""); t != "" {
		fmt.Sscanf(t, "%d", &tail)
	}

	logs, err := deployer.GetContainerLogs(ctx, containerName, tail)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get logs: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"container_name": containerName,
		"tail":  tail,
		"logs":  logs,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDetectEnv(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	level := 2
	if l := request.GetString("level", "2"); l != "" {
		fmt.Sscanf(l, "%d", &level)
	}

	var ports []int
	if p := request.GetString("ports", ""); p != "" {
		for _, ps := range strings.Split(p, ",") {
			ps = strings.TrimSpace(ps)
			var port int
			if _, err := fmt.Sscanf(ps, "%d", &port); err == nil {
				ports = append(ports, port)
			}
		}
	}

	var services []string
	if s := request.GetString("services", ""); s != "" {
		services = strings.Split(s, ",")
		for i := range services {
			services[i] = strings.TrimSpace(services[i])
		}
	}

	env, err := deployer.DetectEnv(ctx, level, ports, services)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("environment detection failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":      "success",
		"environment": env,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleHealthCheck(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	target, err := request.RequireString("target")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	healthType := request.GetString("type", "http")

	health, err := deployer.HealthCheck(ctx, target, healthType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("health check failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"health": health,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetDeployStatus(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status, err := deployer.GetContainerStatus(ctx, containerName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get status: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"container": map[string]string{
			"id":     status.ID,
			"name":   status.Name,
			"image":  status.Image,
			"status": status.Status,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
