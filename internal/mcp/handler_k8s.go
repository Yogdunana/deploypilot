package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"github.com/mark3labs/mcp-go/mcp"
)
func handleListClusters(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tenantID := request.GetString("tenant_id", "tenant-default")

	clusters, err := deployer.ListClusters(ctx, tenantID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list clusters: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"total":   len(clusters),
		"clusters": clusters,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleK8sDeploy(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterID, err := request.RequireString("cluster_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	image, err := request.RequireString("image")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cfg := &K8sDeployConfig{
		Name: name,
		Image: image,
	}

	// Parse replicas
	if r := request.GetString("replicas", ""); r != "" {
		var replicas int32
		_, _ = fmt.Sscanf(r, "%d", &replicas)
		cfg.Replicas = replicas
	}

	// Parse ports
	if p := request.GetString("ports", ""); p != "" {
		for _, ps := range strings.Split(p, ",") {
			ps = strings.TrimSpace(ps)
			var port int32
			if _, err := fmt.Sscanf(ps, "%d", &port); err == nil {
				cfg.Ports = append(cfg.Ports, port)
			}
		}
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

	cfg.Namespace = request.GetString("namespace", "")

	if err := deployer.K8sDeploy(ctx, clusterID, cfg); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("k8s deploy failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("deployment %s created on cluster %s", name, clusterID),
		"deployment": map[string]string{
			"name":       name,
			"image":      image,
			"cluster_id": clusterID,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleK8sListDeployments(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterID, err := request.RequireString("cluster_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.K8sListDeployments(ctx, clusterID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list k8s deployments: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleK8sGetPods(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	clusterID, err := request.RequireString("cluster_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	labelSelector := request.GetString("label_selector", "")

	result, err := deployer.K8sGetPods(ctx, clusterID, labelSelector)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get k8s pods: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
