package service

import (
	"context"
	"fmt"

	"github.com/Yogdunana/deploypilot/internal/mcp"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/provider/server"
	k8sapi "k8s.io/api/core/v1"
)

// ========== Kubernetes Deployment Operations ==========

// K8sDeploy deploys an application to a Kubernetes cluster.
func (b *Bridge) K8sDeploy(ctx context.Context, clusterID string, app *mcp.K8sDeployConfig) error {
	if b.DB == nil {
		return fmt.Errorf("database not available")
	}

	cluster, err := model.GetClusterWithSecrets(clusterID)
	if err != nil {
		return fmt.Errorf("cluster not found: %w", err)
	}

	k8sProvider, err := server.NewK8sProvider(cluster)
	if err != nil {
		return fmt.Errorf("failed to create k8s provider: %w", err)
	}

	deployCfg := &server.K8sDeployConfig{
		Name:      app.Name,
		Image:     app.Image,
		Replicas:  app.Replicas,
		Ports:     app.Ports,
		EnvVars:   app.EnvVars,
		Labels:    app.Labels,
		Namespace: app.Namespace,
	}

	return k8sProvider.Deploy(ctx, deployCfg)
}

// K8sListDeployments lists deployments in a Kubernetes cluster.
func (b *Bridge) K8sListDeployments(ctx context.Context, clusterID string) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	cluster, err := model.GetClusterWithSecrets(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	k8sProvider, err := server.NewK8sProvider(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s provider: %w", err)
	}

	deployments, err := k8sProvider.ListDeployments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	// Convert to serializable format
	items := make([]map[string]interface{}, 0, len(deployments))
	for _, d := range deployments {
		replicas := int32(0)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		items = append(items, map[string]interface{}{
			"name":       d.Name,
			"namespace":  d.Namespace,
			"replicas":   replicas,
			"available":  d.Status.AvailableReplicas,
			"image":      firstContainerImage(d.Spec.Template.Spec.Containers),
			"created_at": d.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	return map[string]interface{}{
		"status":  "success",
		"total":   len(items),
		"cluster": clusterID,
		"items":   items,
	}, nil
}

// K8sGetPods retrieves pods from a Kubernetes cluster.
func (b *Bridge) K8sGetPods(ctx context.Context, clusterID, labelSelector string) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	cluster, err := model.GetClusterWithSecrets(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	k8sProvider, err := server.NewK8sProvider(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s provider: %w", err)
	}

	pods, err := k8sProvider.GetPods(ctx, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	// Convert to serializable format
	items := make([]map[string]interface{}, 0, len(pods))
	for _, p := range pods {
		var containers []map[string]string
		for _, c := range p.Spec.Containers {
			containers = append(containers, map[string]string{
				"name":  c.Name,
				"image": c.Image,
			})
		}

		items = append(items, map[string]interface{}{
			"name":          p.Name,
			"namespace":     p.Namespace,
			"status":        string(p.Status.Phase),
			"pod_ip":        p.Status.PodIP,
			"node":          p.Spec.NodeName,
			"restart_count": totalRestartCount(p.Status.ContainerStatuses),
			"created_at":    p.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			"containers":    containers,
		})
	}

	return map[string]interface{}{
		"status":  "success",
		"total":   len(items),
		"cluster": clusterID,
		"items":   items,
	}, nil
}

// firstContainerImage returns the first container image, or empty if none exist.
func firstContainerImage(containers []k8sapi.Container) string {
	if len(containers) == 0 {
		return ""
	}
	return containers[0].Image
}

// totalRestartCount sums restart counts across all container statuses.
func totalRestartCount(statuses []k8sapi.ContainerStatus) int32 {
	var n int32
	for i := range statuses {
		n += statuses[i].RestartCount
	}
	return n
}
