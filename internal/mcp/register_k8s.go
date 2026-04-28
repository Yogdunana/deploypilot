package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerK8sTools registers k8s tools.
func registerK8sTools(s *server.MCPServer, d Deployer) {
	listClustersTool := mcp.NewTool("list_clusters",
		mcp.WithDescription("List all Kubernetes clusters"),
		mcp.WithString("tenant_id", mcp.Description("Tenant ID (default: tenant-default)")),
	)
	s.AddTool(listClustersTool, withPermissionCheck("list_clusters", withValidation("list_clusters", listClustersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListClusters(ctx, d, request)
	})))
	k8sDeployTool := mcp.NewTool("k8s_deploy",
		mcp.WithDescription("Deploy an application to a Kubernetes cluster"),
		mcp.WithString("cluster_id", mcp.Required(), mcp.Description("Kubernetes cluster ID")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Deployment name")),
		mcp.WithString("image", mcp.Required(), mcp.Description("Container image (e.g. nginx:latest)")),
		mcp.WithString("replicas", mcp.Description("Number of replicas (default: 1)")),
		mcp.WithString("ports", mcp.Description("Comma-separated container ports (e.g. 8080,3000)")),
		mcp.WithString("env_vars", mcp.Description("Environment variables as JSON object (e.g. {\"KEY\":\"value\"})")),
		mcp.WithString("labels", mcp.Description("Labels as JSON object (e.g. {\"app\":\"myapp\"})")),
		mcp.WithString("namespace", mcp.Description("Target namespace (overrides cluster default)")),
	)
	s.AddTool(k8sDeployTool, withPermissionCheck("k8s_deploy", withValidation("k8s_deploy", k8sDeployTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleK8sDeploy(ctx, d, request)
	})))
	k8sListDeploymentsTool := mcp.NewTool("k8s_list_deployments",
		mcp.WithDescription("List deployments in a Kubernetes cluster"),
		mcp.WithString("cluster_id", mcp.Required(), mcp.Description("Kubernetes cluster ID")),
		mcp.WithString("namespace", mcp.Description("Namespace to list deployments in (overrides cluster default)")),
	)
	s.AddTool(k8sListDeploymentsTool, withPermissionCheck("k8s_list_deployments", withValidation("k8s_list_deployments", k8sListDeploymentsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleK8sListDeployments(ctx, d, request)
	})))
	k8sGetPodsTool := mcp.NewTool("k8s_get_pods",
		mcp.WithDescription("Get pods in a Kubernetes cluster"),
		mcp.WithString("cluster_id", mcp.Required(), mcp.Description("Kubernetes cluster ID")),
		mcp.WithString("label_selector", mcp.Description("Label selector to filter pods (e.g. app=myapp)")),
		mcp.WithString("namespace", mcp.Description("Namespace to list pods in (overrides cluster default)")),
	)
	s.AddTool(k8sGetPodsTool, withPermissionCheck("k8s_get_pods", withValidation("k8s_get_pods", k8sGetPodsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleK8sGetPods(ctx, d, request)
	})))

}
