-- Functions: pg_stat_statements access and reset
-- Plus materialized view for historical tracking

CREATE OR REPLACE FUNCTION pgmonitor.pgmonitor_get_pg_stat_statements()
RETURNS SETOF pgmonitor.pg_stat_statements
AS $$
    SELECT * FROM pgmonitor.pg_stat_statements;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;

GRANT EXECUTE ON FUNCTION pgmonitor.pgmonitor_get_pg_stat_statements TO pgmonitor;

CREATE OR REPLACE FUNCTION pgmonitor.pgmonitor_reset_pg_stat_statements()
RETURNS SETOF void
AS $$
    SELECT pgmonitor.pg_stat_statements_reset();
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;

GRANT EXECUTE ON FUNCTION pgmonitor.pgmonitor_reset_pg_stat_statements TO pgmonitor;

-- Materialized view for historical statement stats
CREATE MATERIALIZED VIEW IF NOT EXISTS pgmonitor.pg_stat_statements_history AS
SELECT
    *,
    (CURRENT_DATE - interval '1 day')::date AS from_date,
    CURRENT_DATE AS to_date
FROM pgmonitor.pgmonitor_get_pg_stat_statements()
WITH NO DATA;

ALTER MATERIALIZED VIEW pgmonitor.pg_stat_statements_history OWNER TO pgmonitor;
