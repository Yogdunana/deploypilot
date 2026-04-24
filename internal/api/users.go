package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/auth"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetCurrentUser returns the currently authenticated user's info.
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
