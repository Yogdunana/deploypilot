package mcp

import (
	"github.com/Yogdunana/deploypilot/internal/model"
)

// PreflightErrorInfo is an interface used to detect preflight errors from the deployer.
// The service package's PreflightError implements this interface.
type PreflightErrorInfo interface {
	error
	PreflightCode() string
	PreflightMessage() string
	PreflightChecks() interface{}
}

// Deployer abstracts deployment operations for the MCP server.
type Deployer interface {
	Deploy(ctx context.Context, cfg DeployConfig) (*ContainerStatus, error)
	GetContainerStatus(ctx context.Context, name string) (*ContainerStatus, error)
	ListApps(ctx context.Context) ([]ContainerStatus, error)
	ListServers(ctx context.Context) ([]ServerInfo, error)
	CreateApp(ctx context.Context, cfg CreateAppConfig) (string, error)
	DeleteApp(ctx context.Context, appID string) error
	Stop(ctx context.Context, name string) error
	Remove(ctx context.Context, name string) error
	GetContainerLogs(ctx context.Context, name string, tail int) (string, error)
	Rollback(ctx context.Context, containerName, previousImage string) (*ContainerStatus, error)
	Backup(ctx context.Context, appID string) (string, error)
	Restore(ctx context.Context, backupID string) (*ContainerStatus, error)
	DetectEnv(ctx context.Context, level int, ports []int, services []string) (interface{}, error)
	HealthCheck(ctx context.Context, target, healthType string) (interface{}, error)
	AddServer(ctx context.Context, name, host string, port int, user string) (*ServerInfo, error)
	RemoveServer(ctx context.Context, serverID string) error
	TestServer(ctx context.Context, serverID string) (interface{}, error)
	CreateCredential(ctx context.Context, tenantID, name, credType, plainValue string) (interface{}, error)
	ListCredentials(ctx context.Context, tenantID string) (interface{}, error)
	DeleteCredential(ctx context.Context, credID string) error
	DNSCreateRecord(ctx context.Context, domain, recordType, name, value string) (interface{}, error)
	DNSDeleteRecord(ctx context.Context, recordID string) error
	DNSListRecords(ctx context.Context, domain string) (interface{}, error)
	SendNotification(ctx context.Context, nType, appName, server, status, message string) (interface{}, error)
	ListTemplates(ctx context.Context) (interface{}, error)
	GetTemplate(ctx context.Context, tmplType string) (interface{}, error)
	GetAppDetail(ctx context.Context, appID string) (interface{}, error)
	UpdateApp(ctx context.Context, appID string, config map[string]interface{}) (interface{}, error)
	GetTaskStatus(ctx context.Context, taskID string) (interface{}, error)
	ListTasks(ctx context.Context, limit int, statusFilter string) (interface{}, error)
	SearchAppLogs(ctx context.Context, appID, keyword string, limit int) (interface{}, error)
	UpdateDNSRecord(ctx context.Context, domain, subdomain, recordType, newValue string) (interface{}, error)
	UpdateCredential(ctx context.Context, credID string, value string) (interface{}, error)
	UpdateServer(ctx context.Context, serverID string, config map[string]interface{}) (interface{}, error)
	CheckDeployReadiness(ctx context.Context, appConfig map[string]interface{}) (interface{}, error)
	BatchDeploy(ctx context.Context, apps []map[string]interface{}) (interface{}, error)
	BatchDeployWithConfig(ctx context.Context, config BatchDeployConfig) (*BatchDeployResult, error)
	BatchBackup(ctx context.Context, appIDs []string) (interface{}, error)
	BatchDNS(ctx context.Context, records []map[string]interface{}) (interface{}, error)
	CheckSystemUpdate(ctx context.Context) (interface{}, error)
	GetLatestDeploymentRecord(ctx context.Context, containerName string) (*model.DeploymentRecord, error)
	BuildAndDeploy(ctx context.Context, cfg BuildAndDeployConfig) (*BuildAndDeployResult, error)
	HealContainer(ctx context.Context, containerName string) (interface{}, error)
	GetContainerMetrics(ctx context.Context, containerName string) (interface{}, error)
	GetSystemMetrics(ctx context.Context) (interface{}, error)
	ListAlerts(ctx context.Context) (interface{}, error)
	ListAlertRules(ctx context.Context) (interface{}, error)
	TriggerCIBuild(ctx context.Context, provider, repo, branch string) (interface{}, error)
	GetCIBuildStatus(ctx context.Context, provider, runID string) (interface{}, error)
	ListSSLCertificates(ctx context.Context) (interface{}, error)
	RequestSSLCertificate(ctx context.Context, domain, email string) (interface{}, error)
	RenewSSLCertificate(ctx context.Context, domain string) (interface{}, error)
	DeleteSSLCertificate(ctx context.Context, domain string) (interface{}, error)
	RegistryOps(registryID string, operation string, args map[string]interface{}) (interface{}, error)
	// Kubernetes cluster management
	CreateCluster(ctx context.Context, cluster *model.Cluster) (*model.Cluster, error)
	GetCluster(ctx context.Context, id string) (*model.Cluster, error)
	ListClusters(ctx context.Context, tenantID string) ([]model.Cluster, error)
	UpdateCluster(ctx context.Context, id string, updates map[string]interface{}) (*model.Cluster, error)
	DeleteCluster(ctx context.Context, id string) error
	TestClusterConnection(ctx context.Context, id string) (interface{}, error)
	// Kubernetes deployment operations
	K8sDeploy(ctx context.Context, clusterID string, app *K8sDeployConfig) error
	K8sListDeployments(ctx context.Context, clusterID string) (interface{}, error)
	K8sGetPods(ctx context.Context, clusterID, labelSelector string) (interface{}, error)
	PluginOps(pluginID string, action string) (interface{}, error)
	ListPlugins(provider string) (interface{}, error)
	GetPluginInfo(pluginID string) (interface{}, error)
}

// DeployConfig mirrors deployer.DeployConfig to avoid circular imports.
type DeployConfig struct {
	Image         string            `json:"image"`
	AppName       string            `json:"app_name,omitempty"`
	ContainerName string            `json:"container_name"`
	Ports         string            `json:"ports,omitempty"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	RestartPolicy string            `json:"restart_policy,omitempty"`
	Network       string            `json:"network,omitempty"`
	Volumes       string            `json:"volumes,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	CPU           string            `json:"cpu,omitempty"`
	Memory        string            `json:"memory,omitempty"`
	ServerID      string            `json:"server_id,omitempty"`
}

// ContainerStatus mirrors deployer.ContainerStatus.
type ContainerStatus struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Status    string            `json:"status"`
	Ports     string            `json:"ports,omitempty"`
	CreatedAt string            `json:"created_at,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// DeployStrategy defines the deployment strategy type.
type DeployStrategy string

const (
	StrategySequential DeployStrategy = "sequential"
	StrategyParallel   DeployStrategy = "parallel"
	StrategyRolling    DeployStrategy = "rolling"
)

// BatchDeployConfig holds configuration for batch deployment.
type BatchDeployConfig struct {
	Apps          []map[string]interface{} `json:"apps"`
	Strategy      DeployStrategy           `json:"strategy"`
	MaxConcurrent int                      `json:"max_concurrent"`
	BatchSize     int                      `json:"batch_size"`
	ServerIDs     []string                 `json:"server_ids"`
}

// BatchDeployResult holds the result of a batch deployment.
type BatchDeployResult struct {
	Total    int                        `json:"total"`
	Success  int                        `json:"success"`
	Failed   int                        `json:"failed"`
	Results  []BatchDeployItemResult    `json:"results"`
	Duration float64                    `json:"duration_seconds"`
}

// BatchDeployItemResult holds the result of a single app deployment within a batch.
type BatchDeployItemResult struct {
	Index   int    `json:"index"`
	AppName string `json:"app_name"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ServerInfo represents a registered server.
type ServerInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Status string `json:"status"`
}

// CreateAppConfig holds parameters for creating a new application.
type CreateAppConfig struct {
	Name       string `json:"name"`
	RepoURL    string `json:"repo_url"`
	Branch     string `json:"branch,omitempty"`
	Domain     string `json:"domain,omitempty"`
	TechStack  string `json:"tech_stack,omitempty"`
	DeployMode string `json:"deploy_mode,omitempty"`
	ServerID   string `json:"server_id,omitempty"`
}

// BuildAndDeployConfig holds parameters for a build-and-deploy operation.
type BuildAndDeployConfig struct {
	RepoURL             string            `json:"repo_url"`
	Branch              string            `json:"branch,omitempty"`
	TechStack           string            `json:"tech_stack,omitempty"`
	AppName             string            `json:"app_name"`
	ProjectDir          string            `json:"project_dir,omitempty"`
	BuildArgs           map[string]string `json:"build_args,omitempty"`
	EnvVars             map[string]string `json:"env_vars,omitempty"`
	Ports               string            `json:"ports,omitempty"`
	ServerID            string            `json:"server_id,omitempty"`
	DockerfileOverrides map[string]string `json:"dockerfile_overrides,omitempty"`
	RegistryID          string            `json:"registry_id,omitempty"`
	PushImage           bool              `json:"push_image,omitempty"`
	ImageTag            string            `json:"image_tag,omitempty"`
}

// BuildAndDeployResult holds the result of a build-and-deploy operation.
type BuildAndDeployResult struct {
	Image      string  `json:"image"`
	Digest     string  `json:"digest,omitempty"`
	Size       string  `json:"size,omitempty"`
	BuildLog   string  `json:"build_log"`
	Duration   float64 `json:"duration_seconds"`
	TechStack  string  `json:"tech_stack"`
	CommitHash string  `json:"commit_hash"`
}

// K8sDeployConfig holds parameters for deploying an application to a Kubernetes cluster.
type K8sDeployConfig struct {
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Replicas  int32             `json:"replicas"`
	Ports     []int32           `json:"ports"`
	EnvVars   map[string]string `json:"env_vars"`
	Labels    map[string]string `json:"labels"`
	Namespace string            `json:"namespace"`
}
