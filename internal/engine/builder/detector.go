package builder

import (
	"context"
	"fmt"
	"strings"

	"github.com/Yogdunana/deploypilot/internal/engine/deployer"
)

// DetectTechStack checks for known project files and returns the matching template type.
func DetectTechStack(executor deployer.CommandExecutor, projectDir string) string {
	checks := []struct {
		file  string
		stack string
	}{
		{"next.config.js", "node"},
		{"next.config.mjs", "node"},
		{"vue.config.js", "node"},
		{"vite.config.ts", "node"},
		{"vite.config.js", "node"},
		{"angular.json", "node"},
		{"package.json", "node"},
		{"go.mod", "go"},
		{"Cargo.toml", "rust"},
		{"Gemfile", "ruby"},
		{"composer.json", "php"},
		{"pom.xml", "java"},
		{"build.gradle", "java"},
		{"requirements.txt", "python"},
		{"pyproject.toml", "python"},
		{"Pipfile", "python"},
		{"manage.py", "python"},
		{"Dockerfile", "docker"},
	}

	for _, check := range checks {
		output, _ := executor.RunCommand(context.Background(),
			fmt.Sprintf("test -f %s/%s && echo 'exists'", projectDir, check.file))
		if strings.TrimSpace(output) == "exists" {
			return check.stack
		}
	}
	return "docker" // default: user provides own Dockerfile
}
