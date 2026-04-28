package service

import (
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
)

type giteeOAuthProvider struct{}

// NewGiteeOAuthProvider creates a new Gitee OAuth provider.
func NewGiteeOAuthProvider() OAuthProvider {
	return &giteeOAuthProvider{}
}

func (p *giteeOAuthProvider) Name() string { return "gitee" }
func (p *giteeOAuthProvider) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  "https://gitee.com/oauth/authorize",
		TokenURL: "https://gitee.com/oauth/token",
	}
}
func (p *giteeOAuthProvider) DefaultScopes() []string { return []string{"read:user", "user:email"} }
func (p *giteeOAuthProvider) UserInfoURL() string { return "https://gitee.com/api/v5/user" }

func (p *giteeOAuthProvider) ParseUserInfo(body []byte) (*OAuthUserInfo, error) {
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
}
