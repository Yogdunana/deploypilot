# Phase 7.5: Plugin System Design

> Date: 2026-05-02
> Status: Approved

## Overview

Implement a plugin system for DeployPilot that allows extending functionality through a well-defined Go interface. Plugins can subscribe to events, register custom API routes, and manage their own lifecycle.

## Architecture

```
┌─────────────────────────────────────────────────┐
│              Plugin Manager                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │ Registry  │  │ Lifecycle│  │   Hook System │  │
│  │ (discover)│  │ (init/   │  │ (event hooks) │  │
│  │           │  │  start/  │  │              │  │
│  │           │  │  stop)   │  │              │  │
│  └──────────┘  └──────────┘  └──────────────┘  │
│                                                 │
│  Plugin Interface (Go interface)                 │
│  ├── Name, Version, Description                 │
│  ├── Init(ctx, config) error                    │
│  ├── Start() error                              │
│  ├── Stop() error                               │
│  ├── OnEvent(event)                             │
│  └── RegisterAPIRoutes(rg)                      │
└─────────────────────────────────────────────────┘
```

## Feature Modules

### 1. Plugin Interface

```go
// Plugin is the interface that all plugins must implement.
type Plugin interface {
    // Metadata
    Name() string
    Version() string
    Description() string

    // Lifecycle
    Init(ctx context.Context, config map[string]interface{}) error
    Start() error
    Stop() error

    // Event handling
    OnEvent(event BusEvent)

    // Optional: register custom API routes
    RegisterAPIRoutes(rg *gin.RouterGroup)
}
```

Plugins are compiled into the binary (not dynamically loaded) to avoid cross-compilation issues with Go's plugin package.

### 2. Plugin Manager

```go
type PluginManager struct {
    plugins    map[string]Plugin
    configs    map[string]map[string]interface{}
    statuses   map[string]PluginStatus
    mu         sync.RWMutex
    bus        TypedEventBus
    db         *gorm.DB
}
```

- `Register(plugin Plugin)` — register a plugin
- `InitAll()` — initialize all registered plugins
- `StartAll()` — start all enabled plugins
- `StopAll()` — graceful shutdown
- `GetPlugin(name) Plugin` — get plugin by name
- `ListPlugins() []PluginInfo` — list all plugins with status
- `EnablePlugin(name) error` — enable a plugin
- `DisablePlugin(name) error` — disable a plugin
- `UpdatePluginConfig(name, config) error` — update plugin config

### 3. Hook System

Plugins subscribe to events via the existing TypedEventBus:
- `bus.SubscribeType(ctx, EventDeploy)` — subscribe to deploy events
- `bus.SubscribeType(ctx, EventAlert)` — subscribe to alert events
- etc.

The PluginManager manages subscriptions for all plugins and dispatches events.

### 4. Data Model

```go
type PluginConfig struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    TenantID  string    `json:"tenant_id" gorm:"index"`
    Name      string    `json:"name" gorm:"uniqueIndex"`
    Enabled   bool      `json:"enabled"`
    Config    string    `json:"config" gorm:"type:text"` // JSON
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 5. API Endpoints

```
GET    /api/v1/plugins              - List all plugins (status + config)
GET    /api/v1/plugins/:name        - Get plugin details
PUT    /api/v1/plugins/:name        - Update plugin config/enable
POST   /api/v1/plugins/:name/start  - Start a plugin
POST   /api/v1/plugins/:name/stop   - Stop a plugin
```

### 6. Frontend

- Plugin list page: card grid showing all plugins, status badges, enable/disable toggle
- Plugin detail page: config editor, logs, status

### 7. Built-in Example Plugins

- `hello-world` — minimal example
- `slack-notify` — Slack notification via webhook on deploy events
- `deploy-gate` — deployment approval gate (requires manual approval before deploy)

## File Structure

```
internal/
├── plugin/
│   ├── plugin.go           # Plugin interface + PluginInfo
│   ├── manager.go          # PluginManager
│   └── builtin/            # Built-in plugins
│       ├── hello_world.go
│       ├── slack_notify.go
│       └── deploy_gate.go
├── model/plugin.go         # PluginConfig GORM model
├── api/plugin_api.go       # API handlers
├── api/router.go           # Add /plugins routes
web/src/
├── api/modules/plugin.ts   # API module
├── views/PluginList.vue    # Plugin list page
└── router/index.ts         # Add routes
```
