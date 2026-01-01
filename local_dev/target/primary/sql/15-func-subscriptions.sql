-- Function: subscription stats (PG15+)
-- Handles missing table gracefully for older versions

CREATE OR REPLACE FUNCTION pgmonitor.pg_stat_subscription_stats_if_exists()
RETURNS TABLE (
    subid oid,
    subname name,
    apply_error_count int8,
    sync_error_count int8,
    stats_reset timestamptz
)
LANGUAGE plpgsql PARALLEL SAFE
AS $$
BEGIN
    IF EXISTS (
        SELECT FROM information_schema.tables
        WHERE table_schema = 'pg_catalog'
          AND table_name = 'pg_stat_subscription_stats'
    ) THEN
        RETURN QUERY (
            SELECT
                s.subid,
                s.subname,
                s.apply_error_count,
                s.sync_error_count,
                s.stats_reset
            FROM pg_stat_subscription_stats s
        );
    END IF;
END;
$$;
