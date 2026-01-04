-- Monitoring user with minimal privileges
-- Gets data access through SECURITY DEFINER functions
CREATE USER pgmonitor PASSWORD 'password';

-- pg_monitor role grants read access to various stats views
GRANT pg_monitor TO pgmonitor;

-- Grant access to monitoring schema and config table
GRANT USAGE ON SCHEMA monitoring TO pgmonitor;

GRANT
SELECT
    ON monitoring.cluster_config TO pgmonitor;
