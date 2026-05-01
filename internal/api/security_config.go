package api

import (
	"net/http"
	"sync"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/middleware"
	"github.com/gin-gonic/gin"
)

// securityConfigStore holds the runtime security configuration.
var (
	securityCfgMu sync.RWMutex
	securityCfg   *config.SecurityConfig
)

// SetSecurityConfig stores the security config for API access.
func SetSecurityConfig(cfg *config.SecurityConfig) {
	securityCfgMu.Lock()
	defer securityCfgMu.Unlock()
	securityCfg = cfg
}

// GetSecurityConfig returns the current security configuration (read-only).
// GET /api/v1/system/security/config
func GetSecurityConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		securityCfgMu.RLock()
		cfg := securityCfg
		securityCfgMu.RUnlock()

		if cfg == nil {
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data": gin.H{
				"security_entrance":        cfg.SecurityEntrance,
				"allowed_domains":          cfg.AllowedDomains,
				"allowed_ips":              cfg.AllowedIPs,
				"password_min_len":         cfg.PasswordMinLen,
				"password_require_upper":   cfg.PasswordRequireUpper,
				"password_require_lower":   cfg.PasswordRequireLower,
				"password_require_digit":   cfg.PasswordRequireDigit,
				"password_require_special": cfg.PasswordRequireSpecial,
				"password_max_age_days":    cfg.PasswordMaxAgeDays,
			},
		})
	}
}

// UpdateSecurityConfig updates the security configuration at runtime.
// PUT /api/v1/system/security/config
func UpdateSecurityConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			SecurityEntrance        *string  `json:"security_entrance"`
			AllowedDomains          []string `json:"allowed_domains"`
			AllowedIPs              []string `json:"allowed_ips"`
			PasswordMinLen          *int     `json:"password_min_len"`
			PasswordRequireUpper    *bool    `json:"password_require_upper"`
			PasswordRequireLower    *bool    `json:"password_require_lower"`
			PasswordRequireDigit    *bool    `json:"password_require_digit"`
			PasswordRequireSpecial  *bool    `json:"password_require_special"`
			PasswordMaxAgeDays      *int     `json:"password_max_age_days"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		securityCfgMu.Lock()
		defer securityCfgMu.Unlock()

		if securityCfg == nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error", "security config not initialized")
			return
		}

		// Apply updates (only non-nil fields)
		if input.SecurityEntrance != nil {
			securityCfg.SecurityEntrance = *input.SecurityEntrance
		}
		if input.AllowedDomains != nil {
			securityCfg.AllowedDomains = input.AllowedDomains
		}
		if input.AllowedIPs != nil {
			securityCfg.AllowedIPs = input.AllowedIPs
		}
		if input.PasswordMinLen != nil {
			securityCfg.PasswordMinLen = *input.PasswordMinLen
		}
		if input.PasswordRequireUpper != nil {
			securityCfg.PasswordRequireUpper = *input.PasswordRequireUpper
		}
		if input.PasswordRequireLower != nil {
			securityCfg.PasswordRequireLower = *input.PasswordRequireLower
		}
		if input.PasswordRequireDigit != nil {
			securityCfg.PasswordRequireDigit = *input.PasswordRequireDigit
		}
		if input.PasswordRequireSpecial != nil {
			securityCfg.PasswordRequireSpecial = *input.PasswordRequireSpecial
		}
		if input.PasswordMaxAgeDays != nil {
			securityCfg.PasswordMaxAgeDays = *input.PasswordMaxAgeDays
		}

		// Update the global password validator
		SetPasswordValidator(middleware.NewPasswordValidator(*securityCfg))

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "security configuration updated",
		})
	}
}
