//go:build integration

package extractor

import (
	"context"
	"testing"

	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/store"
	"github.com/robert-sjoblom/pg-inventory/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetServerInfo(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t)
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	st, err := store.NewStore(pool, creds.ConnStr)
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
	creds := testutil.StartPostgres(t)
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	st, err := store.NewStore(pool, creds.ConnStr)
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

	var foundPublic, foundMonitoring bool
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
	}

	assert.True(t, foundPublic, "expected to find public schema in postgres database")
	assert.True(t, foundMonitoring, "expected to find monitoring schema in postgres database")
}

func TestListExtensions(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t)
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	st, err := store.NewStore(pool, creds.ConnStr)
	require.NoError(t, err)

	server := NewServer(st)
	conn := testutil.DialBufconn(t, server)
	client := extractorv1.NewExtractorServiceClient(conn)

	resp, err := client.ListExtensions(ctx, &extractorv1.ListExtensionsRequest{})
	require.NoError(t, err)

	require.NotEmpty(t, resp.Available, "available extensions should not be empty")
	require.Greater(t, len(resp.Available), 0, "should have at least one available extension")

	require.Len(t, resp.Installed, 1, "expected exactly 1 installed extensions")

	var foundPlpgsql bool
	for _, ext := range resp.Installed {
		assert.NotZero(t, ext.Oid, "extension OID should not be zero")
		assert.NotEmpty(t, ext.Name, "extension name should not be empty")
		assert.NotEmpty(t, ext.Version, "extension version should not be empty")
		assert.NotEmpty(t, ext.Schema, "extension schema should not be empty")
		assert.Equal(t, "postgres", ext.Database, "all extensions should be in postgres database")

		if ext.Name == "plpgsql" {
			foundPlpgsql = true
			assert.Equal(t, uint32(13538), ext.Oid)
			assert.Equal(t, "1.0", ext.Version)
			assert.Equal(t, "pg_catalog", ext.Schema)
		}
	}

	assert.True(t, foundPlpgsql, "expected to find plpgsql extension")
}

func TestListSequences(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t)
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	st, err := store.NewStore(pool, creds.ConnStr)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "CREATE SEQUENCE test_seq")
	require.NoError(t, err)

	server := NewServer(st)
	conn := testutil.DialBufconn(t, server)
	client := extractorv1.NewExtractorServiceClient(conn)

	resp, err := client.ListSequences(ctx, &extractorv1.ListSequencesRequest{})
	require.NoError(t, err)

	require.Len(t, resp.Sequences, 1, "expected exactly 1 sequence")

	seq := resp.Sequences[0]
	assert.Equal(t, uint32(16392), seq.Oid)
	assert.Equal(t, "test_seq", seq.Name)
	assert.Equal(t, "postgres", seq.Owner)
	assert.Equal(t, "public", seq.Schema)
	assert.Equal(t, "postgres", seq.Database)
	assert.Equal(t, "int8", seq.DataType)
}

func TestListFunctions(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t)
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	st, err := store.NewStore(pool, creds.ConnStr)
	require.NoError(t, err)

	function := `
	CREATE FUNCTION greet_user(username TEXT)
	RETURNS TEXT AS $$
	BEGIN
		RETURN 'Hello, ' || username || '!';
	END; $$ LANGUAGE plpgsql;`

	_, err = pool.Exec(ctx, function)
	require.NoError(t, err)

	server := NewServer(st)
	conn := testutil.DialBufconn(t, server)
	client := extractorv1.NewExtractorServiceClient(conn)

	resp, err := client.ListFunctions(ctx, &extractorv1.ListFunctionsRequest{})
	require.NoError(t, err)

	require.Len(t, resp.Functions, 1, "expected exactly 1 function")

	fn := resp.Functions[0]
	assert.Equal(t, uint32(16392), fn.Oid)
	assert.Equal(t, "greet_user", fn.Name)
	assert.Equal(t, "public", fn.Schema)
	assert.Equal(t, "postgres", fn.Owner)
	assert.Equal(t, "postgres", fn.Database)
	assert.Equal(t, "plpgsql", fn.Language)
	assert.Equal(t, "text", fn.ReturnType)
	assert.Equal(t, "username text", fn.IdentityArguments)
}
