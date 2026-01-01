-- Function: list installed extensions

CREATE OR REPLACE FUNCTION public.pgmonitor_get_extensions()
RETURNS TABLE (
    extension_name text,
    extension_version text,
    extension_owner text,
    extension_schema text
)
AS $$
    SELECT
        pe.extname::TEXT AS extension_name,
        pe.extversion::TEXT AS extension_version,
        pa.rolname::TEXT AS extension_owner,
        pn.nspname::TEXT AS extension_schema
    FROM pg_catalog.pg_extension pe
    JOIN pg_catalog.pg_authid pa ON pe.extowner = pa.oid
    JOIN pg_catalog.pg_namespace pn ON pe.extnamespace = pn.oid
    WHERE pn.nspname NOT IN ('pg_toast', 'pg_catalog', 'information_schema');
$$ LANGUAGE sql STABLE SECURITY DEFINER;

GRANT EXECUTE ON FUNCTION public.pgmonitor_get_extensions TO pgmonitor;
