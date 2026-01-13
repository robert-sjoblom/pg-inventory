//go:build integration

package store

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/types"
	"github.com/robert-sjoblom/pg-inventory/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sharedCredentials *testutil.TestDbCredentials

func TestMain(m *testing.M) {
	role := testutil.ExtraRole{
		Database: "test-db",
		Schema:   "test-db",
		Role:     "test-db-owner",
		Password: "password",
	}

	ctx := context.Background()

	var err error
	sharedCredentials, err = testutil.StartSharedPostgres(ctx,
		testutil.WithClusterConfig("test-cluster", "test-stanza"),
		testutil.WithExtraDatabases("testdb", "app-db"),
		testutil.WithExtraRoles(role),
		testutil.WithExtraSchemas(
			testutil.ExtraSchema{Name: "testdb-schema", Database: "testdb"},
			testutil.ExtraSchema{Name: "app-db-schema", Database: "app-db"},
		),
		testutil.WithInstalledExtensions(
			testutil.InstalledExtension{Name: "pg_trgm", Database: "testdb"},
		),
		testutil.WithExtraTables(
			testutil.ExtraTable{
				Role:     &role,
				Schema:   "test-db",
				Database: "test-db",
				DDL:      basicTableDDL,
			},
		),
		testutil.WithExtraTables(
			testutil.ExtraTable{
				Role:     &role,
				Schema:   "test-db",
				Database: "test-db",
				DDL:      noPkTableDDL,
			},
		),
		testutil.WithExtraTables(
			testutil.ExtraTable{
				Role:     &role,
				Schema:   "test-db",
				Database: "test-db",
				DDL:      compositePkTableDDL,
			},
		),
		testutil.WithExtraTables(
			testutil.ExtraTable{
				Role:     &role,
				Schema:   "test-db",
				Database: "test-db",
				DDL:      foreignKeyTablesDDL,
			},
		),
		testutil.WithExtraTables(
			testutil.ExtraTable{
				Role:     &role,
				Schema:   "test-db",
				Database: "test-db",
				DDL:      allColumnDataTypes,
			},
		),
		testutil.WithExtraTables(
			testutil.ExtraTable{
				Role:     &role,
				Schema:   "test-db",
				Database: "test-db",
				DDL:      indexTypes,
			},
		),
		testutil.WithExtraTables(
			testutil.ExtraTable{
				Role:     &role,
				Schema:   "test-db",
				Database: "test-db",
				DDL:      emptyTable,
			},
		),
		testutil.WithExtraTables(
			testutil.ExtraTable{
				Role:     &role,
				Schema:   "test-db",
				Database: "test-db",
				DDL:      droppedColumnsTable,
			},
		),
		testutil.WithExtraTables(
			testutil.ExtraTable{
				Role:     &role,
				Schema:   "test-db",
				Database: "test-db",
				DDL:      inheritanceTables,
			},
		),
	)
	if err != nil {
		log.Fatalf("failed to start shared postgres container: %v", err)
	}

	defer sharedCredentials.Terminate(ctx)

	code := m.Run()

	os.Exit(code)
}

func findTableInDb(tablesInfo []*types.TablesInfo, dbName, schema, tableName string) *types.Table {
	db := findDb(tablesInfo, dbName)
	for _, table := range db.Tables {
		if table.Name == tableName && table.Schema == schema {
			return table
		}
	}
	return nil
}

func findDb(tablesInfo []*types.TablesInfo, dbName string) *types.TablesInfo {
	for _, db := range tablesInfo {
		if db.Database == dbName {
			return db
		}
	}
	return nil
}

func setupStore(t *testing.T) (context.Context, *Store) {
	t.Helper()
	ctx := context.Background()
	pool := testutil.ConnectToDatabase(t, sharedCredentials, "postgres")

	store, err := NewStore(pool, sharedCredentials.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed: %v", err)
	}

	return ctx, store
}

func setupStoreAndListTables(t *testing.T) (context.Context, *Store, []*types.TablesInfo) {
	t.Helper()
	ctx, store := setupStore(t)

	actual, err := store.ListTables(ctx)
	if err != nil {
		t.Fatalf("failed to query ListTables: %v", err)
	}

	return ctx, store, actual
}

func TestGetServerInfo(t *testing.T) {
	ctx := context.Background()

	pool := testutil.ConnectToDatabase(t, sharedCredentials, "postgres")

	store, err := NewStore(pool, sharedCredentials.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed")
	}

	var systemIdentifier int64
	err = pool.QueryRow(ctx, "SELECT system_identifier FROM pg_control_system()").Scan(&systemIdentifier)
	if err != nil {
		t.Fatalf("failed to get system identifier: %v", err)
	}

	actual, err := store.GetServerInfo(ctx)
	if err != nil {
		t.Fatalf("GetServerInfo failed: %v", err)
	}

	expected := types.ServerInfo{
		PgVersion:        "PostgreSQL 15.13 (Debian 15.13-1.pgdg110+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 10.2.1-6) 10.2.1 20210110, 64-bit",
		IsInRecovery:     false,
		IsReadOnly:       "off",
		SslEnabled:       "off",
		Port:             5432,
		MaxConnections:   100,
		ArchiveMode:      "off",
		DataDirectory:    "/var/lib/postgresql/data",
		SystemIdentifier: systemIdentifier, // This is unknowable until the container spins up and starts PG
		TimelineID:       1,
		WalLevel:         "replica",
	}

	assert.Equal(t, expected, actual)
}

func TestNewStore(t *testing.T) {
	pool := testutil.ConnectToDatabase(t, sharedCredentials, "postgres")

	store, err := NewStore(pool, sharedCredentials.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed")
	}

	assert.Equal(t, "test-cluster", store.ClusterName)
	assert.Equal(t, "test-stanza", store.Stanza)
}

func TestNewStoreInvalidStanza(t *testing.T) {
	creds := testutil.StartPostgres(t, testutil.WithClusterConfig("another-name", "0000-name"))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid stanza name")
	assert.Nil(t, store, "store should be nil when initialization fails")

}

func TestListDatabases(t *testing.T) {
	ctx := context.Background()
	pool := testutil.ConnectToDatabase(t, sharedCredentials, "postgres")

	store, err := NewStore(pool, sharedCredentials.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed")
	}

	databases, err := store.ListDatabases(ctx)
	if err != nil {
		t.Fatalf("ListDatabases failed: %v", err)
	}

	if len(databases) == 0 {
		t.Fatalf("expected at least one database, got none")
	}

	found := false
	for _, db := range databases {
		if db.Name == "postgres" {
			found = true
			if db.Oid == 0 {
				t.Error("postgres database has invalid OID 0")
			}
			break
		}
	}

	if !found {
		t.Error("expected to find 'postgres' database")
	}
}

func TestListSchemas(t *testing.T) {
	ctx, store := setupStore(t)

	actual, err := store.ListSchemas(ctx)
	if err != nil {
		t.Fatalf("GetServerInfo failed: %v", err)
	}

	schemaMap := make(map[string]*types.Schema)
	for _, schema := range actual {
		key := schema.Database + "." + schema.Name
		schemaMap[key] = schema
	}

	assert.Contains(t, schemaMap, "postgres.public", "postgres.public schema should exist")
	assert.Contains(t, schemaMap, "postgres.monitoring", "postgres.monitoring schema should exist")
	assert.Contains(t, schemaMap, "testdb.public", "testdb.public schema should exist")
	assert.Contains(t, schemaMap, "testdb.testdb-schema", "testdb.testdb-schema should exist")
	assert.Contains(t, schemaMap, "app-db.public", "app-db.public schema should exist")
	assert.Contains(t, schemaMap, "app-db.app-db-schema", "app-db.app-db-schema should exist")

	assert.Equal(t, "postgres", schemaMap["postgres.monitoring"].Owner)
	assert.Equal(t, "pg_database_owner", schemaMap["postgres.public"].Owner)
}

func TestListExtensions(t *testing.T) {
	ctx, store := setupStore(t)

	available, installed, err := store.ListExtensions(ctx)
	if err != nil {
		t.Fatalf("ListExtensions failed: %v", err)
	}

	assert.GreaterOrEqual(t, len(available), 40, "should have many available extensions")

	installedMap := make(map[string]*types.InstalledExtension)
	for _, ext := range installed {
		key := ext.Database + "." + ext.Name
		installedMap[key] = ext
	}

	assert.Contains(t, installedMap, "postgres.plpgsql", "plpgsql should be in postgres")
	assert.Contains(t, installedMap, "testdb.plpgsql", "plpgsql should be in testdb")
	assert.Contains(t, installedMap, "app-db.plpgsql", "plpgsql should be in app-db")

	require.Contains(t, installedMap, "testdb.pg_trgm", "pg_trgm should be in testdb")
	assert.Equal(t, "1.6", installedMap["testdb.pg_trgm"].Version)
	assert.Equal(t, "public", installedMap["testdb.pg_trgm"].Schema)
}

func TestListSequences(t *testing.T) {
	ctx, store := setupStore(t)

	_, err := store.pool.Exec(ctx, "CREATE SEQUENCE test_seq")
	if err != nil {
		t.Fatalf("failed to create sequence: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), "DROP SEQUENCE IF EXISTS test_seq")
	})

	actual, err := store.ListSequences(ctx)
	if err != nil {
		t.Fatalf("failed to list sequences: %v", err)
	}

	var testSeq *types.Sequence
	for _, seq := range actual {
		if seq.Name == "test_seq" && seq.Database == "postgres" && seq.Schema == "public" {
			testSeq = seq
			break
		}
	}

	require.NotNil(t, testSeq, "test_seq should exist")
	assert.Equal(t, "postgres", testSeq.Owner)
	assert.Equal(t, "int8", testSeq.DataType)
}

func TestListFunctions(t *testing.T) {
	ctx, store := setupStore(t)

	function := `
	CREATE FUNCTION sum(a INT, b INT)
	RETURNS INT AS $$
	BEGIN
	RETURN a + b;
	END; $$ LANGUAGE plpgsql;`

	function2 := `
	CREATE FUNCTION sum(a INT)
	RETURNS INT AS $$
	BEGIN
	RETURN a + a;
	END; $$ LANGUAGE plpgsql;`

	_, err := store.pool.Exec(ctx, function)
	if err != nil {
		t.Fatalf("failed to create function: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), "DROP FUNCTION IF EXISTS sum(INT, INT)")
	})

	_, err = store.pool.Exec(ctx, function2)
	if err != nil {
		t.Fatalf("failed to create function: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), "DROP FUNCTION IF EXISTS sum(INT)")
	})

	actual, err := store.ListFunctions(ctx)
	if err != nil {
		t.Fatalf("failed to list functions: %v", err)
	}

	var sumFunctions []*types.Function
	for _, fn := range actual {
		if fn.Name == "sum" && fn.Database == "postgres" && fn.Schema == "public" {
			sumFunctions = append(sumFunctions, fn)
		}
	}

	require.Len(t, sumFunctions, 2, "should have exactly 2 sum functions")

	var sum2Args, sum1Arg *types.Function
	for _, fn := range sumFunctions {
		switch fn.IdentityArguments {
		case "a integer, b integer":
			sum2Args = fn
		case "a integer":
			sum1Arg = fn
		}
	}

	require.NotNil(t, sum2Args, "sum(a integer, b integer) should exist")
	assert.Equal(t, "plpgsql", sum2Args.Language)
	assert.Equal(t, "integer", sum2Args.ReturnType)
	assert.Equal(t, "postgres", sum2Args.Owner)

	require.NotNil(t, sum1Arg, "sum(a integer) should exist")
	assert.Equal(t, "plpgsql", sum1Arg.Language)
	assert.Equal(t, "integer", sum1Arg.ReturnType)
	assert.Equal(t, "postgres", sum1Arg.Owner)
}

func TestBasicTable(t *testing.T) {
	ctx, store := setupStore(t)

	_, err := store.pool.Exec(ctx, "CREATE TABLE foo();")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	t.Cleanup(func() {
		store.pool.Exec(context.Background(), "DROP TABLE IF EXISTS foo")
	})

	actual, err := store.ListTables(ctx)
	if err != nil {
		t.Fatalf("failed to query ListTables: %v", err)
	}

	// Shared container has multiple databases, just verify the ones we need exist
	assert.GreaterOrEqual(t, len(actual), 2, "should have at least postgres and test-db databases")

	postgresDB := findDb(actual, "postgres")
	require.NotNil(t, postgresDB, "postgres database should be present")

	tableNames := make([]string, len(postgresDB.Tables))
	for i, table := range postgresDB.Tables {
		tableNames[i] = table.Schema + "." + table.Name
	}
	assert.Contains(t, tableNames, "public.foo")

	fooTable := findTableInDb(actual, "postgres", "public", "foo")
	require.NotNil(t, fooTable, "foo table should exist in postgres.public")
	assert.Equal(t, "postgres", fooTable.Owner)
	assert.Nil(t, fooTable.Comment)
	assert.Empty(t, fooTable.TableColumns, "foo table has no columns")
	assert.Empty(t, fooTable.TableIndexes, "foo table has no indexes")
	assert.Empty(t, fooTable.TableConstraints, "foo table has no constraints")

	testDB := findDb(actual, "test-db")
	require.NotNil(t, testDB, "test-db database should be present")

	basicTable := findTableInDb(actual, "test-db", "test-db", "basic_table")
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
	_, _, actual := setupStoreAndListTables(t)

	noPkTable := findTableInDb(actual, "test-db", "test-db", "no_pk_table")
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
	_, _, actual := setupStoreAndListTables(t)

	compositePkTable := findTableInDb(actual, "test-db", "test-db", "composite_pk_table")
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
	_, _, actual := setupStoreAndListTables(t)

	childTable := findTableInDb(actual, "test-db", "test-db", "child_table")
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

	compositeFkTable := findTableInDb(actual, "test-db", "test-db", "composite_fk_table")
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

	parentTable := findTableInDb(actual, "test-db", "test-db", "parent_table")
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

func TestListTablesAllColumnTypes(t *testing.T) {
	_, _, actual := setupStoreAndListTables(t)

	allTypesTable := findTableInDb(actual, "test-db", "test-db", "all_column_types")
	require.NotNil(t, allTypesTable, "all_column_types should exist in test-db.test-db")

	require.Len(t, allTypesTable.TableColumns, 36, "should have 36 columns covering all data types")

	columnTypes := make(map[string]string)
	for _, col := range allTypesTable.TableColumns {
		columnTypes[col.Name] = col.Type
	}

	assert.Equal(t, "smallint", columnTypes["col_smallint"])
	assert.Equal(t, "integer", columnTypes["col_integer"])
	assert.Equal(t, "bigint", columnTypes["col_bigint"])
	assert.Equal(t, "numeric(10,2)", columnTypes["col_decimal"])
	assert.Equal(t, "numeric(15,5)", columnTypes["col_numeric"])
	assert.Equal(t, "real", columnTypes["col_real"])
	assert.Equal(t, "double precision", columnTypes["col_double"])
	assert.Equal(t, "integer", columnTypes["col_serial"])   // SERIAL is stored as integer
	assert.Equal(t, "bigint", columnTypes["col_bigserial"]) // BIGSERIAL is stored as bigint

	// SERIAL and BIGSERIAL have implicit NOT NULL constraints
	var serialCol, bigserialCol *types.TableColumn
	for _, col := range allTypesTable.TableColumns {
		switch col.Name {
		case "col_serial":
			serialCol = col
		case "col_bigserial":
			bigserialCol = col
		}
	}
	require.NotNil(t, serialCol)
	require.NotNil(t, bigserialCol)
	assert.True(t, serialCol.NotNull, "SERIAL columns have implicit NOT NULL")
	assert.True(t, bigserialCol.NotNull, "BIGSERIAL columns have implicit NOT NULL")

	assert.Equal(t, "character(10)", columnTypes["col_char"])
	assert.Equal(t, "character varying(255)", columnTypes["col_varchar"])
	assert.Equal(t, "text", columnTypes["col_text"])
	assert.Equal(t, "bytea", columnTypes["col_bytea"])
	assert.Equal(t, "timestamp without time zone", columnTypes["col_timestamp"])
	assert.Equal(t, "timestamp with time zone", columnTypes["col_timestamptz"])
	assert.Equal(t, "date", columnTypes["col_date"])
	assert.Equal(t, "time without time zone", columnTypes["col_time"])
	assert.Equal(t, "time with time zone", columnTypes["col_timetz"])
	assert.Equal(t, "interval", columnTypes["col_interval"])
	assert.Equal(t, "boolean", columnTypes["col_boolean"])
	assert.Equal(t, "uuid", columnTypes["col_uuid"])
	assert.Equal(t, "inet", columnTypes["col_inet"])
	assert.Equal(t, "cidr", columnTypes["col_cidr"])
	assert.Equal(t, "macaddr", columnTypes["col_macaddr"])
	assert.Equal(t, "json", columnTypes["col_json"])
	assert.Equal(t, "jsonb", columnTypes["col_jsonb"])
	assert.Equal(t, "integer[]", columnTypes["col_int_array"])
	assert.Equal(t, "text[]", columnTypes["col_text_array"])
	assert.Equal(t, "jsonb[]", columnTypes["col_jsonb_array"])
	assert.Equal(t, "int4range", columnTypes["col_int4range"])
	assert.Equal(t, "tsrange", columnTypes["col_tsrange"])
	assert.Equal(t, "daterange", columnTypes["col_daterange"])
	assert.Equal(t, "point", columnTypes["col_point"])
	assert.Equal(t, "box", columnTypes["col_box"])
	assert.Equal(t, "tsvector", columnTypes["col_tsvector"])
	assert.Equal(t, "tsquery", columnTypes["col_tsquery"])

	// Verify all non-SERIAL columns are nullable
	for _, col := range allTypesTable.TableColumns {
		if col.Name != "col_serial" && col.Name != "col_bigserial" {
			assert.False(t, col.NotNull, "column %s should be nullable", col.Name)
		}
	}
}

func TestListTablesToastTable(t *testing.T) {
	const toastTableDDL = `
	CREATE TABLE public.toast_table (
		id SERIAL PRIMARY KEY,
		large_text TEXT,
		large_jsonb JSONB
	);
	ALTER TABLE public.toast_table ALTER COLUMN large_text SET STORAGE EXTERNAL;
	
	INSERT INTO public.toast_table (large_text, large_jsonb)
	SELECT
		string_agg(md5(random()::text), '') || repeat('x', 10000),
		jsonb_build_object('data', string_agg(md5(random()::text), ''))
	FROM generate_series(1, 100) i
	CROSS JOIN generate_series(1, 300) j
	GROUP BY i;
	
	-- Analyze to update statistics
	ANALYZE public.toast_table;
	`

	ctx := context.Background()
	credentials := testutil.StartPostgres(t)

	pool, err := pgxpool.New(ctx, credentials.ConnStr("postgres"))
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}

	_, err = pool.Exec(ctx, toastTableDDL)
	if err != nil {
		t.Fatalf("failed to create toasted table: %v", err)
	}

	store, err := NewStore(pool, credentials.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed")
	}

	actual, err := store.ListTables(ctx)
	if err != nil {
		t.Fatalf("failed to query ListTables: %v", err)
	}

	toastTable := findTableInDb(actual, "postgres", "public", "toast_table")
	require.NotNil(t, toastTable, "toast_table should exist in postgres.public")

	assert.Equal(t, toastTable.Stats.RowEstimate, int64(100))
	assert.Equal(t, toastTable.Stats.TotalSizeBytes, uint64(3170304))
	assert.Equal(t, toastTable.Stats.HeapSizeBytes, uint64(8192))
	assert.Equal(t, toastTable.Stats.ToastSizeBytes, uint64(3072000))
}

func TestListTablesIndexTypes(t *testing.T) {
	_, _, actual := setupStoreAndListTables(t)

	indexTypesTable := findTableInDb(actual, "test-db", "test-db", "index_types_table")
	require.NotNil(t, indexTypesTable, "index_types_table should exist in test-db.test-db")

	// Should have primary key + 9 explicit indexes
	require.GreaterOrEqual(t, len(indexTypesTable.TableIndexes), 10, "should have at least 10 indexes")

	indexMap := make(map[string]*types.TableIndex)
	for _, idx := range indexTypesTable.TableIndexes {
		indexMap[idx.Name] = idx
	}

	testCases := []struct {
		name        string
		indexType   string
		columns     []string
		isPrimary   bool
		isUnique    bool
		isPartial   bool
		shouldExist bool
	}{
		{
			name:        "index_types_table_pkey",
			indexType:   "btree",
			columns:     []string{"id"},
			isPrimary:   true,
			isUnique:    true,
			isPartial:   false,
			shouldExist: true,
		},
		{
			name:        "idx_btree",
			indexType:   "btree",
			columns:     []string{"int_col"},
			isPrimary:   false,
			isUnique:    false,
			isPartial:   false,
			shouldExist: true,
		},
		{
			name:        "idx_hash",
			indexType:   "hash",
			columns:     []string{"int_col"},
			isPrimary:   false,
			isUnique:    false,
			isPartial:   false,
			shouldExist: true,
		},
		{
			name:        "idx_gin_jsonb",
			indexType:   "gin",
			columns:     []string{"jsonb_col"},
			isPrimary:   false,
			isUnique:    false,
			isPartial:   false,
			shouldExist: true,
		},
		{
			name:        "idx_gin_array",
			indexType:   "gin",
			columns:     []string{"array_col"},
			isPrimary:   false,
			isUnique:    false,
			isPartial:   false,
			shouldExist: true,
		},
		{
			name:        "idx_gin_tsvector",
			indexType:   "gin",
			columns:     []string{"tsvector_col"},
			isPrimary:   false,
			isUnique:    false,
			isPartial:   false,
			shouldExist: true,
		},
		{
			name:        "idx_gist_point",
			indexType:   "gist",
			columns:     []string{"point_col"},
			isPrimary:   false,
			isUnique:    false,
			isPartial:   false,
			shouldExist: true,
		},
		{
			name:        "idx_spgist_text",
			indexType:   "spgist",
			columns:     []string{"text_col"},
			isPrimary:   false,
			isUnique:    false,
			isPartial:   false,
			shouldExist: true,
		},
		{
			name:        "idx_brin",
			indexType:   "brin",
			columns:     []string{"id"},
			isPrimary:   false,
			isUnique:    false,
			isPartial:   false,
			shouldExist: true,
		},
		{
			name:        "idx_partial",
			indexType:   "btree",
			columns:     []string{"text_col"},
			isPrimary:   false,
			isUnique:    false,
			isPartial:   true,
			shouldExist: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idx := indexMap[tc.name]
			if tc.shouldExist {
				require.NotNil(t, idx, "%s should exist", tc.name)
				assert.Equal(t, tc.indexType, idx.Type)
				assert.ElementsMatch(t, tc.columns, idx.Columns)
				assert.Equal(t, tc.isPrimary, idx.IsPrimary)
				assert.Equal(t, tc.isUnique, idx.IsUnique)
				assert.Equal(t, tc.isPartial, idx.IsPartial)
				assert.True(t, idx.IsValid)
			} else {
				assert.Nil(t, idx, "%s should not exist", tc.name)
			}
		})
	}

	// Find and validate exclusion index separately
	var exclusionIdx *types.TableIndex
	for _, idx := range indexTypesTable.TableIndexes {
		if idx.IsExclusion {
			exclusionIdx = idx
			break
		}
	}
	require.NotNil(t, exclusionIdx, "exclusion index should exist")
	assert.Equal(t, "gist", exclusionIdx.Type)
	assert.ElementsMatch(t, []string{"during"}, exclusionIdx.Columns)
	assert.False(t, exclusionIdx.IsPrimary)
	assert.False(t, exclusionIdx.IsUnique)
	assert.True(t, exclusionIdx.IsExclusion)
	assert.True(t, exclusionIdx.IsValid)
	assert.Greater(t, exclusionIdx.SizeBytes, uint64(0), "exclusion index should have non-zero size")
	assert.Contains(t, exclusionIdx.Definition, "gist", "definition should contain access method")
	assert.Contains(t, exclusionIdx.Definition, "during", "definition should contain column name")
}

func TestListTablesEmptyTable(t *testing.T) {
	_, _, actual := setupStoreAndListTables(t)

	emptyTable := findTableInDb(actual, "test-db", "test-db", "empty_table")
	require.NotNil(t, emptyTable, "empty_table should exist in test-db.test-db")

	assert.Equal(t, "test-db-owner", emptyTable.Owner)
	assert.Nil(t, emptyTable.Comment, "empty_table should have no comment")
	assert.Empty(t, emptyTable.TableColumns, "empty_table should have no columns")
	assert.Empty(t, emptyTable.TableIndexes, "empty_table should have no indexes")
	assert.Empty(t, emptyTable.TableConstraints, "empty_table should have no constraints")

	// Stats should still exist
	assert.Equal(t, int64(-1), emptyTable.Stats.RowEstimate)
	assert.Equal(t, emptyTable.Stats.TotalSizeBytes, uint64(0), "empty tables take no space")
	assert.Equal(t, emptyTable.Stats.HeapSizeBytes, uint64(0), "empty tables have no heap size")
	assert.Equal(t, uint64(0), emptyTable.Stats.ToastSizeBytes, "empty table should have no toast")
}

func TestListTablesDroppedColumns(t *testing.T) {
	_, _, actual := setupStoreAndListTables(t)

	droppedColsTable := findTableInDb(actual, "test-db", "test-db", "dropped_columns_table")
	require.NotNil(t, droppedColsTable, "dropped_columns_table should exist in test-db.test-db")

	assert.Equal(t, "test-db-owner", droppedColsTable.Owner)

	require.Len(t, droppedColsTable.TableColumns, 2, "should have 2 active columns (id, keep_col)")

	columnNames := make([]string, len(droppedColsTable.TableColumns))
	for i, col := range droppedColsTable.TableColumns {
		columnNames[i] = col.Name
	}
	assert.ElementsMatch(t, []string{"id", "keep_col"}, columnNames, "should not include dropped columns")

	columnTypes := make(map[string]string)
	for _, col := range droppedColsTable.TableColumns {
		columnTypes[col.Name] = col.Type
	}
	assert.Equal(t, "integer", columnTypes["id"])
	assert.Equal(t, "text", columnTypes["keep_col"])

	var keep_col *types.TableColumn
	for _, col := range droppedColsTable.TableColumns {
		if col.Name == "keep_col" {
			keep_col = col
			break
		}
	}
	require.NotNil(t, keep_col)
	assert.True(t, keep_col.NotNull, "keep_col should be NOT NULL")
}

func TestListTablesInheritanceSimple(t *testing.T) {
	_, _, actual := setupStoreAndListTables(t)

	baseTable := findTableInDb(actual, "test-db", "test-db", "base_table")
	require.NotNil(t, baseTable, "base_table should exist in test-db.test-db")

	assert.Equal(t, "test-db-owner", baseTable.Owner)
	require.Len(t, baseTable.TableColumns, 2, "base_table should have 2 columns")

	baseColumnNames := make([]string, len(baseTable.TableColumns))
	for i, col := range baseTable.TableColumns {
		baseColumnNames[i] = col.Name
	}
	assert.ElementsMatch(t, []string{"id", "base_col"}, baseColumnNames)
	assert.Nil(t, baseTable.Inheritance.ParentTables, "base_table should have no parents")

	derivedTable := findTableInDb(actual, "test-db", "test-db", "derived_table")
	require.NotNil(t, derivedTable, "derived_table should exist in test-db.test-db")

	assert.Equal(t, "test-db-owner", derivedTable.Owner)
	require.Len(t, derivedTable.TableColumns, 3, "derived_table should have 3 columns (inherited + own)")

	derivedColumnNames := make([]string, len(derivedTable.TableColumns))
	for i, col := range derivedTable.TableColumns {
		derivedColumnNames[i] = col.Name
	}
	assert.ElementsMatch(t, []string{"id", "base_col", "derived_col"}, derivedColumnNames, "derived table should have both inherited and own columns")

	columnTypes := make(map[string]string)
	for _, col := range derivedTable.TableColumns {
		columnTypes[col.Name] = col.Type
	}
	assert.Equal(t, "integer", columnTypes["id"])
	assert.Equal(t, "text", columnTypes["base_col"])
	assert.Equal(t, "integer", columnTypes["derived_col"])

	require.NotNil(t, derivedTable.Inheritance.ParentTables, "derived_table should have parents")
	require.Len(t, derivedTable.Inheritance.ParentTables, 1)
	assert.Equal(t, `"test-db".base_table`, derivedTable.Inheritance.ParentTables[0].Name)
	assert.Equal(t, baseTable.Oid, derivedTable.Inheritance.ParentTables[0].Oid, "parent OID should match base_table OID")
}

func TestListTablesInheritanceMultiLevel(t *testing.T) {
	_, _, actual := setupStoreAndListTables(t)

	grandparentTable := findTableInDb(actual, "test-db", "test-db", "grandparent_table")
	require.NotNil(t, grandparentTable, "grandparent_table should exist")
	assert.Nil(t, grandparentTable.Inheritance.ParentTables, "grandparent should have no parents")

	parentInheritsGp := findTableInDb(actual, "test-db", "test-db", "parent_inherits_gp")
	require.NotNil(t, parentInheritsGp, "parent_inherits_gp should exist")

	require.NotNil(t, parentInheritsGp.Inheritance.ParentTables, "parent should have parents")
	require.Len(t, parentInheritsGp.Inheritance.ParentTables, 1)
	assert.Equal(t, `"test-db".grandparent_table`, parentInheritsGp.Inheritance.ParentTables[0].Name)
	assert.Equal(t, grandparentTable.Oid, parentInheritsGp.Inheritance.ParentTables[0].Oid)

	childInheritsParent := findTableInDb(actual, "test-db", "test-db", "child_inherits_parent")
	require.NotNil(t, childInheritsParent, "child_inherits_parent should exist")

	require.NotNil(t, childInheritsParent.Inheritance.ParentTables, "child should have parents")
	require.Len(t, childInheritsParent.Inheritance.ParentTables, 1)
	assert.Equal(t, `"test-db".parent_inherits_gp`, childInheritsParent.Inheritance.ParentTables[0].Name)
	assert.Equal(t, parentInheritsGp.Oid, childInheritsParent.Inheritance.ParentTables[0].Oid)
}

func TestListTablesInheritanceMultipleParents(t *testing.T) {
	_, _, actual := setupStoreAndListTables(t)

	mixinA := findTableInDb(actual, "test-db", "test-db", "mixin_a")
	require.NotNil(t, mixinA, "mixin_a should exist")

	assert.Nil(t, mixinA.Inheritance.ParentTables, "mixin_a should have no parents")

	mixinB := findTableInDb(actual, "test-db", "test-db", "mixin_b")
	require.NotNil(t, mixinB, "mixin_b should exist")

	assert.Nil(t, mixinB.Inheritance.ParentTables, "mixin_b should have no parents")

	multiInherit := findTableInDb(actual, "test-db", "test-db", "multi_inherit")
	require.NotNil(t, multiInherit, "multi_inherit should exist")

	require.NotNil(t, multiInherit.Inheritance.ParentTables, "multi_inherit should have parents")
	require.Len(t, multiInherit.Inheritance.ParentTables, 2, "multi_inherit should have 2 parents")

	parentNames := []string{
		multiInherit.Inheritance.ParentTables[0].Name,
		multiInherit.Inheritance.ParentTables[1].Name,
	}
	assert.ElementsMatch(t, []string{`"test-db".mixin_a`, `"test-db".mixin_b`}, parentNames)

	// Verify OIDs match
	parentOids := make(map[uint32]bool)
	for _, parent := range multiInherit.Inheritance.ParentTables {
		assert.NotZero(t, parent.Oid)
		parentOids[parent.Oid] = true
	}
	assert.Contains(t, parentOids, mixinA.Oid, "should contain mixin_a OID")
	assert.Contains(t, parentOids, mixinB.Oid, "should contain mixin_b OID")
}

func TestListTablesInheritanceMultipleChildren(t *testing.T) {
	_, _, actual := setupStoreAndListTables(t)

	sharedParent := findTableInDb(actual, "test-db", "test-db", "shared_parent")
	require.NotNil(t, sharedParent, "shared_parent should exist")

	assert.Nil(t, sharedParent.Inheritance.ParentTables, "shared_parent should have no parents")

	childOne := findTableInDb(actual, "test-db", "test-db", "child_one")
	require.NotNil(t, childOne, "child_one should exist")

	require.NotNil(t, childOne.Inheritance.ParentTables, "child_one should have parents")
	require.Len(t, childOne.Inheritance.ParentTables, 1)
	assert.Equal(t, `"test-db".shared_parent`, childOne.Inheritance.ParentTables[0].Name)
	assert.Equal(t, sharedParent.Oid, childOne.Inheritance.ParentTables[0].Oid)

	childTwo := findTableInDb(actual, "test-db", "test-db", "child_two")
	require.NotNil(t, childTwo, "child_two should exist")

	require.NotNil(t, childTwo.Inheritance.ParentTables, "child_two should have parents")
	require.Len(t, childTwo.Inheritance.ParentTables, 1)
	assert.Equal(t, `"test-db".shared_parent`, childTwo.Inheritance.ParentTables[0].Name)
	assert.Equal(t, sharedParent.Oid, childTwo.Inheritance.ParentTables[0].Oid)
}
