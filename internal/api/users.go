package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetCurrentUser returns the currently authenticated user's info.
// @Summary      Get current user
// @Description  Retrieve the currently authenticated user's profile information
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (User with Role and Tenant)"
// @Failure      401 {object} map[string]interface{} "not authenticated"
// @Failure      404 {object} map[string]interface{} "user not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /users/me [get]
func GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get(string(auth.UserIDKey))
	if !exists {
		respondError(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	dbVal, exists := c.Get("db")
	if !exists {
		respondError(c, http.StatusInternalServerError, "database not available")
		return
	}
	db, ok := dbVal.(*gorm.DB)
	if !ok {
		respondError(c, http.StatusInternalServerError, "invalid database connection")
		return
	}

	var user model.User
	if err := db.Preload("Role").Preload("Tenant").Where("id = ?", userID).First(&user).Error; err != nil {
		respondError(c, http.StatusNotFound, "user not found")
		return
	}
	respondSuccess(c, user)
}

// ListUsers lists all users (owner/admin only).
// @Summary      List users
// @Description  Retrieve all users with their role information (owner/admin only)
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (array of User with Role)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      403 {object} map[string]interface{} "forbidden"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /users [get]
func ListUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []model.User
		if err := db.Preload("Role").Find(&users).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if users == nil {
			users = []model.User{}
		}
		respondSuccess(c, users)
	}
}

// DeleteUser deletes a user (owner only).
// @Summary      Delete a user
// @Description  Delete a user by ID (owner only)
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      403 {object} map[string]interface{} "forbidden"
// @Failure      404 {object} map[string]interface{} "user not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /users/{id} [delete]
func DeleteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result := db.Where("id = ?", id).Delete(&model.User{})
		if result.Error != nil {
			respondError(c, http.StatusInternalServerError, result.Error.Error())
			return
		}
		if result.RowsAffected == 0 {
			respondError(c, http.StatusNotFound, "user not found")
			return
		}
		respondSuccess(c, gin.H{"message": "user deleted", "id": id})
	}
}

// UpdateUserRole updates a user's role (owner/admin only).
// @Summary      Update user role
// @Description  Update a user's role assignment (owner/admin only)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Param        request body object{role_id=string} true "Role update request"
// @Success      200 {object} map[string]interface{} "status, data.user_id, data.role_id, data.role_name"
// @Failure      400 {object} map[string]interface{} "invalid request or invalid role_id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      403 {object} map[string]interface{} "forbidden"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /users/{id}/role [put]
func UpdateUserRole(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			RoleID string `json:"role_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}

		// Validate role exists
		var role model.Role
		if err := db.Where("id = ?", input.RoleID).First(&role).Error; err != nil {
			respondError(c, http.StatusBadRequest, "invalid role_id")
			return
		}

		if err := db.Model(&model.User{}).Where("id = ?", id).Update("role_id", input.RoleID).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondSuccess(c, gin.H{"user_id": id, "role_id": input.RoleID, "role_name": role.Name})
	}
}

// ListRoles lists all available roles.
// @Summary      List roles
// @Description  Retrieve all available system roles
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (array of Role)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /roles [get]
func ListRoles(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var roles []model.Role
		if err := db.Find(&roles).Error; err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if roles == nil {
			roles = []model.Role{}
		}
		respondSuccess(c, roles)
	}
}
