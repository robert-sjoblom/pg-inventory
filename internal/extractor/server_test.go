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
