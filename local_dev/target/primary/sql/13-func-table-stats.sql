-- Function: table statistics (sizes, vacuum stats, etc.)

CREATE OR REPLACE FUNCTION public.pgmonitor_table_statistics()
RETURNS TABLE (
    schema_name text,
    table_name text,
    indexes_size bigint,
    total_relation_size bigint,
    table_size bigint,
    toast_size bigint,
    seq_scan bigint,
    seq_tup_read bigint,
    idx_scan bigint,
    idx_tup_fetch bigint,
    n_tup_ins bigint,
    n_tup_upd bigint,
    n_tup_del bigint,
    n_tup_hot_upd bigint,
    n_live_tup bigint,
    n_dead_tup bigint,
    n_mod_since_analyze bigint,
    n_ins_since_vacuum bigint,
    last_vacuum timestamp with time zone,
    last_autovacuum timestamp with time zone,
    last_analyze timestamp with time zone,
    last_autoanalyze timestamp with time zone,
    vacuum_count bigint,
    autovacuum_count bigint,
    analyze_count bigint,
    autoanalyze_count bigint
)
AS $$
BEGIN
    RETURN QUERY
    WITH user_tables AS (
        SELECT
            c.oid AS table_oid,
            n.nspname AS schema_name,
            c.relname AS table_name
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relkind = 'r'
          AND n.nspname NOT IN ('pg_catalog', 'pg_toast', 'information_schema')
    )
    SELECT
        ut.schema_name::text,
        ut.table_name::text,
        pg_indexes_size(ut.table_oid) AS indexes_size,
        pg_total_relation_size(ut.table_oid) AS total_relation_size,
        pg_total_relation_size(ut.table_oid) - pg_indexes_size(ut.table_oid) AS table_size,
        pg_total_relation_size(ut.table_oid)
            - pg_indexes_size(ut.table_oid)
            - pg_relation_size(ut.table_oid, 'main')
            - pg_relation_size(ut.table_oid, 'fsm')
            - pg_relation_size(ut.table_oid, 'vm') AS toast_size,
        t.seq_scan,
        t.seq_tup_read,
        t.idx_scan,
        t.idx_tup_fetch,
        t.n_tup_ins,
        t.n_tup_upd,
        t.n_tup_del,
        t.n_tup_hot_upd,
        t.n_live_tup,
        t.n_dead_tup,
        t.n_mod_since_analyze,
        t.n_ins_since_vacuum,
        t.last_vacuum,
        t.last_autovacuum,
        t.last_analyze,
        t.last_autoanalyze,
        t.vacuum_count,
        t.autovacuum_count,
        t.analyze_count,
        t.autoanalyze_count
    FROM user_tables ut
    INNER JOIN pg_stat_user_tables t
        ON t.schemaname = ut.schema_name AND t.relname = ut.table_name;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

GRANT EXECUTE ON FUNCTION public.pgmonitor_table_statistics TO pgmonitor;
