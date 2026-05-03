-- Create oauth2_clients table
CREATE TABLE IF NOT EXISTS oauth2_clients (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    user_id TEXT,
    name TEXT NOT NULL,
    client_id TEXT UNIQUE NOT NULL,
    client_secret TEXT NOT NULL,
    redirect_uris TEXT,
    scopes TEXT,
    grant_types TEXT,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create oauth2_authorizations table
CREATE TABLE IF NOT EXISTS oauth2_authorizations (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    scopes TEXT,
    code TEXT UNIQUE NOT NULL,
    expires_at DATETIME,
    used BOOLEAN DEFAULT false,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create oauth2_tokens table
CREATE TABLE IF NOT EXISTS oauth2_tokens (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    access_token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE NOT NULL,
    scopes TEXT,
    token_type TEXT DEFAULT 'bearer',
    expires_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
