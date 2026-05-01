package api

import (
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
)

// SSHAPI provides HTTP handlers for SSH key pair and authorization management.
type SSHAPI struct {
	sshSvc *service.SSHService
}

// NewSSHAPI creates a new SSHAPI.
func NewSSHAPI(db *gorm.DB) *SSHAPI {
	return &SSHAPI{
		sshSvc: service.NewSSHService(db),
	}
}

// GenerateKeyPair generates a new RSA SSH key pair.
// POST /api/v1/ssh/keys/generate
func (s *SSHAPI) GenerateKeyPair(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Bits int    `json:"bits"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	keyPair, err := s.sshSvc.GenerateKeyPair(req.Name, req.Bits)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, keyPair)
}

// ImportPublicKey imports an existing public key.
// POST /api/v1/ssh/keys/import
func (s *SSHAPI) ImportPublicKey(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		PublicKey string `json:"public_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	keyPair, err := s.sshSvc.ImportPublicKey(req.Name, req.PublicKey)
	if err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	respondSuccess(c, keyPair)
}

// ListKeyPairs returns all stored SSH key pairs.
// GET /api/v1/ssh/keys
func (s *SSHAPI) ListKeyPairs(c *gin.Context) {
	keys, err := s.sshSvc.ListKeyPairs()
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	respondSuccess(c, keys)
}

// GetKeyPair returns a single key pair by ID.
// GET /api/v1/ssh/keys/:id
func (s *SSHAPI) GetKeyPair(c *gin.Context) {
	id := c.Param("id")
	keyPair, err := s.sshSvc.GetKeyPair(id)
	if err != nil {
		respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
		return
	}
	respondSuccess(c, keyPair)
}

// DeleteKeyPair deletes a key pair and all its authorizations.
// DELETE /api/v1/ssh/keys/:id
func (s *SSHAPI) DeleteKeyPair(c *gin.Context) {
	id := c.Param("id")
	if err := s.sshSvc.DeleteKeyPair(id); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	respondSuccess(c, gin.H{"deleted": true})
}

// AuthorizeKey deploys a public key to a server.
// POST /api/v1/ssh/authorize
func (s *SSHAPI) AuthorizeKey(c *gin.Context) {
	var req struct {
		KeyPairID string `json:"key_pair_id" binding:"required"`
		ServerID  string `json:"server_id" binding:"required"`
		User      string `json:"user"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := s.sshSvc.AuthorizeKey(c.Request.Context(), req.KeyPairID, req.ServerID, req.User); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"key_pair_id": req.KeyPairID, "server_id": req.ServerID, "user": req.User, "status": "authorized"})
}

// RevokeKey removes a public key from a server.
// POST /api/v1/ssh/revoke
func (s *SSHAPI) RevokeKey(c *gin.Context) {
	var req struct {
		KeyPairID string `json:"key_pair_id" binding:"required"`
		ServerID  string `json:"server_id" binding:"required"`
		User      string `json:"user"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	if err := s.sshSvc.RevokeKey(c.Request.Context(), req.KeyPairID, req.ServerID, req.User); err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"key_pair_id": req.KeyPairID, "server_id": req.ServerID, "status": "revoked"})
}

// ListServerAuthorizations returns all authorizations for a server.
// GET /api/v1/servers/:server_id/ssh/authorizations
func (s *SSHAPI) ListServerAuthorizations(c *gin.Context) {
	serverID := c.Param("server_id")
	auths, err := s.sshSvc.ListAuthorizations(serverID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	respondSuccess(c, auths)
}

// ListKeyAuthorizations returns all servers a key is authorized on.
// GET /api/v1/ssh/keys/:id/authorizations
func (s *SSHAPI) ListKeyAuthorizations(c *gin.Context) {
	id := c.Param("id")
	auths, err := s.sshSvc.ListAuthorizationsByKey(id)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}
	respondSuccess(c, auths)
}
