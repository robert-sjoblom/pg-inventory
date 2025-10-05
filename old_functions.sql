CREATE OR REPLACE FUNCTION public.pgmonitor_get_tables() RETURNS TABLE (
    table_name text,
    table_schema text,
    table_owner text,
    table_has_pk bool,
    table_comment text,
    table_columns jsonb,
    table_indexes jsonb
) AS $$ BEGIN RETURN QUERY WITH index_column_names AS (
    SELECT array_agg(a.attname) AS column_names,
      attrelid,
      ic.indexrelid,
      ic.indrelid
    FROM pg_attribute a
      LEFT JOIN (
        SELECT indexrelid,
          indrelid,
          v AS column_key,
          ROW_NUMBER() OVER (
            PARTITION BY indexrelid
            ORDER BY v
          ) AS column_idx
        FROM pg_index,
          LATERAL UNNEST(indkey) WITH ORDINALITY AS a(v)
        ORDER BY column_idx DESC
      ) ic ON ic.indrelid = a.attrelid
      AND ic.column_key = a.attnum
    GROUP BY attrelid,
      ic.indexrelid,
      ic.indrelid
  ),
  index_data AS (
    SELECT jsonb_agg(
        jsonb_build_object(
          'name',
          c.relname,
          'columns',
          icn.column_names,
          'is_unique',
          ix.indisunique,
          'is_primary',
          ix.indisprimary,
          'is_exclusion',
          ix.indisexclusion,
          'is_valid',
          ix.indisvalid,
          'is_partial',
          CASE
            WHEN ix.indpred IS NULL THEN FALSE
            ELSE TRUE
          END,
          'scans',
          psai.idx_scan,
          'tuple_reads',
          psai.idx_tup_read,
          'tuple_fetch',
          psai.idx_tup_fetch,
          'definition',
          ixs.indexdef
        )
      ) AS _table_indexes,
      ix.indrelid,
      ix.indexrelid
    FROM pg_index ix
      LEFT JOIN index_column_names icn ON icn.indexrelid = ix.indexrelid
      AND icn.indrelid = ix.indrelid
      LEFT JOIN pg_class c ON ix.indexrelid = c.oid
      LEFT JOIN pg_indexes ixs ON ixs.indexname = c.relname
      LEFT JOIN pg_stat_all_indexes psai ON psai.indexrelname = c.relname
    GROUP BY ix.indrelid,
      ix.indexrelid
  ),
  index_aggregate AS (
    SELECT c.oid,
      id._table_indexes
    FROM pg_class c
      LEFT JOIN index_data id ON id.indrelid = c.oid
  )
SELECT c.relname::TEXT AS table_name,
  n.nspname::TEXT AS table_schema,
  r.rolname::TEXT AS table_owner,
  EXISTS (
    SELECT 1
    FROM pg_index i
      JOIN pg_attribute a ON a.attrelid = i.indrelid
      AND a.attnum = ANY(i.indkey)
    WHERE i.indrelid = c.oid
      AND i.indisprimary
  ) AS table_has_pk,
  obj_description(c.oid, 'pg_class')::TEXT AS table_comment,
  (
    SELECT jsonb_agg(
        jsonb_build_object(
          'column_name',
          a.attname,
          'column_type',
          format_type(a.atttypid, a.atttypmod)
        )
      )
    FROM pg_attribute a
    WHERE a.attrelid = c.oid
      AND a.attnum > 0
      AND NOT a.attisdropped
  ) AS table_columns,
  _table_indexes AS table_indexes
FROM pg_class c
  JOIN pg_namespace n ON n.oid = c.relnamespace
  JOIN pg_roles r ON r.oid = c.relowner
  LEFT JOIN index_aggregate ia ON ia.oid = c.oid
WHERE c.relkind = 'r'
  AND n.nspname NOT IN (
    'pg_catalog',
    'information_schema',
    'pg_toast'
  );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
GRANT EXECUTE ON FUNCTION public.pgmonitor_get_tables TO pgmonitor;
-- Create secure function for pg_stat_statements
CREATE OR REPLACE FUNCTION pgmonitor.pgmonitor_get_pg_stat_statements() RETURNS SETOF pgmonitor.pg_stat_statements AS $$
  /* pgmonitor */
SELECT *
FROM pgmonitor.pg_stat_statements;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;
-- Create secure function to pg_stat_statements_reset
CREATE OR REPLACE FUNCTION pgmonitor.pgmonitor_reset_pg_stat_statements() RETURNS SETOF void AS $$
  /* pgmonitor */
SELECT pgmonitor.pg_stat_statements_reset() $$ LANGUAGE sql VOLATILE SECURITY DEFINER;
-- Create function to query pg_stat_subscription_stats without errors
CREATE OR REPLACE FUNCTION pgmonitor.pg_stat_subscription_stats_if_exists() RETURNS TABLE (
    subid oid,
    subname name,
    apply_error_count int8,
    sync_error_count int8,
    stats_reset timestamptz
) LANGUAGE plpgsql PARALLEL SAFE AS $func$ BEGIN IF EXISTS (
    SELECT
    FROM information_schema.tables
    WHERE table_schema = 'pg_catalog'
      AND table_name = 'pg_stat_subscription_stats'
  ) THEN RETURN QUERY (
    SELECT s.subid,
      s.subname,
      s.apply_error_count,
      s.sync_error_count,
      s.stats_reset
    FROM pg_stat_subscription_stats s
  );
END IF;
END $func$;
-- Create materialized view pg_stat_statements_history
CREATE MATERIALIZED VIEW IF NOT EXISTS pgmonitor.pg_stat_statements_history AS
SELECT
    *,
    (CURRENT_DATE - interval '1 day')::date AS from_date,
    CURRENT_DATE AS to_date
FROM pgmonitor.pgmonitor_get_pg_stat_statements() WITH NO DATA;
GRANT EXECUTE ON FUNCTION pgmonitor.pgmonitor_reset_pg_stat_statements TO pgmonitor;
ALTER MATERIALIZED VIEW pgmonitor.pg_stat_statements_history OWNER TO pgmonitor;
CREATE OR REPLACE FUNCTION public.pgmonitor_get_extensions() RETURNS TABLE (
    extension_name text,
    extension_version text,
    extension_owner text,
    extension_schema text
) AS $$
SELECT pe.extname::TEXT AS extension_name,
  pe.extversion::TEXT AS extension_version,
  pa.rolname::TEXT AS extension_owner,
  pn.nspname::TEXT AS extension_schema
FROM pg_catalog.pg_extension pe
  JOIN pg_catalog.pg_authid pa ON pe.extowner = pa.oid
  JOIN pg_catalog.pg_namespace pn ON pe.extnamespace = pn.oid
WHERE pn.nspname NOT IN ('pg_toast', 'pg_catalog', 'information_schema');
$$ LANGUAGE sql STABLE SECURITY DEFINER;
GRANT EXECUTE ON FUNCTION public.pgmonitor_get_extensions TO pgmonitor;
CREATE OR REPLACE FUNCTION public.pgmonitor_table_statistics() RETURNS TABLE (
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
) AS $$ BEGIN RETURN QUERY WITH user_tables AS (
    SELECT c.oid AS table_oid,
      n.nspname AS schema_name,
      c.relname AS table_name
    FROM pg_class c
      JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind = 'r'
      AND n.nspname NOT IN ('pg_catalog', 'pg_toast', 'information_schema')
  )
SELECT ut.schema_name::text,
  ut.table_name::text,
  pg_indexes_size(ut.table_oid) AS indexes_size,
  pg_total_relation_size(ut.table_oid) AS total_relation_size,
  pg_total_relation_size(ut.table_oid) - pg_indexes_size(ut.table_oid) AS table_size,
  pg_total_relation_size(ut.table_oid) - pg_indexes_size(ut.table_oid) - pg_relation_size(ut.table_oid, 'main') - pg_relation_size(ut.table_oid, 'fsm') - pg_relation_size(ut.table_oid, 'vm') AS toast_size,
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
  INNER JOIN pg_stat_user_tables t ON t.schemaname = ut.schema_name
  AND t.relname = ut.table_name;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
GRANT EXECUTE ON FUNCTION public.pgmonitor_table_statistics TO pgmonitor;
