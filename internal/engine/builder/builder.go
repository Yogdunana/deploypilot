package builder

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// BuildConfig holds parameters for a build operation.
type BuildConfig struct {
	RepoURL             string            // Git repository URL
	Branch              string            // Git branch (default: main)
	TechStack           string            // Template type (auto-detected if empty)
	AppName             string            // Application name (used for image tag)
	ProjectDir          string            // Local path for git clone (default: /tmp/deploypilot-builds/<app-name>)
	BuildArgs           map[string]string // Additional docker build args
	EnvVars             map[string]string // Runtime environment variables
	Ports               string            // Port mappings (e.g. "8080:80")
	ServerID            string            // Target server ID (deploy locally if empty)
	DockerfileOverrides map[string]string // Override template placeholders

	// Registry configuration for pushing the built image
	RegistryID   string `json:"registry_id,omitempty"`   // ID of the registry to push to
	RegistryURL  string `json:"registry_url,omitempty"`   // Override registry URL
	RegistryUser string `json:"registry_user,omitempty"`  // Override registry username
	RegistryPass string `json:"registry_pass,omitempty"`  // Override registry password
	PushImage    bool   `json:"push_image,omitempty"`     // Whether to push after build
	ImageTag     string `json:"image_tag,omitempty"`      // Custom image tag (default: appname:latest)
}

// BuildResult holds the result of a build operation.
type BuildResult struct {
	Image       string  `json:"image"`
	PushedImage string  `json:"pushed_image,omitempty"`
	Digest      string  `json:"digest,omitempty"`
	Size        string  `json:"size,omitempty"`
	BuildLog    string  `json:"build_log"`
	Duration    float64 `json:"duration_seconds"`
	TechStack   string  `json:"tech_stack"`
	CommitHash  string  `json:"commit_hash"`
}

// Builder orchestrates the build process.
type Builder struct {
	executor    deployer.CommandExecutor
	registry    *TemplateRegistry
	dockerRunner dockerRunner
}

// NewBuilder creates a new Builder.
func NewBuilder(executor deployer.CommandExecutor) *Builder {
	return &Builder{
		executor: executor,
		registry: NewRegistry(),
	}
}

// NewBuilderWithDocker creates a new Builder with a custom dockerRunner for testing.
func NewBuilderWithDocker(executor deployer.CommandExecutor, runner dockerRunner) *Builder {
	return &Builder{
		executor:    executor,
		registry:    NewRegistry(),
		dockerRunner: runner,
	}
}

// BuildAndDeploy executes the full build-and-deploy pipeline.
func (b *Builder) BuildAndDeploy(ctx context.Context, cfg BuildConfig) (*BuildResult, error) {
	start := time.Now()

	// 1. Prepare project directory
	if cfg.ProjectDir == "" {
		cfg.ProjectDir = fmt.Sprintf("/tmp/deploypilot-builds/%s", cfg.AppName)
	}
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}

	// 2. Git clone (or pull if exists)
	commitHash, err := b.gitClone(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("git clone failed: %w", err)
	}

	// 3. Detect tech stack if not specified
	if cfg.TechStack == "" || cfg.TechStack == "auto" {
		cfg.TechStack = DetectTechStack(b.executor, cfg.ProjectDir)
	}

	// 4. Generate Dockerfile from template
	tmpl, err := b.registry.FindByType(cfg.TechStack)
	if err != nil {
		tmpl = b.registry.Get(TemplateDocker) // fallback to custom Dockerfile
	}
	dockerfile := tmpl.GenerateDockerfile(cfg.DockerfileOverrides)

	// Write Dockerfile to project dir
	_, err = b.executor.RunCommand(ctx, fmt.Sprintf("cat > %s/Dockerfile << 'DEPLOYPilot_EOF'\n%s\nDEPLOYPilot_EOF", cfg.ProjectDir, dockerfile))
	if err != nil {
		return nil, fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// 5. Docker build
	imageTag := fmt.Sprintf("%s:%s", cfg.AppName, commitHash[:8])
	buildLog, err := b.dockerBuild(ctx, cfg.ProjectDir, imageTag, cfg.BuildArgs)
	if err != nil {
		return nil, fmt.Errorf("docker build failed: %w\n%s", err, buildLog)
	}

	// 6. Get image digest
	digest, _ := b.executor.RunCommand(ctx, fmt.Sprintf("docker inspect --format='{{.Id}}' %s 2>/dev/null", imageTag))

	duration := time.Since(start).Seconds()

	result := &BuildResult{
		Image:      imageTag,
		Digest:     strings.TrimSpace(digest),
		BuildLog:   buildLog,
		Duration:   duration,
		TechStack:  cfg.TechStack,
		CommitHash: commitHash,
	}

	// 7. Optionally push image to registry
	if cfg.PushImage {
		pushedRef, pushErr := b.PushImage(ctx, &cfg, imageTag)
		if pushErr != nil {
			log.Printf("warning: image push failed (build succeeded): %v", pushErr)
		} else if pushedRef != "" {
			result.PushedImage = pushedRef
		}
	}

	return result, nil
}

// gitClone clones or pulls a git repository.
func (b *Builder) gitClone(ctx context.Context, cfg BuildConfig) (string, error) {
	// Check if directory exists
	output, _ := b.executor.RunCommand(ctx, fmt.Sprintf("test -d %s/.git && echo 'exists'", cfg.ProjectDir))

	if strings.TrimSpace(output) == "exists" {
		// Pull latest
		_, err := b.executor.RunCommand(ctx, fmt.Sprintf("cd %s && git fetch origin && git checkout %s && git pull origin %s", cfg.ProjectDir, cfg.Branch, cfg.Branch))
		if err != nil {
			return "", fmt.Errorf("git pull failed: %w", err)
		}
	} else {
		// Clone
		_, err := b.executor.RunCommand(ctx, fmt.Sprintf("mkdir -p %s && git clone --branch %s --depth 1 %s %s", cfg.ProjectDir, cfg.Branch, cfg.RepoURL, cfg.ProjectDir))
		if err != nil {
			return "", fmt.Errorf("git clone failed: %w", err)
		}
	}

	// Get commit hash
	hash, err := b.executor.RunCommand(ctx, fmt.Sprintf("cd %s && git rev-parse HEAD", cfg.ProjectDir))
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return strings.TrimSpace(hash), nil
}

// dockerBuild executes docker build.
func (b *Builder) dockerBuild(ctx context.Context, projectDir, imageTag string, buildArgs map[string]string) (string, error) {
	cmd := fmt.Sprintf("docker build -t %s", imageTag)
	for k, v := range buildArgs {
		cmd += fmt.Sprintf(" --build-arg %s=%s", k, v)
	}
	cmd += fmt.Sprintf(" %s", projectDir)

	output, err := b.executor.RunCommand(ctx, cmd)
	return output, err
}
