package api

import (
	"net/http"

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
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		// Check if username or email already exists
		var existing model.User
		if err := db.Where("username = ? OR email = ?", input.Username, input.Email).First(&existing).Error; err == nil {
			respondError(c, http.StatusConflict, "username or email already exists")
			return
		}

		// Hash password
		hash, err := crypto.HashPassword(input.Password)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "failed to hash password")
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
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}

		// Generate token
		token, err := auth.GenerateToken(user.ID, "viewer")
		if err != nil {
			respondError(c, http.StatusInternalServerError, "failed to generate token")
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
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		var user model.User
		if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
			respondError(c, http.StatusUnauthorized, "invalid credentials")
			return
		}

		if !crypto.CheckPassword(input.Password, user.PasswordHash) {
			respondError(c, http.StatusUnauthorized, "invalid credentials")
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
			respondError(c, http.StatusInternalServerError, "failed to generate token")
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
