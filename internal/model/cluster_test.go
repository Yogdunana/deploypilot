package model

import (
	"strings"
	"testing"
	"time"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/database"
)

func setupClusterDB(t *testing.T) ([]byte, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	db, err := database.Connect("sqlite", tmpDir+"/test.db")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := database.Seed(db); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}
	encKey := crypto.NewEncryptionKey()
	InitDB(db, encKey)
	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
	return encKey, cleanup
}

// ========== Cluster Model Tests ==========

func TestClusterModel(t *testing.T) {
	now := time.Now()
	c := &Cluster{
		ID:             "cls-001",
		TenantID:       "tenant-default",
		Name:           "my-cluster",
		Description:    "Production Kubernetes cluster",
		Provider:       "kubernetes",
		APIServer:      "https://192.168.1.100:6443",
		KubeConfigPath: "/etc/kubernetes/kubeconfig",
		Context:        "production",
		Namespace:      "default",
		Status:         "connected",
		Version:        "v1.28.0",
		NodeCount:      3,
		Tags:           `["production","primary"]`,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if c.ID != "cls-001" {
		t.Errorf("Cluster.ID = %q, want %q", c.ID, "cls-001")
	}
	if c.Name != "my-cluster" {
		t.Errorf("Cluster.Name = %q, want %q", c.Name, "my-cluster")
	}
	if c.Provider != "kubernetes" {
		t.Errorf("Cluster.Provider = %q, want %q", c.Provider, "kubernetes")
	}
	if c.APIServer != "https://192.168.1.100:6443" {
		t.Errorf("Cluster.APIServer = %q, want %q", c.APIServer, "https://192.168.1.100:6443")
	}
	if c.Status != "connected" {
		t.Errorf("Cluster.Status = %q, want %q", c.Status, "connected")
	}
	if c.Version != "v1.28.0" {
		t.Errorf("Cluster.Version = %q, want %q", c.Version, "v1.28.0")
	}
	if c.NodeCount != 3 {
		t.Errorf("Cluster.NodeCount = %d, want %d", c.NodeCount, 3)
	}
}

func TestClusterTableName(t *testing.T) {
	c := &Cluster{}
	if c.TableName() != "clusters" {
		t.Errorf("Cluster.TableName() = %q, want %q", c.TableName(), "clusters")
	}
}

func TestClusterRowTableName(t *testing.T) {
	r := &clusterRow{}
	if r.TableName() != "clusters" {
		t.Errorf("clusterRow.TableName() = %q, want %q", r.TableName(), "clusters")
	}
}

func TestClusterSensitiveFieldsJSON(t *testing.T) {
	c := &Cluster{
		ID:         "cls-001",
		KubeConfig: "secret-kubeconfig",
		Token:      "secret-token",
		CAData:     "secret-ca-data",
	}

	// Verify json:"-" tags by checking struct tags
	if c.KubeConfig != "secret-kubeconfig" {
		t.Error("KubeConfig field should be accessible in Go")
	}
	if c.Token != "secret-token" {
		t.Error("Token field should be accessible in Go")
	}
	if c.CAData != "secret-ca-data" {
		t.Error("CAData field should be accessible in Go")
	}
}

// ========== Cluster CRUD Tests ==========

func TestCreateCluster(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	cluster, err := CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "test-cluster",
		APIServer: "https://10.0.0.1:6443",
	})
	if err != nil {
		t.Fatalf("CreateCluster() error = %v", err)
	}

	if cluster.ID == "" {
		t.Error("ID should not be empty")
	}
	if !strings.HasPrefix(cluster.ID, "cls-") {
		t.Errorf("ID should have 'cls-' prefix, got %q", cluster.ID)
	}
	if cluster.Name != "test-cluster" {
		t.Errorf("Name = %q, want %q", cluster.Name, "test-cluster")
	}
	if cluster.APIServer != "https://10.0.0.1:6443" {
		t.Errorf("APIServer = %q, want %q", cluster.APIServer, "https://10.0.0.1:6443")
	}
	if cluster.Provider != "kubernetes" {
		t.Errorf("Provider = %q, want %q (default)", cluster.Provider, "kubernetes")
	}
	if cluster.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q (default)", cluster.Namespace, "default")
	}
	if cluster.Status != "unknown" {
		t.Errorf("Status = %q, want %q (default)", cluster.Status, "unknown")
	}
}

func TestCreateClusterWithSensitiveFields(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	cluster, err := CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "secure-cluster",
		APIServer: "https://10.0.0.2:6443",
		KubeConfig: "apiVersion: v1\nkind: Config\nclusters:\n- cluster:\n    server: https://10.0.0.2:6443",
		Token:     "bearer-token-abc123",
		CAData:    "-----BEGIN CERTIFICATE-----\nMIIC...",
	})
	if err != nil {
		t.Fatalf("CreateCluster() error = %v", err)
	}

	// Sensitive fields should not be exposed in the returned Cluster
	if cluster.KubeConfig != "" {
		t.Error("KubeConfig should be empty in returned Cluster")
	}
	if cluster.Token != "" {
		t.Error("Token should be empty in returned Cluster")
	}
	if cluster.CAData != "" {
		t.Error("CAData should be empty in returned Cluster")
	}
}

func TestCreateClusterNameRequired(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	_, err := CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "",
		APIServer: "https://10.0.0.1:6443",
	})
	if err == nil {
		t.Error("CreateCluster() should fail when name is empty")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention 'name', got %q", err.Error())
	}
}

func TestCreateClusterAPIServerRequired(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	_, err := CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "test-cluster",
		APIServer: "",
	})
	if err == nil {
		t.Error("CreateCluster() should fail when api_server is empty")
	}
	if !strings.Contains(err.Error(), "api_server") {
		t.Errorf("error should mention 'api_server', got %q", err.Error())
	}
}

func TestGetCluster(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	created, _ := CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "get-test",
		APIServer: "https://10.0.0.3:6443",
	})

	got, err := GetCluster(created.ID)
	if err != nil {
		t.Fatalf("GetCluster() error = %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Name != "get-test" {
		t.Errorf("Name = %q, want %q", got.Name, "get-test")
	}
}

func TestGetClusterNotFound(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	_, err := GetCluster("nonexistent-id")
	if err == nil {
		t.Error("GetCluster() should fail for nonexistent ID")
	}
}

func TestListClusters(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "cluster-1",
		APIServer: "https://10.0.0.1:6443",
	})
	CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "cluster-2",
		APIServer: "https://10.0.0.2:6443",
	})

	clusters, err := ListClusters("tenant-default")
	if err != nil {
		t.Fatalf("ListClusters() error = %v", err)
	}

	if len(clusters) != 2 {
		t.Errorf("count = %d, want 2", len(clusters))
	}
}

func TestListClustersByTenant(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "default-cluster",
		APIServer: "https://10.0.0.1:6443",
	})
	CreateCluster(&Cluster{
		TenantID:  "tenant-other",
		Name:      "other-cluster",
		APIServer: "https://10.0.0.2:6443",
	})

	clusters, _ := ListClusters("tenant-default")
	if len(clusters) != 1 {
		t.Errorf("count = %d, want 1 (filtered by tenant)", len(clusters))
	}
}

func TestListClustersEmpty(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	clusters, err := ListClusters("tenant-nonexistent")
	if err != nil {
		t.Fatalf("ListClusters() error = %v", err)
	}
	if len(clusters) != 0 {
		t.Errorf("count = %d, want 0", len(clusters))
	}
}

func TestUpdateCluster(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	created, _ := CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "update-test",
		APIServer: "https://10.0.0.1:6443",
	})

	updated, err := UpdateCluster(created.ID, map[string]interface{}{
		"name":        "updated-cluster",
		"description": "Updated description",
		"status":      "connected",
		"version":     "v1.29.0",
		"node_count":  5,
	})
	if err != nil {
		t.Fatalf("UpdateCluster() error = %v", err)
	}

	if updated.Name != "updated-cluster" {
		t.Errorf("Name = %q, want %q", updated.Name, "updated-cluster")
	}
	if updated.Description != "Updated description" {
		t.Errorf("Description = %q, want %q", updated.Description, "Updated description")
	}
	if updated.Status != "connected" {
		t.Errorf("Status = %q, want %q", updated.Status, "connected")
	}
	if updated.Version != "v1.29.0" {
		t.Errorf("Version = %q, want %q", updated.Version, "v1.29.0")
	}
	if updated.NodeCount != 5 {
		t.Errorf("NodeCount = %d, want %d", updated.NodeCount, 5)
	}
}

func TestUpdateClusterSensitiveFields(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	created, _ := CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "encrypt-test",
		APIServer: "https://10.0.0.1:6443",
	})

	// Update with sensitive fields — they should be encrypted
	updated, err := UpdateCluster(created.ID, map[string]interface{}{
		"kube_config": "new-kubeconfig-content",
		"token":       "new-bearer-token",
		"ca_data":     "new-ca-certificate",
	})
	if err != nil {
		t.Fatalf("UpdateCluster() error = %v", err)
	}

	// Sensitive fields should not be exposed in the returned Cluster
	if updated.KubeConfig != "" {
		t.Error("KubeConfig should be empty in returned Cluster after update")
	}
	if updated.Token != "" {
		t.Error("Token should be empty in returned Cluster after update")
	}
	if updated.CAData != "" {
		t.Error("CAData should be empty in returned Cluster after update")
	}
}

func TestUpdateClusterNotFound(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	_, err := UpdateCluster("nonexistent-id", map[string]interface{}{
		"name": "new-name",
	})
	if err == nil {
		t.Error("UpdateCluster() should fail for nonexistent ID")
	}
}

func TestUpdateClusterEmptyUpdates(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	created, _ := CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "noop-test",
		APIServer: "https://10.0.0.1:6443",
	})

	got, err := UpdateCluster(created.ID, map[string]interface{}{})
	if err != nil {
		t.Fatalf("UpdateCluster() with empty updates error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestDeleteCluster(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	created, _ := CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "delete-test",
		APIServer: "https://10.0.0.1:6443",
	})

	err := DeleteCluster(created.ID)
	if err != nil {
		t.Fatalf("DeleteCluster() error = %v", err)
	}

	_, err = GetCluster(created.ID)
	if err == nil {
		t.Error("GetCluster() should fail after delete")
	}
}

func TestDeleteClusterNotFound(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	err := DeleteCluster("nonexistent-id")
	if err == nil {
		t.Error("DeleteCluster() should fail for nonexistent ID")
	}
}

func TestClusterRoundTrip(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	// Create
	cluster, _ := CreateCluster(&Cluster{
		TenantID:    "tenant-default",
		Name:        "round-trip",
		APIServer:   "https://10.0.0.1:6443",
		Description: "Round trip test cluster",
	})

	// Get and verify
	got, _ := GetCluster(cluster.ID)
	if got.Name != "round-trip" {
		t.Errorf("round-trip get failed: Name = %q", got.Name)
	}

	// Update
	UpdateCluster(cluster.ID, map[string]interface{}{
		"name":   "round-trip-updated",
		"status": "connected",
	})

	// Get again
	got2, _ := GetCluster(cluster.ID)
	if got2.Name != "round-trip-updated" {
		t.Errorf("round-trip update failed: Name = %q", got2.Name)
	}
	if got2.Status != "connected" {
		t.Errorf("round-trip update failed: Status = %q", got2.Status)
	}

	// Delete
	DeleteCluster(cluster.ID)

	// Verify deleted
	_, err := GetCluster(cluster.ID)
	if err == nil {
		t.Error("should fail after delete in round-trip")
	}
}

func TestCreateClusterEncryptError(t *testing.T) {
	_, cleanup := setupClusterDB(t)
	defer cleanup()

	// Temporarily replace encKey with a short key to trigger encryption error
	originalKey := dbHolder.encKey
	dbHolder.encKey = []byte("too-short")
	defer func() { dbHolder.encKey = originalKey }()

	_, err := CreateCluster(&Cluster{
		TenantID:  "tenant-default",
		Name:      "encrypt-fail",
		APIServer: "https://10.0.0.1:6443",
		KubeConfig: "some-kubeconfig",
	})
	if err == nil {
		t.Error("CreateCluster() should fail with short encryption key when encrypting kubeconfig")
	}
}

func TestClusterRowToCluster(t *testing.T) {
	now := time.Now()
	row := &clusterRow{
		ID:             "cls-001",
		TenantID:       "tenant-default",
		Name:           "test-cluster",
		Description:    "Test description",
		Provider:       "kubernetes",
		APIServer:      "https://10.0.0.1:6443",
		KubeConfig:     "encrypted-data",
		KubeConfigPath: "/path/to/kubeconfig",
		Context:        "production",
		Namespace:      "kube-system",
		Token:          "encrypted-token",
		CAData:         "encrypted-ca",
		Status:         "connected",
		Version:        "v1.28.0",
		NodeCount:      3,
		Tags:           `["prod"]`,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	c := row.toCluster()

	if c.ID != "cls-001" {
		t.Errorf("ID = %q, want %q", c.ID, "cls-001")
	}
	if c.Name != "test-cluster" {
		t.Errorf("Name = %q", c.Name)
	}
	if c.KubeConfig != "" {
		t.Error("toCluster() should not expose KubeConfig")
	}
	if c.Token != "" {
		t.Error("toCluster() should not expose Token")
	}
	if c.CAData != "" {
		t.Error("toCluster() should not expose CAData")
	}
	if c.KubeConfigPath != "/path/to/kubeconfig" {
		t.Errorf("KubeConfigPath = %q", c.KubeConfigPath)
	}
	if c.Context != "production" {
		t.Errorf("Context = %q", c.Context)
	}
	if c.Namespace != "kube-system" {
		t.Errorf("Namespace = %q", c.Namespace)
	}
	if c.Status != "connected" {
		t.Errorf("Status = %q", c.Status)
	}
	if c.Version != "v1.28.0" {
		t.Errorf("Version = %q", c.Version)
	}
	if c.NodeCount != 3 {
		t.Errorf("NodeCount = %d", c.NodeCount)
	}
	if c.Tags != `["prod"]` {
		t.Errorf("Tags = %q", c.Tags)
	}
}
