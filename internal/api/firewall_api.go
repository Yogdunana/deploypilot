package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/sandbox"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FirewallAPI provides HTTP handlers for remote firewall management.
type FirewallAPI struct {
	fwSvc *service.FirewallService
}

// NewFirewallAPI creates a new FirewallAPI.
func NewFirewallAPI(db *gorm.DB, sb *sandbox.Sandbox) *FirewallAPI {
	return &FirewallAPI{
		fwSvc: service.NewFirewallService(db, sb),
	}
}

// GetFirewallStatus returns the current firewall status and rules.
// GET /api/v1/servers/:server_id/firewall
func (fw *FirewallAPI) GetFirewallStatus(c *gin.Context) {
	serverID := c.Param("server_id")

	status, err := fw.fwSvc.GetStatus(c.Request.Context(), serverID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, status)
}

// DetectFirewall detects which firewall is active on the server.
// GET /api/v1/servers/:server_id/firewall/detect
func (fw *FirewallAPI) DetectFirewall(c *gin.Context) {
	serverID := c.Param("server_id")

	status, err := fw.fwSvc.DetectFirewall(c.Request.Context(), serverID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, status)
}

// OpenPort opens a port on the remote firewall.
// POST /api/v1/servers/:server_id/firewall/ports/open
func (fw *FirewallAPI) OpenPort(c *gin.Context) {
	serverID := c.Param("server_id")

	var req struct {
		Port     string `json:"port" binding:"required"`
		Protocol string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if !service.IsValidPort(req.Port) {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := fw.fwSvc.OpenPort(c.Request.Context(), serverID, req.Port, req.Protocol); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"port": req.Port, "protocol": req.Protocol, "status": "opened"})
}

// ClosePort closes a port on the remote firewall.
// POST /api/v1/servers/:server_id/firewall/ports/close
func (fw *FirewallAPI) ClosePort(c *gin.Context) {
	serverID := c.Param("server_id")

	var req struct {
		Port     string `json:"port" binding:"required"`
		Protocol string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if !service.IsValidPort(req.Port) {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := fw.fwSvc.ClosePort(c.Request.Context(), serverID, req.Port, req.Protocol); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"port": req.Port, "protocol": req.Protocol, "status": "closed"})
}

// BlockIP blocks an IP address on the remote firewall.
// POST /api/v1/servers/:server_id/firewall/blocks
func (fw *FirewallAPI) BlockIP(c *gin.Context) {
	serverID := c.Param("server_id")

	var req struct {
		IP string `json:"ip" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if !service.IsValidIP(req.IP) {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := fw.fwSvc.BlockIP(c.Request.Context(), serverID, req.IP); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"ip": req.IP, "status": "blocked"})
}

// UnblockIP unblocks an IP address on the remote firewall.
// DELETE /api/v1/servers/:server_id/firewall/blocks/:ip
func (fw *FirewallAPI) UnblockIP(c *gin.Context) {
	serverID := c.Param("server_id")
	ip := c.Param("ip")

	if !service.IsValidIP(ip) {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := fw.fwSvc.UnblockIP(c.Request.Context(), serverID, ip); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"ip": ip, "status": "unblocked"})
}

// AllowCommonPorts opens commonly needed ports (22, 80, 443, 8080).
// POST /api/v1/servers/:server_id/firewall/common-ports
func (fw *FirewallAPI) AllowCommonPorts(c *gin.Context) {
	serverID := c.Param("server_id")

	if err := fw.fwSvc.AllowCommonPorts(c.Request.Context(), serverID); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"status": "common ports opened", "ports": []string{"22/tcp", "80/tcp", "443/tcp", "8080/tcp"}})
}
