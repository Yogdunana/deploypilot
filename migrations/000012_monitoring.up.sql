-- Create metric_records table
CREATE TABLE IF NOT EXISTS metric_records (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    server_id TEXT,
    container_name TEXT,
    metric_type TEXT,
    name TEXT,
    value REAL,
    unit TEXT,
    labels TEXT,
    timestamp DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_metric_records_tenant_id ON metric_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_metric_records_server_id ON metric_records(server_id);
CREATE INDEX IF NOT EXISTS idx_metric_records_container_name ON metric_records(container_name);
CREATE INDEX IF NOT EXISTS idx_metric_records_metric_type ON metric_records(metric_type);
CREATE INDEX IF NOT EXISTS idx_metric_records_timestamp ON metric_records(timestamp);

-- Create alert_histories table
CREATE TABLE IF NOT EXISTS alert_histories (
    id TEXT PRIMARY KEY,
    tenant_id TEXT,
    rule_id TEXT,
    rule_name TEXT,
    severity TEXT,
    message TEXT,
    value REAL,
    threshold REAL,
    status TEXT,
    fired_at DATETIME,
    resolved_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_alert_histories_tenant_id ON alert_histories(tenant_id);
CREATE INDEX IF NOT EXISTS idx_alert_histories_rule_id ON alert_histories(rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_histories_status ON alert_histories(status);
CREATE INDEX IF NOT EXISTS idx_alert_histories_fired_at ON alert_histories(fired_at);
