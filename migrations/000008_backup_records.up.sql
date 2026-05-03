-- Create backup_records table with cloud storage columns
CREATE TABLE IF NOT EXISTS backup_records (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL DEFAULT 'database',
    app_id TEXT,
    status TEXT NOT NULL DEFAULT 'completed',
    file_path TEXT,
    file_size INTEGER DEFAULT 0,
    trigger TEXT DEFAULT 'manual',
    error TEXT,
    storage_type TEXT DEFAULT 'local',
    storage_path TEXT DEFAULT '',
    storage_bucket TEXT DEFAULT '',
    file_checksum TEXT DEFAULT '',
    encrypted INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
