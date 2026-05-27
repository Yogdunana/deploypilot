package service

import (
	"database/sql"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupOAuth2TestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&model.OAuth2Client{}, &model.OAuth2Authorization{}, &model.OAuth2Token{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestOAuth2Service_CreateClient(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{
		MaxClientsPerUser: 10,
		CodeExpireMinutes: 5,
		TokenExpireHours:  24,
	}
	svc := NewOAuth2Service(db, cfg)

	client, secret, err := svc.CreateClient("user-1", "test-app", []string{"https://example.com/callback"}, []string{"read", "write"}, []string{"authorization_code", "client_credentials"})
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
	if secret == "" {
		t.Fatal("secret is empty")
	}
	if client.Name != "test-app" {
		t.Errorf("Name = %q, want 'test-app'", client.Name)
	}
	if client.ClientID == "" {
		t.Error("ClientID is empty")
	}
}

func TestOAuth2Service_ListClients(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{MaxClientsPerUser: 10, CodeExpireMinutes: 5, TokenExpireHours: 24}
	svc := NewOAuth2Service(db, cfg)

	for i := 0; i < 3; i++ {
		_, _, err := svc.CreateClient("user-1", "app-"+string(rune('A'+i)), nil, []string{"read"}, nil)
		if err != nil {
			t.Fatalf("CreateClient() error = %v", err)
		}
	}
	// Create for another user
	_, _, _ = svc.CreateClient("user-2", "other-app", nil, []string{"read"}, nil)

	clients, err := svc.ListClients("user-1")
	if err != nil {
		t.Fatalf("ListClients() error = %v", err)
	}
	if len(clients) != 3 {
		t.Errorf("len(clients) = %d, want 3", len(clients))
	}
}

func TestOAuth2Service_GetClient(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{MaxClientsPerUser: 10, CodeExpireMinutes: 5, TokenExpireHours: 24}
	svc := NewOAuth2Service(db, cfg)

	client, _, err := svc.CreateClient("user-1", "test-app", nil, []string{"read"}, nil)
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	// Owner can get
	gotClient, err := svc.GetClient(client.ID, "user-1")
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}
	if gotClient.ID != client.ID {
		t.Errorf("ID mismatch: got %q, want %q", gotClient.ID, client.ID)
	}

	// Other user cannot get
	_, err = svc.GetClient(client.ID, "user-2")
	if err != sql.ErrNoRows {
		t.Error("GetClient() should fail for non-owner")
	}
}

func TestOAuth2Service_UpdateClient(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{MaxClientsPerUser: 10, CodeExpireMinutes: 5, TokenExpireHours: 24}
	svc := NewOAuth2Service(db, cfg)

	client, _, err := svc.CreateClient("user-1", "old-name", nil, []string{"read"}, nil)
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	updates := map[string]interface{}{
		"name": "new-name",
	}
	err = svc.UpdateClient(client.ID, "user-1", updates)
	if err != nil {
		t.Fatalf("UpdateClient() error = %v", err)
	}

	updatedClient, err := svc.GetClient(client.ID, "user-1")
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}
	if updatedClient.Name != "new-name" {
		t.Errorf("Name = %q, want 'new-name'", updatedClient.Name)
	}
}

func TestOAuth2Service_DeleteClient(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{MaxClientsPerUser: 10, CodeExpireMinutes: 5, TokenExpireHours: 24}
	svc := NewOAuth2Service(db, cfg)

	client, _, err := svc.CreateClient("user-1", "to-delete", nil, []string{"read"}, nil)
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	err = svc.DeleteClient(client.ID, "user-1")
	if err != nil {
		t.Fatalf("DeleteClient() error = %v", err)
	}

	_, err = svc.GetClient(client.ID, "user-1")
	if err != sql.ErrNoRows {
		t.Error("GetClient() should return ErrNoRows after deletion")
	}
}

func TestOAuth2Service_RegenerateSecret(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{MaxClientsPerUser: 10, CodeExpireMinutes: 5, TokenExpireHours: 24}
	svc := NewOAuth2Service(db, cfg)

	client, originalSecret, err := svc.CreateClient("user-1", "test-app", nil, []string{"read"}, nil)
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	newSecret, err := svc.RegenerateSecret(client.ID, "user-1")
	if err != nil {
		t.Fatalf("RegenerateSecret() error = %v", err)
	}
	if newSecret == originalSecret {
		t.Error("new secret should be different from original")
	}
}

func TestOAuth2Service_ClientCredentials(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{MaxClientsPerUser: 10, CodeExpireMinutes: 5, TokenExpireHours: 24}
	svc := NewOAuth2Service(db, cfg)

	client, secret, err := svc.CreateClient("user-1", "test-app", nil, []string{"read", "write"}, []string{"client_credentials"})
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	token, err := svc.ClientCredentials(client.ClientID, secret, []string{"read"})
	if err != nil {
		t.Fatalf("ClientCredentials() error = %v", err)
	}
	if token == nil {
		t.Fatal("token is nil")
	}
	if token.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
}

func TestOAuth2Service_RefreshToken(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{MaxClientsPerUser: 10, CodeExpireMinutes: 5, TokenExpireHours: 24}
	svc := NewOAuth2Service(db, cfg)

	client, secret, err := svc.CreateClient("user-1", "test-app", nil, []string{"read"}, []string{"client_credentials", "refresh_token"})
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	token, err := svc.ClientCredentials(client.ClientID, secret, nil)
	if err != nil {
		t.Fatalf("ClientCredentials() error = %v", err)
	}

	refreshed, err := svc.RefreshToken(token.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if refreshed.AccessToken == token.AccessToken {
		t.Error("refreshed access token should be different")
	}
}

func TestOAuth2Service_RevokeToken(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{MaxClientsPerUser: 10, CodeExpireMinutes: 5, TokenExpireHours: 24}
	svc := NewOAuth2Service(db, cfg)

	client, secret, err := svc.CreateClient("user-1", "test-app", nil, []string{"read"}, []string{"client_credentials"})
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	token, err := svc.ClientCredentials(client.ClientID, secret, nil)
	if err != nil {
		t.Fatalf("ClientCredentials() error = %v", err)
	}

	err = svc.RevokeToken(token.AccessToken)
	if err != nil {
		t.Fatalf("RevokeToken() error = %v", err)
	}

	_, err = svc.ValidateAccessToken(token.AccessToken)
	if err == nil {
		t.Error("ValidateAccessToken() should fail for revoked token")
	}
}

func TestOAuth2Service_CreateAuthorization(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{MaxClientsPerUser: 10, CodeExpireMinutes: 5, TokenExpireHours: 24}
	svc := NewOAuth2Service(db, cfg)

	client, _, err := svc.CreateClient("user-1", "test-app", []string{"https://example.com/callback"}, []string{"read"}, []string{"authorization_code"})
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	authz, err := svc.CreateAuthorization(client.ClientID, "user-1", []string{"read"})
	if err != nil {
		t.Fatalf("CreateAuthorization() error = %v", err)
	}
	if authz.Code == "" {
		t.Error("Code is empty")
	}
}

func TestOAuth2Service_ExchangeCode(t *testing.T) {
	db := setupOAuth2TestDB(t)
	cfg := &config.APIPlatformConfig{MaxClientsPerUser: 10, CodeExpireMinutes: 5, TokenExpireHours: 24}
	svc := NewOAuth2Service(db, cfg)

	client, _, err := svc.CreateClient("user-1", "test-app", []string{"https://example.com/callback"}, []string{"read"}, []string{"authorization_code"})
	if err != nil {
		t.Fatalf("CreateClient() error = %v", err)
	}

	authz, err := svc.CreateAuthorization(client.ClientID, "user-1", []string{"read"})
	if err != nil {
		t.Fatalf("CreateAuthorization() error = %v", err)
	}

	token, err := svc.ExchangeCode(authz.Code)
	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	if token == nil {
		t.Fatal("token is nil")
	}
}

func TestValidateScopes(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"valid scopes", []string{"read", "write"}, []string{"read", "write"}},
		{"invalid scopes", []string{"read", "invalid"}, []string{"read"}},
		{"empty", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateScopes(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("validateScopes() len = %d, want %d", len(got), len(tt.want))
			}
		})
	}
}

func TestValidateGrantTypes(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"valid types", []string{"authorization_code", "client_credentials"}, []string{"authorization_code", "client_credentials"}},
		{"invalid type", []string{"authorization_code", "password"}, []string{"authorization_code"}},
		{"empty", nil, []string{"authorization_code"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateGrantTypes(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("validateGrantTypes() len = %d, want %d", len(got), len(tt.want))
			}
		})
	}
}

func TestIntersectScopes(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{"overlap", []string{"read", "write"}, []string{"write", "delete"}, []string{"write"}},
		{"no overlap", []string{"read"}, []string{"write"}, nil},
		{"empty", nil, []string{"read"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intersectScopes(tt.a, tt.b)
			if len(got) != len(tt.want) {
				t.Errorf("intersectScopes() len = %d, want %d", len(got), len(tt.want))
			}
		})
	}
}
