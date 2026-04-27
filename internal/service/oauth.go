package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/crypto"
	"github.com/Yogdunana/deploypilot/internal/model"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"gorm.io/gorm"
)

// OAuthUserInfo represents user information from an OAuth provider.
type OAuthUserInfo struct {
	ID        string
	Username  string
	Email     string
	AvatarURL string
}

// OAuthService handles OAuth2 authentication flows.
type OAuthService struct {
	db      *gorm.DB
	configs map[string]*oauth2.Config
}

// NewOAuthService creates a new OAuth service with the given provider configurations.
func NewOAuthService(db *gorm.DB, providers []config.OAuthProviderConfig) *OAuthService {
	svc := &OAuthService{
		db:      db,
		configs: make(map[string]*oauth2.Config),
	}
	for _, p := range providers {
		var endpoint oauth2.Endpoint
		switch p.Provider {
		case "github":
			endpoint = github.Endpoint
		case "gitee":
			endpoint = oauth2.Endpoint{
				AuthURL:  "https://gitee.com/oauth/authorize",
				TokenURL: "https://gitee.com/oauth/token",
			}
		default:
			slog.Warn("unknown OAuth provider, skipping", "provider", p.Provider)
			continue
		}
		scopes := p.Scopes
		if len(scopes) == 0 {
			scopes = []string{"read:user", "user:email"}
		}
		svc.configs[p.Provider] = &oauth2.Config{
			ClientID:     p.ClientID,
			ClientSecret: p.ClientSecret,
			RedirectURL:  p.RedirectURL,
			Scopes:       scopes,
			Endpoint:     endpoint,
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
	client := &http.Client{}
	var userInfoURL string
	switch provider {
	case "github":
		userInfoURL = "https://api.github.com/user"
	case "gitee":
		userInfoURL = "https://gitee.com/api/v5/user"
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
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

	switch provider {
	case "github":
		var gh struct {
			ID        int    `json:"id"`
			Login     string `json:"login"`
			Email     string `json:"email"`
			AvatarURL string `json:"avatar_url"`
		}
		if err := json.Unmarshal(body, &gh); err != nil {
			return nil, fmt.Errorf("failed to parse GitHub user info: %w", err)
		}
		return &OAuthUserInfo{
			ID:        fmt.Sprintf("%d", gh.ID),
			Username:  gh.Login,
			Email:     gh.Email,
			AvatarURL: gh.AvatarURL,
		}, nil
	case "gitee":
		var ge struct {
			ID        int    `json:"id"`
			Login     string `json:"login"`
			Email     string `json:"email"`
			AvatarURL string `json:"avatar_url"`
		}
		if err := json.Unmarshal(body, &ge); err != nil {
			return nil, fmt.Errorf("failed to parse Gitee user info: %w", err)
		}
		return &OAuthUserInfo{
			ID:        fmt.Sprintf("%d", ge.ID),
			Username:  ge.Login,
			Email:     ge.Email,
			AvatarURL: ge.AvatarURL,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}
