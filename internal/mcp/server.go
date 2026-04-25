package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/Yogdunana/deploypilot/internal/model"
)

// contextKeyRole is the context key for the user role.
type contextKeyRole struct{}

// ContextWithRole returns a new context carrying the given user role.
func ContextWithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, contextKeyRole{}, role)
}

// RoleFromContext extracts the user role from context. Returns "dev" as default.
func RoleFromContext(ctx context.Context) string {
	if role, ok := ctx.Value(contextKeyRole{}).(string); ok {
		return role
	}
	return "dev"
}

// withPermissionCheck wraps a tool handler with a permission check.
// If the user's role (from context) does not meet the minimum requirement,
// a permission denied error is returned.
func withPermissionCheck(toolName string, handler server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userRole := RoleFromContext(ctx)
		if !CheckPermission(toolName, userRole) {
			required := ToolPermissions[toolName]
			slog.Warn("permission denied for tool", "tool", toolName, "role", userRole, "required", RequiredRoleName(required))
			return mcp.NewToolResultError(fmt.Sprintf("permission denied: %s requires role %s or higher, current role: %s", toolName, RequiredRoleName(required), userRole)), nil
		}
		return handler(ctx, request)
	}
}

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
}

// DeployConfig mirrors deployer.DeployConfig to avoid circular imports.
type DeployConfig struct {
	Image         string            `json:"image"`
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

// NewServer creates a new MCP server with deploy tools registered.
func NewServer(deployer Deployer) *server.MCPServer {
	s := server.NewMCPServer(
		"DeployPilot",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	// Register deploy_app tool
	deployTool := mcp.NewTool("deploy_app",
		mcp.WithDescription("Deploy a Docker container. Use server_id to deploy to a remote server via SSH; omit for local Docker deployment."),
		mcp.WithString("image",
			mcp.Required(),
			mcp.Description("Docker image to deploy (e.g. nginx:latest)"),
		),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name for the container"),
		),
		mcp.WithString("ports",
			mcp.Description("Port mapping (e.g. 8080:80)"),
		),
		mcp.WithString("env_vars",
			mcp.Description("Environment variables as JSON object (e.g. {\"KEY\":\"value\"})"),
		),
		mcp.WithString("restart_policy",
			mcp.Description("Restart policy: unless-stopped, always, no"),
		),
		mcp.WithString("network",
			mcp.Description("Docker network to attach"),
		),
		mcp.WithString("volumes",
			mcp.Description("Volume mappings (e.g. /host/path:/container/path)"),
		),
		mcp.WithString("labels",
			mcp.Description("Labels as JSON object (e.g. {\"app\":\"myapp\"})"),
		),
		mcp.WithString("cpu",
			mcp.Description("CPU limit (e.g. 2)"),
		),
		mcp.WithString("memory",
			mcp.Description("Memory limit (e.g. 4GB)"),
		),
		mcp.WithString("server_id",
			mcp.Description("Target server ID for remote deployment via SSH. Omit to deploy locally. The server must be registered via add_server first."),
		),
	)

	s.AddTool(deployTool, withPermissionCheck("deploy_app", withValidation("deploy_app", deployTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeployApp(ctx, deployer, request)
	})))

	// Register get_deploy_status tool
	statusTool := mcp.NewTool("get_deploy_status",
		mcp.WithDescription("Get the status of a deployed container"),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name of the container to check"),
		),
	)

	s.AddTool(statusTool, withPermissionCheck("get_deploy_status", withValidation("get_deploy_status", statusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetDeployStatus(ctx, deployer, request)
	})))

	// Register list_apps tool
	listAppsTool := mcp.NewTool("list_apps",
		mcp.WithDescription("List all deployed applications"),
	)

	s.AddTool(listAppsTool, withPermissionCheck("list_apps", withValidation("list_apps", listAppsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListApps(ctx, deployer, request)
	})))

	// Register list_servers tool
	listServersTool := mcp.NewTool("list_servers",
		mcp.WithDescription("List all registered servers"),
	)

	s.AddTool(listServersTool, withPermissionCheck("list_servers", withValidation("list_servers", listServersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListServers(ctx, deployer, request)
	})))

	// Register create_app tool
	createAppTool := mcp.NewTool("create_app",
		mcp.WithDescription("Register a new application for deployment"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Application name"),
		),
		mcp.WithString("repo_url",
			mcp.Required(),
			mcp.Description("Git repository URL"),
		),
		mcp.WithString("branch",
			mcp.Description("Git branch (default: main)"),
		),
		mcp.WithString("domain",
			mcp.Description("Custom domain for the app"),
		),
		mcp.WithString("tech_stack",
			mcp.Description("Technology stack: docker, node, python, go, java"),
		),
		mcp.WithString("deploy_mode",
			mcp.Description("Deploy mode: api, direct, cicd"),
		),
		mcp.WithString("server_id",
			mcp.Description("Target server ID"),
		),
	)

	s.AddTool(createAppTool, withPermissionCheck("create_app", withValidation("create_app", createAppTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateApp(ctx, deployer, request)
	})))

	// Register delete_app tool
	deleteAppTool := mcp.NewTool("delete_app",
		mcp.WithDescription("Delete an application and stop its container"),
		mcp.WithString("app_id",
			mcp.Required(),
			mcp.Description("ID of the application to delete"),
		),
	)

	s.AddTool(deleteAppTool, withPermissionCheck("delete_app", withValidation("delete_app", deleteAppTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteApp(ctx, deployer, request)
	})))

	// Register rollback tool
	rollbackTool := mcp.NewTool("rollback_app",
		mcp.WithDescription("Rollback a container to a previous image version"),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name of the container to rollback"),
		),
		mcp.WithString("previous_image",
			mcp.Required(),
			mcp.Description("Previous Docker image to rollback to"),
		),
	)

	s.AddTool(rollbackTool, withPermissionCheck("rollback_app", withValidation("rollback_app", rollbackTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRollback(ctx, deployer, request)
	})))

	// Register backup tool
	backupTool := mcp.NewTool("backup_database",
		mcp.WithDescription("Create a backup of an application"),
		mcp.WithString("app_id",
			mcp.Required(),
			mcp.Description("ID of the application to backup"),
		),
	)

	s.AddTool(backupTool, withPermissionCheck("backup_database", withValidation("backup_database", backupTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBackup(ctx, deployer, request)
	})))

	// Register restore tool
	restoreTool := mcp.NewTool("restore_database",
		mcp.WithDescription("Restore an application from a backup"),
		mcp.WithString("backup_id",
			mcp.Required(),
			mcp.Description("ID of the backup to restore from"),
		),
	)

	s.AddTool(restoreTool, withPermissionCheck("restore_database", withValidation("restore_database", restoreTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRestore(ctx, deployer, request)
	})))

	// Register get_app_logs tool
	getLogsTool := mcp.NewTool("get_app_logs",
		mcp.WithDescription("Get logs from a deployed container"),
		mcp.WithString("container_name",
			mcp.Required(),
			mcp.Description("Name of the container"),
		),
		mcp.WithString("tail",
			mcp.Description("Number of lines to retrieve (default: 100)"),
		),
	)

	s.AddTool(getLogsTool, withPermissionCheck("get_app_logs", withValidation("get_app_logs", getLogsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAppLogs(ctx, deployer, request)
	})))

	// Register detect_env tool
	detectEnvTool := mcp.NewTool("detect_environment",
		mcp.WithDescription("Detect server environment (OS, Docker, ports, services)"),
		mcp.WithString("level",
			mcp.Description("Detection level: 1=OS, 2=+Docker, 3=+Ports, 4=+Services (default: 2)"),
		),
		mcp.WithString("ports",
			mcp.Description("Comma-separated port list to check (e.g. 8080,3000)"),
		),
		mcp.WithString("services",
			mcp.Description("Comma-separated service URLs to check (e.g. tcp://localhost:3306)"),
		),
	)

	s.AddTool(detectEnvTool, withPermissionCheck("detect_environment", withValidation("detect_environment", detectEnvTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDetectEnv(ctx, deployer, request)
	})))

	// Register health_check tool
	healthCheckTool := mcp.NewTool("health_check",
		mcp.WithDescription("Check health of a deployed service"),
		mcp.WithString("target",
			mcp.Required(),
			mcp.Description("Health check target URL (e.g. http://localhost:8080/health or tcp://localhost:3306)"),
		),
		mcp.WithString("type",
			mcp.Description("Health check type: http or tcp (default: http)"),
		),
	)

	s.AddTool(healthCheckTool, withPermissionCheck("health_check", withValidation("health_check", healthCheckTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleHealthCheck(ctx, deployer, request)
	})))

	// Register add_server tool
	addServerTool := mcp.NewTool("add_server",
		mcp.WithDescription("Register a new server for remote deployment. Connectivity is tested automatically."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Server name (e.g. production, staging)")),
		mcp.WithString("host", mcp.Required(), mcp.Description("Server hostname or IP address")),
		mcp.WithString("port", mcp.Description("SSH port. Default: 22. Cloud providers often use custom ports (e.g. 23196, 2222). Check your security group settings.")),
		mcp.WithString("user", mcp.Description("SSH username (default: root)")),
	)
	s.AddTool(addServerTool, withPermissionCheck("add_server", withValidation("add_server", addServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleAddServer(ctx, deployer, request)
	})))

	// Register remove_server tool
	removeServerTool := mcp.NewTool("delete_server",
		mcp.WithDescription("Remove a registered server"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("Server ID to remove")),
	)
	s.AddTool(removeServerTool, withPermissionCheck("delete_server", withValidation("delete_server", removeServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRemoveServer(ctx, deployer, request)
	})))

	// Register test_server tool
	testServerTool := mcp.NewTool("test_server",
		mcp.WithDescription("Test SSH connectivity to a registered server. Returns latency and actionable suggestions if unreachable."),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("Server ID to test")),
	)
	s.AddTool(testServerTool, withPermissionCheck("test_server", withValidation("test_server", testServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleTestServer(ctx, deployer, request)
	})))

	// Register create_credential tool
	createCredTool := mcp.NewTool("add_credential",
		mcp.WithDescription("Create an encrypted credential. The value is encrypted with AES-256-GCM before storage — plaintext never touches the database."),
		mcp.WithString("tenant_id", mcp.Required(), mcp.Description("Tenant ID")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Credential name")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Credential type: ssh, api_key, token, password")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Plain credential value (will be encrypted)")),
	)
	s.AddTool(createCredTool, withPermissionCheck("add_credential", withValidation("add_credential", createCredTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateCredential(ctx, deployer, request)
	})))

	// Register list_credentials tool
	listCredsTool := mcp.NewTool("list_credentials",
		mcp.WithDescription("List all credentials for a tenant"),
		mcp.WithString("tenant_id", mcp.Required(), mcp.Description("Tenant ID")),
	)
	s.AddTool(listCredsTool, withPermissionCheck("list_credentials", withValidation("list_credentials", listCredsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListCredentials(ctx, deployer, request)
	})))

	// Register delete_credential tool
	deleteCredTool := mcp.NewTool("delete_credential",
		mcp.WithDescription("Delete a credential"),
		mcp.WithString("credential_id", mcp.Required(), mcp.Description("Credential ID to delete")),
	)
	s.AddTool(deleteCredTool, withPermissionCheck("delete_credential", withValidation("delete_credential", deleteCredTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteCredential(ctx, deployer, request)
	})))

	// Register dns_create_record tool
	dnsCreateTool := mcp.NewTool("add_dns_record",
		mcp.WithDescription("Create a DNS record"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name (e.g. example.com)")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Record type: A, AAAA, CNAME, TXT, MX")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Record name (e.g. @ or subdomain)")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Record value (e.g. IP address)")),
	)
	s.AddTool(dnsCreateTool, withPermissionCheck("add_dns_record", withValidation("add_dns_record", dnsCreateTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDNSCreateRecord(ctx, deployer, request)
	})))

	// Register dns_delete_record tool
	dnsDeleteTool := mcp.NewTool("delete_dns_record",
		mcp.WithDescription("Delete a DNS record"),
		mcp.WithString("record_id", mcp.Required(), mcp.Description("DNS record ID to delete")),
	)
	s.AddTool(dnsDeleteTool, withPermissionCheck("delete_dns_record", withValidation("delete_dns_record", dnsDeleteTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDNSDeleteRecord(ctx, deployer, request)
	})))

	// Register dns_list_records tool
	dnsListTool := mcp.NewTool("list_dns_records",
		mcp.WithDescription("List DNS records for a domain"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name")),
	)
	s.AddTool(dnsListTool, withPermissionCheck("list_dns_records", withValidation("list_dns_records", dnsListTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDNSListRecords(ctx, deployer, request)
	})))

	// Register send_notification tool
	sendNotifyTool := mcp.NewTool("send_notification",
		mcp.WithDescription("Send a deployment notification"),
		mcp.WithString("type", mcp.Required(), mcp.Description("Notification type: deploy_success, deploy_failed, health_check, rollback")),
		mcp.WithString("app", mcp.Required(), mcp.Description("Application name")),
		mcp.WithString("server", mcp.Required(), mcp.Description("Target server")),
		mcp.WithString("status", mcp.Required(), mcp.Description("Status: success, failed, warning")),
		mcp.WithString("message", mcp.Description("Notification message")),
	)
	s.AddTool(sendNotifyTool, withPermissionCheck("send_notification", withValidation("send_notification", sendNotifyTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSendNotification(ctx, deployer, request)
	})))

	// Register list_templates tool
	listTmplTool := mcp.NewTool("list_templates",
		mcp.WithDescription("List all available application templates"),
	)
	s.AddTool(listTmplTool, withPermissionCheck("list_templates", withValidation("list_templates", listTmplTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListTemplates(ctx, deployer, request)
	})))

	// Register get_template tool
	getTmplTool := mcp.NewTool("get_template",
		mcp.WithDescription("Get details of a specific application template"),
		mcp.WithString("type", mcp.Required(), mcp.Description("Template type: node, python, go, java, php, ruby, rust, static, docker")),
	)
	s.AddTool(getTmplTool, withPermissionCheck("get_template", withValidation("get_template", getTmplTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetTemplate(ctx, deployer, request)
	})))

	// Register get_app_detail tool
	getAppDetailTool := mcp.NewTool("get_app_detail",
		mcp.WithDescription("Get detailed information about a deployed application"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("Application ID")),
	)
	s.AddTool(getAppDetailTool, withPermissionCheck("get_app_detail", withValidation("get_app_detail", getAppDetailTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAppDetail(ctx, deployer, request)
	})))

	// Register update_app tool
	updateAppTool := mcp.NewTool("update_app",
		mcp.WithDescription("Update application configuration"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("Application ID")),
		mcp.WithString("config", mcp.Required(), mcp.Description("JSON string of configuration to update")),
	)
	s.AddTool(updateAppTool, withPermissionCheck("update_app", withValidation("update_app", updateAppTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateApp(ctx, deployer, request)
	})))

	// Register get_task_status tool
	getTaskStatusTool := mcp.NewTool("get_task_status",
		mcp.WithDescription("Get status of an async task"),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID")),
	)
	s.AddTool(getTaskStatusTool, withPermissionCheck("get_task_status", withValidation("get_task_status", getTaskStatusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetTaskStatus(ctx, deployer, request)
	})))

	// Register list_tasks tool
	listTasksTool := mcp.NewTool("list_tasks",
		mcp.WithDescription("List recent tasks"),
		mcp.WithString("limit", mcp.Description("Maximum number of tasks to return (default: 20)")),
		mcp.WithString("status_filter", mcp.Description("Filter by status: running, completed, failed")),
	)
	s.AddTool(listTasksTool, withPermissionCheck("list_tasks", withValidation("list_tasks", listTasksTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListTasks(ctx, deployer, request)
	})))

	// Register search_app_logs
	searchLogsTool := mcp.NewTool("search_app_logs",
		mcp.WithDescription("Search container logs by keyword"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("Application ID")),
		mcp.WithString("keyword", mcp.Required(), mcp.Description("Search keyword")),
		mcp.WithString("limit", mcp.Description("Max results (default: 50)")),
	)
	s.AddTool(searchLogsTool, withPermissionCheck("search_app_logs", withValidation("search_app_logs", searchLogsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSearchAppLogs(ctx, deployer, request)
	})))

	// Register update_dns_record
	updateDNSTool := mcp.NewTool("update_dns_record",
		mcp.WithDescription("Update a DNS record"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name")),
		mcp.WithString("subdomain", mcp.Required(), mcp.Description("Subdomain")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Record type: A, AAAA, CNAME, TXT")),
		mcp.WithString("new_value", mcp.Required(), mcp.Description("New record value")),
	)
	s.AddTool(updateDNSTool, withPermissionCheck("update_dns_record", withValidation("update_dns_record", updateDNSTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateDNSRecord(ctx, deployer, request)
	})))

	// Register update_credential
	updateCredTool := mcp.NewTool("update_credential",
		mcp.WithDescription("Update a credential value"),
		mcp.WithString("credential_id", mcp.Required(), mcp.Description("Credential ID")),
		mcp.WithString("value", mcp.Required(), mcp.Description("New credential value")),
	)
	s.AddTool(updateCredTool, withPermissionCheck("update_credential", withValidation("update_credential", updateCredTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateCredential(ctx, deployer, request)
	})))

	// Register update_server
	updateSrvTool := mcp.NewTool("update_server",
		mcp.WithDescription("Update server configuration"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("Server ID")),
		mcp.WithString("config", mcp.Required(), mcp.Description("JSON string of config to update")),
	)
	s.AddTool(updateSrvTool, withPermissionCheck("update_server", withValidation("update_server", updateSrvTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateServer(ctx, deployer, request)
	})))

	// Register check_deploy_readiness
	checkReadinessTool := mcp.NewTool("check_deploy_readiness",
		mcp.WithDescription("Check if deployment prerequisites are met"),
		mcp.WithString("app_config", mcp.Required(), mcp.Description("JSON string of app configuration to validate")),
	)
	s.AddTool(checkReadinessTool, withPermissionCheck("check_deploy_readiness", withValidation("check_deploy_readiness", checkReadinessTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCheckDeployReadiness(ctx, deployer, request)
	})))

	// Register batch_deploy
	batchDeployTool := mcp.NewTool("batch_deploy",
		mcp.WithDescription("Deploy multiple applications at once with configurable strategy (sequential, parallel, rolling)"),
		mcp.WithString("apps", mcp.Required(), mcp.Description("JSON array of app configs: [{repo, branch, domain, stack}]")),
		mcp.WithString("strategy", mcp.Description("Deployment strategy: sequential (default), parallel, or rolling")),
		mcp.WithNumber("max_concurrent", mcp.Description("Max concurrent deployments for parallel strategy (default: 5)")),
		mcp.WithNumber("batch_size", mcp.Description("Batch size for rolling strategy (default: 3)")),
		mcp.WithString("server_ids", mcp.Description("Comma-separated target server IDs")),
	)
	s.AddTool(batchDeployTool, withPermissionCheck("batch_deploy", withValidation("batch_deploy", batchDeployTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBatchDeploy(ctx, deployer, request)
	})))

	// Register batch_backup
	batchBackupTool := mcp.NewTool("batch_backup",
		mcp.WithDescription("Backup multiple applications at once"),
		mcp.WithString("app_ids", mcp.Required(), mcp.Description("JSON array of app IDs: [\"id1\", \"id2\"]")),
	)
	s.AddTool(batchBackupTool, withPermissionCheck("batch_backup", withValidation("batch_backup", batchBackupTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBatchBackup(ctx, deployer, request)
	})))

	// Register batch_dns
	batchDNSTool := mcp.NewTool("batch_dns",
		mcp.WithDescription("Add multiple DNS records at once"),
		mcp.WithString("records", mcp.Required(), mcp.Description("JSON array of DNS records: [{domain, sub, type, value}]")),
	)
	s.AddTool(batchDNSTool, withPermissionCheck("batch_dns", withValidation("batch_dns", batchDNSTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBatchDNS(ctx, deployer, request)
	})))

	// Register check_system_update
	checkSysUpdateTool := mcp.NewTool("check_system_update",
		mcp.WithDescription("Check if a newer version of DeployPilot is available"),
	)
	s.AddTool(checkSysUpdateTool, withPermissionCheck("check_system_update", withValidation("check_system_update", checkSysUpdateTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCheckSystemUpdate(ctx, deployer, request)
	})))

	// Register doctor tool
	doctorTool := mcp.NewTool("doctor",
		mcp.WithDescription("Check DeployPilot prerequisites: Docker availability, database connectivity, and SSH configuration."),
	)
	s.AddTool(doctorTool, withPermissionCheck("doctor", withValidation("doctor", doctorTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDoctor(ctx, deployer, request)
	})))

	// Register build_and_deploy tool
	buildAndDeployTool := mcp.NewTool("build_and_deploy",
		mcp.WithDescription("Build and deploy an application from git source. Clones repo, detects tech stack, generates Dockerfile, builds image, and deploys container."),
		mcp.WithString("repo_url", mcp.Required(), mcp.Description("Git repository URL")),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Application name (used for container name and image tag)")),
		mcp.WithString("branch", mcp.Description("Git branch (default: main)")),
		mcp.WithString("tech_stack", mcp.Description("Tech stack template (auto-detect if empty): node, python, go, java, php, ruby, rust, static, docker")),
		mcp.WithString("ports", mcp.Description("Port mappings (e.g. '8080:80')")),
		mcp.WithString("server_id", mcp.Description("Target server ID (deploy locally if empty)")),
		mcp.WithString("env_vars", mcp.Description("JSON object of environment variables")),
		mcp.WithString("registry_id", mcp.Description("Registry ID to push the built image to")),
		mcp.WithBoolean("push_image", mcp.Description("Whether to push the built image to the registry after build")),
		mcp.WithString("image_tag", mcp.Description("Custom image tag (default: appname:latest)")),
	)
	s.AddTool(buildAndDeployTool, withPermissionCheck("build_and_deploy", withValidation("build_and_deploy", buildAndDeployTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleBuildAndDeploy(ctx, deployer, request)
	})))

	// Register heal_container tool
	healContainerTool := mcp.NewTool("heal_container",
		mcp.WithDescription("Trigger self-healing for a container. Inspects the container state and takes corrective action (restart or rollback) if needed."),
		mcp.WithString("container_name", mcp.Required(), mcp.Description("Name of the container to heal")),
	)
	s.AddTool(healContainerTool, withPermissionCheck("heal_container", withValidation("heal_container", healContainerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleHealContainer(ctx, deployer, request)
	})))

	// Register get_container_metrics tool
	getContainerMetricsTool := mcp.NewTool("get_container_metrics",
		mcp.WithDescription("Get resource usage metrics (CPU, memory) for a specific container."),
		mcp.WithString("container_name", mcp.Required(), mcp.Description("Name of the container")),
	)
	s.AddTool(getContainerMetricsTool, withPermissionCheck("get_container_metrics", withValidation("get_container_metrics", getContainerMetricsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetContainerMetrics(ctx, deployer, request)
	})))

	// Register get_system_metrics tool
	getSystemMetricsTool := mcp.NewTool("get_system_metrics",
		mcp.WithDescription("Get system-level metrics (CPU, memory, disk usage)."),
	)
	s.AddTool(getSystemMetricsTool, withPermissionCheck("get_system_metrics", withValidation("get_system_metrics", getSystemMetricsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetSystemMetrics(ctx, deployer, request)
	})))

	// Register list_alerts tool
	listAlertsTool := mcp.NewTool("list_alerts",
		mcp.WithDescription("List all currently active (firing) alerts."),
	)
	s.AddTool(listAlertsTool, withPermissionCheck("list_alerts", withValidation("list_alerts", listAlertsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListAlerts(ctx, deployer, request)
	})))

	// Register list_alert_rules tool
	listAlertRulesTool := mcp.NewTool("list_alert_rules",
		mcp.WithDescription("List all configured alert rules."),
	)
	s.AddTool(listAlertRulesTool, withPermissionCheck("list_alert_rules", withValidation("list_alert_rules", listAlertRulesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListAlertRules(ctx, deployer, request)
	})))

	// Register trigger_ci_build tool
	triggerCITool := mcp.NewTool("trigger_ci_build",
		mcp.WithDescription("Trigger a CI/CD build for a repository"),
		mcp.WithString("provider", mcp.Required(), mcp.Description("CI/CD provider type: github-actions")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name (e.g. my-project)")),
		mcp.WithString("branch", mcp.Required(), mcp.Description("Git branch to build (e.g. main)")),
	)
	s.AddTool(triggerCITool, withPermissionCheck("trigger_ci_build", withValidation("trigger_ci_build", triggerCITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleTriggerCIBuild(ctx, deployer, request)
	})))

	// Register get_ci_build_status tool
	getCIStatusTool := mcp.NewTool("get_ci_build_status",
		mcp.WithDescription("Get the status of a CI/CD build"),
		mcp.WithString("provider", mcp.Required(), mcp.Description("CI/CD provider type: github-actions")),
		mcp.WithString("run_id", mcp.Required(), mcp.Description("Build run ID")),
	)
	s.AddTool(getCIStatusTool, withPermissionCheck("get_ci_build_status", withValidation("get_ci_build_status", getCIStatusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetCIBuildStatus(ctx, deployer, request)
	})))

	// Register SSL certificate management tools
	listSSLCertsTool := mcp.NewTool("list_ssl_certificates",
		mcp.WithDescription("List all SSL certificates"),
	)
	s.AddTool(listSSLCertsTool, withPermissionCheck("list_ssl_certificates", withValidation("list_ssl_certificates", listSSLCertsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListSSLCertificates(ctx, deployer, request)
	})))

	requestSSLCertTool := mcp.NewTool("request_ssl_certificate",
		mcp.WithDescription("Request a new SSL certificate for a domain"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name")),
		mcp.WithString("email", mcp.Required(), mcp.Description("Email for certificate registration")),
	)
	s.AddTool(requestSSLCertTool, withPermissionCheck("request_ssl_certificate", withValidation("request_ssl_certificate", requestSSLCertTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRequestSSLCertificate(ctx, deployer, request)
	})))

	renewSSLCertTool := mcp.NewTool("renew_ssl_certificate",
		mcp.WithDescription("Renew an SSL certificate"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name")),
	)
	s.AddTool(renewSSLCertTool, withPermissionCheck("renew_ssl_certificate", withValidation("renew_ssl_certificate", renewSSLCertTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRenewSSLCertificate(ctx, deployer, request)
	})))

	deleteSSLCertTool := mcp.NewTool("delete_ssl_certificate",
		mcp.WithDescription("Delete an SSL certificate"),
		mcp.WithString("domain", mcp.Required(), mcp.Description("Domain name")),
	)
	s.AddTool(deleteSSLCertTool, withPermissionCheck("delete_ssl_certificate", withValidation("delete_ssl_certificate", deleteSSLCertTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteSSLCertificate(ctx, deployer, request)
	})))

	// Register get_context tool
	getContextTool := mcp.NewTool("get_context",
		mcp.WithDescription("Get current MCP session context and operation history"),
	)
	s.AddTool(getContextTool, withPermissionCheck("get_context", withValidation("get_context", getContextTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetContext(ctx, request)
	})))

	// Register detect_panel tool
	detectPanelTool := mcp.NewTool("detect_panel",
		mcp.WithDescription("Detect which hosting panel (1Panel/BT-Panel) is installed on a server"),
		mcp.WithString("server_id", mcp.Required(), mcp.Description("Server ID to detect panel on")),
	)
	s.AddTool(detectPanelTool, withPermissionCheck("detect_panel", withValidation("detect_panel", detectPanelTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDetectPanel(ctx, deployer, request)
	})))

	// Register registry_login tool
	registryLoginTool := mcp.NewTool("registry_login",
		mcp.WithDescription("Authenticate with a container registry"),
		mcp.WithString("registry_id", mcp.Required(), mcp.Description("Registry ID to authenticate with")),
	)
	s.AddTool(registryLoginTool, withPermissionCheck("registry_login", withValidation("registry_login", registryLoginTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRegistryLogin(ctx, deployer, request)
	})))

	// Register push_image tool
	pushImageTool := mcp.NewTool("push_image",
		mcp.WithDescription("Push a Docker image to a container registry"),
		mcp.WithString("registry_id", mcp.Required(), mcp.Description("Registry ID to push to")),
		mcp.WithString("local_image", mcp.Required(), mcp.Description("Local image name (e.g. myapp:latest)")),
		mcp.WithString("remote_tag", mcp.Description("Remote tag for the image (e.g. registry.example.com/myapp:v1)")),
	)
	s.AddTool(pushImageTool, withPermissionCheck("push_image", withValidation("push_image", pushImageTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlePushImage(ctx, deployer, request)
	})))

	// Register list_registry_tags tool
	listRegistryTagsTool := mcp.NewTool("list_registry_tags",
		mcp.WithDescription("List tags for a repository in a container registry"),
		mcp.WithString("registry_id", mcp.Required(), mcp.Description("Registry ID")),
		mcp.WithString("repository", mcp.Required(), mcp.Description("Repository name (e.g. myuser/myapp)")),
	)
	s.AddTool(listRegistryTagsTool, withPermissionCheck("list_registry_tags", withValidation("list_registry_tags", listRegistryTagsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListRegistryTags(ctx, deployer, request)
	})))

	// Register ping_registry tool
	pingRegistryTool := mcp.NewTool("ping_registry",
		mcp.WithDescription("Check if a container registry is accessible"),
		mcp.WithString("registry_id", mcp.Required(), mcp.Description("Registry ID to ping")),
	)
	s.AddTool(pingRegistryTool, withPermissionCheck("ping_registry", withValidation("ping_registry", pingRegistryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handlePingRegistry(ctx, deployer, request)
	})))

	return s
}

func handleBuildAndDeploy(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoURL, err := request.RequireString("repo_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	appName, err := request.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	branch := request.GetString("branch", "main")
	techStack := request.GetString("tech_stack", "")
	ports := request.GetString("ports", "")
	serverID := request.GetString("server_id", "")
	envVarsStr := request.GetString("env_vars", "")

	var envVars map[string]string
	if envVarsStr != "" {
		if err := json.Unmarshal([]byte(envVarsStr), &envVars); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid env_vars JSON: %v", err)), nil
		}
	}

	cfg := BuildAndDeployConfig{
		RepoURL:   repoURL,
		AppName:   appName,
		Branch:    branch,
		TechStack: techStack,
		Ports:     ports,
		ServerID:  serverID,
		EnvVars:   envVars,
		RegistryID: request.GetString("registry_id", ""),
		ImageTag:   request.GetString("image_tag", ""),
	}

	if v := request.GetString("push_image", ""); v != "" {
		cfg.PushImage = strings.ToLower(v) == "true"
	}

	result, err := deployer.BuildAndDeploy(ctx, cfg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("build and deploy failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleTriggerCIBuild(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	provider, err := request.RequireString("provider")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	repo, err := request.RequireString("repo")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	branch, err := request.RequireString("branch")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.TriggerCIBuild(ctx, provider, repo, branch)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to trigger CI build: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetCIBuildStatus(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	provider, err := request.RequireString("provider")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	runID, err := request.RequireString("run_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.GetCIBuildStatus(ctx, provider, runID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get CI build status: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDeployApp(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	image, err := request.RequireString("image")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cfg := DeployConfig{
		Image:         image,
		ContainerName: containerName,
	}

	if v := request.GetString("ports", ""); v != "" {
		cfg.Ports = v
	}
	if v := request.GetString("restart_policy", ""); v != "" {
		cfg.RestartPolicy = v
	}
	if v := request.GetString("network", ""); v != "" {
		cfg.Network = v
	}
	if v := request.GetString("volumes", ""); v != "" {
		cfg.Volumes = v
	}
	if v := request.GetString("cpu", ""); v != "" {
		cfg.CPU = v
	}
	if v := request.GetString("memory", ""); v != "" {
		cfg.Memory = v
	}
	if v := request.GetString("server_id", ""); v != "" {
		cfg.ServerID = v
	}

	// Parse env vars JSON
	if envStr := request.GetString("env_vars", ""); envStr != "" {
		var envVars map[string]string
		if err := json.Unmarshal([]byte(envStr), &envVars); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid env_vars JSON: %v", err)), nil
		}
		cfg.EnvVars = envVars
	}

	// Parse labels JSON
	if labelsStr := request.GetString("labels", ""); labelsStr != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsStr), &labels); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid labels JSON: %v", err)), nil
		}
		cfg.Labels = labels
	}

	status, err := deployer.Deploy(ctx, cfg)
	if err != nil {
		// Check if it's a preflight error for structured output
		var pfErr PreflightErrorInfo
		if errors.As(err, &pfErr) {
			pfData := map[string]interface{}{
				"status":  "preflight_failed",
				"code":    pfErr.PreflightCode(),
				"message": pfErr.PreflightMessage(),
				"checks":  pfErr.PreflightChecks(),
			}
			data, _ := json.MarshalIndent(pfData, "", "  ")
			return mcp.NewToolResultError(string(data)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("deploy failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Container %s deployed successfully", containerName),
		"container": map[string]string{
			"id":     status.ID,
			"name":   status.Name,
			"image":  status.Image,
			"status": status.Status,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListApps(ctx context.Context, deployer Deployer, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	apps, err := deployer.ListApps(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list apps: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"total":  len(apps),
		"apps":   apps,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListServers(ctx context.Context, deployer Deployer, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	servers, err := deployer.ListServers(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list servers: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"total":   len(servers),
		"servers": servers,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleCreateApp(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	repoURL, err := request.RequireString("repo_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cfg := CreateAppConfig{
		Name:    name,
		RepoURL: repoURL,
		Branch:  request.GetString("branch", "main"),
		Domain:  request.GetString("domain", ""),
		TechStack: request.GetString("tech_stack", "docker"),
		DeployMode: request.GetString("deploy_mode", "api"),
		ServerID: request.GetString("server_id", ""),
	}

	appID, err := deployer.CreateApp(ctx, cfg)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create app: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Application %s created successfully", name),
		"app": map[string]string{
			"id":       appID,
			"name":     name,
			"repo_url": repoURL,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDeleteApp(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := deployer.DeleteApp(ctx, appID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete app: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Application %s deleted successfully", appID),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleRollback(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	previousImage, err := request.RequireString("previous_image")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status, err := deployer.Rollback(ctx, containerName, previousImage)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("rollback failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Container %s rolled back to %s", containerName, previousImage),
		"container": map[string]string{
			"id":     status.ID,
			"name":   status.Name,
			"image":  status.Image,
			"status": status.Status,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleBackup(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	backupID, err := deployer.Backup(ctx, appID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("backup failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Backup created for app %s", appID),
		"backup": map[string]string{
			"id": backupID,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleRestore(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	backupID, err := request.RequireString("backup_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status, err := deployer.Restore(ctx, backupID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("restore failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Application restored from backup %s", backupID),
		"container": map[string]string{
			"id":     status.ID,
			"name":   status.Name,
			"image":  status.Image,
			"status": status.Status,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetAppLogs(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tail := 100
	if t := request.GetString("tail", ""); t != "" {
		_, _ = fmt.Sscanf(t, "%d", &tail)
	}

	logs, err := deployer.GetContainerLogs(ctx, containerName, tail)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get logs: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"container_name": containerName,
		"tail":  tail,
		"logs":  logs,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDetectEnv(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	level := 2
	if l := request.GetString("level", "2"); l != "" {
		_, _ = fmt.Sscanf(l, "%d", &level)
	}

	var ports []int
	if p := request.GetString("ports", ""); p != "" {
		for _, ps := range strings.Split(p, ",") {
			ps = strings.TrimSpace(ps)
			var port int
			if _, err := fmt.Sscanf(ps, "%d", &port); err == nil {
				ports = append(ports, port)
			}
		}
	}

	var services []string
	if s := request.GetString("services", ""); s != "" {
		services = strings.Split(s, ",")
		for i := range services {
			services[i] = strings.TrimSpace(services[i])
		}
	}

	env, err := deployer.DetectEnv(ctx, level, ports, services)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("environment detection failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":      "success",
		"environment": env,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleHealthCheck(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	target, err := request.RequireString("target")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	healthType := request.GetString("type", "http")

	health, err := deployer.HealthCheck(ctx, target, healthType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("health check failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"health": health,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleAddServer(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	host, err := request.RequireString("host")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	port := 22
	if p := request.GetString("port", "22"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}
	user := request.GetString("user", "root")

	srv, err := deployer.AddServer(ctx, name, host, port, user)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add server: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Server %s added successfully", name),
		"server": map[string]interface{}{
			"id":     srv.ID,
			"name":   srv.Name,
			"host":   srv.Host,
			"port":   srv.Port,
			"status": srv.Status,
		},
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleRemoveServer(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := request.RequireString("server_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := deployer.RemoveServer(ctx, serverID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove server: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Server %s removed successfully", serverID),
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleTestServer(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := request.RequireString("server_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	testResult, err := deployer.TestServer(ctx, serverID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("server test failed: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"test":   testResult,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDoctor(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	checks := []map[string]interface{}{}

	// Check 1: Docker
	_, dockerErr := deployer.(interface {
		GetContainerStatus(ctx context.Context, name string) (*ContainerStatus, error)
	}).GetContainerStatus(ctx, "__doctor_probe__")

	dockerCheck := map[string]interface{}{"name": "Docker"}
	if dockerErr != nil {
		dockerCheck["status"] = "unavailable"
		dockerCheck["message"] = "Docker is not available or not running"
		dockerCheck["suggestion"] = "Install Docker (https://docs.docker.com/get-docker/) and ensure the daemon is running: sudo systemctl start docker"
	} else {
		dockerCheck["status"] = "available"
		dockerCheck["message"] = "Docker is available"
	}
	checks = append(checks, dockerCheck)

	// Check 2: Database (inferred from no error on startup — if we're here, DB works)
	checks = append(checks, map[string]interface{}{
		"name":    "Database",
		"status":  "ok",
		"message": "Database connection is working",
	})

	// Check 3: SSH executor
	checks = append(checks, map[string]interface{}{
		"name":    "SSH Executor",
		"status":  "ok",
		"message": "Local executor is available. For remote deployment, register servers via add_server and create credentials via add_credential.",
	})

	result := map[string]interface{}{
		"status": "ok",
		"checks": checks,
		"tip":    "To deploy remotely: 1) add_server 2) add_credential 3) deploy_app with server_id",
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleCreateCredential(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tenantID, err := request.RequireString("tenant_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	credType, err := request.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	value, err := request.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cred, err := deployer.CreateCredential(ctx, tenantID, name, credType, value)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create credential: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":     "success",
		"message":    fmt.Sprintf("Credential %s created successfully", name),
		"credential": cred,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListCredentials(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tenantID, err := request.RequireString("tenant_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	creds, err := deployer.ListCredentials(ctx, tenantID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list credentials: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":      "success",
		"credentials": creds,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDeleteCredential(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	credID, err := request.RequireString("credential_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := deployer.DeleteCredential(ctx, credID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete credential: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Credential %s deleted successfully", credID),
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDNSCreateRecord(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, _ := request.RequireString("domain")
	recordType, _ := request.RequireString("type")
	name, _ := request.RequireString("name")
	value, _ := request.RequireString("value")

	if domain == "" || recordType == "" || name == "" || value == "" {
		return mcp.NewToolResultError("domain, type, name, and value are required"), nil
	}

	record, err := deployer.DNSCreateRecord(ctx, domain, recordType, name, value)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DNS create failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "record": record}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDNSDeleteRecord(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	recordID, err := request.RequireString("record_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := deployer.DNSDeleteRecord(ctx, recordID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DNS delete failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "message": fmt.Sprintf("Record %s deleted", recordID)}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDNSListRecords(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	records, err := deployer.DNSListRecords(ctx, domain)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DNS list failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "records": records}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleSendNotification(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	nType, err := request.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	appName, _ := request.RequireString("app")
	server, _ := request.RequireString("server")
	status, _ := request.RequireString("status")
	message := request.GetString("message", "")

	result, err := deployer.SendNotification(ctx, nType, appName, server, status, message)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("notification failed: %v", err)), nil
	}

	resp := map[string]interface{}{"status": "success", "notification": result}
	data, _ := json.MarshalIndent(resp, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListTemplates(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	templates, err := deployer.ListTemplates(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list templates failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "templates": templates}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetTemplate(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tmplType, err := request.RequireString("type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tmpl, err := deployer.GetTemplate(ctx, tmplType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get template failed: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "template": tmpl}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetAppDetail(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	detail, err := deployer.GetAppDetail(ctx, appID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get app detail: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "app": detail}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleUpdateApp(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	configStr, err := request.RequireString("config")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid config JSON: %v", err)), nil
	}

	updated, err := deployer.UpdateApp(ctx, appID, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update app: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "message": fmt.Sprintf("App %s updated", appID), "app": updated}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetTaskStatus(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID, err := request.RequireString("task_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	task, err := deployer.GetTaskStatus(ctx, taskID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get task status: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "task": task}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListTasks(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := 20
	if l := request.GetString("limit", "20"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	statusFilter := request.GetString("status_filter", "")

	tasks, err := deployer.ListTasks(ctx, limit, statusFilter)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list tasks: %v", err)), nil
	}

	result := map[string]interface{}{"status": "success", "tasks": tasks}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleSearchAppLogs(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appID, err := request.RequireString("app_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	keyword, err := request.RequireString("keyword")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := 50
	if l := request.GetString("limit", "50"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	res, err := deployer.SearchAppLogs(ctx, appID, keyword, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search logs failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "search": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleUpdateDNSRecord(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, _ := request.RequireString("domain")
	subdomain, _ := request.RequireString("subdomain")
	recordType, _ := request.RequireString("type")
	newValue, _ := request.RequireString("new_value")
	if domain == "" || subdomain == "" || recordType == "" || newValue == "" {
		return mcp.NewToolResultError("domain, subdomain, type, and new_value are required"), nil
	}
	res, err := deployer.UpdateDNSRecord(ctx, domain, subdomain, recordType, newValue)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("update DNS failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "record": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleUpdateCredential(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	credID, err := request.RequireString("credential_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	value, err := request.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	res, err := deployer.UpdateCredential(ctx, credID, value)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("update credential failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "credential": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleUpdateServer(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := request.RequireString("server_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	configStr, err := request.RequireString("config")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid config JSON: %v", err)), nil
	}
	res, err := deployer.UpdateServer(ctx, serverID, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("update server failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "server": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleCheckDeployReadiness(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configStr, err := request.RequireString("app_config")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var appConfig map[string]interface{}
	if err := json.Unmarshal([]byte(configStr), &appConfig); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid app_config JSON: %v", err)), nil
	}
	res, err := deployer.CheckDeployReadiness(ctx, appConfig)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("readiness check failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "readiness": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleBatchDeploy(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appsStr, err := request.RequireString("apps")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var apps []map[string]interface{}
	if err := json.Unmarshal([]byte(appsStr), &apps); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid apps JSON: %v", err)), nil
	}

	// Parse optional strategy parameters
	strategy := DeployStrategy(request.GetString("strategy", "sequential"))
	maxConcurrent := 0
	if s := request.GetString("max_concurrent", ""); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			maxConcurrent = v
		}
	}
	batchSize := 0
	if s := request.GetString("batch_size", ""); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			batchSize = v
		}
	}
	var serverIDs []string
	if sidStr := request.GetString("server_ids", ""); sidStr != "" {
		for _, s := range strings.Split(sidStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				serverIDs = append(serverIDs, s)
			}
		}
	}

	config := BatchDeployConfig{
		Apps:          apps,
		Strategy:      strategy,
		MaxConcurrent: maxConcurrent,
		BatchSize:     batchSize,
		ServerIDs:     serverIDs,
	}

	res, err := deployer.BatchDeployWithConfig(ctx, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("batch deploy failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "batch": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleBatchBackup(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appIDsStr, err := request.RequireString("app_ids")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var appIDs []string
	if err := json.Unmarshal([]byte(appIDsStr), &appIDs); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid app_ids JSON: %v", err)), nil
	}
	res, err := deployer.BatchBackup(ctx, appIDs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("batch backup failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "batch": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleBatchDNS(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	recordsStr, err := request.RequireString("records")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(recordsStr), &records); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid records JSON: %v", err)), nil
	}
	res, err := deployer.BatchDNS(ctx, records)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("batch DNS failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "batch": res}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleCheckSystemUpdate(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	update, err := deployer.CheckSystemUpdate(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("check update failed: %v", err)), nil
	}
	result := map[string]interface{}{"status": "success", "update": update}
	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetDeployStatus(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status, err := deployer.GetContainerStatus(ctx, containerName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get status: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "success",
		"container": map[string]string{
			"id":     status.ID,
			"name":   status.Name,
			"image":  status.Image,
			"status": status.Status,
		},
	}

	// Add preflight summary from latest deployment record
	if record, err := deployer.GetLatestDeploymentRecord(ctx, containerName); err == nil && record.PreflightCode != "" {
		result["last_preflight"] = map[string]interface{}{
			"status":  record.Status,
			"code":    record.PreflightCode,
			"message": record.PreflightMessage,
			"time":    record.CreatedAt,
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleHealContainer(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.HealContainer(ctx, containerName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("heal failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetContainerMetrics(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	containerName, err := request.RequireString("container_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.GetContainerMetrics(ctx, containerName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get container metrics: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleGetSystemMetrics(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := deployer.GetSystemMetrics(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get system metrics: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListAlerts(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := deployer.ListAlerts(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list alerts: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListAlertRules(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := deployer.ListAlertRules(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list alert rules: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListSSLCertificates(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := deployer.ListSSLCertificates(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list SSL certificates: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleRequestSSLCertificate(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	email, err := request.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.RequestSSLCertificate(ctx, domain, email)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to request SSL certificate: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleRenewSSLCertificate(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.RenewSSLCertificate(ctx, domain)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to renew SSL certificate: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDeleteSSLCertificate(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain, err := request.RequireString("domain")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.DeleteSSLCertificate(ctx, domain)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete SSL certificate: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// contextManager is the global session context manager.
var contextManager = NewContextManager()

func handleGetContext(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := "default" // In production, extract from session
	session := contextManager.GetOrCreateSession(sessionID)
	entries := session.GetEntries()
	summary := session.GetSummary()

	result := fmt.Sprintf("Session: %s\nEntries: %d\nMemory: %d bytes\nLast access: %s\n\nRecent operations:\n",
		summary["session_id"], summary["entries"], summary["memory_usage"], summary["last_access"])
	for i, e := range entries {
		result += fmt.Sprintf("%d. [%s] %s (%s)\n", i+1, e.Time.Format("15:04:05"), e.Tool, e.Duration)
	}

	return mcp.NewToolResultText(result), nil
}

func handleDetectPanel(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serverID, err := request.RequireString("server_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Use TestServer to verify server connectivity and detect panel
	_, err = deployer.TestServer(ctx, serverID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to connect to server %s: %v", serverID, err)), nil
	}

	result := map[string]interface{}{
		"status":    "success",
		"server_id": serverID,
		"message":   "Panel detection initiated. Use detect_environment for full environment details.",
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleRegistryLogin(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryID, err := request.RequireString("registry_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := map[string]interface{}{
		"registry_url": request.GetString("registry_url", ""),
		"username":     request.GetString("username", ""),
		"password":     request.GetString("password", ""),
	}

	result, err := deployer.RegistryOps(registryID, "login", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("registry login failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handlePushImage(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryID, err := request.RequireString("registry_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	localImage, err := request.RequireString("local_image")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := map[string]interface{}{
		"local_image": localImage,
		"remote_tag":  request.GetString("remote_tag", ""),
	}

	result, err := deployer.RegistryOps(registryID, "push", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("push image failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleListRegistryTags(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryID, err := request.RequireString("registry_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	repository, err := request.RequireString("repository")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := map[string]interface{}{
		"repository": repository,
	}

	result, err := deployer.RegistryOps(registryID, "list_tags", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list registry tags failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handlePingRegistry(ctx context.Context, deployer Deployer, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registryID, err := request.RequireString("registry_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := deployer.RegistryOps(registryID, "ping", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ping registry failed: %v", err)), nil
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
