-- Create audit_hashes table for hash chain verification
CREATE TABLE IF NOT EXISTS audit_hashes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    audit_id INTEGER UNIQUE NOT NULL,
    hash TEXT NOT NULL,
    previous_hash TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create ip_whitelists table for per-user IP whitelist
CREATE TABLE IF NOT EXISTS ip_whitelists (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    user_id TEXT NOT NULL,
    description TEXT,
    cidr TEXT NOT NULL,
    created_by TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create devices table for device binding
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    user_id TEXT,
    device_id TEXT UNIQUE NOT NULL,
    device_name TEXT,
    user_agent TEXT,
    ip TEXT,
    last_ip TEXT,
    trusted BOOLEAN DEFAULT false,
    trust_expires_at DATETIME,
    last_seen_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create signing_keys table for Ed25519 code signing
CREATE TABLE IF NOT EXISTS signing_keys (
    id TEXT PRIMARY KEY,
    key_version INTEGER UNIQUE NOT NULL,
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL,
    fingerprint TEXT,
    is_active BOOLEAN DEFAULT 1,
    created_by TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
