-- Test schemas and tables for development
-- These simulate a real application database

CREATE SCHEMA schema1;

CREATE TABLE schema1.testtable1 (id int8 PRIMARY KEY);

CREATE TABLE schema1.testtable2 (id int8, id_data text);
COMMENT ON TABLE schema1.testtable2 IS 'test comment';

CREATE TABLE schema1.testtable3 (id serial PRIMARY KEY, cool_col jsonb);

-- Edge case: table with no columns
CREATE TABLE schema1.nasty_table_without_cols ();

CREATE SCHEMA schema2;
