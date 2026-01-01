-- Function: list schemas from information_schema
-- SECURITY DEFINER allows pgmonitor to read schema info

CREATE OR REPLACE FUNCTION public.pgmonitor_get_information_schema_schemata()
RETURNS SETOF information_schema.schemata
AS $$
    SELECT * FROM information_schema.schemata;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;

GRANT EXECUTE ON FUNCTION public.pgmonitor_get_information_schema_schemata TO pgmonitor;
