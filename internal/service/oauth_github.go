package service

import (
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type githubOAuthProvider struct{}

// NewGithubOAuthProvider creates a new GitHub OAuth provider.
func NewGithubOAuthProvider() OAuthProvider {
	return &githubOAuthProvider{}
}

func (p *githubOAuthProvider) Name() string { return "github" }
func (p *githubOAuthProvider) Endpoint() oauth2.Endpoint { return github.Endpoint }
func (p *githubOAuthProvider) DefaultScopes() []string { return []string{"read:user", "user:email"} }
func (p *githubOAuthProvider) UserInfoURL() string { return "https://api.github.com/user" }

func (p *githubOAuthProvider) ParseUserInfo(body []byte) (*OAuthUserInfo, error) {
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
}
