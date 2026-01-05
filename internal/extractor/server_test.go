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

	st, err := store.NewStore(pool)
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
