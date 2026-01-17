//go:build integration

package store

const basicTableDDL = `
	CREATE TABLE "test-db".basic_table (
    	id SERIAL PRIMARY KEY,
    	name VARCHAR(255) NOT NULL,
    	email TEXT UNIQUE,
    	created_at TIMESTAMPTZ DEFAULT now(),
    	metadata JSONB,
    	CONSTRAINT name_not_empty CHECK (length(name) > 0)
	);
	COMMENT ON TABLE "test-db".basic_table IS 'Basic table with PK, unique, check constraint';

	-- Index: partial, expression-based
	CREATE INDEX idx_basic_metadata_gin ON "test-db".basic_table USING gin (metadata);
	CREATE INDEX idx_basic_name_lower ON "test-db".basic_table (lower(name));
	CREATE UNIQUE INDEX idx_basic_email_unique ON "test-db".basic_table (email) WHERE email IS NOT NULL;
`

const noPkTableDDL = `
	CREATE TABLE "test-db".no_pk_table (
		data TEXT,
		value INTEGER
	);
	COMMENT ON TABLE "test-db".no_pk_table IS 'Table without primary key';
`

const compositePkTableDDL = `
	CREATE TABLE "test-db".composite_pk_table (
		tenant_id UUID NOT NULL,
		record_id BIGINT NOT NULL,
		data TEXT,
		PRIMARY KEY (tenant_id, record_id)
	);
`

const foreignKeyTablesDDL = `
	CREATE TABLE "test-db".parent_table (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL
	);

	CREATE TABLE "test-db".child_table (
		id BIGSERIAL PRIMARY KEY,
		parent_id BIGINT NOT NULL REFERENCES "test-db".parent_table(id) ON DELETE CASCADE,
		child_name TEXT NOT NULL,
		CONSTRAINT fk_parent FOREIGN KEY (parent_id) REFERENCES "test-db".parent_table(id)
	);

	CREATE TABLE "test-db".composite_fk_table (
		id SERIAL PRIMARY KEY,
		tenant_id UUID NOT NULL,
		record_id BIGINT NOT NULL,
		CONSTRAINT fk_composite FOREIGN KEY (tenant_id, record_id)
			REFERENCES "test-db".composite_pk_table(tenant_id, record_id)
	);
`

const allColumnDataTypes = `
CREATE TABLE "test-db".all_column_types (
    -- Numeric types
    col_smallint SMALLINT,
    col_integer INTEGER,
    col_bigint BIGINT,
    col_decimal DECIMAL(10, 2),
    col_numeric NUMERIC(15, 5),
    col_real REAL,
    col_double DOUBLE PRECISION,
    col_serial SERIAL,
    col_bigserial BIGSERIAL,

    -- Character types
    col_char CHAR(10),
    col_varchar VARCHAR(255),
    col_text TEXT,

    -- Binary types
    col_bytea BYTEA,

    -- Date/Time types
    col_timestamp TIMESTAMP,
    col_timestamptz TIMESTAMPTZ,
    col_date DATE,
    col_time TIME,
    col_timetz TIMETZ,
    col_interval INTERVAL,

    -- Boolean
    col_boolean BOOLEAN,

    -- UUID
    col_uuid UUID,

    -- Network types
    col_inet INET,
    col_cidr CIDR,
    col_macaddr MACADDR,

    -- JSON types
    col_json JSON,
    col_jsonb JSONB,

    -- Array types
    col_int_array INTEGER[],
    col_text_array TEXT[],
    col_jsonb_array JSONB[],

    -- Range types
    col_int4range INT4RANGE,
    col_tsrange TSRANGE,
    col_daterange DATERANGE,

    -- Geometric types
    col_point POINT,
    col_box BOX,

    -- Full-text search
    col_tsvector TSVECTOR,
    col_tsquery TSQUERY
);
`

const indexTypes = `
 CREATE TABLE "test-db".index_types_table (
     id SERIAL PRIMARY KEY,
     int_col INTEGER,
     text_col TEXT,
     tsvector_col TSVECTOR,
     point_col POINT,
     jsonb_col JSONB,
     array_col INTEGER[],
     during INT4RANGE,
     EXCLUDE USING gist (during WITH &&)
 );
 
 CREATE INDEX idx_btree ON "test-db".index_types_table (int_col);
 CREATE INDEX idx_hash ON "test-db".index_types_table USING hash (int_col);
 CREATE INDEX idx_gin_jsonb ON "test-db".index_types_table USING gin (jsonb_col);
 CREATE INDEX idx_gin_array ON "test-db".index_types_table USING gin (array_col);
 CREATE INDEX idx_gin_tsvector ON "test-db".index_types_table USING gin (tsvector_col);
 CREATE INDEX idx_gist_point ON "test-db".index_types_table USING gist (point_col);
 CREATE INDEX idx_spgist_text ON "test-db".index_types_table USING spgist (text_col);
 CREATE INDEX idx_brin ON "test-db".index_types_table USING brin (id);
 CREATE INDEX idx_partial ON "test-db".index_types_table (text_col) WHERE text_col IS NOT NULL;
 `

const emptyTable = `
CREATE TABLE "test-db".empty_table ();
`

const droppedColumnsTable = `
CREATE TABLE "test-db".dropped_columns_table (
    id SERIAL PRIMARY KEY,
    keep_col TEXT NOT NULL,
    drop_me_1 INTEGER
);
ALTER TABLE "test-db".dropped_columns_table DROP COLUMN drop_me_1;
`

const inheritanceTables = `
-- Simple inheritance: single parent
CREATE TABLE "test-db".base_table (
    id SERIAL PRIMARY KEY,
    base_col TEXT
);

CREATE TABLE "test-db".derived_table (
    derived_col INTEGER
) INHERITS ("test-db".base_table);

-- Multi-level inheritance: grandparent -> parent -> child
CREATE TABLE "test-db".grandparent_table (
    gp_col TEXT
);

CREATE TABLE "test-db".parent_inherits_gp (
    parent_col INTEGER
) INHERITS ("test-db".grandparent_table);

CREATE TABLE "test-db".child_inherits_parent (
    child_col BOOLEAN
) INHERITS ("test-db".parent_inherits_gp);

-- Multiple inheritance: inherits from two parents
CREATE TABLE "test-db".mixin_a (
    mixin_a_col VARCHAR(50)
);

CREATE TABLE "test-db".mixin_b (
    mixin_b_col TIMESTAMPTZ
);

CREATE TABLE "test-db".multi_inherit (
    own_col JSONB
) INHERITS ("test-db".mixin_a, "test-db".mixin_b);

-- Parent with multiple children
CREATE TABLE "test-db".shared_parent (
    shared_col UUID
);

CREATE TABLE "test-db".child_one (
    child_one_col INTEGER
) INHERITS ("test-db".shared_parent);

CREATE TABLE "test-db".child_two (
    child_two_col TEXT
) INHERITS ("test-db".shared_parent);
`

const partitionedTablesDDL = `
CREATE TABLE "test-db".partitioned_table (
    id SERIAL,
    created_at DATE NOT NULL,
    data TEXT,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE "test-db".partitioned_table_2024 PARTITION OF "test-db".partitioned_table
    FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');

CREATE TABLE "test-db".partitioned_table_2025 PARTITION OF "test-db".partitioned_table
    FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');

-- Index on partitioned table (propagates to partitions)
CREATE INDEX idx_partitioned_data ON "test-db".partitioned_table (data);

-- Index on a specific partition
CREATE INDEX idx_special_data ON "test-db".partitioned_table_2025 (data);
`

const testSequenceDDL = `
CREATE SEQUENCE "test-db".test_seq;
`

const testFunctionsDDL = `
CREATE FUNCTION "test-db".sum(a INT, b INT)
RETURNS INT AS $$
BEGIN
    RETURN a + b;
END; $$ LANGUAGE plpgsql;

CREATE FUNCTION "test-db".sum(a INT)
RETURNS INT AS $$
BEGIN
    RETURN a + a;
END; $$ LANGUAGE plpgsql;
`

const fooTableDDL = `
CREATE TABLE "test-db".foo();
`

const specialCharsTableDDL = `
CREATE TABLE "test-db"."Table-With-Dashes" (
    "Column With Spaces" TEXT,
    "column-with-dashes" INTEGER,
    "MixedCase" BOOLEAN
);
COMMENT ON TABLE "test-db"."Table-With-Dashes" IS 'Table with special characters in identifiers';
`
