package builder

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// builderMockExecutor is a command executor that returns pre-configured responses.
type builderMockExecutor struct {
	responses map[string]string // command substring -> response
	errs      map[string]error  // command substring -> error
}

func (e *builderMockExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	// Check both responses and errs maps for a matching substring
	for substr := range e.errs {
		if strings.Contains(cmd, substr) {
			resp := e.responses[substr] // may be empty
			return resp, e.errs[substr]
		}
	}
	for substr, resp := range e.responses {
		if strings.Contains(cmd, substr) {
			return resp, nil
		}
	}
	return "", nil
}

var _ deployer.CommandExecutor = (*builderMockExecutor)(nil)

func TestBuilder_GitClone_New(t *testing.T) {
	exec := &builderMockExecutor{
		responses: map[string]string{
			"test -d '/tmp/builds'/.git": "", // directory does not exist
			"git rev-parse HEAD":            "abcdef1234567890",
		},
	}
	b := NewBuilder(exec)
	cfg := BuildConfig{
		RepoURL:    "https://github.com/user/repo.git",
		Branch:     "main",
		AppName:    "myapp",
		ProjectDir: "/tmp/builds",
	}

	hash, err := b.gitClone(context.Background(), cfg)
	if err != nil {
		t.Fatalf("gitClone() error = %v", err)
	}
	if hash != "abcdef1234567890" {
		t.Errorf("gitClone() hash = %q, want %q", hash, "abcdef1234567890")
	}
}

func TestBuilder_GitClone_Existing(t *testing.T) {
	exec := &builderMockExecutor{
		responses: map[string]string{
			"test -d '/tmp/builds'/.git": "exists",
			"git rev-parse HEAD":         "fedcba0987654321",
		},
	}
	b := NewBuilder(exec)
	cfg := BuildConfig{
		RepoURL:    "https://github.com/user/repo.git",
		Branch:     "main",
		AppName:    "myapp",
		ProjectDir: "/tmp/builds",
	}

	hash, err := b.gitClone(context.Background(), cfg)
	if err != nil {
		t.Fatalf("gitClone() error = %v", err)
	}
	if hash != "fedcba0987654321" {
		t.Errorf("gitClone() hash = %q, want %q", hash, "fedcba0987654321")
	}
}

func TestBuilder_GitClone_Error(t *testing.T) {
	exec := &builderMockExecutor{
		responses: map[string]string{},
		errs: map[string]error{
			"git clone": fmt.Errorf("repository not found"),
		},
	}
	b := NewBuilder(exec)
	cfg := BuildConfig{
		RepoURL:    "https://github.com/user/bad.git",
		Branch:     "main",
		AppName:    "badapp",
		ProjectDir: "/tmp/bad-builds",
	}

	_, err := b.gitClone(context.Background(), cfg)
	if err == nil {
		t.Fatal("gitClone() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "git clone failed") {
		t.Errorf("gitClone() error = %v, want git clone failed", err)
	}
}

func TestBuilder_DockerBuild(t *testing.T) {
	exec := &builderMockExecutor{
		responses: map[string]string{
			"docker build": "Successfully built abc123",
		},
	}
	b := NewBuilder(exec)

	log, err := b.dockerBuild(context.Background(), "/tmp/builds", "myapp:v1", nil)
	if err != nil {
		t.Fatalf("dockerBuild() error = %v", err)
	}
	if !strings.Contains(log, "Successfully built") {
		t.Errorf("dockerBuild() log = %q, want to contain 'Successfully built'", log)
	}
}

func TestBuilder_DockerBuild_WithArgs(t *testing.T) {
	exec := &builderMockExecutor{
		responses: map[string]string{
			"docker build": "built with args",
		},
	}
	b := NewBuilder(exec)

	args := map[string]string{
		"NODE_ENV": "production",
		"VERSION":  "1.0",
	}
	log, err := b.dockerBuild(context.Background(), "/tmp/builds", "myapp:v1", args)
	if err != nil {
		t.Fatalf("dockerBuild() error = %v", err)
	}
	if log != "built with args" {
		t.Errorf("dockerBuild() log = %q, want %q", log, "built with args")
	}
}

func TestBuilder_DockerBuild_Error(t *testing.T) {
	exec := &builderMockExecutor{
		responses: map[string]string{
			"docker build": "error: Dockerfile not found",
		},
		errs: map[string]error{
			"docker build": fmt.Errorf("build failed"),
		},
	}
	b := NewBuilder(exec)

	_, err := b.dockerBuild(context.Background(), "/tmp/builds", "myapp:v1", nil)
	if err == nil {
		t.Fatal("dockerBuild() expected error, got nil")
	}
}

func TestBuildAndDeploy_Full(t *testing.T) {
	commitHash := "a1b2c3d4e5f6g7h8i9j0"
	exec := &builderMockExecutor{
		responses: map[string]string{
			"test -d '/tmp/deploypilot-builds/myapp'/.git": "", // new clone
			"git rev-parse HEAD":                              commitHash,
			"test -f /tmp/deploypilot-builds/myapp/go.mod":   "exists",
			"docker build":                                    "Successfully built",
			"docker inspect":                                  "sha256:image123",
		},
	}
	b := NewBuilder(exec)

	cfg := BuildConfig{
		RepoURL: "https://github.com/user/gorepo.git",
		Branch:  "main",
		AppName: "myapp",
	}

	result, err := b.BuildAndDeploy(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildAndDeploy() error = %v", err)
	}

	if result.TechStack != "go" {
		t.Errorf("BuildAndDeploy() TechStack = %q, want %q", result.TechStack, "go")
	}
	if result.CommitHash != commitHash {
		t.Errorf("BuildAndDeploy() CommitHash = %q, want %q", result.CommitHash, commitHash)
	}
	expectedImage := "myapp:" + commitHash[:8]
	if result.Image != expectedImage {
		t.Errorf("BuildAndDeploy() Image = %q, want %q", result.Image, expectedImage)
	}
	if result.Duration <= 0 {
		t.Errorf("BuildAndDeploy() Duration = %v, want > 0", result.Duration)
	}
}

func TestBuildAndDeploy_AutoDetect(t *testing.T) {
	commitHash := "11223344556677889900"
	exec := &builderMockExecutor{
		responses: map[string]string{
			"test -d '/tmp/deploypilot-builds/pyapp'/.git": "", // new clone
			"git rev-parse HEAD":                               commitHash,
			// No specific framework files, but requirements.txt exists
			"test -f /tmp/deploypilot-builds/pyapp/requirements.txt": "exists",
			"docker build": "built ok",
			"docker inspect": "sha256:pyimg",
		},
	}
	b := NewBuilder(exec)

	cfg := BuildConfig{
		RepoURL:  "https://github.com/user/pyrepo.git",
		AppName:  "pyapp",
		TechStack: "auto",
	}

	result, err := b.BuildAndDeploy(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildAndDeploy() error = %v", err)
	}

	if result.TechStack != "python" {
		t.Errorf("BuildAndDeploy() TechStack = %q, want %q", result.TechStack, "python")
	}
}

func TestBuildAndDeploy_Defaults(t *testing.T) {
	commitHash := "ffeeddccbbaa99887766"
	exec := &builderMockExecutor{
		responses: map[string]string{
			"test -d '/tmp/deploypilot-builds/testapp'/.git": "", // new clone
			"git rev-parse HEAD":                                commitHash,
			// No framework files detected -> default to "docker"
			"docker build": "built",
			"docker inspect": "sha256:default",
		},
	}
	b := NewBuilder(exec)

	cfg := BuildConfig{
		RepoURL: "https://github.com/user/custom.git",
		AppName: "testapp",
		// Branch and ProjectDir should default
	}

	result, err := b.BuildAndDeploy(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildAndDeploy() error = %v", err)
	}

	if result.TechStack != "docker" {
		t.Errorf("BuildAndDeploy() TechStack = %q, want %q (default)", result.TechStack, "docker")
	}
}

func TestNewBuilder(t *testing.T) {
	exec := &builderMockExecutor{}
	b := NewBuilder(exec)
	if b == nil {
		t.Fatal("NewBuilder() returned nil")
	}
	if b.registry == nil {
		t.Fatal("NewBuilder() registry is nil")
	}
}

func TestBuilder_GitClone_PullError(t *testing.T) {
	exec := &builderMockExecutor{
		responses: map[string]string{
			"test -d '/tmp/builds'/.git": "exists",
		},
		errs: map[string]error{
			"git pull": fmt.Errorf("pull failed"),
		},
	}
	b := NewBuilder(exec)
	cfg := BuildConfig{
		RepoURL:    "https://github.com/user/repo.git",
		Branch:     "main",
		AppName:    "myapp",
		ProjectDir: "/tmp/builds",
	}

	_, err := b.gitClone(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when git pull fails")
	}
	if !strings.Contains(err.Error(), "git pull failed") {
		t.Errorf("expected 'git pull failed' error, got: %v", err)
	}
}

func TestBuilder_GitClone_RevParseError(t *testing.T) {
	exec := &builderMockExecutor{
		responses: map[string]string{
			"test -d '/tmp/builds'/.git": "",
		},
		errs: map[string]error{
			"git rev-parse HEAD": fmt.Errorf("not a git repo"),
		},
	}
	b := NewBuilder(exec)
	cfg := BuildConfig{
		RepoURL:    "https://github.com/user/repo.git",
		Branch:     "main",
		AppName:    "myapp",
		ProjectDir: "/tmp/builds",
	}

	_, err := b.gitClone(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when rev-parse fails")
	}
}

func TestBuildAndDeploy_DockerfileWriteError(t *testing.T) {
	commitHash := "a1b2c3d4e5f6g7h8i9j0"
	exec := &builderMockExecutor{
		responses: map[string]string{
			"test -d '/tmp/deploypilot-builds/writefail'/.git": "",
			"git rev-parse HEAD":                                  commitHash,
			"test -f /tmp/deploypilot-builds/writefail/go.mod":   "exists",
		},
		errs: map[string]error{
			"cat > '/tmp/deploypilot-builds/writefail'/Dockerfile": fmt.Errorf("permission denied"),
		},
	}
	b := NewBuilder(exec)

	cfg := BuildConfig{
		RepoURL: "https://github.com/user/gorepo.git",
		Branch:  "main",
		AppName: "writefail",
	}

	_, err := b.BuildAndDeploy(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when Dockerfile write fails")
	}
	if !strings.Contains(err.Error(), "failed to write Dockerfile") {
		t.Errorf("expected 'failed to write Dockerfile' error, got: %v", err)
	}
}

func TestBuildAndDeploy_DockerBuildError(t *testing.T) {
	commitHash := "a1b2c3d4e5f6g7h8i9j0"
	exec := &builderMockExecutor{
		responses: map[string]string{
			"test -d '/tmp/deploypilot-builds/buildfail'/.git": "",
			"git rev-parse HEAD":                                 commitHash,
			"docker build":                                      "error: no space left on device",
		},
		errs: map[string]error{
			"docker build": fmt.Errorf("no space left on device"),
		},
	}
	b := NewBuilder(exec)

	cfg := BuildConfig{
		RepoURL: "https://github.com/user/gorepo.git",
		Branch:  "main",
		AppName: "buildfail",
	}

	_, err := b.BuildAndDeploy(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when docker build fails")
	}
	if !strings.Contains(err.Error(), "docker build failed") {
		t.Errorf("expected 'docker build failed' error, got: %v", err)
	}
}

func TestBuildAndDeploy_ExplicitBranchAndProjectDir(t *testing.T) {
	commitHash := "aabbccddeeff00112233"
	exec := &builderMockExecutor{
		responses: map[string]string{
			"test -d '/custom/dir'/.git": "", // new clone
			"git rev-parse HEAD":          commitHash,
			"docker build":                "built",
			"docker inspect":              "sha256:custom",
		},
	}
	b := NewBuilder(exec)

	cfg := BuildConfig{
		RepoURL:    "https://github.com/user/repo.git",
		Branch:     "develop",
		AppName:    "customapp",
		ProjectDir: "/custom/dir",
		TechStack:  "docker",
	}

	result, err := b.BuildAndDeploy(context.Background(), cfg)
	if err != nil {
		t.Fatalf("BuildAndDeploy() error = %v", err)
	}
	if result.TechStack != "docker" {
		t.Errorf("expected tech_stack=docker, got %q", result.TechStack)
	}
	if result.Image != "customapp:aabbccdd" {
		t.Errorf("expected image customapp:aabbccdd, got %q", result.Image)
	}
}
