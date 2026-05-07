package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/bruteforce"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/middleware"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// passwordValidator is the global password validator instance.
var passwordValidator *middleware.PasswordValidator

// auditSvcForAuth is the global audit service for auth events.
var auditSvcForAuth *service.AuditService

// registerRateLimiter provides per-IP rate limiting for registration attempts.
// Max 5 registrations per IP per 15 minutes.
type registerRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*rateLimitEntry
	max      int
	window   time.Duration
}

type rateLimitEntry struct {
	count    int
	expireAt time.Time
}

var registerRL = &registerRateLimiter{
	attempts: make(map[string]*rateLimitEntry),
	max:      5,
	window:   15 * time.Minute,
}

// Allow checks if the given IP is allowed to register.
func (rl *registerRateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	entry, ok := rl.attempts[ip]
	if !ok || now.After(entry.expireAt) {
		rl.attempts[ip] = &rateLimitEntry{count: 1, expireAt: now.Add(rl.window)}
		return true
	}
	if entry.count >= rl.max {
		return false
	}
	entry.count++
	return true
}

// Remaining returns the remaining registration attempts for the given IP.
func (rl *registerRateLimiter) Remaining(ip string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	entry, ok := rl.attempts[ip]
	if !ok || now.After(entry.expireAt) {
		return rl.max
	}
	return rl.max - entry.count
}

// SetPasswordValidator sets the global password validator for registration and password changes.
func SetPasswordValidator(v *middleware.PasswordValidator) {
	passwordValidator = v
}

// SetAuditServiceForAuth sets the global audit service for auth event logging.
func SetAuditServiceForAuth(svc *service.AuditService) {
	auditSvcForAuth = svc
}

// Register handles user registration.
// @Summary      Register a new user
// @Description  Create a new user account with username, email, and password
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body object{username=string,email=string,password=string} true "Registration request"
// @Success      201 {object} map[string]interface{} "status, data.user, data.token"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      409 {object} map[string]interface{} "username or email already exists"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /auth/register [post]
func Register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Rate limit: max 5 registrations per IP per 15 minutes
		clientIP := c.ClientIP()
		if !registerRL.Allow(clientIP) {
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many registration attempts, please try again later",
			})
			c.Abort()
			return
		}
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", registerRL.Remaining(clientIP)))

		var input struct {
			Username string `json:"username" binding:"required"`
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		// Input length limits
		if len(input.Username) > 255 || len(input.Email) > 255 || len(input.Password) > 128 {
			respondErrori18n(c, http.StatusBadRequest, "error.common.input_too_long")
			return
		}

		// Validate password complexity
		if passwordValidator != nil {
			if err := passwordValidator.Validate(input.Password); err != nil {
				var pwdErr *middleware.PasswordValidationError
				errorsAs := false
				if errors.As(err, &pwdErr) {
					errorsAs = true
				}
				resp := gin.H{
					"status":  "error",
					"message": "password does not meet security requirements",
				}
				if errorsAs {
					resp["errors"] = pwdErr.Errors
				}
				c.JSON(http.StatusBadRequest, resp)
				return
			}
		}

		// Check if username or email already exists
		var existing model.User
		if err := db.Where("username = ? OR email = ?", input.Username, input.Email).First(&existing).Error; err == nil {
			respondErrori18n(c, http.StatusConflict, "error.auth.username_or_email_exists")
			return
		}

		// Hash password
		hash, err := crypto.HashPassword(input.Password)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.auth.failed_to_hash_password")
			return
		}

		user := model.User{
			ID:           uuid.New().String(),
			TenantID:     "tenant-default",
			RoleID:       "role-viewer",
			Username:     input.Username,
			Email:        input.Email,
			PasswordHash: hash,
		}

		if err := db.Create(&user).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		// Generate token
		token, err := auth.GenerateToken(user.ID, "viewer")
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.auth.failed_to_generate_token")
			return
		}

		// Set auth cookies if refresh store is available
		if refreshStore != nil {
			refreshID, rErr := auth.GenerateRefreshTokenID()
			if rErr == nil {
				if err := refreshStore.Store(auth.RefreshTokenEntry{
					TokenID: refreshID, UserID: user.ID, Role: "viewer",
					DeviceInfo: c.GetHeader("User-Agent"), IPAddress: c.ClientIP(),
					CreatedAt: time.Now(), ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
				}); err != nil {
					slog.Warn("failed to store refresh token", "error", err)
				}
				setAuthCookies(c, token, refreshID)
			}
		}

		// Record register audit event
		if auditSvcForAuth != nil {
			if err := auditSvcForAuth.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(user.ID),
				Username:     user.Username,
				Action:       "user.register",
				ResourceType: "user",
				ResourceID:   user.ID,
				IPAddress:    c.ClientIP(),
				UserAgent:    c.GetHeader("User-Agent"),
			}); err != nil {
				slog.Warn("failed to record audit log", "error", err)
			}
		}

		respondSuccess(c, gin.H{
			"user": model.User{
				ID:       user.ID,
				TenantID: user.TenantID,
				RoleID:   user.RoleID,
				Username: user.Username,
				Email:    user.Email,
			},
			"token": token,
		})
	}
}

// RevokeToken revokes the current JWT token.
// @Summary      Revoke token (logout)
// @Description  Revoke the current JWT token, preventing further use
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data.message"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /auth/revoke [post]
func RevokeToken(blacklist auth.TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.authorization_header_required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_authorization_format")
			return
		}

		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_or_expired_token")
			return
		}

		if claims.ID == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		// Calculate remaining TTL
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl <= 0 {
			// Token already expired, no need to revoke
			respondSuccess(c, gin.H{"message": "token_already_expired"})
			return
		}

		if err := blacklist.Revoke(claims.ID, ttl); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		// Clear auth cookies on logout
		clearAuthCookies(c)

		respondSuccess(c, gin.H{"message": "token_revoked"})
	}
}

// WSTicket generates a one-time WebSocket authentication ticket.
// The client must present a valid JWT in the Authorization header.
// The returned ticket can be used once within the expiration window to authenticate a WebSocket connection.
// @Summary      Generate WebSocket ticket
// @Description  Exchange a valid JWT for a one-time WebSocket authentication ticket (valid for 30 seconds)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data.ticket, data.expires_in"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /auth/ws-ticket [post]
func WSTicket(ticketStore *auth.WSTicketStore, expire time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.authorization_header_required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_authorization_format")
			return
		}

		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_or_expired_token")
			return
		}

		ticket, err := ticketStore.GenerateTicket(claims.UserID, claims.Role, expire)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.auth.ws_ticket_generate_failed")
			return
		}

		respondSuccess(c, gin.H{
			"ticket":     ticket,
			"expires_in": int(expire.Seconds()),
		})
	}
}

// Login handles user authentication.
// @Summary      Login
// @Description  Authenticate with username and password to obtain a JWT token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body object{username=string,password=string} true "Login request"
// @Success      200 {object} map[string]interface{} "status, data.user, data.token"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "invalid credentials"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /auth/login [post]
func Login(db *gorm.DB, bf *bruteforce.Protector) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		clientIP := c.ClientIP()

		// Check brute-force protection
		if bf != nil {
			check := bf.Check(input.Username, clientIP)
			if !check.Allowed {
				c.Header("Retry-After", fmt.Sprintf("%d", int(time.Until(check.LockedUntil).Seconds())))
				respondError(c, http.StatusTooManyRequests, bruteforce.IsAccountLockedError(check))
				return
			}
			// Apply progressive delay
			if check.Delay > 0 {
				time.Sleep(check.Delay)
			}
		}

		var user model.User
		if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
			if bf != nil {
				bf.RecordFailure(input.Username, clientIP, "user_not_found")
			}
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_credentials")
			return
		}

		if !crypto.CheckPassword(input.Password, user.PasswordHash) {
			if bf != nil {
				bf.RecordFailure(input.Username, clientIP, "invalid_password")
			}
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_credentials")
			return
		}

		// Successful login — clear failures
		if bf != nil {
			bf.RecordSuccess(input.Username, clientIP)
		}

		// Determine role name
		roleName := "viewer"
		var role model.Role
		if err := db.Where("id = ?", user.RoleID).First(&role).Error; err == nil {
			roleName = role.Name
		}

		// If 2FA is enabled, return a pending token instead of a full JWT
		if user.TOTPEnabled {
			pendingToken, err := auth.Generate2FAPendingToken(user.ID, roleName)
			if err != nil {
				respondErrori18n(c, http.StatusInternalServerError, "error.auth.failed_to_generate_token")
				return
			}
			respondSuccess(c, gin.H{
				"requires_2fa": true,
				"two_fa_token": pendingToken,
				"user_id":      user.ID,
			})
			return
		}

		token, err := auth.GenerateToken(user.ID, roleName)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.auth.failed_to_generate_token")
			return
		}

		// Set auth cookies if refresh store is available
		if refreshStore != nil {
			refreshID, rErr := auth.GenerateRefreshTokenID()
			if rErr == nil {
				if err := refreshStore.Store(auth.RefreshTokenEntry{
					TokenID: refreshID, UserID: user.ID, Role: roleName,
					DeviceInfo: c.GetHeader("User-Agent"), IPAddress: c.ClientIP(),
					CreatedAt: time.Now(), ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
				}); err != nil {
					slog.Warn("failed to store refresh token", "error", err)
				}
				setAuthCookies(c, token, refreshID)
			}
		}

		// Record login audit event
		if auditSvcForAuth != nil {
			if err := auditSvcForAuth.Record(c.Request.Context(), service.AuditEntry{
				UserID:       parseUserID(user.ID),
				Username:     user.Username,
				Action:       "user.login",
				ResourceType: "user",
				ResourceID:   user.ID,
				IPAddress:    c.ClientIP(),
				UserAgent:    c.GetHeader("User-Agent"),
			}); err != nil {
				slog.Warn("failed to record audit log", "error", err)
			}
		}

		// Detect first login and update last_login_at
		isFirstLogin := user.LastLoginAt == nil && !user.OnboardingCompleted
		now := time.Now()
		db.Model(&user).Updates(map[string]interface{}{
			"last_login_at": now,
		})

		respondSuccess(c, gin.H{
			"user": model.User{
				ID:                 user.ID,
				TenantID:           user.TenantID,
				RoleID:             user.RoleID,
				Username:           user.Username,
				Email:              user.Email,
				OnboardingCompleted: user.OnboardingCompleted,
				LastLoginAt:        &now,
			},
			"token":          token,
			"is_first_login": isFirstLogin,
		})
	}
}


// OAuthLogin initiates an OAuth2 login flow.
// @Summary      OAuth login
// @Description  Redirect to OAuth provider for authentication
// @Tags         Auth
// @Produce      json
// @Param        provider path string true "OAuth provider (github, gitee)"
// @Success      302 {string} string "Redirect to OAuth provider"
// @Failure      400 {object} map[string]interface{} "invalid provider"
// @Router       /auth/oauth/{provider} [get]
func OAuthLogin(oauthSvc *service.OAuthService, stateStore auth.StateStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("provider")
		if !oauthSvc.IsProviderConfigured(provider) {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		state := uuid.New().String()
		if err := stateStore.Generate(state, 5*time.Minute); err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		authURL, err := oauthSvc.AuthURL(provider, state)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, authURL)
	}
}

// OAuthCallback handles the OAuth2 callback.
// @Summary      OAuth callback
// @Description  Handle OAuth provider callback and return JWT token
// @Tags         Auth
// @Produce      json
// @Param        provider path string true "OAuth provider (github, gitee)"
// @Param        code query string true "Authorization code"
// @Param        state query string true "State parameter"
// @Success      200 {object} map[string]interface{} "status, data.user, data.token"
// @Failure      400 {object} map[string]interface{} "invalid state or provider"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /auth/oauth/{provider}/callback [get]
func OAuthCallback(oauthSvc *service.OAuthService, stateStore auth.StateStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("provider")
		code := c.Query("code")
		state := c.Query("state")

		if code == "" || state == "" {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		if !stateStore.Validate(state) {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		user, roleName, err := oauthSvc.HandleCallback(c.Request.Context(), provider, code)
		if err != nil {
			slog.Error("OAuth callback failed", "provider", provider, "error", err)
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}

		token, err := auth.GenerateToken(user.ID, roleName)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.auth.failed_to_generate_token")
			return
		}

		respondSuccess(c, gin.H{
			"user": model.User{
				ID:           user.ID,
				TenantID:     user.TenantID,
				RoleID:       user.RoleID,
				Username:     user.Username,
				Email:        user.Email,
				AuthProvider: user.AuthProvider,
				AvatarURL:    user.AvatarURL,
			},
			"token": token,
		})
	}
}
