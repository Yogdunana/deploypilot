package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListTemplates lists all available templates (built-in + custom).
// @Summary      List templates
// @Description  Retrieve all available deployment templates (built-in and custom)
// @Tags         Templates
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "status, data (array of Template)"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /templates [get]
func ListTemplates(bridge *service.Bridge) gin.HandlerFunc {
	return func(c *gin.Context) {
		templates, err := bridge.ListTemplates(c.Request.Context())
		if err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, templates)
	}
}

// CreateTemplate creates a custom template.
// @Summary      Create a template
// @Description  Create a custom deployment template
// @Tags         Templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{name=string,type=string,description=string,build_cmd=string,run_cmd=string,port=int} true "Template creation request"
// @Success      200 {object} map[string]interface{} "status, data.id, data.name, data.type"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /templates [post]
func CreateTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Name        string `json:"name" binding:"required"`
			Type        string `json:"type" binding:"required"`
			Description string `json:"description"`
			BuildCmd    string `json:"build_cmd"`
			RunCmd      string `json:"run_cmd"`
			Port        int    `json:"port"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		id := uuid.New().String()
		// Store custom templates in the providers table with type "template"
		if err := db.Exec(
			`INSERT INTO providers (id, tenant_id, type, name, config, enabled) VALUES (?, ?, 'template', ?, ?, 1)`,
			id, model.DefaultTenantID, input.Name,
			map[string]interface{}{
				"type":        input.Type,
				"description": input.Description,
				"build_cmd":   input.BuildCmd,
				"run_cmd":     input.RunCmd,
				"port":        input.Port,
			},
		).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{"id": id, "name": input.Name, "type": input.Type})
	}
}

// UpdateTemplate updates a custom template.
// @Summary      Update a template
// @Description  Update an existing custom deployment template
// @Tags         Templates
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Template ID"
// @Param        request body object{name=string,description=string,build_cmd=string,run_cmd=string,port=int} true "Template update request"
// @Success      200 {object} map[string]interface{} "status, data.id, data.message"
// @Failure      400 {object} map[string]interface{} "invalid request"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /templates/{id} [put]
func UpdateTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var input struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			BuildCmd    string `json:"build_cmd"`
			RunCmd      string `json:"run_cmd"`
			Port        int    `json:"port"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}

		if err := db.Exec(
			`UPDATE providers SET config = ?, name = ? WHERE id = ? AND type = 'template'`,
			map[string]interface{}{
				"description": input.Description,
				"build_cmd":   input.BuildCmd,
				"run_cmd":     input.RunCmd,
				"port":        input.Port,
			},
			input.Name, id,
		).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		respondSuccess(c, gin.H{"id": id, "message": "template updated"})
	}
}

// DeleteTemplate deletes a custom template.
// @Summary      Delete a template
// @Description  Delete a custom deployment template by ID
// @Tags         Templates
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Template ID"
// @Success      200 {object} map[string]interface{} "status, data.message, data.id"
// @Failure      401 {object} map[string]interface{} "unauthorized"
// @Failure      404 {object} map[string]interface{} "template not found"
// @Failure      500 {object} map[string]interface{} "internal error"
// @Router       /templates/{id} [delete]
func DeleteTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result := db.Exec(`DELETE FROM providers WHERE id = ? AND type = 'template'`, id)
		if result.Error != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
			return
		}
		if result.RowsAffected == 0 {
			respondErrori18n(c, http.StatusNotFound, "error.template.not_found")
			return
		}
		respondSuccess(c, gin.H{"message": "template deleted", "id": id})
	}
}
