package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ========== GHCR Provider Tests ==========

func newGHCRTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Ping endpoint: /v2/
		if r.URL.Path == "/v2/" && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Tags list endpoint: /v2/<owner>/<image>/tags/list
		if r.Method == http.MethodGet && r.URL.Path == "/v2/OWNER/myapp/tags/list" {
			// Verify Bearer token (GitHub token) is present
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test_token_123" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"errors":[{"message":"unauthorized"}]}`))
				return
			}
			json.NewEncoder(w).Encode(tagsResponse{
				Name: "OWNER/myapp",
				Tags: []string{"latest", "v1.0.0", "v1.1.0", "sha-abc123"},
			})
			return
		}

		// Tags list for empty repo
		if r.Method == http.MethodGet && r.URL.Path == "/v2/OWNER/empty/tags/list" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test_token_123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			json.NewEncoder(w).Encode(tagsResponse{
				Name: "OWNER/empty",
				Tags: []string{},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestGHCRName(t *testing.T) {
	g := NewGHCRProvider("", "user", "token")
	if g.Name() != "ghcr" {
		t.Errorf("Name() = %q, want %q", g.Name(), "ghcr")
	}
}

func TestGHCRDefaultURL(t *testing.T) {
	g := NewGHCRProvider("", "user", "token")
	if g.baseURL != defaultGHCRURL {
		t.Errorf("baseURL = %q, want %q", g.baseURL, defaultGHCRURL)
	}
}

func TestGHCRDefaultUsername(t *testing.T) {
	g := NewGHCRProvider("", "", "token")
	if g.username != "OWNER" {
		t.Errorf("username = %q, want %q when empty", g.username, "OWNER")
	}
}

func TestGHCRSetBaseURL(t *testing.T) {
	g := NewGHCRProvider("", "user", "token")
	g.SetBaseURL("http://custom-registry/v2/")
	if g.baseURL != "http://custom-registry/v2/" {
		t.Errorf("baseURL = %q, want %q", g.baseURL, "http://custom-registry/v2/")
	}
}

func TestGHCRPing(t *testing.T) {
	ts := newGHCRTestServer(t)
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "test_token_123")
	ctx := context.Background()

	err := g.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestGHCRPing401(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "token")
	ctx := context.Background()

	err := g.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping() should accept 401 as valid response, got error: %v", err)
	}
}

func TestGHCRPing500(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "token")
	ctx := context.Background()

	err := g.Ping(ctx)
	if err == nil {
		t.Error("Ping() should return error on HTTP 500")
	}
}

func TestGHCRPingConnectionRefused(t *testing.T) {
	g := NewGHCRProvider("http://127.0.0.1:1/v2/", "OWNER", "token")
	ctx := context.Background()

	err := g.Ping(ctx)
	if err == nil {
		t.Error("Ping() should return error when connection is refused")
	}
}

func TestGHCRListTags(t *testing.T) {
	ts := newGHCRTestServer(t)
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "test_token_123")
	ctx := context.Background()

	tags, err := g.ListTags(ctx, "OWNER/myapp")
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if len(tags) != 4 {
		t.Errorf("ListTags() returned %d tags, want 4", len(tags))
	}

	expectedTags := map[string]bool{
		"latest":    false,
		"v1.0.0":    false,
		"v1.1.0":    false,
		"sha-abc123": false,
	}
	for _, tag := range tags {
		if _, ok := expectedTags[tag]; !ok {
			t.Errorf("unexpected tag %q", tag)
		}
		expectedTags[tag] = true
	}
	for tag, found := range expectedTags {
		if !found {
			t.Errorf("missing tag %q", tag)
		}
	}
}

func TestGHCRListTagsEmpty(t *testing.T) {
	ts := newGHCRTestServer(t)
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "test_token_123")
	ctx := context.Background()

	tags, err := g.ListTags(ctx, "OWNER/empty")
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("ListTags() returned %d tags, want 0", len(tags))
	}
}

func TestGHCRListTagsUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"message":"unauthorized"}]}`))
	}))
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "invalid-token")
	ctx := context.Background()

	_, err := g.ListTags(ctx, "OWNER/myapp")
	if err == nil {
		t.Error("ListTags() should return error on 401")
	}
}

func TestGHCRListTagsInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "test_token_123")
	ctx := context.Background()

	_, err := g.ListTags(ctx, "OWNER/myapp")
	if err == nil {
		t.Error("ListTags() should return error on invalid JSON")
	}
}

func TestGHCRListTagsNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "test_token_123")
	ctx := context.Background()

	_, err := g.ListTags(ctx, "OWNER/nonexistent")
	if err == nil {
		t.Error("ListTags() should return error on 404")
	}
}

func TestGHCRListTagsBearerToken(t *testing.T) {
	var capturedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tagsResponse{
			Name: "OWNER/myapp",
			Tags: []string{"latest"},
		})
	}))
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "test_token_123")
	ctx := context.Background()

	_, err := g.ListTags(ctx, "OWNER/myapp")
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}

	if capturedAuth != "Bearer test_token_123" {
		t.Errorf("Authorization = %q, want %q", capturedAuth, "Bearer test_token_123")
	}
}

func TestGHCRPingContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "token")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := g.Ping(ctx)
	if err == nil {
		t.Error("Ping() should return error when context is cancelled")
	}
}

func TestGHCRListTagsContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer ts.Close()

	g := NewGHCRProvider(ts.URL+"/v2/", "OWNER", "token")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.ListTags(ctx, "OWNER/myapp")
	if err == nil {
		t.Error("ListTags() should return error when context is cancelled")
	}
}
