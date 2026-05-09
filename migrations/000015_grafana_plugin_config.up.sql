-- Create grafana_custom_dashboards table
CREATE TABLE IF NOT EXISTS grafana_custom_dashboards (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    name TEXT NOT NULL,
    uid TEXT UNIQUE NOT NULL,
    description TEXT,
    json TEXT NOT NULL,
    tags TEXT,
    is_built_in BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create grafana_sync_logs table
CREATE TABLE IF NOT EXISTS grafana_sync_logs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    action TEXT NOT NULL,
    status TEXT NOT NULL,
    message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create plugin_configs table
CREATE TABLE IF NOT EXISTS plugin_configs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    name TEXT UNIQUE NOT NULL,
    enabled BOOLEAN DEFAULT 1,
    config TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
