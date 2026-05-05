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

// GetSecurityConfig godoc
// @Summary      Get security config
// @Description  Get the current security configuration (read-only)
// @Tags         Security
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "security configuration settings"
// @Router       /system/security/config [get]
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
				"force_2fa":                cfg.Force2FA,
				"force_2fa_roles":          cfg.Force2FARoles,
				"force_2fa_grace_days":     cfg.Force2FAGraceDays,
			},
		})
	}
}

// UpdateSecurityConfig godoc
// @Summary      Update security config
// @Description  Update the security configuration at runtime (requires owner or admin role)
// @Tags         Security
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{security_entrance=string,allowed_domains=[]string,allowed_ips=[]string,password_min_len=int,password_require_upper=bool,password_require_lower=bool,password_require_digit=bool,password_require_special=bool,password_max_age_days=int,force_2fa=bool,force_2fa_roles=[]string,force_2fa_grace_days=int} true "Security config update request (only non-nil fields are updated)"
// @Success      200 {object} map[string]interface{} "update confirmation"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      403 {object} map[string]interface{} "forbidden"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /system/security/config [put]
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
			Force2FA                *bool    `json:"force_2fa"`
			Force2FARoles           []string `json:"force_2fa_roles"`
			Force2FAGraceDays       *int     `json:"force_2fa_grace_days"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		// Validate input ranges
		if input.PasswordMinLen != nil && *input.PasswordMinLen > 0 && *input.PasswordMinLen < 8 {
			respondErrori18n(c, http.StatusBadRequest, "error.security.password_min_length_too_short")
			return
		}
		if input.PasswordMaxAgeDays != nil && *input.PasswordMaxAgeDays < 0 {
			respondErrori18n(c, http.StatusBadRequest, "error.security.invalid_max_age_days")
			return
		}
		if input.Force2FAGraceDays != nil && *input.Force2FAGraceDays < 0 {
			respondErrori18n(c, http.StatusBadRequest, "error.security.invalid_grace_days")
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
		if input.Force2FA != nil {
			securityCfg.Force2FA = *input.Force2FA
		}
		if input.Force2FARoles != nil {
			securityCfg.Force2FARoles = input.Force2FARoles
		}
		if input.Force2FAGraceDays != nil {
			securityCfg.Force2FAGraceDays = *input.Force2FAGraceDays
		}

		// Update the global password validator
		SetPasswordValidator(middleware.NewPasswordValidator(*securityCfg))

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "security configuration updated",
		})
	}
}
