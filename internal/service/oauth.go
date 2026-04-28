package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// OAuthUserInfo represents user information from an OAuth provider.
type OAuthUserInfo struct {
	ID        string
	Username  string
	Email     string
	AvatarURL string
}

// OAuthProvider defines the unified interface for OAuth providers.
type OAuthProvider interface {
	Name() string
	Endpoint() oauth2.Endpoint
	DefaultScopes() []string
	UserInfoURL() string
	ParseUserInfo(body []byte) (*OAuthUserInfo, error)
}

// OAuthService handles OAuth2 authentication flows.
type OAuthService struct {
	db        *gorm.DB
	configs   map[string]*oauth2.Config
	providers map[string]OAuthProvider
}

// NewOAuthService creates a new OAuth service with the given provider configurations.
func NewOAuthService(db *gorm.DB, providers []config.OAuthProviderConfig) *OAuthService {
	svc := &OAuthService{
		db:        db,
		configs:   make(map[string]*oauth2.Config),
		providers: make(map[string]OAuthProvider),
	}

	// Register built-in providers
	builtinProviders := map[string]OAuthProvider{
		"github": NewGithubOAuthProvider(),
		"gitee":  NewGiteeOAuthProvider(),
	}
	for name, p := range builtinProviders {
		svc.providers[name] = p
	}

	// Build configs from provider configurations
	for _, p := range providers {
		provider, ok := svc.providers[p.Provider]
		if !ok {
			slog.Warn("unknown OAuth provider, skipping", "provider", p.Provider)
			continue
		}
		scopes := p.Scopes
		if len(scopes) == 0 {
			scopes = provider.DefaultScopes()
		}
		svc.configs[p.Provider] = &oauth2.Config{
			ClientID:     p.ClientID,
			ClientSecret: p.ClientSecret,
			RedirectURL:  p.RedirectURL,
			Scopes:       scopes,
			Endpoint:     provider.Endpoint(),
		}
	}
	return svc
}

// IsProviderConfigured returns true if the given OAuth provider is configured.
func (s *OAuthService) IsProviderConfigured(provider string) bool {
	_, ok := s.configs[provider]
	return ok
}

// AuthURL returns the OAuth authorization URL for the given provider.
func (s *OAuthService) AuthURL(provider, state string) (string, error) {
	cfg, ok := s.configs[provider]
	if !ok {
		return "", fmt.Errorf("OAuth provider %q is not configured", provider)
	}
	return cfg.AuthCodeURL(state), nil
}

// HandleCallback processes the OAuth callback and returns the user and role name.
func (s *OAuthService) HandleCallback(ctx context.Context, provider, code string) (*model.User, string, error) {
	cfg, ok := s.configs[provider]
	if !ok {
		return nil, "", fmt.Errorf("OAuth provider %q is not configured", provider)
	}

	// Exchange code for token
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange OAuth code: %w", err)
	}

	// Get user info from provider
	userInfo, err := s.getUserInfo(ctx, provider, token.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get OAuth user info: %w", err)
	}

	// Find or create user
	var user model.User
	err = s.db.Where("auth_provider = ? AND auth_uid = ?", provider, userInfo.ID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new user
			user = model.User{
				ID:           uuid.New().String(),
				TenantID:     "tenant-default",
				RoleID:       "role-viewer",
				Username:     userInfo.Username,
				Email:        userInfo.Email,
				AuthProvider: provider,
				AuthUID:      userInfo.ID,
				AvatarURL:    userInfo.AvatarURL,
			}
			// Generate random password for OAuth users (they won't use it)
			randomPwd := uuid.New().String()
			hash, hashErr := crypto.HashPassword(randomPwd)
			if hashErr != nil {
				return nil, "", fmt.Errorf("failed to hash password for OAuth user: %w", hashErr)
			}
			user.PasswordHash = hash

			if createErr := s.db.Create(&user).Error; createErr != nil {
				return nil, "", fmt.Errorf("failed to create OAuth user: %w", createErr)
			}
		} else {
			return nil, "", fmt.Errorf("failed to query user: %w", err)
		}
	}

	// Determine role name
	roleName := "viewer"
	var role model.Role
	if err := s.db.Where("id = ?", user.RoleID).First(&role).Error; err == nil {
		roleName = role.Name
	}

	return &user, roleName, nil
}

func (s *OAuthService) getUserInfo(ctx context.Context, provider, accessToken string) (*OAuthUserInfo, error) {
	p, ok := s.providers[provider]
	if !ok {
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", p.UserInfoURL(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OAuth user info request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return p.ParseUserInfo(body)
}
