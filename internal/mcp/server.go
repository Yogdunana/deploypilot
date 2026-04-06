package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Deployer abstracts deployment operations for the MCP server.
type Deployer interface {
	Deploy(ctx context.Context, cfg DeployConfig) (*ContainerStatus, error)
	GetContainerStatus(ctx context.Context, name string) (*ContainerStatus, error)
	ListApps(ctx context.Context) ([]ContainerStatus, error)
	ListServers(ctx context.Context) ([]ServerInfo, error)
	Stop(ctx context.Context, name string) error
	Remove(ctx context.Context, name string) error
	GetContainerLogs(ctx context.Context, name string, tail int) (string, error)
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
