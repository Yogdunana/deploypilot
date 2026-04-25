package api

import (
	"net/http"
	"strconv"
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
		var certs []model.SSLCertificate
		if result := db.Find(&certs); result.Error != nil {
			respondError(c, http.StatusInternalServerError, "failed to list certificates")
			return
		}
		respondSuccess(c, certs)
	}
}

// RequestSSLCertificate creates a new SSL certificate request.
// @Summary      Request a new SSL certificate
// @Tags         SSL
// @Accept       json
// @Produce      json
// @Param        request body object{domain=string,email=string,provider=string} true "Certificate request"
// @Success      201 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /ssl/certificates [post]
// @Security     BearerAuth
func RequestSSLCertificate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Domain   string `json:"domain" binding:"required"`
			Email    string `json:"email" binding:"required"`
			Provider string `json:"provider"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
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
			respondError(c, http.StatusInternalServerError, "failed to create certificate record")
			return
		}
		c.JSON(http.StatusCreated, gin.H{"status": "success", "data": cert})
	}
}

// DeleteSSLCertificate deletes an SSL certificate by ID.
// @Summary      Delete an SSL certificate
// @Tags         SSL
// @Produce      json
// @Param        id path int true "Certificate ID"
// @Success      200 {object} map[string]interface{}
// @Router       /ssl/certificates/{id} [delete]
// @Security     BearerAuth
func DeleteSSLCertificate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid id")
			return
		}
		if result := db.Delete(&model.SSLCertificate{}, id); result.RowsAffected == 0 {
			respondError(c, http.StatusNotFound, "certificate not found")
			return
		}
		respondSuccess(c, gin.H{"message": "deleted"})
	}
}

// RenewSSLCertificate initiates renewal of an SSL certificate.
// @Summary      Renew an SSL certificate
// @Tags         SSL
// @Produce      json
// @Param        id path int true "Certificate ID"
// @Success      200 {object} map[string]interface{}
// @Router       /ssl/certificates/{id}/renew [post]
// @Security     BearerAuth
func RenewSSLCertificate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid id")
			return
		}
		var cert model.SSLCertificate
		if result := db.First(&cert, id); result.Error != nil {
			respondError(c, http.StatusNotFound, "certificate not found")
			return
		}
		now := time.Now()
		cert.Status = "renewing"
		cert.RetryCount++
		cert.LastRenewed = &now
		db.Save(&cert)
		respondSuccess(c, gin.H{"data": cert, "message": "renewal initiated"})
	}
}
