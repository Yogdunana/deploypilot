package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
		var input struct {
			Username string `json:"username" binding:"required"`
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
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
func Login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		var user model.User
		if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_credentials")
			return
		}

		if !crypto.CheckPassword(input.Password, user.PasswordHash) {
			respondErrori18n(c, http.StatusUnauthorized, "error.auth.invalid_credentials")
			return
		}

		// Determine role name
		roleName := "viewer"
		var role model.Role
		if err := db.Where("id = ?", user.RoleID).First(&role).Error; err == nil {
			roleName = role.Name
		}

		token, err := auth.GenerateToken(user.ID, roleName)
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.auth.failed_to_generate_token")
			return
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
