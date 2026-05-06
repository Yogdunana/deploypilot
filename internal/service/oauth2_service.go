package service

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/gorm"
)

// validOAuth2Scopes is the set of valid scopes for OAuth2 clients.
// Duplicated from auth package to avoid import cycle (auth → service → auth).
var validOAuth2Scopes = map[string]bool{
	"read": true, "write": true, "delete": true, "deploy": true, "admin": true,
	"monitor:read": true, "monitor:write": true,
	"server:read": true, "server:exec": true,
	"credential:read": true, "credential:write": true,
	"dns:write": true, "ssl:write": true,
	"backup:read": true, "backup:write": true,
	"webhook:manage": true, "grafana:manage": true,
}

// validateScopes filters a list of scopes to only include valid ones.
func validateScopes(scopes []string) []string {
	valid := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if validOAuth2Scopes[s] {
			valid = append(valid, s)
		}
	}
	return valid
}

// OAuth2Service handles OAuth2 client management and token operations.
type OAuth2Service struct {
	db  *gorm.DB
	cfg *config.APIPlatformConfig
}

// NewOAuth2Service creates a new OAuth2Service.
func NewOAuth2Service(db *gorm.DB, cfg *config.APIPlatformConfig) *OAuth2Service {
	return &OAuth2Service{db: db, cfg: cfg}
}

// CreateClient creates a new OAuth2 client application.
// Returns the client model and the raw client_secret (shown only once).
func (s *OAuth2Service) CreateClient(userID, name string, redirectURIs, scopes, grantTypes []string) (*model.OAuth2Client, string, error) {
	// Enforce max clients per user
	if s.cfg.MaxClientsPerUser > 0 {
		var count int64
		if err := s.db.Model(&model.OAuth2Client{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
			return nil, "", fmt.Errorf("failed to count client applications: %w", err)
		}
		if int(count) >= s.cfg.MaxClientsPerUser {
			return nil, "", fmt.Errorf("maximum number of client applications (%d) reached", s.cfg.MaxClientsPerUser)
		}
	}

	// Validate scopes
	validScopes := validateScopes(scopes)

	// Validate grant types
	validGrantTypes := validateGrantTypes(grantTypes)

	clientID := generateClientID()
	clientSecret := generateClientSecret()

	redirectURIsJSON, err := json.Marshal(redirectURIs)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal redirect URIs: %w", err)
	}

	scopesJSON, err := json.Marshal(validScopes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal scopes: %w", err)
	}

	grantTypesJSON, err := json.Marshal(validGrantTypes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal grant types: %w", err)
	}

	client := &model.OAuth2Client{
		ID:           generateOAuth2ID(),
		UserID:       userID,
		Name:         name,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURIs: string(redirectURIsJSON),
		Scopes:       string(scopesJSON),
		GrantTypes:   string(grantTypesJSON),
		Enabled:      true,
	}

	if err := s.db.Create(client).Error; err != nil {
		return nil, "", fmt.Errorf("failed to create OAuth2 client: %w", err)
	}

	slog.Info("OAuth2 client created", "id", client.ID, "client_id", clientID, "name", name, "user_id", userID)
	return client, clientSecret, nil
}

// ListClients returns all OAuth2 clients for a given user.
func (s *OAuth2Service) ListClients(userID string) ([]model.OAuth2Client, error) {
	var clients []model.OAuth2Client
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&clients).Error; err != nil {
		return nil, fmt.Errorf("failed to list OAuth2 clients: %w", err)
	}
	return clients, nil
}

// GetClient returns a single OAuth2 client by ID. Only the owner can access.
func (s *OAuth2Service) GetClient(id, userID string) (*model.OAuth2Client, error) {
	var client model.OAuth2Client
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get OAuth2 client: %w", err)
	}
	return &client, nil
}

// UpdateClient modifies an OAuth2 client's metadata.
func (s *OAuth2Service) UpdateClient(id, userID string, updates map[string]interface{}) error {
	result := s.db.Model(&model.OAuth2Client{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update OAuth2 client: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return sql.ErrNoRows
	}
	slog.Info("OAuth2 client updated", "id", id, "user_id", userID)
	return nil
}

// DeleteClient removes an OAuth2 client by ID. Only the owner can delete.
func (s *OAuth2Service) DeleteClient(id, userID string) error {
	result := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.OAuth2Client{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete OAuth2 client: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return sql.ErrNoRows
	}
	slog.Info("OAuth2 client deleted", "id", id, "user_id", userID)
	return nil
}

// RegenerateSecret generates a new client_secret for an existing client.
// Returns the new secret (shown only once).
func (s *OAuth2Service) RegenerateSecret(id, userID string) (string, error) {
	var client model.OAuth2Client
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", sql.ErrNoRows
		}
		return "", fmt.Errorf("failed to get OAuth2 client: %w", err)
	}

	newSecret := generateClientSecret()
	if err := s.db.Model(&client).Update("client_secret", newSecret).Error; err != nil {
		return "", fmt.Errorf("failed to regenerate client secret: %w", err)
	}

	slog.Info("OAuth2 client secret regenerated", "id", id, "user_id", userID)
	return newSecret, nil
}

// CreateAuthorization creates an authorization code for the authorization code grant flow.
func (s *OAuth2Service) CreateAuthorization(clientID, userID string, scopes []string) (*model.OAuth2Authorization, error) {
	// Validate client exists and is enabled
	var client model.OAuth2Client
	if err := s.db.Where("client_id = ? AND enabled = ?", clientID, true).First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid or disabled client")
		}
		return nil, fmt.Errorf("failed to validate client: %w", err)
	}

	// Validate scopes against client's registered scopes
	clientScopes := ParseScopes(client.Scopes)
	validScopes := validateScopes(scopes)
	requestedScopes := intersectScopes(validScopes, clientScopes)
	if len(requestedScopes) == 0 {
		return nil, fmt.Errorf("no valid scopes requested")
	}

	scopesJSON, err := json.Marshal(requestedScopes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scopes: %w", err)
	}

	code := generateToken()
	expiresAt := time.Now().Add(time.Duration(s.cfg.CodeExpireMinutes) * time.Minute)

	authz := &model.OAuth2Authorization{
		ID:        generateOAuth2ID(),
		ClientID:  clientID,
		UserID:    userID,
		Scopes:    string(scopesJSON),
		Code:      code,
		ExpiresAt: expiresAt,
		Used:      false,
	}

	if err := s.db.Create(authz).Error; err != nil {
		return nil, fmt.Errorf("failed to create authorization: %w", err)
	}

	return authz, nil
}

// ExchangeCode exchanges an authorization code for an access/refresh token pair.
func (s *OAuth2Service) ExchangeCode(code string) (*model.OAuth2Token, error) {
	var authz model.OAuth2Authorization
	if err := s.db.Where("code = ? AND used = ?", code, false).First(&authz).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid or expired authorization code")
		}
		return nil, fmt.Errorf("failed to lookup authorization code: %w", err)
	}

	// Check expiration
	if authz.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("authorization code expired")
	}

	// Mark code as used
	if err := s.db.Model(&authz).Update("used", true).Error; err != nil {
		return nil, fmt.Errorf("failed to invalidate authorization code: %w", err)
	}

	// Create token
	scopes := ParseScopes(authz.Scopes)
	token, err := s.createToken(authz.ClientID, authz.UserID, scopes)
	if err != nil {
		return nil, err
	}

	return token, nil
}

// ClientCredentials handles the client_credentials grant type.
func (s *OAuth2Service) ClientCredentials(clientID, clientSecret string, scopes []string) (*model.OAuth2Token, error) {
	// Validate client
	var client model.OAuth2Client
	if err := s.db.Where("client_id = ? AND enabled = ?", clientID, true).First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid or disabled client")
		}
		return nil, fmt.Errorf("failed to validate client: %w", err)
	}

	// Validate client secret using constant-time comparison
	if subtle.ConstantTimeCompare([]byte(client.ClientSecret), []byte(clientSecret)) != 1 {
		return nil, fmt.Errorf("invalid client credentials")
	}

	// Validate grant type includes client_credentials
	grantTypes := parseGrantTypes(client.GrantTypes)
	if !containsString(grantTypes, "client_credentials") {
		return nil, fmt.Errorf("client does not support client_credentials grant type")
	}

	// Validate scopes against client's registered scopes
	clientScopes := ParseScopes(client.Scopes)
	validScopes := validateScopes(scopes)
	requestedScopes := intersectScopes(validScopes, clientScopes)
	if len(requestedScopes) == 0 {
		requestedScopes = clientScopes // use client defaults if none requested
	}

	token, err := s.createToken(clientID, client.UserID, requestedScopes)
	if err != nil {
		return nil, err
	}

	return token, nil
}

// RefreshToken exchanges a refresh token for a new access/refresh token pair.
func (s *OAuth2Service) RefreshToken(refreshToken string) (*model.OAuth2Token, error) {
	var existing model.OAuth2Token
	if err := s.db.Where("refresh_token = ?", refreshToken).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid refresh token")
		}
		return nil, fmt.Errorf("failed to lookup refresh token: %w", err)
	}

	// Delete old token
	if err := s.db.Delete(&existing).Error; err != nil {
		return nil, fmt.Errorf("failed to revoke old token: %w", err)
	}

	// Create new token with same scopes
	scopes := ParseScopes(existing.Scopes)
	token, err := s.createToken(existing.ClientID, existing.UserID, scopes)
	if err != nil {
		return nil, err
	}

	return token, nil
}

// RevokeToken deletes an access token, effectively revoking it.
func (s *OAuth2Service) RevokeToken(accessToken string) error {
	result := s.db.Where("access_token = ?", accessToken).Delete(&model.OAuth2Token{})
	if result.Error != nil {
		return fmt.Errorf("failed to revoke token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("token not found")
	}
	slog.Info("OAuth2 token revoked", "access_token_prefix", accessToken[:min(8, len(accessToken))])
	return nil
}

// ValidateAccessToken looks up an access token in the database and returns the token record.
func (s *OAuth2Service) ValidateAccessToken(token string) (*model.OAuth2Token, error) {
	var oauthToken model.OAuth2Token
	if err := s.db.Where("access_token = ?", token).First(&oauthToken).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid access token")
		}
		return nil, fmt.Errorf("failed to validate access token: %w", err)
	}

	// Check expiration
	if oauthToken.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("access token expired")
	}

	return &oauthToken, nil
}

// createToken creates a new access/refresh token pair.
func (s *OAuth2Service) createToken(clientID, userID string, scopes []string) (*model.OAuth2Token, error) {
	accessToken := generateToken()
	refreshToken := generateToken()

	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scopes: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(s.cfg.TokenExpireHours) * time.Hour)

	token := &model.OAuth2Token{
		ID:           generateOAuth2ID(),
		ClientID:     clientID,
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Scopes:       string(scopesJSON),
		TokenType:    "bearer",
		ExpiresAt:    expiresAt,
	}

	if err := s.db.Create(token).Error; err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	slog.Info("OAuth2 token created", "id", token.ID, "client_id", clientID, "user_id", userID)
	return token, nil
}

// generateToken generates a random 32-byte hex string.
func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		slog.Error("failed to generate token", "error", err)
		panic("failed to generate token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// generateClientID generates a random 16-byte hex string prefixed with "dp_".
func generateClientID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		slog.Error("failed to generate client ID", "error", err)
		panic("failed to generate client ID: " + err.Error())
	}
	return "dp_" + hex.EncodeToString(b)
}

// generateClientSecret generates a random 32-byte hex string.
func generateClientSecret() string {
	return generateToken()
}

// generateOAuth2ID generates a random hex ID for OAuth2 records.
func generateOAuth2ID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate OAuth2 ID: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// validateGrantTypes filters grant types to only allowed values.
func validateGrantTypes(grantTypes []string) []string {
	allowed := map[string]bool{
		"authorization_code": true,
		"client_credentials": true,
		"refresh_token":      true,
	}
	valid := make([]string, 0, len(grantTypes))
	for _, gt := range grantTypes {
		if allowed[gt] {
			valid = append(valid, gt)
		}
	}
	if len(valid) == 0 {
		return []string{"authorization_code"}
	}
	return valid
}

// parseGrantTypes parses the JSON grant types string into a slice.
func parseGrantTypes(grantTypesJSON string) []string {
	var grantTypes []string
	if err := json.Unmarshal([]byte(grantTypesJSON), &grantTypes); err != nil {
		return nil
	}
	return grantTypes
}

// intersectScopes returns the intersection of two scope slices.
func intersectScopes(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, s := range b {
		set[s] = true
	}
	result := make([]string, 0, len(a))
	for _, s := range a {
		if set[s] {
			result = append(result, s)
		}
	}
	return result
}

// containsString checks if a string slice contains a given string.
func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}

// min returns the smaller of two integers (Go 1.18 compat).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
