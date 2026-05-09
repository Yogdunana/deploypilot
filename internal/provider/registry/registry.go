package registry

import (
	"context"
	"fmt"
)

// RegistryProvider defines the interface for container registry operations.
type RegistryProvider interface {
	// Login authenticates with the registry (docker login equivalent).
	Login(ctx context.Context) error

	// Push pushes an image to the registry.
	Push(ctx context.Context, localImage, remoteTag string) error

	// Pull pulls an image from the registry.
	Pull(ctx context.Context, image string) error

	// Tag creates a tag for an image.
	Tag(ctx context.Context, sourceImage, targetTag string) error

	// ListTags lists tags for a repository.
	ListTags(ctx context.Context, repository string) ([]string, error)

	// Ping checks if the registry is accessible.
	Ping(ctx context.Context) error

	// Name returns the provider name.
	Name() string
}

// NewRegistryProvider creates a new RegistryProvider based on the provider type.
func NewRegistryProvider(provider, url, username, password string) (RegistryProvider, error) {
	switch provider {
	case "docker_hub":
		return NewDockerHubProvider(url, username, password), nil
	case "ghcr":
		return NewGHCRProvider(url, username, password), nil
	default:
		return nil, fmt.Errorf("unsupported registry provider: %s", provider)
	}
}
