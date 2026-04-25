package plugin

import (
	"fmt"
	"sync"
)

// PluginFactory is a function that creates a provider instance from plugin config.
type PluginFactory func(config map[string]interface{}) (interface{}, error)

// PluginDescriptor describes a plugin's capabilities.
type PluginDescriptor struct {
	Name        string
	DisplayName string
	Version     string
	Description string
	Author      string
	Provider    string // dns, notify, registry, cicd, server, ssl
	Type        string // cloudflare, webhook, docker_hub, etc.
	Factory     PluginFactory
}

// descriptorKey returns the composite key used for descriptor lookup.
func descriptorKey(provider, pluginType string) string {
	return provider + ":" + pluginType
}

// Registry manages plugin registration and lifecycle.
type Registry struct {
	mu          sync.RWMutex
	descriptors map[string]*PluginDescriptor // key: "provider:type"
	instances   map[string]interface{}       // key: plugin ID from DB
}

var globalRegistry *Registry
var once sync.Once

// Global returns the global plugin registry singleton.
func Global() *Registry {
	once.Do(func() {
		globalRegistry = NewRegistry()
	})
	return globalRegistry
}

// NewRegistry creates a new empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		descriptors: make(map[string]*PluginDescriptor),
		instances:   make(map[string]interface{}),
	}
}

// Register registers a plugin descriptor. Returns an error if a plugin
// with the same provider:type is already registered.
func (r *Registry) Register(desc *PluginDescriptor) error {
	if desc == nil {
		return fmt.Errorf("plugin descriptor must not be nil")
	}
	if desc.Provider == "" {
		return fmt.Errorf("plugin provider must not be empty")
	}
	if desc.Type == "" {
		return fmt.Errorf("plugin type must not be empty")
	}
	if desc.Factory == nil {
		return fmt.Errorf("plugin factory must not be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := descriptorKey(desc.Provider, desc.Type)
	if _, exists := r.descriptors[key]; exists {
		return fmt.Errorf("plugin already registered: %s", key)
	}

	r.descriptors[key] = desc
	return nil
}

// Unregister removes a plugin descriptor by provider and type.
func (r *Registry) Unregister(provider, pluginType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := descriptorKey(provider, pluginType)
	if _, exists := r.descriptors[key]; !exists {
		return fmt.Errorf("plugin not registered: %s", key)
	}

	delete(r.descriptors, key)
	return nil
}

// GetDescriptor returns a plugin descriptor by provider and type.
func (r *Registry) GetDescriptor(provider, pluginType string) (*PluginDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	desc, ok := r.descriptors[descriptorKey(provider, pluginType)]
	return desc, ok
}

// ListDescriptors returns all registered descriptors, optionally filtered by provider.
func (r *Registry) ListDescriptors(provider string) []*PluginDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*PluginDescriptor
	for _, desc := range r.descriptors {
		if provider == "" || desc.Provider == provider {
			result = append(result, desc)
		}
	}
	return result
}

// CreateInstance creates a provider instance using the descriptor's factory
// and stores it in the registry under the given pluginID.
func (r *Registry) CreateInstance(pluginID string, desc *PluginDescriptor, config map[string]interface{}) (interface{}, error) {
	if desc == nil {
		return nil, fmt.Errorf("plugin descriptor must not be nil")
	}
	if desc.Factory == nil {
		return nil, fmt.Errorf("plugin %s:%s has no factory", desc.Provider, desc.Type)
	}

	instance, err := desc.Factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance for plugin %s:%s: %w", desc.Provider, desc.Type, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.instances[pluginID] = instance
	return instance, nil
}

// GetInstance returns a previously created instance by plugin ID.
func (r *Registry) GetInstance(pluginID string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instance, ok := r.instances[pluginID]
	return instance, ok
}

// RemoveInstance removes a cached instance by plugin ID.
func (r *Registry) RemoveInstance(pluginID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.instances, pluginID)
}

// ResetGlobal resets the global registry (for testing only).
func ResetGlobal() {
	once.Do(func() {}) // ensure once has run
	globalRegistry = NewRegistry()
}
