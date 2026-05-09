-- Combined initial schema for new installations
-- Creates core tables: tenants, roles, users, credentials, providers, servers, apps

CREATE TABLE IF NOT EXISTS tenants (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    plan TEXT DEFAULT 'free',
    max_servers INTEGER DEFAULT 5,
    max_apps INTEGER DEFAULT 20,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    permissions TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES tenants(id),
    role_id TEXT REFERENCES roles(id),
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    auth_provider TEXT DEFAULT '',
    auth_uid TEXT DEFAULT '',
    avatar_url TEXT DEFAULT '',
    totp_secret TEXT DEFAULT '',
    totp_enabled BOOLEAN DEFAULT false,
    backup_codes TEXT DEFAULT '',
    onboarding_completed BOOLEAN DEFAULT false,
    last_login_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS credentials (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES tenants(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    encrypted_value TEXT NOT NULL,
    expires_at DATETIME,
    last_rotated DATETIME,
    rotation_days INTEGER DEFAULT 90,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_credentials_expires_at ON credentials(expires_at);

CREATE TABLE IF NOT EXISTS providers (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES tenants(id),
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    config TEXT,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS servers (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES tenants(id),
    credential_id TEXT REFERENCES credentials(id),
    provider_id TEXT REFERENCES providers(id),
    name TEXT NOT NULL,
    host TEXT NOT NULL,
    port INTEGER DEFAULT 22,
    tags TEXT,
    status TEXT DEFAULT 'unknown',
    detected_info TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS apps (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES tenants(id),
    server_id TEXT REFERENCES servers(id),
    name TEXT NOT NULL,
    repo_url TEXT NOT NULL,
    branch TEXT DEFAULT 'main',
    domain TEXT,
    tech_stack TEXT DEFAULT 'docker',
    deploy_mode TEXT DEFAULT 'api',
    status TEXT DEFAULT 'pending',
    current_version TEXT,
    container_name TEXT,
    env_vars TEXT,
    resource_limits TEXT,
    environment TEXT DEFAULT 'production',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_apps_environment ON apps(environment);
