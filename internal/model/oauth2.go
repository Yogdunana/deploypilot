package model

import "time"

// OAuth2Client represents a registered OAuth2 client application.
type OAuth2Client struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	TenantID     string    `gorm:"index" json:"tenant_id"`
	UserID       string    `gorm:"index" json:"user_id"`
	Name         string    `gorm:"not null;size:100" json:"name"`
	ClientID     string    `gorm:"uniqueIndex;not null;size:40" json:"client_id"`
	ClientSecret string    `gorm:"not null;size:100" json:"-"`
	RedirectURIs string    `gorm:"type:text" json:"redirect_uris"` // JSON array of URIs
	Scopes       string    `gorm:"type:text" json:"scopes"`        // JSON array of scopes
	GrantTypes   string    `gorm:"type:text" json:"grant_types"`   // JSON array of grant types
	Enabled      bool      `gorm:"default:true" json:"enabled"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Tenant Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	User   User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (OAuth2Client) TableName() string { return "oauth2_clients" }

// OAuth2Authorization represents an authorization code grant.
type OAuth2Authorization struct {
	ID        string     `gorm:"primaryKey" json:"id"`
	ClientID  string     `gorm:"index;not null;size:40" json:"client_id"`
	UserID    string     `gorm:"index;not null" json:"user_id"`
	Scopes    string     `gorm:"type:text" json:"scopes"` // JSON array of scopes
	Code      string     `gorm:"uniqueIndex;not null;size:100" json:"-"`
	ExpiresAt time.Time  `gorm:"index" json:"expires_at"`
	Used      bool       `gorm:"default:false" json:"used"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (OAuth2Authorization) TableName() string { return "oauth2_authorizations" }

// OAuth2Token represents an issued OAuth2 access/refresh token pair.
type OAuth2Token struct {
	ID           string     `gorm:"primaryKey" json:"id"`
	ClientID     string     `gorm:"index;not null;size:40" json:"client_id"`
	UserID       string     `gorm:"index;not null" json:"user_id"`
	AccessToken  string     `gorm:"uniqueIndex;not null;size:100" json:"access_token"`
	RefreshToken string     `gorm:"uniqueIndex;not null;size:100" json:"refresh_token"`
	Scopes       string     `gorm:"type:text" json:"scopes"` // JSON array of scopes
	TokenType    string     `gorm:"size:20;default:bearer" json:"token_type"`
	ExpiresAt    time.Time  `gorm:"index" json:"expires_at"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (OAuth2Token) TableName() string { return "oauth2_tokens" }
