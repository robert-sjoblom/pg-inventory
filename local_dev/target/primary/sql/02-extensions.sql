-- Extensions and pgmonitor schema setup

CREATE SCHEMA IF NOT EXISTS pgmonitor;

GRANT USAGE ON SCHEMA pgmonitor TO pgmonitor;

CREATE EXTENSION pg_stat_statements SCHEMA pgmonitor;

-- Set search_path so pgmonitor can use pg_stat_activity transparently
ALTER USER pgmonitor SET search_path = pgmonitor, pg_catalog;
