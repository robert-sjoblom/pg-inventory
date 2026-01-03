-- Monitoring infrastructure for pg-inventory
-- This must exist before extractor can connect
CREATE SCHEMA IF NOT EXISTS monitoring;

CREATE TABLE IF NOT EXISTS monitoring.cluster_config (key TEXT PRIMARY KEY, value TEXT NOT NULL);

-- Default cluster name for local dev
INSERT INTO
    monitoring.cluster_config (key, value)
VALUES
    ('cluster_name', 'main') ON CONFLICT (key) DO NOTHING;

-- Force WAL switch to trigger archive_command and verify it works
SELECT
    pg_switch_wal ();
