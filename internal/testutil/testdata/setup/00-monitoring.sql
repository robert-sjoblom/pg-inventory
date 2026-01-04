-- Monitoring infrastructure for pg-inventory
-- This must exist before extractor can connect
CREATE SCHEMA IF NOT EXISTS monitoring;

CREATE TABLE IF NOT EXISTS monitoring.cluster_config (key TEXT PRIMARY KEY, value TEXT NOT NULL);

INSERT INTO
    monitoring.cluster_config (key, value)
VALUES
    ('cluster_name', 'test-cluster'),
    ('stanza', 'test-stanza');