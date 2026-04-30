package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the root configuration structure.
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Deploy     DeployConfig     `mapstructure:"deploy"`
	Cache      CacheConfig      `mapstructure:"cache"`
	Security   SecurityConfig   `mapstructure:"security"`
	Log        LogConfig        `mapstructure:"log"`
	Notify     NotifyConfig     `mapstructure:"notify"`
	Monitor    MonitorConfig    `mapstructure:"monitor"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	Audit      AuditConfig      `mapstructure:"audit"`
	Backup     BackupConfig     `mapstructure:"backup"`
	BruteForce BruteForceConfig `mapstructure:"bruteforce"`
}

// BackupConfig holds database auto-backup settings.
type BackupConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Interval       string `mapstructure:"interval"`         // e.g. "6h", "30m"
	RetentionCount int    `mapstructure:"retention_count"`  // max backup files (default: 10)
	RetentionDays  int    `mapstructure:"retention_days"`   // max days to keep (default: 30)
	BackupDir      string `mapstructure:"backup_dir"`       // backup directory (default: ./data/backups)
}

// BruteForceConfig holds brute-force protection settings.
type BruteForceConfig struct {
	MaxAttempts       int    `mapstructure:"max_attempts"`
	LockoutDuration   string `mapstructure:"lockout_duration"`
	WindowDuration    string `mapstructure:"window_duration"`
	ProgressiveDelay  bool   `mapstructure:"progressive_delay"`
	BaseDelay         string `mapstructure:"base_delay"`
	MaxDelay          string `mapstructure:"max_delay"`
	IPMaxAttempts     int    `mapstructure:"ip_max_attempts"`
	IPLockoutDuration string `mapstructure:"ip_lockout_duration"`
}

// AuditConfig holds configuration for audit logging.
type AuditConfig struct {
	RetentionDays int    `mapstructure:"retention_days"` // default: 90
	ExternalLogPath string `mapstructure:"external_log_path"`
}

// RedisConfig holds Redis connection settings for Pub/Sub.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`     // default: "localhost:6379"
	Password string `mapstructure:"password"` // default: ""
	DB       int    `mapstructure:"db"`       // default: 0
}

// ServerConfig holds HTTP/MCP/Web server settings.
type ServerConfig struct {
	Host               string   `mapstructure:"host"`
	Port               int      `mapstructure:"port"`
	MCPPort            int      `mapstructure:"mcp_port"`
	WebPort            int      `mapstructure:"web_port"`
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Type string `mapstructure:"type"`
	DSN  string `mapstructure:"dsn"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTSecret      string `mapstructure:"jwt_secret"`
	TokenExpire    string `mapstructure:"token_expire"`
	WSTicketExpire string `mapstructure:"ws_ticket_expire"`
	OAuthProviders []OAuthProviderConfig `mapstructure:"oauth_providers"`
}



// OAuthProviderConfig holds configuration for an OAuth2 provider.
type OAuthProviderConfig struct {
	Provider     string   `mapstructure:"provider"`
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"`
}
// DeployConfig holds deployment engine settings.
type DeployConfig struct {
	DefaultMode         string `mapstructure:"default_mode"`
	BuildTimeout        string `mapstructure:"build_timeout"`
	HealthCheckInterval string `mapstructure:"health_check_interval"`
	HealthCheckRetries  int    `mapstructure:"health_check_retries"`
	RollbackOnFailure   bool   `mapstructure:"rollback_on_failure"`
	SandboxCPU          int    `mapstructure:"sandbox_cpu"`
	SandboxMemory       string `mapstructure:"sandbox_memory"`
	RuntimeCPU          int    `mapstructure:"runtime_cpu"`
	RuntimeMemory       string `mapstructure:"runtime_memory"`
}

// CacheConfig holds cache settings.
type CacheConfig struct {
	BuildCacheTTL string `mapstructure:"build_cache_ttl"`
	BuildCacheMax int    `mapstructure:"build_cache_max"`
}

// SecurityConfig holds security/rate-limit settings.
type SecurityConfig struct {
	RateLimitDefault int `mapstructure:"rate_limit_default"`
	RateLimitOwner   int `mapstructure:"rate_limit_owner"`
	RateLimitAdmin   int `mapstructure:"rate_limit_admin"`
	RateLimitDev     int `mapstructure:"rate_limit_dev"`
	RateLimitViewer  int `mapstructure:"rate_limit_viewer"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level         string `mapstructure:"level"`
	Format        string `mapstructure:"format"`
	File          string `mapstructure:"file"`
	MaxSize       string `mapstructure:"max_size"`
	MaxBackups    int    `mapstructure:"max_backups"`
	EnableTracing bool   `mapstructure:"enable_tracing"`
}

// NotifyConfig holds notification settings.
type NotifyConfig struct {
	DefaultChannels []string `mapstructure:"default_channels"`
}

// MonitorConfig holds monitoring settings.
type MonitorConfig struct {
	Enabled     bool `mapstructure:"enabled"`
	MetricsPort int  `mapstructure:"metrics_port"`
}

// KubernetesConfig holds Kubernetes cluster settings.
type KubernetesConfig struct {
	Enabled          bool            `mapstructure:"enabled"`
	DefaultNamespace string          `mapstructure:"default_namespace"`
	Clusters         []ClusterConfig `mapstructure:"clusters"`
}

// ClusterConfig holds a single Kubernetes cluster configuration.
type ClusterConfig struct {
	Name      string `mapstructure:"name"`
	APIServer string `mapstructure:"api_server"`
	Context   string `mapstructure:"context"`
	Namespace string `mapstructure:"namespace"`
}

// Load reads the configuration file and applies environment variable overrides.
// Environment variables use the prefix DEPLOYPILOT_ and are mapped to config keys.
// For example, DEPLOYPILOT_SERVER_PORT overrides server.port.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure file reading
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Configure environment variable overrides
	v.SetEnvPrefix("DEPLOYPILOT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// DefaultConfig returns a Config populated with default values.
func DefaultConfig() *Config {
	v := viper.New()
	setDefaults(v)
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		// Fallback to minimal defaults if unmarshal fails
		return &Config{
			Database: DatabaseConfig{
				Type: "sqlite",
				DSN:  "./data/deploypilot.db",
			},
		}
	}
	return &cfg
}

// setDefaults sets default values for all configuration fields.
func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mcp_port", 9090)
	v.SetDefault("server.web_port", 3000)
	// Security: CORS defaults to empty (no origins allowed).
	// Users must explicitly configure allowed origins via config or DEPLOYPILOT_SERVER_CORS_ALLOWED_ORIGINS.
	v.SetDefault("server.cors_allowed_origins", []string{})

	// Database
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.dsn", "./data/deploypilot.db")

	// Auth — no default JWT secret; must be set via JWT_SECRET env var or config
	v.SetDefault("auth.token_expire", "24h")
	v.SetDefault("auth.ws_ticket_expire", "30s")

	// Deploy
	v.SetDefault("deploy.default_mode", "api")
	v.SetDefault("deploy.build_timeout", "10m")
	v.SetDefault("deploy.health_check_interval", "30s")
	v.SetDefault("deploy.health_check_retries", 3)
	v.SetDefault("deploy.rollback_on_failure", true)
	v.SetDefault("deploy.sandbox_cpu", 2)
	v.SetDefault("deploy.sandbox_memory", "4GB")
	v.SetDefault("deploy.runtime_cpu", 2)
	v.SetDefault("deploy.runtime_memory", "2GB")

	// Cache
	v.SetDefault("cache.build_cache_ttl", "720h")
	v.SetDefault("cache.build_cache_max", 50)

	// Security
	v.SetDefault("security.rate_limit_default", 100)
	v.SetDefault("security.rate_limit_owner", 200)
	v.SetDefault("security.rate_limit_admin", 150)
	v.SetDefault("security.rate_limit_dev", 100)
	v.SetDefault("security.rate_limit_viewer", 50)

	// Log
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.file", "./logs/deploypilot.log")
	v.SetDefault("log.max_size", "100MB")
	v.SetDefault("log.max_backups", 10)
	v.SetDefault("log.enable_tracing", true)

	// Notify
	v.SetDefault("notify.default_channels", []string{"webhook"})

	// Monitor
	v.SetDefault("monitor.enabled", true)
	v.SetDefault("monitor.metrics_port", 9091)

	// Redis
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// Kubernetes
	v.SetDefault("kubernetes.enabled", false)
	v.SetDefault("kubernetes.default_namespace", "default")

	// Audit
	v.SetDefault("audit.external_log_path", "")

	// Backup
	v.SetDefault("backup.enabled", true)
	v.SetDefault("backup.interval", "6h")
	v.SetDefault("backup.retention_count", 10)
	v.SetDefault("backup.retention_days", 30)
	v.SetDefault("backup.backup_dir", "./data/backups")

	// BruteForce
	v.SetDefault("bruteforce.max_attempts", 5)
	v.SetDefault("bruteforce.lockout_duration", "15m")
	v.SetDefault("bruteforce.window_duration", "15m")
	v.SetDefault("bruteforce.progressive_delay", true)
	v.SetDefault("bruteforce.base_delay", "1s")
	v.SetDefault("bruteforce.max_delay", "30s")
	v.SetDefault("bruteforce.ip_max_attempts", 20)
	v.SetDefault("bruteforce.ip_lockout_duration", "30m")
}
