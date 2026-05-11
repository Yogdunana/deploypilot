-- Create audit_logs table with all enhancement columns
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    username TEXT,
    action TEXT,
    resource_type TEXT,
    resource_id TEXT,
    detail TEXT,
    ip_address TEXT,
    user_agent TEXT,
    trace_id TEXT DEFAULT '',
    record_hash TEXT DEFAULT '',
    log_type TEXT DEFAULT 'operation',
    archived BOOLEAN DEFAULT false,
    archived_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
