package deployer

import (
	"context"
	"fmt"

	"github.com/Yogdunana/deploypilot/internal/sandbox"
)

// SandboxExecutor wraps a CommandExecutor with sandbox validation.
// It validates every command against sandbox rules before execution.
type SandboxExecutor struct {
	inner    CommandExecutor
	sandbox  *sandbox.Sandbox
}

// NewSandboxExecutor creates a new sandboxed executor.
func NewSandboxExecutor(inner CommandExecutor, sb *sandbox.Sandbox) *SandboxExecutor {
	return &SandboxExecutor{inner: inner, sandbox: sb}
}

// RunCommand validates the command against sandbox rules, then delegates to the inner executor.
func (e *SandboxExecutor) RunCommand(ctx context.Context, cmd string) (string, error) {
	if err := e.sandbox.Validate(cmd); err != nil {
		return "", fmt.Errorf("sandbox: %w", err)
	}
	return e.inner.RunCommand(ctx, cmd)
}

// Sandbox returns the underlying sandbox for configuration access.
func (e *SandboxExecutor) Sandbox() *sandbox.Sandbox {
	return e.sandbox
}
