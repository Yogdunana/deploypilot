package model

import (
	"fmt"
	"time"

	"github.com/Yogdunana/deploypilot/internal/crypto"
)

// Cluster represents a Kubernetes cluster configuration.
type Cluster struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	TenantID       string    `gorm:"index" json:"tenant_id"`
	Name           string    `gorm:"not null" json:"name"`
	Description    string    `json:"description"`
	Provider       string    `gorm:"not null;default:kubernetes" json:"provider"` // always "kubernetes"
	APIServer      string    `gorm:"not null" json:"api_server"`                  // e.g., https://192.168.1.100:6443
	KubeConfig     string    `gorm:"type:text" json:"-"`                          // kubeconfig content (encrypted), never expose
	KubeConfigPath string    `json:"kube_config_path"`                            // path to kubeconfig file (alternative)
	Context        string    `json:"context"`                                     // kubeconfig context name
	Namespace      string    `gorm:"default:default" json:"namespace"`            // default namespace
	Token          string    `gorm:"type:text" json:"-"`                          // bearer token (encrypted)
	CAData         string    `gorm:"type:text" json:"-"`                          // CA certificate data (encrypted)
	Status         string    `gorm:"default:unknown" json:"status"`               // connected, disconnected, unknown, error
	Version        string    `json:"version"`                                     // k8s version (e.g., v1.28.0)
	NodeCount      int       `json:"node_count"`
	Tags           string    `gorm:"type:text" json:"tags"`                       // JSON array of tags
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Cluster) TableName() string { return "clusters" }

// clusterRow is the DB representation with encrypted sensitive fields.
type clusterRow struct {
	ID             string    `gorm:"primaryKey"`
	TenantID       string    `gorm:"index"`
	Name           string    `gorm:"not null"`
	Description    string
	Provider       string    `gorm:"not null;default:kubernetes"`
	APIServer      string    `gorm:"not null"`
	KubeConfig     string    `gorm:"type:text"`
	KubeConfigPath string
	Context        string
	Namespace      string    `gorm:"default:default"`
	Token          string    `gorm:"type:text"`
	CAData         string    `gorm:"type:text"`
	Status         string    `gorm:"default:unknown"`
	Version        string
	NodeCount      int
	Tags           string    `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (clusterRow) TableName() string { return "clusters" }

// toCluster converts a clusterRow to a Cluster (without sensitive fields).
func (r *clusterRow) toCluster() Cluster {
	return Cluster{
		ID:             r.ID,
		TenantID:       r.TenantID,
		Name:           r.Name,
		Description:    r.Description,
		Provider:       r.Provider,
		APIServer:      r.APIServer,
		KubeConfigPath: r.KubeConfigPath,
		Context:        r.Context,
		Namespace:      r.Namespace,
		Status:         r.Status,
		Version:        r.Version,
		NodeCount:      r.NodeCount,
		Tags:           r.Tags,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

// CreateCluster creates a new Kubernetes cluster with encrypted sensitive fields.
func CreateCluster(cluster *Cluster) (*Cluster, error) {
	if cluster.Name == "" {
		return nil, fmt.Errorf("cluster name is required")
	}
	if cluster.APIServer == "" {
		return nil, fmt.Errorf("cluster api_server is required")
	}

	row := &clusterRow{
		ID:             "cls-" + generateUUID(),
		TenantID:       cluster.TenantID,
		Name:           cluster.Name,
		Description:    cluster.Description,
		Provider:       cluster.Provider,
		APIServer:      cluster.APIServer,
		KubeConfigPath: cluster.KubeConfigPath,
		Context:        cluster.Context,
		Namespace:      cluster.Namespace,
		Status:         cluster.Status,
		Version:        cluster.Version,
		NodeCount:      cluster.NodeCount,
		Tags:           cluster.Tags,
	}

	if cluster.Provider == "" {
		row.Provider = "kubernetes"
	}
	if row.Namespace == "" {
		row.Namespace = "default"
	}
	if row.Status == "" {
		row.Status = "unknown"
	}

	// Encrypt sensitive fields
	if cluster.KubeConfig != "" {
		encrypted, err := crypto.Encrypt(getEncKey(), cluster.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt kubeconfig: %w", err)
		}
		row.KubeConfig = encrypted
	}
	if cluster.Token != "" {
		encrypted, err := crypto.Encrypt(getEncKey(), cluster.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt token: %w", err)
		}
		row.Token = encrypted
	}
	if cluster.CAData != "" {
		encrypted, err := crypto.Encrypt(getEncKey(), cluster.CAData)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt ca_data: %w", err)
		}
		row.CAData = encrypted
	}

	result := getDB().Create(row)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", result.Error)
	}

	c := row.toCluster()
	return &c, nil
}

// GetCluster retrieves a cluster by ID (without decrypted sensitive fields).
func GetCluster(id string) (*Cluster, error) {
	var row clusterRow
	result := getDB().First(&row, "id = ?", id)
	if result.Error != nil {
		return nil, fmt.Errorf("cluster not found: %w", result.Error)
	}

	c := row.toCluster()
	return &c, nil
}

// ListClusters returns all clusters for a tenant (without decrypted sensitive fields).
func ListClusters(tenantID string) ([]Cluster, error) {
	var rows []clusterRow
	result := getDB().Where("tenant_id = ?", tenantID).Find(&rows)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", result.Error)
	}

	clusters := make([]Cluster, 0, len(rows))
	for _, row := range rows {
		clusters = append(clusters, row.toCluster())
	}
	return clusters, nil
}

// UpdateCluster updates a cluster's fields. Sensitive fields (kubeConfig, token, caData)
// are re-encrypted if provided.
func UpdateCluster(id string, updates map[string]interface{}) (*Cluster, error) {
	if len(updates) == 0 {
		return GetCluster(id)
	}

	// Encrypt sensitive fields before updating
	dbUpdates := make(map[string]interface{})
	for k, v := range updates {
		switch k {
		case "kube_config":
			if plain, ok := v.(string); ok && plain != "" {
				encrypted, err := crypto.Encrypt(getEncKey(), plain)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt kubeconfig: %w", err)
				}
				dbUpdates["kube_config"] = encrypted
			}
		case "token":
			if plain, ok := v.(string); ok && plain != "" {
				encrypted, err := crypto.Encrypt(getEncKey(), plain)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt token: %w", err)
				}
				dbUpdates["token"] = encrypted
			}
		case "ca_data":
			if plain, ok := v.(string); ok && plain != "" {
				encrypted, err := crypto.Encrypt(getEncKey(), plain)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt ca_data: %w", err)
				}
				dbUpdates["ca_data"] = encrypted
			}
		default:
			dbUpdates[k] = v
		}
	}

	result := getDB().Model(&clusterRow{}).Where("id = ?", id).Updates(dbUpdates)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update cluster: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("cluster not found")
	}

	return GetCluster(id)
}

// DeleteCluster removes a cluster by ID.
func DeleteCluster(id string) error {
	result := getDB().Delete(&clusterRow{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete cluster: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("cluster not found")
	}
	return nil
}

// GetClusterWithSecrets retrieves a cluster by ID and decrypts sensitive fields.
// This should only be used internally when a k8s client needs to be created.
func GetClusterWithSecrets(id string) (*Cluster, error) {
	var row clusterRow
	result := getDB().First(&row, "id = ?", id)
	if result.Error != nil {
		return nil, fmt.Errorf("cluster not found: %w", result.Error)
	}

	c := row.toCluster()

	// Decrypt sensitive fields
	if row.KubeConfig != "" {
		decrypted, err := crypto.Decrypt(getEncKey(), row.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt kubeconfig: %w", err)
		}
		c.KubeConfig = decrypted
	}
	if row.Token != "" {
		decrypted, err := crypto.Decrypt(getEncKey(), row.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt token: %w", err)
		}
		c.Token = decrypted
	}
	if row.CAData != "" {
		decrypted, err := crypto.Decrypt(getEncKey(), row.CAData)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt ca_data: %w", err)
		}
		c.CAData = decrypted
	}

	return &c, nil
}
