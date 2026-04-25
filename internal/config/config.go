package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the root configuration structure.
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Deploy    DeployConfig    `mapstructure:"deploy"`
	Cache     CacheConfig     `mapstructure:"cache"`
	Security  SecurityConfig  `mapstructure:"security"`
	Log       LogConfig       `mapstructure:"log"`
	Notify    NotifyConfig    `mapstructure:"notify"`
	Monitor   MonitorConfig   `mapstructure:"monitor"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
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
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"` // default: ["*"]
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
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	File       string `mapstructure:"file"`
	MaxSize    string `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
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
	return &Config{
		Database: DatabaseConfig{
			Type: "sqlite",
			DSN:  "./data/deploypilot.db",
		},
	}
}

// setDefaults sets default values for all configuration fields.
func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mcp_port", 9090)
	v.SetDefault("server.web_port", 3000)
	v.SetDefault("server.cors_allowed_origins", []string{"*"})

	// Database
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.dsn", "./data/deploypilot.db")

	// Auth
	v.SetDefault("auth.jwt_secret", "change-me-in-production")
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
}
