package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

func setupClusterTest(t *testing.T) (*gin.Engine, *service.Bridge, string) {
	t.Helper()
	db := setupTestDB(t)
	encKey := crypto.NewEncryptionKey()
	model.InitDB(db, encKey)
	bridge := service.NewBridge(db, nil, encKey, nil)
	router := setupFullTestRouter(db, bridge)
	token := getTestToken(t, "user-1", "owner")
	return router, bridge, token
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	return resp
}

func TestCreateCluster(t *testing.T) {
	router, _, token := setupClusterTest(t)

	body := map[string]interface{}{
		"name":       "test-cluster",
		"api_server": "https://10.0.0.1:6443",
		"description": "Test cluster",
		"namespace":  "default",
		"tags":       `["test"]`,
	}

	w := makeRequest(router, "POST", "/api/v1/clusters", body, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if resp["status"] != "success" {
		t.Errorf("status = %v, want success", resp["status"])
	}

	data := resp["data"].(map[string]interface{})
	if data["name"] != "test-cluster" {
		t.Errorf("name = %v, want test-cluster", data["name"])
	}
	if data["api_server"] != "https://10.0.0.1:6443" {
		t.Errorf("api_server = %v, want https://10.0.0.1:6443", data["api_server"])
	}
	if data["provider"] != "kubernetes" {
		t.Errorf("provider = %v, want kubernetes", data["provider"])
	}
}

func TestCreateCluster_MissingName(t *testing.T) {
	router, _, token := setupClusterTest(t)

	body := map[string]interface{}{
		"api_server": "https://10.0.0.1:6443",
	}

	w := makeRequest(router, "POST", "/api/v1/clusters", body, token)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCluster_MissingAPIServer(t *testing.T) {
	router, _, token := setupClusterTest(t)

	body := map[string]interface{}{
		"name": "test-cluster",
	}

	w := makeRequest(router, "POST", "/api/v1/clusters", body, token)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCluster_WithSensitiveFields(t *testing.T) {
	router, _, token := setupClusterTest(t)

	body := map[string]interface{}{
		"name":        "secure-cluster",
		"api_server":  "https://10.0.0.2:6443",
		"kube_config": "apiVersion: v1\nkind: Config",
		"token":       "bearer-token-abc123",
		"ca_data":     "-----BEGIN CERTIFICATE-----\nMIIC...",
	}

	w := makeRequest(router, "POST", "/api/v1/clusters", body, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].(map[string]interface{})

	// Sensitive fields should not be in response
	if _, ok := data["kube_config"]; ok {
		t.Error("kube_config should not be in response")
	}
	if _, ok := data["token"]; ok {
		t.Error("token should not be in response")
	}
	if _, ok := data["ca_data"]; ok {
		t.Error("ca_data should not be in response")
	}
}

func TestListClusters(t *testing.T) {
	router, bridge, token := setupClusterTest(t)

	// Create some clusters
	bridge.CreateCluster(context.TODO(), &model.Cluster{
		TenantID:  "tenant-default",
		Name:      "cluster-1",
		APIServer: "https://10.0.0.1:6443",
	})
	bridge.CreateCluster(context.TODO(), &model.Cluster{
		TenantID:  "tenant-default",
		Name:      "cluster-2",
		APIServer: "https://10.0.0.2:6443",
	})

	w := makeRequest(router, "GET", "/api/v1/clusters?tenant_id=tenant-default", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].([]interface{})

	if len(data) != 2 {
		t.Errorf("count = %d, want 2", len(data))
	}
}

func TestListClusters_Empty(t *testing.T) {
	router, _, token := setupClusterTest(t)

	w := makeRequest(router, "GET", "/api/v1/clusters", nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].([]interface{})

	if len(data) != 0 {
		t.Errorf("count = %d, want 0", len(data))
	}
}

func TestGetCluster(t *testing.T) {
	router, bridge, token := setupClusterTest(t)

	created, _ := bridge.CreateCluster(context.TODO(), &model.Cluster{
		TenantID:  "tenant-default",
		Name:      "get-test",
		APIServer: "https://10.0.0.3:6443",
	})

	w := makeRequest(router, "GET", "/api/v1/clusters/"+created.ID, nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].(map[string]interface{})

	if data["name"] != "get-test" {
		t.Errorf("name = %v, want get-test", data["name"])
	}
	if data["id"] != created.ID {
		t.Errorf("id = %v, want %v", data["id"], created.ID)
	}
}

func TestGetCluster_NotFound(t *testing.T) {
	router, _, token := setupClusterTest(t)

	w := makeRequest(router, "GET", "/api/v1/clusters/nonexistent-id", nil, token)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCluster(t *testing.T) {
	router, bridge, token := setupClusterTest(t)

	created, _ := bridge.CreateCluster(context.TODO(), &model.Cluster{
		TenantID:  "tenant-default",
		Name:      "update-test",
		APIServer: "https://10.0.0.4:6443",
	})

	body := map[string]interface{}{
		"name":        "updated-cluster",
		"description": "Updated description",
		"status":      "connected",
		"version":     "v1.29.0",
		"node_count":  5,
	}

	w := makeRequest(router, "PUT", "/api/v1/clusters/"+created.ID, body, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].(map[string]interface{})

	if data["name"] != "updated-cluster" {
		t.Errorf("name = %v, want updated-cluster", data["name"])
	}
	if data["status"] != "connected" {
		t.Errorf("status = %v, want connected", data["status"])
	}
	if data["version"] != "v1.29.0" {
		t.Errorf("version = %v, want v1.29.0", data["version"])
	}
}

func TestDeleteCluster(t *testing.T) {
	router, bridge, token := setupClusterTest(t)

	created, _ := bridge.CreateCluster(context.TODO(), &model.Cluster{
		TenantID:  "tenant-default",
		Name:      "delete-test",
		APIServer: "https://10.0.0.5:6443",
	})

	w := makeRequest(router, "DELETE", "/api/v1/clusters/"+created.ID, nil, token)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	data := resp["data"].(map[string]interface{})

	if data["id"] != created.ID {
		t.Errorf("id = %v, want %v", data["id"], created.ID)
	}

	// Verify it's gone
	w2 := makeRequest(router, "GET", "/api/v1/clusters/"+created.ID, nil, token)
	if w2.Code != http.StatusNotFound {
		t.Errorf("expected status 404 after delete, got %d", w2.Code)
	}
}

func TestCluster_Unauthorized(t *testing.T) {
	router, _, _ := setupClusterTest(t)

	w := makeRequest(router, "GET", "/api/v1/clusters", nil, "")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}
