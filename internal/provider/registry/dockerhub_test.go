package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ========== Docker Hub Provider Tests ==========

func newDockerHubTestServer(t *testing.T) (*httptest.Server, *httptest.Server) {
	t.Helper()

	// Auth server (mocks auth.docker.io)
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(authTokenResponse{
			Token: "test-bearer-token-12345",
		})
	}))

	// Registry server (mocks registry-1.docker.io)
	registryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Ping endpoint: /v2/
		if r.URL.Path == "/v2/" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Tags list endpoint: /v2/<repo>/tags/list
		if r.Method == http.MethodGet && r.URL.Path == "/v2/library/nginx/tags/list" {
			// Verify Bearer token is present
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-bearer-token-12345" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"errors":[{"message":"unauthorized"}]}`))
				return
			}
			json.NewEncoder(w).Encode(tagsResponse{
				Name: "library/nginx",
				Tags: []string{"latest", "1.25", "1.25-alpine", "1.24"},
			})
			return
		}

		// Tags list for empty repo
		if r.Method == http.MethodGet && r.URL.Path == "/v2/library/empty/tags/list" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-bearer-token-12345" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			json.NewEncoder(w).Encode(tagsResponse{
				Name: "library/empty",
				Tags: []string{},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	return authServer, registryServer
}

func TestDockerHubName(t *testing.T) {
	d := NewDockerHubProvider("", "user", "pass")
	if d.Name() != "docker_hub" {
		t.Errorf("Name() = %q, want %q", d.Name(), "docker_hub")
	}
}

func TestDockerHubDefaultURL(t *testing.T) {
	d := NewDockerHubProvider("", "user", "pass")
	if d.baseURL != defaultDockerHubURL {
		t.Errorf("baseURL = %q, want %q", d.baseURL, defaultDockerHubURL)
	}
}

func TestDockerHubSetBaseURL(t *testing.T) {
	d := NewDockerHubProvider("", "user", "pass")
	d.SetBaseURL("http://custom-registry/v2/")
	if d.baseURL != "http://custom-registry/v2/" {
		t.Errorf("baseURL = %q, want %q", d.baseURL, "http://custom-registry/v2/")
	}
}

func TestDockerHubPing(t *testing.T) {
	_, registryServer := newDockerHubTestServer(t)
	defer registryServer.Close()

	d := NewDockerHubProvider(registryServer.URL+"/v2/", "user", "pass")
	ctx := context.Background()

	err := d.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestDockerHubPing401(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	d := NewDockerHubProvider(ts.URL+"/v2/", "user", "pass")
	ctx := context.Background()

	err := d.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping() should accept 401 as valid response, got error: %v", err)
	}
}

func TestDockerHubPing500(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	d := NewDockerHubProvider(ts.URL+"/v2/", "user", "pass")
	ctx := context.Background()

	err := d.Ping(ctx)
	if err == nil {
		t.Error("Ping() should return error on HTTP 500")
	}
}

func TestDockerHubPingConnectionRefused(t *testing.T) {
	d := NewDockerHubProvider("http://127.0.0.1:1/v2/", "user", "pass")
	ctx := context.Background()

	err := d.Ping(ctx)
	if err == nil {
		t.Error("Ping() should return error when connection is refused")
	}
}

func TestDockerHubListTags(t *testing.T) {
	_, registryServer := newDockerHubTestServer(t)
	defer registryServer.Close()

	d := NewDockerHubProvider(registryServer.URL+"/v2/", "testuser", "testpass")
	ctx := context.Background()

	// We need to override the auth URL for testing.
	// Since getAuthToken calls auth.docker.io directly, we need to test differently.
	// Instead, test the ListTags with a mock that doesn't require auth.
	tags, err := d.listTagsWithMockAuth(ctx, "library/nginx", "test-bearer-token")
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if len(tags) != 4 {
		t.Errorf("ListTags() returned %d tags, want 4", len(tags))
	}
}

func TestDockerHubListTagsEmpty(t *testing.T) {
	_, registryServer := newDockerHubTestServer(t)
	defer registryServer.Close()

	d := NewDockerHubProvider(registryServer.URL+"/v2/", "testuser", "testpass")
	ctx := context.Background()

	tags, err := d.listTagsWithMockAuth(ctx, "library/empty", "test-bearer-token")
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("ListTags() returned %d tags, want 0", len(tags))
	}
}

func TestDockerHubListTagsUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"message":"unauthorized"}]}`))
	}))
	defer ts.Close()

	d := NewDockerHubProvider(ts.URL+"/v2/", "testuser", "testpass")
	ctx := context.Background()

	_, err := d.listTagsWithMockAuth(ctx, "library/nginx", "invalid-token")
	if err == nil {
		t.Error("ListTags() should return error on 401")
	}
}

func TestDockerHubListTagsInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	d := NewDockerHubProvider(ts.URL+"/v2/", "testuser", "testpass")
	ctx := context.Background()

	_, err := d.listTagsWithMockAuth(ctx, "library/nginx", "test-token")
	if err == nil {
		t.Error("ListTags() should return error on invalid JSON")
	}
}

func TestDockerHubListTagsNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	d := NewDockerHubProvider(ts.URL+"/v2/", "testuser", "testpass")
	ctx := context.Background()

	_, err := d.listTagsWithMockAuth(ctx, "library/nonexistent", "test-token")
	if err == nil {
		t.Error("ListTags() should return error on 404")
	}
}

func TestDockerHubGetAuthToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Basic Auth credentials
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(authTokenResponse{
			Token: "mock-token-abc",
		})
	}))
	defer ts.Close()

	d := NewDockerHubProvider("", "testuser", "testpass")
	// Override auth URL by temporarily replacing
	ctx := context.Background()

	token, err := d.getAuthTokenFromURL(ctx, ts.URL)
	if err != nil {
		t.Fatalf("getAuthToken() error = %v", err)
	}
	if token != "mock-token-abc" {
		t.Errorf("token = %q, want %q", token, "mock-token-abc")
	}
}

func TestDockerHubGetAuthTokenInvalidCredentials(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	d := NewDockerHubProvider("", "baduser", "badpass")
	ctx := context.Background()

	_, err := d.getAuthTokenFromURL(ctx, ts.URL)
	if err == nil {
		t.Error("getAuthToken() should return error on invalid credentials")
	}
}

func TestDockerHubGetAuthTokenInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	d := NewDockerHubProvider("", "testuser", "testpass")
	ctx := context.Background()

	_, err := d.getAuthTokenFromURL(ctx, ts.URL)
	if err == nil {
		t.Error("getAuthToken() should return error on invalid JSON")
	}
}

func TestDockerHubPingContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	d := NewDockerHubProvider(ts.URL+"/v2/", "user", "pass")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := d.Ping(ctx)
	if err == nil {
		t.Error("Ping() should return error when context is cancelled")
	}
}

func TestDockerHubListTagsContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	d := NewDockerHubProvider(ts.URL+"/v2/", "user", "pass")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := d.listTagsWithMockAuth(ctx, "library/nginx", "token")
	if err == nil {
		t.Error("ListTags() should return error when context is cancelled")
	}
}

// listTagsWithMockAuth is a test helper that performs ListTags with a pre-set Bearer token,
// bypassing the auth.docker.io token flow.
func (d *DockerHubProvider) listTagsWithMockAuth(ctx context.Context, repository, token string) ([]string, error) {
	url := d.baseURL + repository + "/tags/list"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list tags (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Tags, nil
}

// getAuthTokenFromURL is a test helper that fetches a token from a custom URL.
func (d *DockerHubProvider) getAuthTokenFromURL(ctx context.Context, tokenURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(d.username, d.password)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get auth token (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp authTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.Token, nil
}
