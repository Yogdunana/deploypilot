-- Create registries table
CREATE TABLE IF NOT EXISTS registries (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES tenants(id),
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    url TEXT NOT NULL,
    username TEXT,
    password TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create clusters table
CREATE TABLE IF NOT EXISTS clusters (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES tenants(id),
    name TEXT NOT NULL,
    description TEXT,
    provider TEXT NOT NULL DEFAULT 'kubernetes',
    api_server TEXT NOT NULL,
    kube_config TEXT,
    kube_config_path TEXT,
    context TEXT,
    namespace TEXT DEFAULT 'default',
    token TEXT,
    ca_data TEXT,
    status TEXT DEFAULT 'unknown',
    version TEXT,
    node_count INTEGER DEFAULT 0,
    tags TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create plugins table
CREATE TABLE IF NOT EXISTS plugins (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES tenants(id),
    name TEXT NOT NULL,
    display_name TEXT,
    version TEXT DEFAULT '1.0.0',
    description TEXT,
    author TEXT,
    provider TEXT NOT NULL,
    type TEXT NOT NULL,
    config TEXT,
    enabled INTEGER DEFAULT 1,
    priority INTEGER DEFAULT 0,
    status TEXT DEFAULT 'active',
    error_msg TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, name)
);
