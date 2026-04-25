package builder

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// mockDockerRunner records calls and returns pre-configured errors.
type mockDockerRunner struct {
	calls  []mockDockerCall
	runErr map[string]error // substring of args -> error to return
}

type mockDockerCall struct {
	name string
	args []string
}

func (m *mockDockerRunner) Run(ctx context.Context, name string, args ...string) error {
	m.calls = append(m.calls, mockDockerCall{name: name, args: args})
	for substr, err := range m.runErr {
		for _, a := range args {
			if strings.Contains(a, substr) {
				return err
			}
		}
	}
	return nil
}

func (m *mockDockerRunner) RunWithStdin(ctx context.Context, name string, stdin io.Reader, args ...string) error {
	m.calls = append(m.calls, mockDockerCall{name: name, args: args})
	for substr, err := range m.runErr {
		for _, a := range args {
			if strings.Contains(a, substr) {
				return err
			}
		}
	}
	return nil
}

func (m *mockDockerRunner) callCount() int {
	return len(m.calls)
}

func (m *mockDockerRunner) nthCall(i int) mockDockerCall {
	if i < 0 || i >= len(m.calls) {
		return mockDockerCall{}
	}
	return m.calls[i]
}

var _ dockerRunner = (*mockDockerRunner)(nil)

// TestPushImage_NoRegistryConfig tests that PushImage returns an error
// when no registry is configured.
func TestPushImage_NoRegistryConfig(t *testing.T) {
	exec := &builderMockExecutor{}
	runner := &mockDockerRunner{}
	b := NewBuilderWithDocker(exec, runner)

	cfg := &BuildConfig{
		AppName: "myapp",
	}

	ref, err := b.PushImage(context.Background(), cfg, "myapp:abc12345")
	if err == nil {
		t.Fatal("expected error when no registry is configured, got nil")
	}
	if ref != "" {
		t.Errorf("expected empty ref, got %q", ref)
	}
	if !strings.Contains(err.Error(), "no registry configured") {
		t.Errorf("expected 'no registry configured' error, got: %v", err)
	}
	// No docker commands should have been executed
	if runner.callCount() != 0 {
		t.Errorf("expected 0 docker calls, got %d", runner.callCount())
	}
}

// TestPushImage_DirectRegistryURL tests pushing with a directly configured registry URL.
func TestPushImage_DirectRegistryURL(t *testing.T) {
	exec := &builderMockExecutor{}
	runner := &mockDockerRunner{}
	b := NewBuilderWithDocker(exec, runner)

	cfg := &BuildConfig{
		AppName:     "myapp",
		RegistryURL: "https://registry.example.com",
		RegistryUser: "myuser",
		RegistryPass: "mypass",
	}

	ref, err := b.PushImage(context.Background(), cfg, "myapp:abc12345")
	if err != nil {
		t.Fatalf("PushImage() error = %v", err)
	}
	if ref == "" {
		t.Fatal("expected non-empty remote ref")
	}
	if !strings.Contains(ref, "registry.example.com") {
		t.Errorf("expected ref to contain registry URL, got %q", ref)
	}
	if !strings.Contains(ref, "myapp:latest") {
		t.Errorf("expected ref to contain myapp:latest, got %q", ref)
	}

	// Should have: docker tag, docker login, docker push
	if runner.callCount() != 3 {
		t.Fatalf("expected 3 docker calls, got %d", runner.callCount())
	}

	// First call: docker tag
	call0 := runner.nthCall(0)
	if call0.name != "docker" || call0.args[0] != "tag" {
		t.Errorf("expected 'docker tag' call, got %s %v", call0.name, call0.args)
	}

	// Second call: docker login with --password-stdin
	call1 := runner.nthCall(1)
	if call1.name != "docker" || call1.args[0] != "login" {
		t.Errorf("expected 'docker login' call, got %s %v", call1.name, call1.args)
	}
	hasPasswordStdin := false
	for _, a := range call1.args {
		if a == "--password-stdin" {
			hasPasswordStdin = true
		}
	}
	if !hasPasswordStdin {
		t.Error("expected --password-stdin flag in docker login call")
	}

	// Third call: docker push
	call2 := runner.nthCall(2)
	if call2.name != "docker" || call2.args[0] != "push" {
		t.Errorf("expected 'docker push' call, got %s %v", call2.name, call2.args)
	}
	if call2.args[1] != ref {
		t.Errorf("expected push ref %q, got %q", ref, call2.args[1])
	}
}

// TestPushImage_DirectRegistryURL_CustomTag tests pushing with a custom image tag.
func TestPushImage_DirectRegistryURL_CustomTag(t *testing.T) {
	exec := &builderMockExecutor{}
	runner := &mockDockerRunner{}
	b := NewBuilderWithDocker(exec, runner)

	cfg := &BuildConfig{
		AppName:     "myapp",
		RegistryURL: "https://registry.example.com",
		RegistryUser: "myuser",
		RegistryPass: "mypass",
		ImageTag:    "v1.2.3",
	}

	ref, err := b.PushImage(context.Background(), cfg, "myapp:abc12345")
	if err != nil {
		t.Fatalf("PushImage() error = %v", err)
	}
	if !strings.HasSuffix(ref, ":v1.2.3") {
		t.Errorf("expected ref to end with :v1.2.3, got %q", ref)
	}
}

// TestPushImage_DockerHub tests tag format for Docker Hub.
func TestPushImage_DockerHub(t *testing.T) {
	exec := &builderMockExecutor{}
	runner := &mockDockerRunner{}
	b := NewBuilderWithDocker(exec, runner)

	cfg := &BuildConfig{
		AppName:     "myuser",
		RegistryURL: "https://registry.hub.docker.com/v2/",
		RegistryUser: "myuser",
		RegistryPass: "mypass",
	}

	ref, err := b.PushImage(context.Background(), cfg, "myuser:abc12345")
	if err != nil {
		t.Fatalf("PushImage() error = %v", err)
	}
	// Docker Hub format: myuser/myuser:latest
	if !strings.HasPrefix(ref, "myuser/") {
		t.Errorf("expected Docker Hub format 'myuser/...', got %q", ref)
	}
}

// TestPushImage_GHCR tests tag format for GitHub Container Registry.
func TestPushImage_GHCR(t *testing.T) {
	exec := &builderMockExecutor{}
	runner := &mockDockerRunner{}
	b := NewBuilderWithDocker(exec, runner)

	cfg := &BuildConfig{
		AppName:     "myowner/myapp",
		RegistryURL: "https://ghcr.io",
		RegistryUser: "myowner",
		RegistryPass: "ghp_token",
	}

	ref, err := b.PushImage(context.Background(), cfg, "myapp:abc12345")
	if err != nil {
		t.Fatalf("PushImage() error = %v", err)
	}
	if !strings.HasPrefix(ref, "ghcr.io/") {
		t.Errorf("expected GHCR format 'ghcr.io/...', got %q", ref)
	}
}

// TestPushImage_TagFailure tests error propagation when docker tag fails.
func TestPushImage_TagFailure(t *testing.T) {
	exec := &builderMockExecutor{}
	runner := &mockDockerRunner{
		runErr: map[string]error{
			"tag": errors.New("no such image"),
		},
	}
	b := NewBuilderWithDocker(exec, runner)

	cfg := &BuildConfig{
		AppName:     "myapp",
		RegistryURL: "https://registry.example.com",
		RegistryUser: "myuser",
		RegistryPass: "mypass",
	}

	ref, err := b.PushImage(context.Background(), cfg, "nonexistent:abc12345")
	if err == nil {
		t.Fatal("expected error when docker tag fails, got nil")
	}
	if !strings.Contains(err.Error(), "docker tag failed") {
		t.Errorf("expected 'docker tag failed' error, got: %v", err)
	}
	if ref != "" {
		t.Errorf("expected empty ref on error, got %q", ref)
	}
	// Only tag should have been attempted
	if runner.callCount() != 1 {
		t.Errorf("expected 1 docker call, got %d", runner.callCount())
	}
}

// TestPushImage_LoginFailure tests error propagation when docker login fails.
func TestPushImage_LoginFailure(t *testing.T) {
	exec := &builderMockExecutor{}
	runner := &mockDockerRunner{
		runErr: map[string]error{
			"login": errors.New("authentication failed"),
		},
	}
	b := NewBuilderWithDocker(exec, runner)

	cfg := &BuildConfig{
		AppName:     "myapp",
		RegistryURL: "https://registry.example.com",
		RegistryUser: "myuser",
		RegistryPass: "wrongpass",
	}

	ref, err := b.PushImage(context.Background(), cfg, "myapp:abc12345")
	if err == nil {
		t.Fatal("expected error when docker login fails, got nil")
	}
	if !strings.Contains(err.Error(), "docker login failed") {
		t.Errorf("expected 'docker login failed' error, got: %v", err)
	}
	// tag + login should have been attempted
	if runner.callCount() != 2 {
		t.Errorf("expected 2 docker calls, got %d", runner.callCount())
	}
}

// TestPushImage_PushFailure tests error propagation when docker push fails.
func TestPushImage_PushFailure(t *testing.T) {
	exec := &builderMockExecutor{}
	runner := &mockDockerRunner{
		runErr: map[string]error{
			"push": errors.New("denied: access forbidden"),
		},
	}
	b := NewBuilderWithDocker(exec, runner)

	cfg := &BuildConfig{
		AppName:     "myapp",
		RegistryURL: "https://registry.example.com",
		RegistryUser: "myuser",
		RegistryPass: "mypass",
	}

	ref, err := b.PushImage(context.Background(), cfg, "myapp:abc12345")
	if err == nil {
		t.Fatal("expected error when docker push fails, got nil")
	}
	if !strings.Contains(err.Error(), "docker push failed") {
		t.Errorf("expected 'docker push failed' error, got: %v", err)
	}
	// tag + login + push should have been attempted
	if runner.callCount() != 3 {
		t.Errorf("expected 3 docker calls, got %d", runner.callCount())
	}
}

// TestPushImage_NoUsername tests that login is skipped when username is empty.
func TestPushImage_NoUsername(t *testing.T) {
	exec := &builderMockExecutor{}
	runner := &mockDockerRunner{}
	b := NewBuilderWithDocker(exec, runner)

	cfg := &BuildConfig{
		AppName:     "myapp",
		RegistryURL: "https://registry.example.com",
		// No username/password — anonymous push
	}

	ref, err := b.PushImage(context.Background(), cfg, "myapp:abc12345")
	if err != nil {
		t.Fatalf("PushImage() error = %v", err)
	}
	if ref == "" {
		t.Fatal("expected non-empty ref")
	}
	// Should have: docker tag, docker push (no login)
	if runner.callCount() != 2 {
		t.Fatalf("expected 2 docker calls (no login), got %d", runner.callCount())
	}
	// Verify no login call
	for _, c := range runner.calls {
		if c.name == "docker" && len(c.args) > 0 && c.args[0] == "login" {
			t.Error("expected no docker login call when username is empty")
		}
	}
}

// TestBuildRemoteTag tests the buildRemoteTag function.
func TestBuildRemoteTag(t *testing.T) {
	tests := []struct {
		name    string
		regURL  string
		appName string
		tag     string
		want    string
	}{
		{
			name:    "docker hub registry-1.docker.io",
			regURL:  "https://registry-1.docker.io",
			appName: "myuser",
			tag:     "latest",
			want:    "myuser/myuser:latest",
		},
		{
			name:    "docker hub docker.io",
			regURL:  "https://docker.io",
			appName: "myuser",
			tag:     "v1",
			want:    "myuser/myuser:v1",
		},
		{
			name:    "ghcr.io",
			regURL:  "https://ghcr.io",
			appName: "myowner/myapp",
			tag:     "latest",
			want:    "ghcr.io/myowner/myapp:latest",
		},
		{
			name:    "custom registry",
			regURL:  "https://registry.example.com",
			appName: "myapp",
			tag:     "latest",
			want:    "registry.example.com/myapp:latest",
		},
		{
			name:    "custom registry with v2 suffix",
			regURL:  "https://registry.example.com/v2/",
			appName: "myapp",
			tag:     "v2.0",
			want:    "registry.example.com/myapp:v2.0",
		},
		{
			name:    "harbor registry",
			regURL:  "https://harbor.mycompany.com",
			appName: "project/myapp",
			tag:     "dev",
			want:    "harbor.mycompany.com/project/myapp:dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRemoteTag(tt.regURL, tt.appName, tt.tag)
			if got != tt.want {
				t.Errorf("buildRemoteTag() = %q, want %q", got, tt.want)
			}
		})
	}
}
