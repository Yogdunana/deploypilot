package server

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	k8sapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"

	"github.com/Yogdunana/deploypilot/internal/model"
)

func newTestProvider(t *testing.T) (*K8sProvider, *fake.Clientset) {
	t.Helper()
	clientset := fake.NewSimpleClientset()
	provider := NewK8sProviderWithClient(clientset, "default", "cls-test")
	return provider, clientset
}

func TestNewK8sProviderWithClient(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	p := NewK8sProviderWithClient(clientset, "default", "cls-001")

	if p.namespace != "default" {
		t.Errorf("namespace = %q, want %q", p.namespace, "default")
	}
	if p.clusterID != "cls-001" {
		t.Errorf("clusterID = %q, want %q", p.clusterID, "cls-001")
	}
	if p.client == nil {
		t.Error("client should not be nil")
	}
}

func TestNewK8sProviderWithClient_EmptyNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	p := NewK8sProviderWithClient(clientset, "", "cls-002")

	if p.namespace != "default" {
		t.Errorf("namespace = %q, want %q (default)", p.namespace, "default")
	}
}

func TestK8sProvider_Deploy(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	cfg := &K8sDeployConfig{
		Name:     "my-app",
		Image:    "nginx:latest",
		Replicas: 3,
		Ports:    []int32{80, 443},
		EnvVars:  map[string]string{"ENV": "production"},
		Labels:   map[string]string{"tier": "frontend"},
	}

	err := p.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	// Verify the deployment was created
	deploy, err := p.GetDeployment(ctx, "my-app")
	if err != nil {
		t.Fatalf("GetDeployment() error = %v", err)
	}

	if deploy.Name != "my-app" {
		t.Errorf("deployment name = %q, want %q", deploy.Name, "my-app")
	}
	if *deploy.Spec.Replicas != 3 {
		t.Errorf("replicas = %d, want 3", *deploy.Spec.Replicas)
	}
	if deploy.Spec.Template.Spec.Containers[0].Image != "nginx:latest" {
		t.Errorf("image = %q, want %q", deploy.Spec.Template.Spec.Containers[0].Image, "nginx:latest")
	}
	if len(deploy.Spec.Template.Spec.Containers[0].Ports) != 2 {
		t.Errorf("ports count = %d, want 2", len(deploy.Spec.Template.Spec.Containers[0].Ports))
	}
	if deploy.Labels["tier"] != "frontend" {
		t.Errorf("label tier = %q, want %q", deploy.Labels["tier"], "frontend")
	}
}

func TestK8sProvider_Deploy_UpdateExisting(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	// Create initial deployment
	cfg := &K8sDeployConfig{
		Name:     "update-app",
		Image:    "nginx:1.0",
		Replicas: 1,
	}
	if err := p.Deploy(ctx, cfg); err != nil {
		t.Fatalf("initial Deploy() error = %v", err)
	}

	// Update with new image
	cfg.Image = "nginx:2.0"
	cfg.Replicas = 2
	if err := p.Deploy(ctx, cfg); err != nil {
		t.Fatalf("update Deploy() error = %v", err)
	}

	deploy, _ := p.GetDeployment(ctx, "update-app")
	if deploy.Spec.Template.Spec.Containers[0].Image != "nginx:2.0" {
		t.Errorf("image = %q, want %q after update", deploy.Spec.Template.Spec.Containers[0].Image, "nginx:2.0")
	}
	if *deploy.Spec.Replicas != 2 {
		t.Errorf("replicas = %d, want 2 after update", *deploy.Spec.Replicas)
	}
}

func TestK8sProvider_Deploy_DefaultReplicas(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	cfg := &K8sDeployConfig{
		Name:  "default-replicas",
		Image: "nginx:latest",
		// Replicas is 0 (default)
	}

	err := p.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	deploy, _ := p.GetDeployment(ctx, "default-replicas")
	if *deploy.Spec.Replicas != 1 {
		t.Errorf("replicas = %d, want 1 (default)", *deploy.Spec.Replicas)
	}
}

func TestK8sProvider_Deploy_NamespaceOverride(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	cfg := &K8sDeployConfig{
		Name:      "ns-override",
		Image:     "nginx:latest",
		Namespace: "kube-system",
	}

	err := p.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	// Verify in the overridden namespace
	deploy, err := p.client.AppsV1().Deployments("kube-system").Get(ctx, "ns-override", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get deployment in kube-system: %v", err)
	}
	if deploy.Namespace != "kube-system" {
		t.Errorf("namespace = %q, want %q", deploy.Namespace, "kube-system")
	}
}

func TestK8sProvider_ListDeployments(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	// Create two deployments
	p.Deploy(ctx, &K8sDeployConfig{Name: "app-1", Image: "nginx:latest"})
	p.Deploy(ctx, &K8sDeployConfig{Name: "app-2", Image: "redis:latest"})

	deployments, err := p.ListDeployments(ctx)
	if err != nil {
		t.Fatalf("ListDeployments() error = %v", err)
	}

	if len(deployments) != 2 {
		t.Errorf("count = %d, want 2", len(deployments))
	}
}

func TestK8sProvider_DeleteDeployment(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	p.Deploy(ctx, &K8sDeployConfig{Name: "to-delete", Image: "nginx:latest"})

	err := p.DeleteDeployment(ctx, "to-delete")
	if err != nil {
		t.Fatalf("DeleteDeployment() error = %v", err)
	}

	_, err = p.GetDeployment(ctx, "to-delete")
	if err == nil {
		t.Error("GetDeployment() should fail after delete")
	}
}

func TestK8sProvider_ScaleDeployment(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	p.Deploy(ctx, &K8sDeployConfig{Name: "scale-me", Image: "nginx:latest", Replicas: 1})

	err := p.ScaleDeployment(ctx, "scale-me", 5)
	if err != nil {
		t.Fatalf("ScaleDeployment() error = %v", err)
	}

	deploy, _ := p.GetDeployment(ctx, "scale-me")
	if *deploy.Spec.Replicas != 5 {
		t.Errorf("replicas = %d, want 5", *deploy.Spec.Replicas)
	}
}

func TestK8sProvider_GetPods(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	// Create a deployment first
	p.Deploy(ctx, &K8sDeployConfig{Name: "pod-app", Image: "nginx:latest"})

	// Create a pod manually (fake client doesn't auto-create pods)
	pod := &k8sapi.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-app-abc123",
			Namespace: "default",
			Labels:    map[string]string{"app": "pod-app"},
		},
		Spec: k8sapi.PodSpec{
			Containers: []k8sapi.Container{
				{Name: "pod-app", Image: "nginx:latest"},
			},
		},
	}
	_, err := p.client.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create pod: %v", err)
	}

	pods, err := p.GetPods(ctx, "app=pod-app")
	if err != nil {
		t.Fatalf("GetPods() error = %v", err)
	}

	if len(pods) != 1 {
		t.Errorf("pod count = %d, want 1", len(pods))
	}
}

func TestK8sProvider_GetPods_NoSelector(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	// Create pods
	for _, name := range []string{"pod-1", "pod-2"} {
		p.client.CoreV1().Pods("default").Create(ctx, &k8sapi.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: k8sapi.PodSpec{
				Containers: []k8sapi.Container{{Name: name, Image: "nginx"}},
			},
		}, metav1.CreateOptions{})
	}

	pods, err := p.GetPods(ctx, "")
	if err != nil {
		t.Fatalf("GetPods() error = %v", err)
	}

	if len(pods) != 2 {
		t.Errorf("pod count = %d, want 2", len(pods))
	}
}

func TestK8sProvider_GetDeployment_NotFound(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	_, err := p.GetDeployment(ctx, "nonexistent")
	if err == nil {
		t.Error("GetDeployment() should fail for nonexistent deployment")
	}
}

func TestK8sProvider_DeleteDeployment_NotFound(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	err := p.DeleteDeployment(ctx, "nonexistent")
	if err == nil {
		t.Error("DeleteDeployment() should fail for nonexistent deployment")
	}
}

func TestK8sProvider_CreateService(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	// Create a deployment first
	p.Deploy(ctx, &K8sDeployConfig{Name: "svc-app", Image: "nginx:latest"})

	svcCfg := &K8sServiceConfig{
		Name:           "svc-app-svc",
		DeploymentName: "svc-app",
		Ports:          []int32{80},
		Type:           "ClusterIP",
	}

	err := p.CreateService(ctx, svcCfg)
	if err != nil {
		t.Fatalf("CreateService() error = %v", err)
	}

	services, err := p.GetServices(ctx)
	if err != nil {
		t.Fatalf("GetServices() error = %v", err)
	}

	found := false
	for _, svc := range services {
		if svc.Name == "svc-app-svc" {
			found = true
			if svc.Spec.Type != k8sapi.ServiceTypeClusterIP {
				t.Errorf("service type = %q, want %q", svc.Spec.Type, k8sapi.ServiceTypeClusterIP)
			}
		}
	}
	if !found {
		t.Error("service svc-app-svc not found")
	}
}

func TestK8sProvider_CreateService_NodePort(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	p.Deploy(ctx, &K8sDeployConfig{Name: "np-app", Image: "nginx:latest"})

	svcCfg := &K8sServiceConfig{
		Name:           "np-app-svc",
		DeploymentName: "np-app",
		Ports:          []int32{8080},
		Type:           "NodePort",
	}

	err := p.CreateService(ctx, svcCfg)
	if err != nil {
		t.Fatalf("CreateService() error = %v", err)
	}

	svc, err := p.client.CoreV1().Services("default").Get(ctx, "np-app-svc", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get service: %v", err)
	}
	if svc.Spec.Type != k8sapi.ServiceTypeNodePort {
		t.Errorf("service type = %q, want %q", svc.Spec.Type, k8sapi.ServiceTypeNodePort)
	}
}

func TestK8sProvider_CreateService_UpdateExisting(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	p.Deploy(ctx, &K8sDeployConfig{Name: "update-svc-app", Image: "nginx:latest"})

	svcCfg := &K8sServiceConfig{
		Name:           "update-svc-app-svc",
		DeploymentName: "update-svc-app",
		Ports:          []int32{80},
		Type:           "ClusterIP",
	}
	p.CreateService(ctx, svcCfg)

	// Update with new type
	svcCfg.Type = "LoadBalancer"
	err := p.CreateService(ctx, svcCfg)
	if err != nil {
		t.Fatalf("update CreateService() error = %v", err)
	}

	svc, _ := p.client.CoreV1().Services("default").Get(ctx, "update-svc-app-svc", metav1.GetOptions{})
	if svc.Spec.Type != k8sapi.ServiceTypeLoadBalancer {
		t.Errorf("service type = %q, want %q after update", svc.Spec.Type, k8sapi.ServiceTypeLoadBalancer)
	}
}

func TestK8sProvider_GetServices_Empty(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	services, err := p.GetServices(ctx)
	if err != nil {
		t.Fatalf("GetServices() error = %v", err)
	}

	// kubernetes service is always present in the default namespace with fake client
	if len(services) == 0 {
		t.Log("no services found (expected in empty namespace)")
	}
}

func TestK8sProvider_GetClusterInfo(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	// Create some nodes
	for _, name := range []string{"node-1", "node-2"} {
		p.client.CoreV1().Nodes().Create(ctx, &k8sapi.Node{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}, metav1.CreateOptions{})
	}

	info, err := p.GetClusterInfo(ctx)
	if err != nil {
		t.Fatalf("GetClusterInfo() error = %v", err)
	}

	if info["cluster_id"] != "cls-test" {
		t.Errorf("cluster_id = %v, want %q", info["cluster_id"], "cls-test")
	}
	if info["namespace"] != "default" {
		t.Errorf("namespace = %v, want %q", info["namespace"], "default")
	}
	if info["node_count"] != 2 {
		t.Errorf("node_count = %v, want 2", info["node_count"])
	}
}

func TestK8sProvider_TestConnection(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()

	err := p.TestConnection(ctx)
	if err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}
}

func TestBuildClientFromCluster_NoCredentials(t *testing.T) {
	cluster := &model.Cluster{
		APIServer: "https://localhost:6443",
	}

	_, err := BuildClientFromCluster(cluster)
	if err == nil {
		t.Error("BuildClientFromCluster() should fail without credentials")
	}
}

func TestBuildClientFromCluster_InvalidKubeConfig(t *testing.T) {
	cluster := &model.Cluster{
		APIServer:  "https://localhost:6443",
		KubeConfig: "invalid: yaml: content: [",
	}

	_, err := BuildClientFromCluster(cluster)
	if err == nil {
		t.Error("BuildClientFromCluster() should fail with invalid kubeconfig")
	}
}

func TestBuildClientFromToken_NoAPIServer(t *testing.T) {
	cluster := &model.Cluster{
		Token: "some-token",
	}

	_, err := buildClientFromToken(cluster)
	if err == nil {
		t.Error("buildClientFromToken() should fail without api_server")
	}
}

func TestBuildClientFromToken_InvalidCA(t *testing.T) {
	cluster := &model.Cluster{
		APIServer: "https://localhost:6443",
		Token:     "some-token",
		CAData:    "not-valid-pem-data",
	}

	_, err := buildClientFromToken(cluster)
	// Should fail because CA data is invalid PEM
	if err == nil {
		t.Error("buildClientFromToken() should fail with invalid CA data")
	}
}

func TestNewTLSConfig_NoCA(t *testing.T) {
	cfg, err := newTLSConfig(nil)
	if err != nil {
		t.Fatalf("newTLSConfig() error = %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true when no CA data")
	}
}

func TestNewTLSConfig_InvalidCA(t *testing.T) {
	_, err := newTLSConfig([]byte("not-valid-pem"))
	if err == nil {
		t.Error("newTLSConfig() should fail with invalid CA data")
	}
}

// ensure appsv1 and k8sapi are used
var _ = (*appsv1.Deployment)(nil)
var _ = (*k8sapi.Pod)(nil)
var _ = (*k8sapi.Service)(nil)
