package service

import (
	"context"
	"fmt"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/provider/server"
)

// CreateCluster creates a new Kubernetes cluster record.
func (b *Bridge) CreateCluster(ctx context.Context, cluster *model.Cluster) (*model.Cluster, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	return model.CreateCluster(cluster)
}

// GetCluster retrieves a cluster by ID.
func (b *Bridge) GetCluster(ctx context.Context, id string) (*model.Cluster, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	return model.GetCluster(id)
}

// ListClusters returns all clusters for a tenant.
func (b *Bridge) ListClusters(ctx context.Context, tenantID string) ([]model.Cluster, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	return model.ListClusters(tenantID)
}

// UpdateCluster updates a cluster's fields.
func (b *Bridge) UpdateCluster(ctx context.Context, id string, updates map[string]interface{}) (*model.Cluster, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}
	return model.UpdateCluster(id, updates)
}

// DeleteCluster removes a cluster by ID.
func (b *Bridge) DeleteCluster(ctx context.Context, id string) error {
	if b.DB == nil {
		return fmt.Errorf("database not available")
	}
	return model.DeleteCluster(id)
}

// TestClusterConnection tests connectivity to a Kubernetes cluster.
func (b *Bridge) TestClusterConnection(ctx context.Context, id string) (interface{}, error) {
	if b.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	cluster, err := model.GetClusterWithSecrets(id)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	k8sProvider, err := server.NewK8sProvider(cluster)
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("failed to create k8s provider: %v", err),
		}, nil
	}

	if err := k8sProvider.TestConnection(ctx); err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("connection failed: %v", err),
		}, nil
	}

	info, err := k8sProvider.GetClusterInfo(ctx)
	if err != nil {
		return map[string]interface{}{
			"status":  "success",
			"message": "connected",
		}, nil
	}

	return map[string]interface{}{
		"status": "success",
		"message": "connected",
		"info":   info,
	}, nil
}
