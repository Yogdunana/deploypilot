package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/Yogdunana/deploypilot/internal/config"
	"github.com/Yogdunana/deploypilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupOAuth2TestAPI(t *testing.T) (*OAuth2API, *gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.OAuth2Client{}, &model.OAuth2Authorization{}, &model.OAuth2Token{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	testUser := &model.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
	}
	db.Create(testUser)

	cfg := &config.APIPlatformConfig{
		MaxClientsPerUser: 10,
		CodeExpireMinutes: 5,
		TokenExpireHours:  24,
	}
	api := NewOAuth2API(db, cfg)

	r := gin.New()
	apiGroup := r.Group("/api/v1")
	{
		apiGroup.Use(func(c *gin.Context) {
			c.Set("user_id", "1")
			c.Next()
		})
		oauthGroup := apiGroup.Group("/oauth")
		{
			oauthGroup.GET("/clients", api.ListClients)
			oauthGroup.POST("/clients", api.CreateClient)
			oauthGroup.GET("/clients/:id", api.GetClient)
			oauthGroup.PUT("/clients/:id", api.UpdateClient)
			oauthGroup.DELETE("/clients/:id", api.DeleteClient)
			oauthGroup.POST("/clients/:id/secret", api.RegenerateSecret)
			oauthGroup.POST("/authorize", api.Authorize)
			oauthGroup.POST("/token", api.Token)
			oauthGroup.POST("/token/refresh", api.RefreshToken)
			oauthGroup.POST("/token/revoke", api.RevokeToken)
		}
	}

	return api, r, db
}

func TestOAuth2CreateClient(t *testing.T) {
	_, r, db := setupOAuth2TestAPI(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	reqBody := map[string]interface{}{
		"name":          "test-app",
		"redirect_uris": []string{"https://example.com/callback"},
		"scopes":        []string{"read", "write"},
		"grant_types":   []string{"authorization_code", "client_credentials"},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/oauth/clients", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])
	assert.NotNil(t, response["data"])
	assert.NotNil(t, response["client_secret"])
}

func TestOAuth2ListClients(t *testing.T) {
	_, r, db := setupOAuth2TestAPI(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/oauth/clients", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
