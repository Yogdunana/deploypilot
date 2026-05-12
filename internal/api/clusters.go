package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// CreateCluster creates a new Kubernetes cluster.
// @Summary      Create a cluster
// @Description  Register a new Kubernetes cluster with encrypted credentials
// @Tags         Clusters
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{tenant_id=string,name=string,description=string,api_server=string,kube_config=string,kube_config_path=string,context=string,namespace=string,token=string,ca_data=string,tags=string} true "Cluster creation request"
// @Success      200 {object} map[string]interface{} "status, data (Cluster)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /clusters [post]
func CreateCluster(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			TenantID       string `json:"tenant_id"`
			Name           string `json:"name" binding:"required"`
			Description    string `json:"description"`
			APIServer      string `json:"api_server" binding:"required"`
			KubeConfig     string `json:"kube_config"`
			KubeConfigPath string `json:"kube_config_path"`
			Context        string `json:"context"`
			Namespace      string `json:"namespace"`
			Token          string `json:"token"`
			CAData         string `json:"ca_data"`
			Tags           string `json:"tags"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}
		if input.TenantID == "" {
			input.TenantID = model.DefaultTenantID
		}

		cluster, err := bridge.CreateCluster(c.Request.Context(), &model.Cluster{
			TenantID:       input.TenantID,
			Name:           input.Name,
			Description:    input.Description,
			APIServer:      input.APIServer,
			KubeConfig:     input.KubeConfig,
			KubeConfigPath: input.KubeConfigPath,
			Context:        input.Context,
			Namespace:      input.Namespace,
			Token:          input.Token,
			CAData:         input.CAData,
			Tags:           input.Tags,
		})
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, cluster)
	}
}

// ListClusters lists all Kubernetes clusters for a tenant.
// @Summary      List clusters
// @Description  Retrieve all Kubernetes clusters for a tenant
// @Tags         Clusters
// @Produce      json
// @Security     BearerAuth
// @Param        tenant_id query string false "Tenant ID" default(model.DefaultTenantID)
// @Success      200 {object} map[string]interface{} "status, data (array of Cluster)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /clusters [get]
func ListClusters(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Query("tenant_id")
		if tenantID == "" {
			tenantID = model.DefaultTenantID
		}

		clusters, err := bridge.ListClusters(c.Request.Context(), tenantID)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		if clusters == nil {
			clusters = []model.Cluster{}
		}
		respondSuccess(c, clusters)
	}
}

// GetCluster retrieves a single Kubernetes cluster by ID.
// @Summary      Get a cluster
// @Description  Retrieve a Kubernetes cluster by ID
// @Tags         Clusters
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Cluster ID"
// @Success      200 {object} map[string]interface{} "status, data (Cluster)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /clusters/{id} [get]
func GetCluster(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		cluster, err := bridge.GetCluster(c.Request.Context(), id)
		if err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.cluster.not_found")
			return
		}
		respondSuccess(c, cluster)
	}
}

// UpdateCluster updates a Kubernetes cluster.
// @Summary      Update a cluster
// @Description  Update fields of a Kubernetes cluster
// @Tags         Clusters
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Cluster ID"
// @Param        request body object{name=string,description=string,api_server=string,kube_config=string,kube_config_path=string,context=string,namespace=string,token=string,ca_data=string,tags=string,status=string,version=string,node_count=int} true "Cluster update request"
// @Success      200 {object} map[string]interface{} "status, data (updated Cluster)"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /clusters/{id} [put]
func UpdateCluster(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var input struct {
			Name           string `json:"name"`
			Description    string `json:"description"`
			APIServer      string `json:"api_server"`
			KubeConfig     string `json:"kube_config"`
			KubeConfigPath string `json:"kube_config_path"`
			Context        string `json:"context"`
			Namespace      string `json:"namespace"`
			Token          string `json:"token"`
			CAData         string `json:"ca_data"`
			Tags           string `json:"tags"`
			Status         string `json:"status"`
			Version        string `json:"version"`
			NodeCount      *int   `json:"node_count"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		updates := make(map[string]interface{})
		if input.Name != "" {
			updates["name"] = input.Name
		}
		if input.Description != "" {
			updates["description"] = input.Description
		}
		if input.APIServer != "" {
			updates["api_server"] = input.APIServer
		}
		if input.KubeConfig != "" {
			updates["kube_config"] = input.KubeConfig
		}
		if input.KubeConfigPath != "" {
			updates["kube_config_path"] = input.KubeConfigPath
		}
		if input.Context != "" {
			updates["context"] = input.Context
		}
		if input.Namespace != "" {
			updates["namespace"] = input.Namespace
		}
		if input.Token != "" {
			updates["token"] = input.Token
		}
		if input.CAData != "" {
			updates["ca_data"] = input.CAData
		}
		if input.Tags != "" {
			updates["tags"] = input.Tags
		}
		if input.Status != "" {
			updates["status"] = input.Status
		}
		if input.Version != "" {
			updates["version"] = input.Version
		}
		if input.NodeCount != nil {
			updates["node_count"] = *input.NodeCount
		}

		cluster, err := bridge.UpdateCluster(c.Request.Context(), id, updates)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, cluster)
	}
}

// DeleteCluster deletes a Kubernetes cluster.
// @Summary      Delete a cluster
// @Description  Delete a Kubernetes cluster by ID
// @Tags         Clusters
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Cluster ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /clusters/{id} [delete]
func DeleteCluster(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		if err := bridge.DeleteCluster(c.Request.Context(), id); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{"message": "cluster deleted", "id": id})
	}
}

// TestClusterConnection tests the connection to a Kubernetes cluster.
// @Summary      Test cluster connection
// @Description  Test connectivity to a Kubernetes cluster
// @Tags         Clusters
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Cluster ID"
// @Success      200 {object} map[string]interface{} "status, data (cluster info)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /clusters/{id}/test [post]
func TestClusterConnection(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		result, err := bridge.TestClusterConnection(c.Request.Context(), id)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.cluster.connection_failed")
			return
		}
		respondSuccess(c, result)
	}
}
