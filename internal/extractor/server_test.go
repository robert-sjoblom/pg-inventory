//go:build integration

package extractor

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/store"
	"github.com/robert-sjoblom/pg-inventory/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sharedCredentials *testutil.TestDbCredentials

func TestMain(m *testing.M) {
	testRole := testutil.ExtraRole{
		Database: "testdb",
		Schema:   "testschema",
		Role:     "test_owner",
		Password: "password",
	}

	ctx := context.Background()

	var err error
	sharedCredentials, err = testutil.StartSharedPostgres(ctx,
		testutil.WithClusterConfig("test-cluster", "test-stanza"),
		testutil.WithExtraRoles(testRole),
	)
	if err != nil {
		log.Fatalf("failed to start shared postgres container: %v", err)
	}

	defer sharedCredentials.Terminate(ctx)

	code := m.Run()

	os.Exit(code)
}

func TestGetServerInfo(t *testing.T) {
	ctx := context.Background()
	pool := testutil.ConnectAsPgmonitor(t, sharedCredentials, "postgres")

	st, err := store.NewStore(pool, sharedCredentials.PgmonitorConnStrFunc())
	require.NoError(t, err)

	server := NewServer(st)
	conn := testutil.DialBufconn(t, server)
	client := extractorv1.NewExtractorServiceClient(conn)

	resp, err := client.GetServerInfo(ctx, &extractorv1.GetServerInfoRequest{})
	require.NoError(t, err)

	assert.Equal(t, "test-cluster", resp.ClusterName)
	assert.Contains(t, resp.PgVersion, "15.")
	assert.False(t, resp.IsInRecovery)
	assert.Equal(t, int32(5432), resp.Port)
	assert.Equal(t, int32(100), resp.MaxConnections)
	assert.NotEmpty(t, resp.DataDirectory)
	assert.NotZero(t, resp.SystemIdentifier)
	assert.Equal(t, int32(1), resp.TimelineId)
	assert.NotEmpty(t, resp.WalLevel)
}

func TestListSchemas(t *testing.T) {
	ctx := context.Background()
	pool := testutil.ConnectAsPgmonitor(t, sharedCredentials, "postgres")

	st, err := store.NewStore(pool, sharedCredentials.PgmonitorConnStrFunc())
	require.NoError(t, err)

	server := NewServer(st)
	conn := testutil.DialBufconn(t, server)
	client := extractorv1.NewExtractorServiceClient(conn)

	resp, err := client.ListSchemas(ctx, &extractorv1.ListSchemasRequest{})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(resp.Schemas), 2, "expected at least 2 schemas")

	for _, schema := range resp.Schemas {
		assert.NotZero(t, schema.Oid, "schema OID should not be zero")
		assert.NotEmpty(t, schema.Name, "schema name should not be empty")
		assert.NotEmpty(t, schema.Owner, "schema owner should not be empty")
		assert.NotEmpty(t, schema.Database, "schema database should not be empty")
	}

	var foundPublic, foundMonitoring, foundTestSchema bool
	for _, schema := range resp.Schemas {
		if schema.Database == "postgres" && schema.Name == "public" {
			foundPublic = true
			assert.Equal(t, uint32(2200), schema.Oid, "public schema has fixed OID")
			assert.Equal(t, "pg_database_owner", schema.Owner)
		}
		if schema.Database == "postgres" && schema.Name == "monitoring" {
			foundMonitoring = true
			assert.Equal(t, "postgres", schema.Owner)
		}
		if schema.Database == "testdb" && schema.Name == "testschema" {
			foundTestSchema = true
			assert.Equal(t, "test_owner", schema.Owner)
		}
	}

	assert.True(t, foundPublic, "expected to find public schema in postgres database")
	assert.True(t, foundMonitoring, "expected to find monitoring schema in postgres database")
	assert.True(t, foundTestSchema, "expected to find testschema in testdb database")
}

func TestListExtensions(t *testing.T) {
	ctx := context.Background()
	pool := testutil.ConnectAsPgmonitor(t, sharedCredentials, "postgres")

	st, err := store.NewStore(pool, sharedCredentials.PgmonitorConnStrFunc())
	require.NoError(t, err)

	server := NewServer(st)
	conn := testutil.DialBufconn(t, server)
	client := extractorv1.NewExtractorServiceClient(conn)

	resp, err := client.ListExtensions(ctx, &extractorv1.ListExtensionsRequest{})
	require.NoError(t, err)

	require.NotEmpty(t, resp.Available, "available extensions should not be empty")
	require.Greater(t, len(resp.Available), 0, "should have at least one available extension")

	require.GreaterOrEqual(t, len(resp.Installed), 1, "expected at least 1 installed extension")

	var foundPlpgsql bool
	for _, ext := range resp.Installed {
		assert.NotZero(t, ext.Oid, "extension OID should not be zero")
		assert.NotEmpty(t, ext.Name, "extension name should not be empty")
		assert.NotEmpty(t, ext.Version, "extension version should not be empty")
		assert.NotEmpty(t, ext.Schema, "extension schema should not be empty")

		if ext.Name == "plpgsql" && ext.Database == "postgres" {
			foundPlpgsql = true
			assert.Equal(t, uint32(13538), ext.Oid)
			assert.Equal(t, "1.0", ext.Version)
			assert.Equal(t, "pg_catalog", ext.Schema)
		}
	}

	assert.True(t, foundPlpgsql, "expected to find plpgsql extension in postgres database")
}

func TestListSequences(t *testing.T) {
	ctx := context.Background()
	pool := testutil.ConnectAsPgmonitor(t, sharedCredentials, "postgres")

	st, err := store.NewStore(pool, sharedCredentials.PgmonitorConnStrFunc())
	require.NoError(t, err)

	// Connect as test_owner to create the sequence
	ownerConnStr := sharedCredentials.ConnStrForUser("testdb", "test_owner", "password")
	ownerPool, err := pgxpool.New(ctx, ownerConnStr)
	require.NoError(t, err)
	defer ownerPool.Close()

	_, err = ownerPool.Exec(ctx, "CREATE SEQUENCE testschema.test_seq")
	require.NoError(t, err)

	server := NewServer(st)
	conn := testutil.DialBufconn(t, server)
	client := extractorv1.NewExtractorServiceClient(conn)

	resp, err := client.ListSequences(ctx, &extractorv1.ListSequencesRequest{})
	require.NoError(t, err)

	require.Len(t, resp.Sequences, 1, "expected exactly 1 sequence")

	seq := resp.Sequences[0]
	assert.Greater(t, seq.Oid, uint32(0), "OID should be greater than 0")
	assert.Equal(t, "test_seq", seq.Name)
	assert.Equal(t, "test_owner", seq.Owner)
	assert.Equal(t, "testschema", seq.Schema)
	assert.Equal(t, "testdb", seq.Database)
	assert.Equal(t, "int8", seq.DataType)
}

func TestListFunctions(t *testing.T) {
	ctx := context.Background()
	pool := testutil.ConnectAsPgmonitor(t, sharedCredentials, "postgres")

	st, err := store.NewStore(pool, sharedCredentials.PgmonitorConnStrFunc())
	require.NoError(t, err)

	// Connect as test_owner to create the function
	ownerConnStr := sharedCredentials.ConnStrForUser("testdb", "test_owner", "password")
	ownerPool, err := pgxpool.New(ctx, ownerConnStr)
	require.NoError(t, err)
	defer ownerPool.Close()

	function := `
	CREATE FUNCTION testschema.greet_user(username TEXT)
	RETURNS TEXT AS $$
	BEGIN
		RETURN 'Hello, ' || username || '!';
	END; $$ LANGUAGE plpgsql;`

	_, err = ownerPool.Exec(ctx, function)
	require.NoError(t, err)

	server := NewServer(st)
	conn := testutil.DialBufconn(t, server)
	client := extractorv1.NewExtractorServiceClient(conn)

	resp, err := client.ListFunctions(ctx, &extractorv1.ListFunctionsRequest{})
	require.NoError(t, err)

	require.Len(t, resp.Functions, 1, "expected exactly 1 function")

	fn := resp.Functions[0]
	assert.Greater(t, fn.Oid, uint32(0), "OID should be greater than 0")
	assert.Equal(t, "greet_user", fn.Name)
	assert.Equal(t, "testschema", fn.Schema)
	assert.Equal(t, "test_owner", fn.Owner)
	assert.Equal(t, "testdb", fn.Database)
	assert.Equal(t, "plpgsql", fn.Language)
	assert.Equal(t, "text", fn.ReturnType)
	assert.Equal(t, "username text", fn.IdentityArguments)
}

func TestListTables(t *testing.T) {
	ctx := context.Background()
	pool := testutil.ConnectAsPgmonitor(t, sharedCredentials, "postgres")

	st, err := store.NewStore(pool, sharedCredentials.PgmonitorConnStrFunc())
	require.NoError(t, err)

	// Connect as test_owner to create the table
	ownerConnStr := sharedCredentials.ConnStrForUser("testdb", "test_owner", "password")
	ownerPool, err := pgxpool.New(ctx, ownerConnStr)
	require.NoError(t, err)
	defer ownerPool.Close()

	tableDDL := `
	CREATE TABLE testschema.users (
		id SERIAL PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		created_at TIMESTAMPTZ DEFAULT now()
	);
	COMMENT ON TABLE testschema.users IS 'User accounts';
	CREATE INDEX idx_users_created ON testschema.users (created_at);
	`

	_, err = ownerPool.Exec(ctx, tableDDL)
	require.NoError(t, err)

	server := NewServer(st)
	conn := testutil.DialBufconn(t, server)
	client := extractorv1.NewExtractorServiceClient(conn)

	resp, err := client.ListTables(ctx, &extractorv1.ListTablesRequest{})
	require.NoError(t, err)

	var testdbTables *extractorv1.DatabaseTables
	for _, dt := range resp.DatabaseTables {
		if dt.Database == "testdb" {
			testdbTables = dt
			break
		}
	}
	require.NotNil(t, testdbTables, "expected to find testdb in response")

	var usersTable *extractorv1.Table
	for _, tbl := range testdbTables.Tables {
		if tbl.Name == "users" && tbl.Schema == "testschema" {
			usersTable = tbl
			break
		}
	}
	require.NotNil(t, usersTable, "expected to find users table")

	assert.Greater(t, usersTable.Oid, uint32(0))
	assert.Equal(t, "test_owner", usersTable.Owner)
	require.NotNil(t, usersTable.Comment)
	assert.Equal(t, "User accounts", *usersTable.Comment)

	require.Len(t, usersTable.Columns, 3)
	columnNames := make([]string, len(usersTable.Columns))
	for i, col := range usersTable.Columns {
		columnNames[i] = col.Name
	}
	assert.ElementsMatch(t, []string{"id", "email", "created_at"}, columnNames)

	// Verify indexes (PK + unique + explicit)
	require.GreaterOrEqual(t, len(usersTable.Indexes), 3)
	var pkIndex, createdIndex *extractorv1.TableIndex
	for _, idx := range usersTable.Indexes {
		if idx.IsPrimary {
			pkIndex = idx
		}
		if idx.Name == "idx_users_created" {
			createdIndex = idx
		}
	}
	require.NotNil(t, pkIndex, "expected primary key index")
	assert.True(t, pkIndex.IsUnique)
	assert.ElementsMatch(t, []string{"id"}, pkIndex.Columns)

	require.NotNil(t, createdIndex, "expected idx_users_created index")
	assert.Equal(t, "btree", createdIndex.Type)
	assert.ElementsMatch(t, []string{"created_at"}, createdIndex.Columns)

	// Verify constraints
	require.GreaterOrEqual(t, len(usersTable.Constraints), 2) // PK + unique
	var pkConstraint *extractorv1.TableConstraint
	for _, con := range usersTable.Constraints {
		if con.Type == "primary_key" {
			pkConstraint = con
			break
		}
	}
	require.NotNil(t, pkConstraint)
	assert.ElementsMatch(t, []string{"id"}, pkConstraint.LocalColumns)

	// Verify stats
	require.NotNil(t, usersTable.Stats)
	assert.GreaterOrEqual(t, usersTable.Stats.TotalSizeBytes, uint64(0))

	// Verify inheritance (should be empty for non-inherited table)
	require.NotNil(t, usersTable.Inheritance)
	assert.Empty(t, usersTable.Inheritance.ParentTables)
}
