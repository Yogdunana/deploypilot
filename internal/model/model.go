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
	PasswordHash string    `gorm:"not null" json:"-"`
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
	ResourceLimits string    `gorm:"type:text" json:"resource_limits"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Server Server `gorm:"foreignKey:ServerID" json:"server,omitempty"`
}

func (App) TableName() string { return "apps" }

// Credential represents encrypted credentials for server access.
type Credential struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	TenantID       string    `gorm:"index" json:"tenant_id"`
	Name           string    `gorm:"not null" json:"name"`
	Type           string    `gorm:"not null" json:"type"` // ssh, api_key, token
	EncryptedValue string    `gorm:"not null" json:"-"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`

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
	ContainerName    string    `json:"container_name"`
	Image            string    `json:"image"`
	Status           string    `json:"status"` // "preflight_failed", "deploying", "success", "failed"
	PreflightCode    string    `gorm:"column:preflight_code" json:"preflight_code,omitempty"`
	PreflightMessage string    `gorm:"column:preflight_message" json:"preflight_message,omitempty"`
	PreflightChecks  string    `gorm:"column:preflight_checks;type:text" json:"preflight_checks,omitempty"` // JSON string
	ErrorMessage     string    `gorm:"column:error_message" json:"error_message,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (DeploymentRecord) TableName() string { return "deployments" }
