package store

import (
	"context"
	"testing"

	"github.com/robert-sjoblom/pg-inventory/internal/extractor/types"
	"github.com/robert-sjoblom/pg-inventory/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicTable(t *testing.T) {
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

	ctx := context.Background()
	role := testutil.ExtraRole{
		Database: "test-db",
		Schema:   "test-db",
		Role:     "test-db-owner",
		Password: "password",
	}
	creds := testutil.StartPostgres(t, testutil.WithExtraRoles(role), testutil.WithExtraTables(
		testutil.ExtraTable{
			Role:     &role,
			Schema:   "test-db",
			Database: "test-db",
			DDL:      basicTableDDL,
		},
	))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	_, err := pool.Exec(ctx, "CREATE TABLE foo();")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	store, err := NewStore(pool, creds.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed")
	}

	actual, err := store.ListTables(ctx)
	if err != nil {
		t.Fatalf("failed to query ListTables: %v", err)
	}

	assert.Len(t, actual, 2)

	dbNames := make([]string, len(actual))
	for i, db := range actual {
		dbNames[i] = db.Database
	}
	assert.ElementsMatch(t, []string{"postgres", "test-db"}, dbNames)

	var postgresDB *types.TablesInfo
	for _, db := range actual {
		if db.Database == "postgres" {
			postgresDB = db
			break
		}
	}
	require.NotNil(t, postgresDB, "postgres database should be present")

	tableNames := make([]string, len(postgresDB.Tables))
	for i, table := range postgresDB.Tables {
		tableNames[i] = table.Schema + "." + table.Name
	}
	assert.Contains(t, tableNames, "public.foo")

	var fooTable *types.Table
	for _, table := range postgresDB.Tables {
		if table.Name == "foo" && table.Schema == "public" {
			fooTable = table
			break
		}
	}
	require.NotNil(t, fooTable, "foo table should exist in postgres.public")
	assert.Equal(t, "postgres", fooTable.Owner)
	assert.Nil(t, fooTable.Comment)
	assert.Empty(t, fooTable.TableColumns, "foo table has no columns")
	assert.Empty(t, fooTable.TableIndexes, "foo table has no indexes")
	assert.Empty(t, fooTable.TableConstraints, "foo table has no constraints")

	var testDB *types.TablesInfo
	for _, db := range actual {
		if db.Database == "test-db" {
			testDB = db
			break
		}
	}
	require.NotNil(t, testDB, "test-db database should be present")

	var basicTable *types.Table
	for _, table := range testDB.Tables {
		if table.Name == "basic_table" && table.Schema == "test-db" {
			basicTable = table
			break
		}
	}
	require.NotNil(t, basicTable, "basic_table should exist in test-db.test-db")

	assert.Equal(t, "test-db-owner", basicTable.Owner)
	assert.NotNil(t, basicTable.Comment)
	assert.Equal(t, "Basic table with PK, unique, check constraint", *basicTable.Comment)

	require.Len(t, basicTable.TableColumns, 5)
	columnNames := make([]string, len(basicTable.TableColumns))
	for i, col := range basicTable.TableColumns {
		columnNames[i] = col.Name
	}
	assert.ElementsMatch(t, []string{"id", "name", "email", "created_at", "metadata"}, columnNames)

	var nameCol, emailCol *types.TableColumn
	for _, col := range basicTable.TableColumns {
		if col.Name == "name" {
			nameCol = col
		}
		if col.Name == "email" {
			emailCol = col
		}
	}
	require.NotNil(t, nameCol)
	assert.True(t, nameCol.NotNull, "name column should be NOT NULL")
	assert.Equal(t, "character varying(255)", nameCol.Type)

	require.NotNil(t, emailCol)
	assert.False(t, emailCol.NotNull, "email column should be nullable")
	assert.Equal(t, "text", emailCol.Type)

	require.GreaterOrEqual(t, len(basicTable.TableIndexes), 4, "should have at least 4 indexes (PK + 3 explicit)")
	indexNames := make([]string, len(basicTable.TableIndexes))
	for i, idx := range basicTable.TableIndexes {
		indexNames[i] = idx.Name
	}
	assert.Contains(t, indexNames, "basic_table_pkey")
	assert.Contains(t, indexNames, "idx_basic_metadata_gin")
	assert.Contains(t, indexNames, "idx_basic_name_lower")
	assert.Contains(t, indexNames, "idx_basic_email_unique")

	var pkIndex *types.TableIndex
	for _, idx := range basicTable.TableIndexes {
		if idx.IsPrimary {
			pkIndex = idx
			break
		}
	}
	require.NotNil(t, pkIndex, "should have primary key index")
	assert.True(t, pkIndex.IsUnique)
	assert.True(t, pkIndex.IsValid)
	assert.ElementsMatch(t, []string{"id"}, pkIndex.Columns)

	var partialIdx *types.TableIndex
	for _, idx := range basicTable.TableIndexes {
		if idx.Name == "idx_basic_email_unique" {
			partialIdx = idx
			break
		}
	}
	require.NotNil(t, partialIdx, "partial unique index should exist")
	assert.True(t, partialIdx.IsUnique)
	assert.True(t, partialIdx.IsPartial, "should be a partial index")
	assert.ElementsMatch(t, []string{"email"}, partialIdx.Columns)

	require.GreaterOrEqual(t, len(basicTable.TableConstraints), 3, "should have at least 3 constraints (PK, unique, check)")
	constraintNames := make([]string, len(basicTable.TableConstraints))
	for i, con := range basicTable.TableConstraints {
		constraintNames[i] = con.Name
	}
	assert.Contains(t, constraintNames, "basic_table_pkey")
	assert.Contains(t, constraintNames, "basic_table_email_key")
	assert.Contains(t, constraintNames, "name_not_empty")

	var checkConstraint *types.TableConstraint
	for _, con := range basicTable.TableConstraints {
		if con.Name == "name_not_empty" {
			checkConstraint = con
			break
		}
	}
	require.NotNil(t, checkConstraint, "check constraint should exist")
	assert.Equal(t, "check", checkConstraint.Type)
	assert.Contains(t, checkConstraint.Definition, "length")
}

func TestListTablesNoPrimaryKey(t *testing.T) {
	const noPkTableDDL = `
	CREATE TABLE "test-db".no_pk_table (
		data TEXT,
		value INTEGER
	);
	COMMENT ON TABLE "test-db".no_pk_table IS 'Table without primary key';
	`

	ctx := context.Background()
	role := testutil.ExtraRole{
		Database: "test-db",
		Schema:   "test-db",
		Role:     "test-db-owner",
		Password: "password",
	}
	creds := testutil.StartPostgres(t, testutil.WithExtraRoles(role), testutil.WithExtraTables(
		testutil.ExtraTable{
			Role:     &role,
			Schema:   "test-db",
			Database: "test-db",
			DDL:      noPkTableDDL,
		},
	))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed")
	}

	actual, err := store.ListTables(ctx)
	if err != nil {
		t.Fatalf("failed to query ListTables: %v", err)
	}

	var testDB *types.TablesInfo
	for _, db := range actual {
		if db.Database == "test-db" {
			testDB = db
			break
		}
	}
	require.NotNil(t, testDB, "test-db database should be present")

	var noPkTable *types.Table
	for _, table := range testDB.Tables {
		if table.Name == "no_pk_table" && table.Schema == "test-db" {
			noPkTable = table
			break
		}
	}
	require.NotNil(t, noPkTable, "no_pk_table should exist in test-db.test-db")

	for _, idx := range noPkTable.TableIndexes {
		assert.False(t, idx.IsPrimary, "no_pk_table should not have a primary key index")
	}

	for _, con := range noPkTable.TableConstraints {
		assert.NotEqual(t, "primary_key", con.Type, "no_pk_table should not have a primary key constraint")
	}

	require.Len(t, noPkTable.TableColumns, 2)
	columnNames := make([]string, len(noPkTable.TableColumns))
	for i, col := range noPkTable.TableColumns {
		columnNames[i] = col.Name
	}
	assert.ElementsMatch(t, []string{"data", "value"}, columnNames)

	require.NotNil(t, noPkTable.Comment)
	assert.Equal(t, "Table without primary key", *noPkTable.Comment)
}

func TestListTablesCompositePrimaryKey(t *testing.T) {
	const compositePkTableDDL = `
	CREATE TABLE "test-db".composite_pk_table (
		tenant_id UUID NOT NULL,
		record_id BIGINT NOT NULL,
		data TEXT,
		PRIMARY KEY (tenant_id, record_id)
	);
	`

	ctx := context.Background()
	role := testutil.ExtraRole{
		Database: "test-db",
		Schema:   "test-db",
		Role:     "test-db-owner",
		Password: "password",
	}
	creds := testutil.StartPostgres(t, testutil.WithExtraRoles(role), testutil.WithExtraTables(
		testutil.ExtraTable{
			Role:     &role,
			Schema:   "test-db",
			Database: "test-db",
			DDL:      compositePkTableDDL,
		},
	))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed")
	}

	actual, err := store.ListTables(ctx)
	if err != nil {
		t.Fatalf("failed to query ListTables: %v", err)
	}

	var testDB *types.TablesInfo
	for _, db := range actual {
		if db.Database == "test-db" {
			testDB = db
			break
		}
	}
	require.NotNil(t, testDB, "test-db database should be present")

	var compositePkTable *types.Table
	for _, table := range testDB.Tables {
		if table.Name == "composite_pk_table" && table.Schema == "test-db" {
			compositePkTable = table
			break
		}
	}
	require.NotNil(t, compositePkTable, "composite_pk_table should exist in test-db.test-db")

	require.Len(t, compositePkTable.TableColumns, 3)
	columnNames := make([]string, len(compositePkTable.TableColumns))
	for i, col := range compositePkTable.TableColumns {
		columnNames[i] = col.Name
	}
	assert.ElementsMatch(t, []string{"tenant_id", "record_id", "data"}, columnNames)

	var pkIndex *types.TableIndex
	for _, idx := range compositePkTable.TableIndexes {
		if idx.IsPrimary {
			pkIndex = idx
			break
		}
	}
	require.NotNil(t, pkIndex, "should have primary key index")
	assert.True(t, pkIndex.IsUnique, "composite PK should be unique")
	assert.True(t, pkIndex.IsPrimary, "should be marked as primary")
	assert.True(t, pkIndex.IsValid, "PK index should be valid")
	assert.ElementsMatch(t, []string{"tenant_id", "record_id"}, pkIndex.Columns, "PK should have both columns in order")

	var pkConstraint *types.TableConstraint
	for _, con := range compositePkTable.TableConstraints {
		if con.Type == "primary_key" {
			pkConstraint = con
			break
		}
	}
	require.NotNil(t, pkConstraint, "should have primary key constraint")
	assert.ElementsMatch(t, []string{"tenant_id", "record_id"}, pkConstraint.LocalColumns, "PK constraint should include both columns")
}

func TestListTablesForeignKeys(t *testing.T) {
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

	CREATE TABLE "test-db".composite_pk_table (
		tenant_id UUID NOT NULL,
		record_id BIGINT NOT NULL,
		data TEXT,
		PRIMARY KEY (tenant_id, record_id)
	);

	CREATE TABLE "test-db".composite_fk_table (
		id SERIAL PRIMARY KEY,
		tenant_id UUID NOT NULL,
		record_id BIGINT NOT NULL,
		CONSTRAINT fk_composite FOREIGN KEY (tenant_id, record_id)
			REFERENCES "test-db".composite_pk_table(tenant_id, record_id)
	);
	`

	ctx := context.Background()
	role := testutil.ExtraRole{
		Database: "test-db",
		Schema:   "test-db",
		Role:     "test-db-owner",
		Password: "password",
	}
	creds := testutil.StartPostgres(t, testutil.WithExtraRoles(role), testutil.WithExtraTables(
		testutil.ExtraTable{
			Role:     &role,
			Schema:   "test-db",
			Database: "test-db",
			DDL:      foreignKeyTablesDDL,
		},
	))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed")
	}

	actual, err := store.ListTables(ctx)
	if err != nil {
		t.Fatalf("failed to query ListTables: %v", err)
	}

	var testDB *types.TablesInfo
	for _, db := range actual {
		if db.Database == "test-db" {
			testDB = db
			break
		}
	}
	require.NotNil(t, testDB, "test-db database should be present")

	var childTable *types.Table
	for _, table := range testDB.Tables {
		if table.Name == "child_table" && table.Schema == "test-db" {
			childTable = table
			break
		}
	}
	require.NotNil(t, childTable, "child_table should exist")

	var fkParent *types.TableConstraint
	for _, con := range childTable.TableConstraints {
		if con.Name == "fk_parent" {
			fkParent = con
			break
		}
	}
	require.NotNil(t, fkParent, "fk_parent constraint should exist")
	assert.Equal(t, "foreign_key", fkParent.Type)
	assert.ElementsMatch(t, []string{"parent_id"}, fkParent.LocalColumns)
	assert.Contains(t, fkParent.ForeignTable, "parent_table", "should reference parent_table")
	assert.ElementsMatch(t, []string{"id"}, fkParent.ForeignColumns)
	assert.NotEmpty(t, fkParent.Definition, "should have constraint definition")

	require.Len(t, childTable.TableColumns, 3)
	columnNames := make([]string, len(childTable.TableColumns))
	for i, col := range childTable.TableColumns {
		columnNames[i] = col.Name
	}
	assert.ElementsMatch(t, []string{"id", "parent_id", "child_name"}, columnNames)

	var compositeFkTable *types.Table
	for _, table := range testDB.Tables {
		if table.Name == "composite_fk_table" && table.Schema == "test-db" {
			compositeFkTable = table
			break
		}
	}
	require.NotNil(t, compositeFkTable, "composite_fk_table should exist")

	var fkComposite *types.TableConstraint
	for _, con := range compositeFkTable.TableConstraints {
		if con.Name == "fk_composite" {
			fkComposite = con
			break
		}
	}
	require.NotNil(t, fkComposite, "fk_composite constraint should exist")
	assert.Equal(t, "foreign_key", fkComposite.Type)
	assert.ElementsMatch(t, []string{"tenant_id", "record_id"}, fkComposite.LocalColumns)
	assert.Contains(t, fkComposite.ForeignTable, "composite_pk_table", "should reference composite_pk_table")
	assert.ElementsMatch(t, []string{"tenant_id", "record_id"}, fkComposite.ForeignColumns)

	var parentTable *types.Table
	for _, table := range testDB.Tables {
		if table.Name == "parent_table" && table.Schema == "test-db" {
			parentTable = table
			break
		}
	}
	require.NotNil(t, parentTable, "parent_table should exist")

	for _, con := range parentTable.TableConstraints {
		assert.NotEqual(t, "foreign_key", con.Type, "parent_table should not have foreign key constraints")
	}

	var parentPK *types.TableConstraint
	for _, con := range parentTable.TableConstraints {
		if con.Type == "primary_key" {
			parentPK = con
			break
		}
	}
	require.NotNil(t, parentPK, "parent_table should have primary key")
	assert.ElementsMatch(t, []string{"id"}, parentPK.LocalColumns)
}
