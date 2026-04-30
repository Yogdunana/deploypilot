package model

import "time"

// Tenant represents a multi-tenant organization.
type Tenant struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"not null" json:"name"`
	Slug       string    `gorm:"uniqueIndex;not null" json:"slug"`
	Plan       string    `gorm:"default:free" json:"plan"`
	MaxServers int       `gorm:"default:5" json:"max_servers"`
	MaxApps    int       `gorm:"default:20" json:"max_apps"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Tenant) TableName() string { return "tenants" }

// Role represents a user role with permissions.
type Role struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Permissions string    `gorm:"type:text" json:"permissions"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Role) TableName() string { return "roles" }

// User represents a system user.
type User struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	TenantID     string    `gorm:"index" json:"tenant_id"`
	RoleID       string    `gorm:"index" json:"role_id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"" json:"-"`
	AuthProvider string `gorm:"size:20;index" json:"auth_provider,omitempty"`
	AuthUID      string `gorm:"size:100;index" json:"auth_uid,omitempty"`
	AvatarURL    string `gorm:"size:500" json:"avatar_url,omitempty"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Role   Role   `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

func (User) TableName() string { return "users" }

// Server represents a target deployment server.
type Server struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	TenantID     string    `gorm:"index" json:"tenant_id"`
	CredentialID string    `gorm:"index" json:"credential_id"`
	ProviderID   string    `gorm:"index" json:"provider_id"`
	Name         string    `gorm:"not null" json:"name"`
	Host         string    `gorm:"not null" json:"host"`
	Port         int       `gorm:"default:22" json:"port"`
	Tags         string    `gorm:"type:text" json:"tags"`
	Status       string    `gorm:"default:unknown" json:"status"` // unknown, reachable, unreachable
	DetectedInfo string    `gorm:"type:text" json:"detected_info"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Tenant     Tenant     `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Credential Credential `gorm:"foreignKey:CredentialID" json:"credential,omitempty"`
	Provider   Provider   `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

func (Server) TableName() string { return "servers" }

// App represents a deployable application.
type App struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	TenantID       string    `gorm:"index" json:"tenant_id"`
	ServerID       string    `gorm:"index" json:"server_id"`
	Name           string    `gorm:"not null" json:"name"`
	RepoURL        string    `gorm:"not null" json:"repo_url"`
	Branch         string    `gorm:"default:main" json:"branch"`
	Domain         string    `json:"domain"`
	TechStack      string    `gorm:"default:docker" json:"tech_stack"`
	DeployMode     string    `gorm:"default:api" json:"deploy_mode"` // api, direct, cicd
	Status         string    `gorm:"default:pending" json:"status"`  // pending, deploying, running, failed, stopped
	CurrentVersion string    `json:"current_version"`
	ContainerName  string    `json:"container_name"`
	EnvVars        string    `gorm:"type:text" json:"env_vars"`
	ResourceLimits    string    `gorm:"type:text" json:"resource_limits"`
	ComposeContent    string    `gorm:"type:text" json:"compose_content,omitempty"`
	ComposeProjectName string   `json:"compose_project_name,omitempty"`
	Environment       string   `gorm:"size:20;default:production;index" json:"environment"` // production, staging, development, testing
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Server Server `gorm:"foreignKey:ServerID" json:"server,omitempty"`
}

func (App) TableName() string { return "apps" }

// Credential represents encrypted credentials for server access.
type Credential struct {
	ID             string     `gorm:"primaryKey" json:"id"`
	TenantID       string     `gorm:"index" json:"tenant_id"`
	Name           string     `gorm:"not null" json:"name"`
	Type           string     `gorm:"not null" json:"type"` // ssh, api_key, token
	EncryptedValue string     `gorm:"not null" json:"-"`
	ExpiresAt      *time.Time `gorm:"index" json:"expires_at,omitempty"`
	LastRotated    *time.Time `json:"last_rotated,omitempty"`
	RotationDays   int        `gorm:"default:90" json:"rotation_days"` // 0 = never expires
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (Credential) TableName() string { return "credentials" }

// Provider represents a deployment provider configuration.
type Provider struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	TenantID  string    `gorm:"index" json:"tenant_id"`
	Type      string    `gorm:"not null" json:"type"` // docker, ssh, 1panel
	Name      string    `gorm:"not null" json:"name"`
	Config    string    `gorm:"type:text" json:"config"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (Provider) TableName() string { return "providers" }

// DeploymentRecord tracks deployment attempts and their results.
type DeploymentRecord struct {
	ID               string    `gorm:"primaryKey" json:"id"`
	TenantID         string    `gorm:"index" json:"tenant_id"`
	ServerID         string    `gorm:"index" json:"server_id"`
	AppName          string    `json:"app_name"`
	AppID            string    `gorm:"index" json:"app_id,omitempty"`
	ContainerName    string    `gorm:"index" json:"container_name"`
	Image            string    `json:"image"`
	PreviousImage    string    `json:"previous_image,omitempty"`    // image before this deployment (for rollback tracking)
	DeployType       string    `gorm:"default:deploy" json:"deploy_type"` // deploy, rollback, auto_rollback
	ConfigSnapshot   string    `gorm:"type:text" json:"config_snapshot,omitempty"` // JSON snapshot of full deploy config at time of deployment
	Status           string    `json:"status"` // "preflight_failed", "deploying", "success", "failed"
	PreflightCode    string    `gorm:"column:preflight_code" json:"preflight_code,omitempty"`
	PreflightMessage string    `gorm:"column:preflight_message" json:"preflight_message,omitempty"`
	PreflightChecks  string    `gorm:"column:preflight_checks;type:text" json:"preflight_checks,omitempty"` // JSON string
	ErrorMessage     string    `gorm:"column:error_message" json:"error_message,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (DeploymentRecord) TableName() string { return "deployments" }

// AuditLog records system actions for compliance and debugging.
type AuditLog struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"index"`
	Username     string    `json:"username"`
	Action       string    `json:"action" gorm:"index;size:100"`
	ResourceType string    `json:"resource_type" gorm:"size:50"`
	ResourceID   string    `json:"resource_id" gorm:"size:100"`
	Detail       string    `json:"detail" gorm:"type:text"`
	IPAddress    string    `json:"ip_address" gorm:"size:45"`
	UserAgent    string    `json:"user_agent" gorm:"size:500"`
	RecordHash   string    `json:"record_hash" gorm:"column:record_hash;size:128"`
	TraceID      string    `json:"trace_id,omitempty" gorm:"size:36;index"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime;index"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// SSLCertificate represents an SSL/TLS certificate managed via ACME.
type SSLCertificate struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Domain      string     `gorm:"uniqueIndex;not null" json:"domain"`
	Email       string     `gorm:"not null" json:"email"`
	Provider    string     `gorm:"not null;default:cloudflare" json:"provider"` // cloudflare, aliyun, tencent
	Status      string     `gorm:"not null;default:pending" json:"status"`      // pending, active, expired, failed
	CertPath    string     `json:"cert_path"`
	KeyPath     string     `json:"key_path"`
	IssuedAt    *time.Time `json:"issued_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	AutoRenew   bool       `gorm:"default:true" json:"auto_renew"`
	LastRenewed *time.Time `json:"last_renewed"`
	RetryCount  int        `gorm:"default:0" json:"retry_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (SSLCertificate) TableName() string { return "ssl_certificates" }

// MetricRecord stores historical monitoring metrics.
type MetricRecord struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	TenantID      string    `gorm:"index" json:"tenant_id,omitempty"`
	ServerID      string    `gorm:"index" json:"server_id,omitempty"`
	ContainerName string    `gorm:"index" json:"container_name,omitempty"`
	MetricType    string    `gorm:"index;size:20" json:"metric_type"` // cpu, memory, disk, network, disk_io, container
	Name          string    `gorm:"size:50" json:"name"`               // cpu_usage, memory_used_mb, etc.
	Value         float64   `json:"value"`
	Unit          string    `gorm:"size:20" json:"unit"`               // percent, bytes, megabytes, etc.
	Labels        string    `gorm:"type:text" json:"labels,omitempty"` // JSON string
	Timestamp     time.Time `gorm:"index" json:"timestamp"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (MetricRecord) TableName() string { return "metric_records" }

// AlertHistory stores resolved alert records for historical analysis.
type AlertHistory struct {
	ID         string     `gorm:"primaryKey" json:"id"`
	TenantID   string     `gorm:"index" json:"tenant_id,omitempty"`
	RuleID     string     `gorm:"index" json:"rule_id"`
	RuleName   string     `json:"rule_name"`
	Severity   string     `gorm:"size:20" json:"severity"`   // critical, warning, info
	Message    string     `json:"message"`
	Value      float64    `json:"value"`
	Threshold  float64    `json:"threshold"`
	Status     string     `gorm:"size:20;index" json:"status"` // firing, resolved
	FiredAt    time.Time  `gorm:"index" json:"fired_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (AlertHistory) TableName() string { return "alert_histories" }

// ScheduledTask represents a cron-based scheduled task.
type ScheduledTask struct {
	ID          string     `gorm:"primaryKey" json:"id"`
	TenantID    string     `gorm:"index" json:"tenant_id"`
	Name        string     `gorm:"not null;size:100" json:"name"`
	Description string     `json:"description"`
	CronExpr    string     `gorm:"not null;size:50" json:"cron_expr"` // e.g., "0 */6 * * *"
	TaskType    string     `gorm:"size:30;index" json:"task_type"`    // shell, backup, health_check, log_cleanup
	Command     string     `gorm:"type:text" json:"command"`          // shell command or task-specific config
	ServerID    string     `gorm:"index" json:"server_id,omitempty"`
	Enabled     bool       `gorm:"default:true" json:"enabled"`
	Timeout     int        `gorm:"default:300" json:"timeout"`        // seconds
	LastRunAt   *time.Time `json:"last_run_at,omitempty"`
	LastStatus  string     `gorm:"size:20" json:"last_status,omitempty"` // success, failed, running
	LastError   string     `gorm:"type:text" json:"last_error,omitempty"`
	RunCount    int        `gorm:"default:0" json:"run_count"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ScheduledTask) TableName() string { return "scheduled_tasks" }

// TaskExecution stores execution reports for scheduled tasks.
type TaskExecution struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	TaskID    string    `gorm:"index" json:"task_id"`
	TenantID  string    `gorm:"index" json:"tenant_id"`
	Status    string    `gorm:"size:20;index" json:"status"` // success, failed, timeout
	Output    string    `gorm:"type:text" json:"output,omitempty"`
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Duration  int64     `json:"duration"` // milliseconds
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (TaskExecution) TableName() string { return "task_executions" }
