package server

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	k8sapi "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/Yogdunana/deploypilot/internal/model"
)

// K8sDeployConfig holds parameters for deploying an application to Kubernetes.
type K8sDeployConfig struct {
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Replicas  int32             `json:"replicas"`
	Ports     []int32           `json:"ports"`
	EnvVars   map[string]string `json:"env_vars"`
	Labels    map[string]string `json:"labels"`
	Namespace string            `json:"namespace"` // override default
}

// K8sServiceConfig holds parameters for creating a Kubernetes service.
type K8sServiceConfig struct {
	Name           string  `json:"name"`
	DeploymentName string  `json:"deployment_name"`
	Ports          []int32 `json:"ports"`
	Type           string  `json:"type"` // ClusterIP, NodePort, LoadBalancer
}

// K8sProvider manages Kubernetes cluster operations.
type K8sProvider struct {
	client    k8sclient.Interface
	namespace string
	clusterID string
}

// NewK8sProvider creates a K8s provider from a cluster model.
// It retrieves the cluster with decrypted secrets and builds a k8s client.
func NewK8sProvider(cluster *model.Cluster) (*K8sProvider, error) {
	clientset, err := BuildClientFromCluster(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to build k8s client: %w", err)
	}

	ns := cluster.Namespace
	if ns == "" {
		ns = "default"
	}

	return &K8sProvider{
		client:    clientset,
		namespace: ns,
		clusterID: cluster.ID,
	}, nil
}

// NewK8sProviderWithClient creates a K8s provider with an explicit client (for testing).
func NewK8sProviderWithClient(client k8sclient.Interface, namespace, clusterID string) *K8sProvider {
	if namespace == "" {
		namespace = "default"
	}
	return &K8sProvider{
		client:    client,
		namespace: namespace,
		clusterID: clusterID,
	}
}

// effectiveNamespace returns the namespace to use, preferring the config override.
func (k *K8sProvider) effectiveNamespace(cfg *K8sDeployConfig) string {
	if cfg != nil && cfg.Namespace != "" {
		return cfg.Namespace
	}
	return k.namespace
}

// Deploy creates or updates a Kubernetes Deployment for the given app config.
func (k *K8sProvider) Deploy(ctx context.Context, app *K8sDeployConfig) error {
	ns := k.effectiveNamespace(app)

	// Build container ports
	var ports []k8sapi.ContainerPort
	for _, p := range app.Ports {
		ports = append(ports, k8sapi.ContainerPort{
			ContainerPort: p,
		})
	}

	// Build env vars
	var envVars []k8sapi.EnvVar
	for key, value := range app.EnvVars {
		envVars = append(envVars, k8sapi.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// Build labels
	labels := map[string]string{
		"app": app.Name,
	}
	for key, value := range app.Labels {
		labels[key] = value
	}

	replicas := app.Replicas
	if replicas <= 0 {
		replicas = 1
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": app.Name},
			},
			Template: k8sapi.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: k8sapi.PodSpec{
					Containers: []k8sapi.Container{
						{
							Name:            app.Name,
							Image:           app.Image,
							Ports:           ports,
							Env:             envVars,
							ImagePullPolicy: k8sapi.PullIfNotPresent,
						},
					},
				},
			},
		},
	}

	// Try to create; if it exists, update instead
	_, err := k.client.AppsV1().Deployments(ns).Create(ctx, deployment, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		// Get the existing deployment to preserve resource version
		existing, getErr := k.client.AppsV1().Deployments(ns).Get(ctx, app.Name, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get existing deployment: %w", getErr)
		}
		deployment.ResourceVersion = existing.ResourceVersion
		_, err = k.client.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{})
	}
	if err != nil {
		return fmt.Errorf("failed to deploy %s: %w", app.Name, err)
	}

	return nil
}

// GetDeployment retrieves a deployment by name.
func (k *K8sProvider) GetDeployment(ctx context.Context, name string) (*appsv1.Deployment, error) {
	deploy, err := k.client.AppsV1().Deployments(k.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment %s: %w", name, err)
	}
	return deploy, nil
}

// ListDeployments lists all deployments in the provider's namespace.
func (k *K8sProvider) ListDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	list, err := k.client.AppsV1().Deployments(k.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	return list.Items, nil
}

// DeleteDeployment deletes a deployment by name.
func (k *K8sProvider) DeleteDeployment(ctx context.Context, name string) error {
	err := k.client.AppsV1().Deployments(k.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete deployment %s: %w", name, err)
	}
	return nil
}

// ScaleDeployment scales a deployment to the specified number of replicas.
func (k *K8sProvider) ScaleDeployment(ctx context.Context, name string, replicas int32) error {
	deploy, err := k.client.AppsV1().Deployments(k.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment %s: %w", name, err)
	}

	deploy.Spec.Replicas = &replicas
	_, err = k.client.AppsV1().Deployments(k.namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale deployment %s to %d replicas: %w", name, replicas, err)
	}

	return nil
}

// GetPods retrieves pods, optionally filtered by label selector.
func (k *K8sProvider) GetPods(ctx context.Context, labelSelector string) ([]k8sapi.Pod, error) {
	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	list, err := k.client.CoreV1().Pods(k.namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	return list.Items, nil
}

// GetLogs retrieves container logs for a deployment's pods.
func (k *K8sProvider) GetLogs(ctx context.Context, name string, tailLines int64) (string, error) {
	// Find pods belonging to this deployment
	pods, err := k.GetPods(ctx, fmt.Sprintf("app=%s", name))
	if err != nil {
		return "", fmt.Errorf("failed to get pods for logs: %w", err)
	}

	if len(pods) == 0 {
		return "", fmt.Errorf("no pods found for deployment %s", name)
	}

	opts := &k8sapi.PodLogOptions{
		Container: name,
	}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}

	// Get logs from the first pod (most recent)
	req := k.client.CoreV1().Pods(k.namespace).GetLogs(pods[0].Name, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to stream logs: %w", err)
	}
	defer func() { _ = stream.Close() }()

	var builder strings.Builder
	buf := make([]byte, 4096)
	for {
		n, readErr := stream.Read(buf)
		if n > 0 {
			builder.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}

	return builder.String(), nil
}

// GetServices lists all services in the provider's namespace.
func (k *K8sProvider) GetServices(ctx context.Context) ([]k8sapi.Service, error) {
	list, err := k.client.CoreV1().Services(k.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	return list.Items, nil
}

// CreateService creates a Kubernetes service for a deployment.
func (k *K8sProvider) CreateService(ctx context.Context, svc *K8sServiceConfig) error {
	ns := k.namespace

	var ports []k8sapi.ServicePort
	for _, p := range svc.Ports {
		ports = append(ports, k8sapi.ServicePort{
			Port:       p,
			TargetPort: getTargetPort(p),
			Protocol:   k8sapi.ProtocolTCP,
		})
	}

	svcType := k8sapi.ServiceTypeClusterIP
	switch strings.ToLower(svc.Type) {
	case "nodeport":
		svcType = k8sapi.ServiceTypeNodePort
	case "loadbalancer":
		svcType = k8sapi.ServiceTypeLoadBalancer
	}

	service := &k8sapi.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: ns,
			Labels: map[string]string{
				"app": svc.DeploymentName,
			},
		},
		Spec: k8sapi.ServiceSpec{
			Type:     svcType,
			Selector: map[string]string{"app": svc.DeploymentName},
			Ports:    ports,
		},
	}

	_, err := k.client.CoreV1().Services(ns).Create(ctx, service, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		existing, getErr := k.client.CoreV1().Services(ns).Get(ctx, svc.Name, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get existing service: %w", getErr)
		}
		service.ResourceVersion = existing.ResourceVersion
		_, err = k.client.CoreV1().Services(ns).Update(ctx, service, metav1.UpdateOptions{})
	}
	if err != nil {
		return fmt.Errorf("failed to create service %s: %w", svc.Name, err)
	}

	return nil
}

// TestConnection verifies connectivity to the Kubernetes cluster.
func (k *K8sProvider) TestConnection(ctx context.Context) error {
	_, err := k.client.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to kubernetes cluster: %w", err)
	}
	return nil
}

// GetClusterInfo retrieves cluster information.
func (k *K8sProvider) GetClusterInfo(ctx context.Context) (map[string]interface{}, error) {
	version, err := k.client.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster version: %w", err)
	}

	nodes, err := k.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	info := map[string]interface{}{
		"version":     version.GitVersion,
		"platform":    version.Platform,
		"node_count":  len(nodes.Items),
		"cluster_id":  k.clusterID,
		"namespace":   k.namespace,
	}

	return info, nil
}

// getTargetPort converts a service port to an intstr.IntOrString target port.
func getTargetPort(port int32) intstr.IntOrString {
	return intstr.IntOrString{
		IntVal: port,
	}
}
