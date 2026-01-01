-- Function: list tables with columns and indexes
-- This is the main inventory function

CREATE OR REPLACE FUNCTION public.pgmonitor_get_tables()
RETURNS TABLE (
    table_name text,
    table_schema text,
    table_owner text,
    table_has_pk bool,
    table_comment text,
    table_columns jsonb,
    table_indexes jsonb
)
AS $$
BEGIN
    RETURN QUERY
    WITH index_column_names AS (
        SELECT
            array_agg(a.attname) AS column_names,
            attrelid,
            ic.indexrelid,
            ic.indrelid
        FROM pg_attribute a
        LEFT JOIN (
            SELECT
                indexrelid,
                indrelid,
                v AS column_key,
                ROW_NUMBER() OVER (PARTITION BY indexrelid ORDER BY v) AS column_idx
            FROM pg_index,
            LATERAL UNNEST(indkey) WITH ORDINALITY AS a(v)
            ORDER BY column_idx DESC
        ) ic ON ic.indrelid = a.attrelid AND ic.column_key = a.attnum
        GROUP BY attrelid, ic.indexrelid, ic.indrelid
    ),
    index_data AS (
        SELECT
            jsonb_agg(
                jsonb_build_object(
                    'name', c.relname,
                    'columns', icn.column_names,
                    'is_unique', ix.indisunique,
                    'is_primary', ix.indisprimary,
                    'is_exclusion', ix.indisexclusion,
                    'is_valid', ix.indisvalid,
                    'is_partial', CASE WHEN ix.indpred IS NULL THEN FALSE ELSE TRUE END,
                    'scans', psai.idx_scan,
                    'tuple_reads', psai.idx_tup_read,
                    'tuple_fetch', psai.idx_tup_fetch,
                    'definition', ixs.indexdef
                )
            ) AS _table_indexes,
            ix.indrelid,
            ix.indexrelid
        FROM pg_index ix
        LEFT JOIN index_column_names icn ON icn.indexrelid = ix.indexrelid AND icn.indrelid = ix.indrelid
        LEFT JOIN pg_class c ON ix.indexrelid = c.oid
        LEFT JOIN pg_indexes ixs ON ixs.indexname = c.relname
        LEFT JOIN pg_stat_all_indexes psai ON psai.indexrelname = c.relname
        GROUP BY ix.indrelid, ix.indexrelid
    ),
    index_aggregate AS (
        SELECT c.oid, id._table_indexes
        FROM pg_class c
        LEFT JOIN index_data id ON id.indrelid = c.oid
    )
    SELECT
        c.relname::TEXT AS table_name,
        n.nspname::TEXT AS table_schema,
        r.rolname::TEXT AS table_owner,
        EXISTS (
            SELECT 1
            FROM pg_index i
            JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
            WHERE i.indrelid = c.oid AND i.indisprimary
        ) AS table_has_pk,
        obj_description(c.oid, 'pg_class')::TEXT AS table_comment,
        (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'column_name', a.attname,
                    'column_type', format_type(a.atttypid, a.atttypmod)
                )
            )
            FROM pg_attribute a
            WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
        ) AS table_columns,
        _table_indexes AS table_indexes
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_roles r ON r.oid = c.relowner
    LEFT JOIN index_aggregate ia ON ia.oid = c.oid
    WHERE c.relkind = 'r'
      AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast');
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

GRANT EXECUTE ON FUNCTION public.pgmonitor_get_tables TO pgmonitor;
