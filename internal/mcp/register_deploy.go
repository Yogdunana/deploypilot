package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerDeployTools registers deploy tools.
func registerDeployTools(s *server.MCPServer, d Deployer) {
	deployTool := mcp.NewTool("deploy_app",
		mcp.WithDescription("Deploy a Docker container. Use server_id to deploy to a remote server via SSH; omit for local Docker deployment."),
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
		mcp.WithString("server_id",
			mcp.Description("Target server ID for remote deployment via SSH. Omit to deploy locally. The server must be registered via add_server first."),
		),
	)

	s.AddTool(deployTool, withPermissionCheck("deploy_app", withValidation("deploy_app", deployTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeployApp(ctx, d, request)
	})))
	statusTool := mcp.NewTool("get_deploy_status",
		mcp.WithDescription("Get the status of a deployed container"),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name of the container to check"),
		),
	)

	s.AddTool(statusTool, withPermissionCheck("get_deploy_status", withValidation("get_deploy_status", statusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetDeployStatus(ctx, d, request)
	})))
	listAppsTool := mcp.NewTool("list_apps",
		mcp.WithDescription("List all deployed applications"),
	)

	s.AddTool(listAppsTool, withPermissionCheck("list_apps", withValidation("list_apps", listAppsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListApps(ctx, d, request)
	})))
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

	s.AddTool(createAppTool, withPermissionCheck("create_app", withValidation("create_app", createAppTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateApp(ctx, d, request)
	})))
	deleteAppTool := mcp.NewTool("delete_app",
		mcp.WithDescription("Delete an application and stop its container"),
		mcp.WithString("app_id",
			mcp.Required(),
			mcp.Description("ID of the application to delete"),
		),
	)

	s.AddTool(deleteAppTool, withPermissionCheck("delete_app", withValidation("delete_app", deleteAppTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteApp(ctx, d, request)
	})))
	rollbackTool := mcp.NewTool("rollback_app",
		mcp.WithDescription("Rollback a container to a previous image version. If previous_image is omitted, automatically resolves the last successful deployment image."),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name of the container to rollback"),
		),
		mcp.WithString("previous_image",
			mcp.Description("Previous Docker image to rollback to (optional — auto-resolves from deployment history if omitted)"),
		),
	)

	s.AddTool(rollbackTool, withPermissionCheck("rollback_app", withValidation("rollback_app", rollbackTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRollback(ctx, d, request)
	})))
	buildAndDeployTool := mcp.NewTool("build_and_deploy",
		mcp.WithDescription("Build and deploy an application from git source. Clones repo, detects tech stack, generates Dockerfile, builds image, and deploys container."),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Git repository URL")),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Application name (used for container name and image tag)")),
		mcp.WithString("branch", mcp.Description("Git branch (default: main)")),
		mcp.WithString("tech_stack", mcp.Description("Tech stack template (auto-detect if empty): node, python, go, java, php, ruby, rust, static, docker")),
		mcp.WithString("ports", mcp.Description("Port mappings (e.g. '8080:80')")),
		mcp.WithString("server_id", mcp.Description("Target server ID (deploy locally if empty)")),
		mcp.WithString("env_vars", mcp.Description("JSON object of environment variables")),
		mcp.WithString("registry_id", mcp.Description("Registry ID to push the built image to")),
		mcp.WithBoolean("push_image", mcp.Description("Whether to push the built image to the registry after build")),
		mcp.WithString("image_tag", mcp.Description("Custom image tag (default: appname:latest)")),
	)
	s.AddTool(buildAndDeployTool, withPermissionCheck("build_and_deploy", withValidation("build_and_deploy", buildAndDeployTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBuildAndDeploy(ctx, d, request)
	})))
	getAppDetailTool := mcp.NewTool("get_app_detail",
		mcp.WithDescription("Get detailed information about a deployed application"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("Application ID")),
	)
	s.AddTool(getAppDetailTool, withPermissionCheck("get_app_detail", withValidation("get_app_detail", getAppDetailTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAppDetail(ctx, d, request)
	})))
	updateAppTool := mcp.NewTool("update_app",
		mcp.WithDescription("Update application configuration"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("Application ID")),
		mcp.WithString("config", mcp.Required(), mcp.Description("JSON string of configuration to update")),
	)
	s.AddTool(updateAppTool, withPermissionCheck("update_app", withValidation("update_app", updateAppTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateApp(ctx, d, request)
	})))
	checkReadinessTool := mcp.NewTool("check_deploy_readiness",
		mcp.WithDescription("Check if deployment prerequisites are met"),
		mcp.WithString("app_config", mcp.Required(), mcp.Description("JSON string of app configuration to validate")),
	)
	s.AddTool(checkReadinessTool, withPermissionCheck("check_deploy_readiness", withValidation("check_deploy_readiness", checkReadinessTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCheckDeployReadiness(ctx, d, request)
	})))
	batchDeployTool := mcp.NewTool("batch_deploy",
		mcp.WithDescription("Deploy multiple applications at once with configurable strategy (sequential, parallel, rolling)"),
		mcp.WithString("apps", mcp.Required(), mcp.Description("JSON array of app configs: [{repo, branch, domain, stack}]")),
		mcp.WithString("strategy", mcp.Description("Deployment strategy: sequential (default), parallel, or rolling")),
		mcp.WithNumber("max_concurrent", mcp.Description("Max concurrent deployments for parallel strategy (default: 5)")),
		mcp.WithNumber("batch_size", mcp.Description("Batch size for rolling strategy (default: 3)")),
		mcp.WithString("server_ids", mcp.Description("Comma-separated target server IDs")),
	)
	s.AddTool(batchDeployTool, withPermissionCheck("batch_deploy", withValidation("batch_deploy", batchDeployTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBatchDeploy(ctx, d, request)
	})))
	healContainerTool := mcp.NewTool("heal_container",
		mcp.WithDescription("Trigger self-healing for a container. Inspects the container state and takes corrective action (restart or rollback) if needed."),
		mcp.WithString("container_name", mcp.Required(), mcp.Description("Name of the container to heal")),
	)
	s.AddTool(healContainerTool, withPermissionCheck("heal_container", withValidation("heal_container", healContainerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleHealContainer(ctx, d, request)
	})))
	listImagesTool := mcp.NewTool("list_images",
		mcp.WithDescription("List Docker images on a server. Runs locally if server_id is omitted, or remotely via SSH if server_id is provided."),
		mcp.WithString("server_id", mcp.Description("Target server ID (omit for local execution)")),
		mcp.WithString("filter", mcp.Description("Grep filter to apply to the output")),
	)
	s.AddTool(listImagesTool, withPermissionCheck("list_images", withValidation("list_images", listImagesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListImages(ctx, d, request)
	})))

}
