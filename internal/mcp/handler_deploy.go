package mcp

import (
	"context"
	"fmt",
	"strconv",
	"strings",
	"github.com/mark3labs/mcp-go/mcp"
)
func handleBuildAndDeploy(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoURL, err := request.RequireString("repo_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	appName, err := request.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	branch := request.GetString("branch", "main")
	techStack := request.GetString("tech_stack", "")
	ports := request.GetString("ports", "")
	serverID := request.GetString("server_id", "")
	envVarsStr := request.GetString("env_vars", "")

	var envVars map[string]string
	if envVarsStr != "" {
		if err := json.Unmarshal([]byte(envVarsStr), &envVars); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid env_vars JSON: %v", err)), nil
		}
	}

	cfg := BuildAndDeployConfig{
		RepoURL:   repoURL,
		AppName:   appName,
		Branch:    branch,
		TechStack: techStack,
		Ports:     ports,
		ServerID:  serverID,
		EnvVars:   envVars,
		RegistryID: request.GetString("registry_id", ""),
		ImageTag:   request.GetString("image_tag", ""),
	}

	if v := request.GetString("push_image", ""); v != "" {
		cfg.PushImage = strings.ToLower(v) == "true"
	}

	result, err := deployer.BuildAndDeploy(ctx, cfg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("build and deploy failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleDeployApp(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	image, err := request.RequireString("image")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	// M-13: Validate image registry
	if err := validateImageRegistry(image); err != nil {
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
	// M-12: Validate volume paths to prevent path traversal
	if cfg.Volumes != "" {
		volParts := strings.Split(cfg.Volumes, ",")
		for _, vol := range volParts {
			vol = strings.TrimSpace(vol)
			if vol == "" {
				continue
			}
			pathParts := strings.SplitN(vol, ":", 2)
			if len(pathParts) >= 1 && pathParts[0] != "" {
				if err := validateVolumePath(pathParts[0]); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("invalid volume path: %v", err)), nil
				}
			}
		}
	}
	if v := request.GetString("cpu", ""); v != "" {
		cfg.CPU = v
	}
	if v := request.GetString("memory", ""); v != "" {
		cfg.Memory = v
	}
	if v := request.GetString("server_id", ""); v != "" {
		cfg.ServerID = v
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
		// Check if it's a preflight error for structured output
		var pfErr PreflightErrorInfo
		if errors.As(err, &pfErr) {
			pfData := map[string]interface{}{
				"status":  "preflight_failed",
				"code":    pfErr.PreflightCode(),
				"message": pfErr.PreflightMessage(),
				"checks":  pfErr.PreflightChecks(),
			}
			data, _ := json.MarshalIndent(pfData, "", "  ")
			return mcp.NewToolResultError(string(data)), nil
		}
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

	// previous_image is now optional — auto-resolves from deployment history
	previousImage := request.GetString("previous_image", "")

	status, err := deployer.Rollback(ctx, containerName, previousImage)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("rollback failed: %v", err)), nil
	}

	resolvedImage := previousImage
	if resolvedImage == "" {
		resolvedImage = "(auto-resolved)"
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Container %s rolled back to %s", containerName, resolvedImage),
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
func handleGetAppDetail(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	detail, err := deployer.GetAppDetail(ctx, appID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get app detail: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "app": detail}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleUpdateApp(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	configStr, err := request.RequireString("config")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid config JSON: %v", err)), nil
	}

	updated, err := deployer.UpdateApp(ctx, appID, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update app: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "message": fmt.Sprintf("App %s updated", appID), "app": updated}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleCheckDeployReadiness(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configStr, err := request.RequireString("app_config")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var appConfig map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &appConfig); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid app_config JSON: %v", err)), nil
	}
	res, err := deployer.CheckDeployReadiness(ctx, appConfig)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("readiness check failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "readiness": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
func handleBatchDeploy(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appsStr, err := request.RequireString("apps")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var apps []map[string]interface{}
	if err := json.Unmarshal([]byte(appsStr), &apps); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid apps JSON: %v", err)), nil
	}

	// Parse optional strategy parameters
	strategy := DeployStrategy(request.GetString("strategy", "sequential"))
	maxConcurrent := 0
	if s := request.GetString("max_concurrent", ""); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			maxConcurrent = v
		}
	}
	batchSize := 0
	if s := request.GetString("batch_size", ""); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			batchSize = v
		}
	}
	var serverIDs []string
	if sidStr := request.GetString("server_ids", ""); sidStr != "" {
		for _, s := range strings.Split(sidStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				serverIDs = append(serverIDs, s)
			}
		}
	}

	config := BatchDeployConfig{
		Apps:          apps,
		Strategy:      strategy,
		MaxConcurrent: maxConcurrent,
		BatchSize:     batchSize,
		ServerIDs:     serverIDs,
	}

	res, err := deployer.BatchDeployWithConfig(ctx, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("batch deploy failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "batch": res}
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

	// Add preflight summary from latest deployment record
	if record, err := deployer.GetLatestDeploymentRecord(ctx, containerName); err == nil && record.PreflightCode != "" {
		result["last_preflight"] = map[string]interface{}{
			"status":  record.Status,
			"code":    record.PreflightCode,
			"message": record.PreflightMessage,
			"time":    record.CreatedAt,
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
