-- Create ssl_certificates table
CREATE TABLE IF NOT EXISTS ssl_certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT UNIQUE NOT NULL,
    email TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'cloudflare',
    status TEXT NOT NULL DEFAULT 'pending',
    cert_path TEXT,
    key_path TEXT,
    issued_at DATETIME,
    expires_at DATETIME,
    auto_renew INTEGER DEFAULT 1,
    last_renewed DATETIME,
    retry_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
