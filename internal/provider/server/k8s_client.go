package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Yogdunana/deploypilot/internal/model"
)

// BuildClientFromCluster creates a kubernetes.Clientset from a Cluster model.
// It supports two auth modes:
// 1. KubeConfig: if cluster.KubeConfig is set, parse it and create client.
// 2. Token + CA: if cluster.Token is set, use token auth with CA cert.
func BuildClientFromCluster(cluster *model.Cluster) (*k8sclient.Clientset, error) {
	if cluster.KubeConfig != "" {
		return buildClientFromKubeConfig(cluster)
	}
	if cluster.Token != "" {
		return buildClientFromToken(cluster)
	}
	return nil, fmt.Errorf("cluster has no authentication credentials: either kube_config or token is required")
}

// buildClientFromKubeConfig creates a client from kubeconfig content.
func buildClientFromKubeConfig(cluster *model.Cluster) (*k8sclient.Clientset, error) {
	config, err := clientcmd.Load([]byte(cluster.KubeConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	// Override context if specified
	if cluster.Context != "" {
		if _, ok := config.Contexts[cluster.Context]; !ok {
			return nil, fmt.Errorf("context %q not found in kubeconfig", cluster.Context)
		}
		config.CurrentContext = cluster.Context
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest config from kubeconfig: %w", err)
	}

	clientset, err := k8sclient.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// buildClientFromToken creates a client using bearer token authentication.
func buildClientFromToken(cluster *model.Cluster) (*k8sclient.Clientset, error) {
	if cluster.APIServer == "" {
		return nil, fmt.Errorf("cluster api_server is required for token auth")
	}

	restConfig := &rest.Config{
		Host:        cluster.APIServer,
		BearerToken: cluster.Token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: false,
		},
	}

	// Configure CA cert if provided
	if cluster.CAData != "" {
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM([]byte(cluster.CAData)) {
			return nil, fmt.Errorf("failed to parse CA certificate data")
		}
		restConfig.TLSClientConfig.CAData = []byte(cluster.CAData)
		restConfig.TLSClientConfig.Insecure = false
	} else {
		// No CA data: use insecure mode (not recommended for production)
		restConfig.TLSClientConfig.Insecure = true
	}

	clientset, err := k8sclient.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// newTLSConfig creates a tls.Config from CA data.
func newTLSConfig(caData []byte) (*tls.Config, error) {
	if len(caData) == 0 {
		return &tls.Config{InsecureSkipVerify: true}, nil
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}
	return &tls.Config{
		RootCAs: caPool,
	}, nil
}
