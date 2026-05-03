-- Create alert_rules table
CREATE TABLE IF NOT EXISTS alert_rules (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    name TEXT NOT NULL,
    metric_type TEXT NOT NULL,
    condition TEXT NOT NULL,
    threshold REAL DEFAULT 0,
    severity TEXT NOT NULL DEFAULT 'warning',
    enabled INTEGER DEFAULT 1,
    cooldown_seconds INTEGER DEFAULT 900,
    notify_channels TEXT,
    server_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create scheduled_tasks table
CREATE TABLE IF NOT EXISTS scheduled_tasks (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    cron_expr TEXT NOT NULL,
    task_type TEXT,
    command TEXT,
    server_id TEXT,
    enabled INTEGER DEFAULT 1,
    timeout INTEGER DEFAULT 300,
    last_run_at DATETIME,
    last_status TEXT,
    last_error TEXT,
    run_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create task_executions table
CREATE TABLE IF NOT EXISTS task_executions (
    id TEXT PRIMARY KEY,
    task_id TEXT,
    tenant_id TEXT,
    status TEXT,
    output TEXT,
    error TEXT,
    started_at DATETIME,
    ended_at DATETIME,
    duration BIGINT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create alert_silences table
CREATE TABLE IF NOT EXISTS alert_silences (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    name TEXT,
    reason TEXT,
    matchers TEXT,
    starts_at DATETIME,
    ends_at DATETIME,
    created_by TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_silences_ends_at ON alert_silences(ends_at);

-- Create alert_escalations table
CREATE TABLE IF NOT EXISTS alert_escalations (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    name TEXT,
    rule_ids TEXT,
    steps TEXT,
    repeat_interval INTEGER DEFAULT 60,
    enabled BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create alert_groups table
CREATE TABLE IF NOT EXISTS alert_groups (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    group_key TEXT,
    rule_id TEXT,
    severity TEXT,
    alert_count INTEGER DEFAULT 1,
    first_alert_at DATETIME,
    last_alert_at DATETIME,
    status TEXT DEFAULT 'firing',
    resolved_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create event_logs table
CREATE TABLE IF NOT EXISTS event_logs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    event_type TEXT NOT NULL,
    topic TEXT,
    source TEXT,
    payload TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create ssh_key_pairs table
CREATE TABLE IF NOT EXISTS ssh_key_pairs (
    id TEXT PRIMARY KEY,
    name TEXT,
    public_key TEXT,
    private_key TEXT,
    fingerprint TEXT,
    key_type TEXT DEFAULT 'rsa',
    key_bits INTEGER DEFAULT 4096,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create ssh_authorizations table
CREATE TABLE IF NOT EXISTS ssh_authorizations (
    id TEXT PRIMARY KEY,
    key_pair_id TEXT,
    server_id TEXT,
    user TEXT DEFAULT 'root',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create process_rules table
CREATE TABLE IF NOT EXISTS process_rules (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    server_id TEXT,
    name TEXT NOT NULL,
    process_pattern TEXT NOT NULL,
    restart_command TEXT,
    auto_restart INTEGER DEFAULT 0,
    max_restarts INTEGER DEFAULT 5,
    restart_count INTEGER DEFAULT 0,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create system_snapshots table
CREATE TABLE IF NOT EXISTS system_snapshots (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    server_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    file_count INTEGER DEFAULT 0,
    total_size INTEGER DEFAULT 0,
    checksum TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create toolbox_scripts table
CREATE TABLE IF NOT EXISTS toolbox_scripts (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    category TEXT,
    script TEXT NOT NULL,
    is_built_in INTEGER DEFAULT 0,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create monitors table
CREATE TABLE IF NOT EXISTS monitors (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    target TEXT NOT NULL,
    interval INTEGER DEFAULT 60,
    timeout INTEGER DEFAULT 10,
    retries INTEGER DEFAULT 3,
    status TEXT DEFAULT 'unknown',
    enabled INTEGER DEFAULT 1,
    last_check TEXT,
    last_status TEXT,
    uptime REAL DEFAULT 100,
    total_checks INTEGER DEFAULT 0,
    up_checks INTEGER DEFAULT 0,
    avg_latency REAL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create monitor_check_results table
CREATE TABLE IF NOT EXISTS monitor_check_results (
    id TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL,
    status TEXT,
    status_code INTEGER DEFAULT 0,
    latency REAL DEFAULT 0,
    message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create heartbeats table
CREATE TABLE IF NOT EXISTS heartbeats (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    name TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    interval INTEGER DEFAULT 60,
    timeout INTEGER DEFAULT 120,
    status TEXT DEFAULT 'unknown',
    last_beat TEXT,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
