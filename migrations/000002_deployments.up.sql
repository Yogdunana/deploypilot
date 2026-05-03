-- Create deployments table
CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES tenants(id),
    server_id TEXT,
    app_name TEXT,
    app_id TEXT,
    container_name TEXT,
    image TEXT,
    previous_image TEXT,
    deploy_type TEXT DEFAULT 'deploy',
    config_snapshot TEXT,
    status TEXT DEFAULT 'deploying',
    preflight_code TEXT,
    preflight_message TEXT,
    preflight_checks TEXT,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_deployments_container_name ON deployments(container_name);
CREATE INDEX IF NOT EXISTS idx_deployments_app_id ON deployments(app_id);
