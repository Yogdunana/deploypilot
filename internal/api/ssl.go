package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListSSLCertificates lists all SSL certificates.
// @Summary      List SSL certificates
// @Tags         SSL
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /ssl/certificates [get]
// @Security     BearerAuth
func ListSSLCertificates(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		var certs []model.SSLCertificate
		var total int64

		if err := db.Model(&model.SSLCertificate{}).Count(&total).Error; err != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.ssl.failed_to_list_certificates")
			return
		}

		if result := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&certs); result.Error != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.ssl.failed_to_list_certificates")
			return
		}
		// Return paginated response with data key for backward compatibility
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data":   certs,
			"pagination": gin.H{
				"page":      page,
				"page_size": pageSize,
				"total":     total,
			},
		})
	}
}

// RequestSSLCertificate creates a new SSL certificate request.
// @Summary      Create a new SSL certificate
// @Tags         SSL
// @Accept       json
// @Produce      json
// @Param        request body object{domain=string,email=string,provider=string} true "Certificate request"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /ssl/certificates [post]
// @Security     BearerAuth
func RequestSSLCertificate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Domain   string `json:"domain" binding:"required"`
			Email    string `json:"email" binding:"required,email"`
			Provider string `json:"provider"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", err.Error())
			return
		}

		// Validate domain format
		if !isValidDomain(req.Domain) {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request", "invalid domain format")
			return
		}

		cert := model.SSLCertificate{
			Domain:    req.Domain,
			Email:     req.Email,
			Provider:  req.Provider,
			Status:    "pending",
			AutoRenew: true,
		}
		if result := db.Create(&cert); result.Error != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.ssl.failed_to_create_certificate")
			return
		}
		c.JSON(http.StatusCreated, gin.H{"status": "success", "data": cert})
	}
}

// DeleteSSLCertificate deletes an SSL certificate.
// @Summary      Delete an SSL certificate
// @Tags         SSL
// @Produce      json
// @Param        id path string true "Certificate ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /ssl/certificates/{id} [delete]
// @Security     BearerAuth
func DeleteSSLCertificate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.ssl.invalid_id")
			return
		}
		if result := db.Delete(&model.SSLCertificate{}, id); result.RowsAffected == 0 {
			respondErrori18n(c, http.StatusNotFound, "error.ssl.certificate_not_found")
			return
		}
		respondSuccess(c, gin.H{"message": "deleted"})
	}
}

// RenewSSLCertificate initiates renewal of an SSL certificate.
// @Summary      Renew an SSL certificate
// @Tags         SSL
// @Produce      json
// @Param        id path string true "Certificate ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /ssl/certificates/{id}/renew [post]
// @Security     BearerAuth
func RenewSSLCertificate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.ssl.invalid_id")
			return
		}
		var cert model.SSLCertificate
		if err := db.First(&cert, id).Error; err != nil {
			respondErrori18n(c, http.StatusNotFound, "error.ssl.certificate_not_found")
			return
		}
		// In a real implementation, this would trigger the renewal process
		cert.Status = "renewing"
		cert.RetryCount++
		newExpiresAt := time.Now().AddDate(0, 0, 90)
		cert.ExpiresAt = &newExpiresAt
		if result := db.Save(&cert); result.Error != nil {
			respondErrori18n(c, http.StatusInternalServerError, "error.ssl.failed_to_renew_certificate")
			return
		}
		respondSuccess(c, gin.H{"data": cert, "message": "renewal initiated"})
	}
}

// isValidDomain checks whether the given string looks like a valid domain name.
func isValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	// Simple domain validation: must contain at least one dot, no spaces
	return strings.Contains(domain, ".") && !strings.ContainsAny(domain, " \t\n")
}
