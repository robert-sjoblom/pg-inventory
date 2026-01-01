-- Monitoring user with minimal privileges
-- Gets data access through SECURITY DEFINER functions

CREATE USER pgmonitor PASSWORD 'password';

-- pg_monitor role grants read access to various stats views
GRANT pg_monitor TO pgmonitor;
