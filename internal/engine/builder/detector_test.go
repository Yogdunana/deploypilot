package builder

import (
	"context"
	"testing"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// mockExecutor implements deployer.CommandExecutor for testing.
type mockExecutor struct {
	responses map[string]string // cmd -> output
}

func (m *mockExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	if resp, ok := m.responses[cmd]; ok {
		return resp, nil
	}
	return "", nil
}

func TestDetectTechStack(t *testing.T) {
	tests := []struct {
		name      string
		existFile string
		want      string
	}{
		{"Next.js config js", "next.config.js", "node"},
		{"Next.js config mjs", "next.config.mjs", "node"},
		{"Vue config", "vue.config.js", "node"},
		{"Vite config ts", "vite.config.ts", "node"},
		{"Vite config js", "vite.config.js", "node"},
		{"Angular", "angular.json", "node"},
		{"Generic Node.js", "package.json", "node"},
		{"Go", "go.mod", "go"},
		{"Rust", "Cargo.toml", "rust"},
		{"Ruby", "Gemfile", "ruby"},
		{"PHP", "composer.json", "php"},
		{"Java Maven", "pom.xml", "java"},
		{"Java Gradle", "build.gradle", "java"},
		{"Python requirements", "requirements.txt", "python"},
		{"Python pyproject", "pyproject.toml", "python"},
		{"Python Pipfile", "Pipfile", "python"},
		{"Django", "manage.py", "python"},
		{"Custom Dockerfile", "Dockerfile", "docker"},
		{"Unknown - default", "", "docker"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &mockExecutor{
				responses: map[string]string{},
			}
			if tt.existFile != "" {
				exec.responses[getTestCmd("/project", tt.existFile)] = "exists"
			}

			got := DetectTechStack(exec, "/project")
			if got != tt.want {
				t.Errorf("DetectTechStack() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectTechStack_Priority(t *testing.T) {
	// When multiple files exist, the first match should win.
	// next.config.js should be detected before package.json.
	exec := &mockExecutor{
		responses: map[string]string{
			getTestCmd("/project", "next.config.js"): "exists",
			getTestCmd("/project", "package.json"):   "exists",
			getTestCmd("/project", "go.mod"):         "exists",
		},
	}

	got := DetectTechStack(exec, "/project")
	if got != "node" {
		t.Errorf("DetectTechStack() = %q, want %q (next.config.js should take priority)", got, "node")
	}
}

// mockDetectorExecutor is a minimal executor for detector tests.
var _ deployer.CommandExecutor = (*mockExecutor)(nil)

func getTestCmd(projectDir, file string) string {
	return "test -f " + projectDir + "/" + file + " && echo 'exists'"
}
