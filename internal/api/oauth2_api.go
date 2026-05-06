package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/Yogdunana/deploypilot/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// globalOAuth2API is the package-level OAuth2API instance, accessible via GetGlobalOAuth2API().
var globalOAuth2API *OAuth2API

// GetGlobalOAuth2API returns the global OAuth2API instance.
func GetGlobalOAuth2API() *OAuth2API { return globalOAuth2API }

// OAuth2API provides HTTP handlers for OAuth2 client and token management.
type OAuth2API struct {
	db  *gorm.DB
	cfg *config.APIPlatformConfig
}

// NewOAuth2API creates a new OAuth2API.
func NewOAuth2API(db *gorm.DB, cfg *config.APIPlatformConfig) *OAuth2API {
	return &OAuth2API{db: db, cfg: cfg}
}

// createClientRequest is the request body for creating an OAuth2 client.
type createClientRequest struct {
	Name         string   `json:"name" binding:"required"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	GrantTypes   []string `json:"grant_types"`
}

// updateClientRequest is the request body for updating an OAuth2 client.
type updateClientRequest struct {
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	GrantTypes   []string `json:"grant_types"`
	Enabled      *bool    `json:"enabled"`
}

// tokenRequest is the OAuth2 token request body (RFC 6749).
type tokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type"`
	Code         string `json:"code" form:"code"`
	RedirectURI  string `json:"redirect_uri" form:"redirect_uri"`
	ClientID     string `json:"client_id" form:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret"`
	Scope        string `json:"scope" form:"scope"`
	RefreshToken string `json:"refresh_token" form:"refresh_token"`
}

// authorizeRequest is the request body for starting an authorization code flow.
type authorizeRequest struct {
	ClientID string   `json:"client_id" binding:"required"`
	Scopes   []string `json:"scopes"`
}

// ListClients lists all OAuth2 client applications for the authenticated user.
// GET /api/v1/oauth/clients
func (a *OAuth2API) ListClients(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.unauthorized")
		return
	}

	svc := service.NewOAuth2Service(a.db, a.cfg)
	clients, err := svc.ListClients(userID)
	if err != nil {
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	// Sanitize: remove client_secret from response
	sanitized := make([]map[string]interface{}, 0, len(clients))
	for _, cl := range clients {
		sanitized = append(sanitized, map[string]interface{}{
			"id":            cl.ID,
			"user_id":       cl.UserID,
			"name":          cl.Name,
			"client_id":     cl.ClientID,
			"redirect_uris": cl.RedirectURIs,
			"scopes":        cl.Scopes,
			"grant_types":   cl.GrantTypes,
			"enabled":       cl.Enabled,
			"created_at":    cl.CreatedAt,
			"updated_at":    cl.UpdatedAt,
		})
	}

	respondSuccess(c, sanitized)
}

// CreateClient creates a new OAuth2 client application.
// POST /api/v1/oauth/clients
func (a *OAuth2API) CreateClient(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.unauthorized")
		return
	}

	var req createClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	svc := service.NewOAuth2Service(a.db, a.cfg)
	client, secret, err := svc.CreateClient(userID, req.Name, req.RedirectURIs, req.Scopes, req.GrantTypes)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid request")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":         "success",
		"data":           client,
		"client_secret":  secret,
		"message":        "save the client_secret now, it will not be shown again",
	})
}

// GetClient returns a single OAuth2 client by ID.
// GET /api/v1/oauth/clients/:id
func (a *OAuth2API) GetClient(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.unauthorized")
		return
	}

	id := c.Param("id")
	svc := service.NewOAuth2Service(a.db, a.cfg)
	client, err := svc.GetClient(id, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, client)
}

// UpdateClient updates an OAuth2 client's metadata.
// PUT /api/v1/oauth/clients/:id
func (a *OAuth2API) UpdateClient(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.unauthorized")
		return
	}

	id := c.Param("id")

	var req updateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.RedirectURIs != nil {
		urisJSON, err := json.Marshal(req.RedirectURIs)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}
		updates["redirect_uris"] = string(urisJSON)
	}
	if req.Scopes != nil {
		scopesJSON, err := json.Marshal(req.Scopes)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}
		updates["scopes"] = string(scopesJSON)
	}
	if req.GrantTypes != nil {
		grantTypesJSON, err := json.Marshal(req.GrantTypes)
		if err != nil {
			respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
			return
		}
		updates["grant_types"] = string(grantTypesJSON)
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if len(updates) == 0 {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	svc := service.NewOAuth2Service(a.db, a.cfg)
	if err := svc.UpdateClient(id, userID, updates); err != nil {
		if err == sql.ErrNoRows {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"id": id})
}

// DeleteClient deletes an OAuth2 client by ID.
// DELETE /api/v1/oauth/clients/:id
func (a *OAuth2API) DeleteClient(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.unauthorized")
		return
	}

	id := c.Param("id")
	svc := service.NewOAuth2Service(a.db, a.cfg)
	if err := svc.DeleteClient(id, userID); err != nil {
		if err == sql.ErrNoRows {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	respondSuccess(c, gin.H{"id": id})
}

// RegenerateSecret generates a new client_secret for an existing client.
// POST /api/v1/oauth/clients/:id/secret
func (a *OAuth2API) RegenerateSecret(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.unauthorized")
		return
	}

	id := c.Param("id")
	svc := service.NewOAuth2Service(a.db, a.cfg)
	newSecret, err := svc.RegenerateSecret(id, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondErrori18n(c, http.StatusNotFound, "error.common.not_found")
			return
		}
		respondErrori18n(c, http.StatusInternalServerError, "error.common.internal_error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"client_secret": newSecret,
		"message":       "save the new client_secret now, it will not be shown again",
	})
}

// Authorize starts an authorization code flow and returns the authorization URL with code.
// POST /api/v1/oauth/authorize
func (a *OAuth2API) Authorize(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		respondErrori18n(c, http.StatusUnauthorized, "error.auth.unauthorized")
		return
	}

	var req authorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	svc := service.NewOAuth2Service(a.db, a.cfg)
	authz, err := svc.CreateAuthorization(req.ClientID, userID, req.Scopes)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid request")
		return
	}

	respondSuccess(c, gin.H{
		"code":       authz.Code,
		"expires_at": authz.ExpiresAt,
	})
}

// Token handles OAuth2 token requests (RFC 6749 compliant).
// Supports grant_type: authorization_code, client_credentials, refresh_token.
// POST /api/v1/oauth/token
// This endpoint does NOT require auth middleware — it authenticates via client_id/client_secret.
func (a *OAuth2API) Token(c *gin.Context) {
	// Try Basic Auth first, then fall back to POST body
	clientID, clientSecret := extractClientCredentials(c)

	var req tokenRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "invalid or missing parameters",
		})
		return
	}

	// POST body overrides Basic Auth if present
	if req.ClientID != "" {
		clientID = req.ClientID
	}
	if req.ClientSecret != "" {
		clientSecret = req.ClientSecret
	}

	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_client",
			"error_description": "client_id is required",
		})
		return
	}

	svc := service.NewOAuth2Service(a.db, a.cfg)

	switch req.GrantType {
	case "authorization_code":
		if req.Code == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "code is required for authorization_code grant",
			})
			return
		}
		token, err := svc.ExchangeCode(req.Code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_grant",
				"error_description": "internal server error",
			})
			return
		}
		respondOAuth2Token(c, token)

	case "client_credentials":
		if clientSecret == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_client",
				"error_description": "client_secret is required for client_credentials grant",
			})
			return
		}
		var scopes []string
		if req.Scope != "" {
			scopes = splitScopes(req.Scope)
		}
		token, err := svc.ClientCredentials(clientID, clientSecret, scopes)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_client",
				"error_description": "internal server error",
			})
			return
		}
		respondOAuth2Token(c, token)

	case "refresh_token":
		if req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "refresh_token is required for refresh_token grant",
			})
			return
		}
		token, err := svc.RefreshToken(req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_grant",
				"error_description": "internal server error",
			})
			return
		}
		respondOAuth2Token(c, token)

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "unsupported_grant_type",
			"error_description": "grant_type must be authorization_code, client_credentials, or refresh_token",
		})
	}
}

// RefreshToken refreshes an access token using a refresh token.
// POST /api/v1/oauth/token/refresh
func (a *OAuth2API) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	svc := service.NewOAuth2Service(a.db, a.cfg)
	token, err := svc.RefreshToken(req.RefreshToken)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid request")
		return
	}

	respondSuccess(c, token)
}

// RevokeToken revokes an OAuth2 access token.
// POST /api/v1/oauth/token/revoke
func (a *OAuth2API) RevokeToken(c *gin.Context) {
	var req struct {
		AccessToken string `json:"access_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrori18n(c, http.StatusBadRequest, "error.common.invalid_request")
		return
	}

	svc := service.NewOAuth2Service(a.db, a.cfg)
	if err := svc.RevokeToken(req.AccessToken); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request")
		return
	}

	respondSuccess(c, gin.H{"revoked": true})
}

// respondOAuth2Token returns an OAuth2-compliant token response (RFC 6749).
func respondOAuth2Token(c *gin.Context, token *model.OAuth2Token) {
	expiresIn := int(time.Until(token.ExpiresAt).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  token.AccessToken,
		"token_type":    token.TokenType,
		"expires_in":    expiresIn,
		"refresh_token": token.RefreshToken,
		"scope":         token.Scopes,
	})
}

// extractClientCredentials extracts client_id and client_secret from Basic Auth header.
func extractClientCredentials(c *gin.Context) (clientID, clientSecret string) {
	clientID, clientSecret, ok := c.Request.BasicAuth()
	if !ok {
		return "", ""
	}
	return clientID, clientSecret
}

// splitScopes splits a space-separated scope string into a slice.
func splitScopes(scope string) []string {
	if scope == "" {
		return nil
	}
	var scopes []string
	seen := make(map[string]bool)
	for _, s := range splitBySpace(scope) {
		if s != "" && !seen[s] {
			seen[s] = true
			scopes = append(scopes, s)
		}
	}
	return scopes
}

// splitBySpace splits a string by spaces.
func splitBySpace(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}
