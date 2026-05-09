package auth

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/i18n"
	"github.com/gin-gonic/gin"
)

// OAuth2ScopesKey is the context key for OAuth2 token scopes.
const OAuth2ScopesKey contextKey = "oauth2_scopes"

// Scope constants for the API Open Platform.
const (
	ScopeRead           = "read"
	ScopeWrite          = "write"
	ScopeDelete         = "delete"
	ScopeDeploy         = "deploy"
	ScopeAdmin          = "admin"
	ScopeMonitorRead    = "monitor:read"
	ScopeMonitorWrite   = "monitor:write"
	ScopeServerRead     = "server:read"
	ScopeServerExec     = "server:exec"
	ScopeCredentialRead = "credential:read"
	ScopeCredentialWrite = "credential:write"
	ScopeDNSWrite       = "dns:write"
	ScopeSSLWrite       = "ssl:write"
	ScopeBackupRead     = "backup:read"
	ScopeBackupWrite    = "backup:write"
	ScopeWebhookManage  = "webhook:manage"
	ScopeGrafanaManage  = "grafana:manage"
)

// AllScopes is the complete list of valid scope strings.
var AllScopes = []string{
	ScopeRead,
	ScopeWrite,
	ScopeDelete,
	ScopeDeploy,
	ScopeAdmin,
	ScopeMonitorRead,
	ScopeMonitorWrite,
	ScopeServerRead,
	ScopeServerExec,
	ScopeCredentialRead,
	ScopeCredentialWrite,
	ScopeDNSWrite,
	ScopeSSLWrite,
	ScopeBackupRead,
	ScopeBackupWrite,
	ScopeWebhookManage,
	ScopeGrafanaManage,
}

// allScopesSet is a lookup map for fast scope validation.
var allScopesSet = func() map[string]bool {
	m := make(map[string]bool, len(AllScopes))
	for _, s := range AllScopes {
		m[s] = true
	}
	return m
}()

// ScopeDescriptions provides human-readable descriptions for each scope.
var ScopeDescriptions = map[string]string{
	ScopeRead:           "Read-only access to resources",
	ScopeWrite:          "Create and update resources",
	ScopeDelete:         "Delete resources",
	ScopeDeploy:         "Trigger deployments and rollbacks",
	ScopeAdmin:          "Full administrative access (bypasses all scope checks)",
	ScopeMonitorRead:    "View monitoring data, metrics, and alerts",
	ScopeMonitorWrite:   "Create and manage monitors, alert rules, and silences",
	ScopeServerRead:     "View server information and status",
	ScopeServerExec:     "Execute commands on servers",
	ScopeCredentialRead: "View credentials and SSH keys",
	ScopeCredentialWrite: "Create, update, and rotate credentials",
	ScopeDNSWrite:       "Manage DNS records",
	ScopeSSLWrite:       "Manage SSL/TLS certificates",
	ScopeBackupRead:     "View backup records",
	ScopeBackupWrite:    "Create and manage backups",
	ScopeWebhookManage:  "Create and manage outbound webhooks",
	ScopeGrafanaManage:  "Manage Grafana dashboards and sync",
}

// IsValidScope checks if a single scope string is valid.
func IsValidScope(scope string) bool {
	return allScopesSet[scope]
}

// ValidateScopes filters a list of scopes to only include valid ones.
func ValidateScopes(scopes []string) []string {
	valid := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if allScopesSet[s] {
			valid = append(valid, s)
		}
	}
	return valid
}

// RequireScope returns middleware that checks if the current request has
// one of the required scopes. It checks both APIKeyScopesKey and
// OAuth2ScopesKey from the Gin context. The "admin" scope bypasses all checks.
func RequireScope(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try API key scopes first
		var tokenScopes []string
		if v, exists := c.Get(string(APIKeyScopesKey)); exists {
			if s, ok := v.([]string); ok {
				tokenScopes = s
			}
		}

		// Fall back to OAuth2 scopes
		if len(tokenScopes) == 0 {
			if v, exists := c.Get(string(OAuth2ScopesKey)); exists {
				if s, ok := v.([]string); ok {
					tokenScopes = s
				}
			}
		}

		// If no scopes found (e.g., JWT auth without scopes), allow through
		// JWT auth permissions are handled by RoleRequired middleware
		if len(tokenScopes) == 0 {
			c.Next()
			return
		}

		// Check admin bypass
		for _, s := range tokenScopes {
			if s == ScopeAdmin {
				c.Next()
				return
			}
		}

		// Check if any required scope is satisfied
		for _, required := range scopes {
			for _, s := range tokenScopes {
				if s == required {
					c.Next()
					return
				}
			}
		}

		// Insufficient scope
		locale := i18n.GetLocaleFromContext(c)
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": i18n.T(locale, "error.auth.insufficient_permissions"),
		})
		c.Abort()
	}
}
