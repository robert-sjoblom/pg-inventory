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
