//go:build integration

package store

import (
	"context"
	"os"
	"testing"

	"github.com/robert-sjoblom/pg-inventory/internal/extractor/types"
	"github.com/robert-sjoblom/pg-inventory/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewStore(t *testing.T) {
	creds := testutil.StartPostgres(t, testutil.WithClusterConfig("another-name", "stanza-name"))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed")
	}

	assert.Equal(t, "another-name", store.ClusterName)
	assert.Equal(t, "stanza-name", store.Stanza)
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
	creds := testutil.StartPostgres(t, testutil.WithExtraDatabases("testdb", "app-db"))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)
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

func TestBackupInfoJsonToLocalType(t *testing.T) {
	data, err := os.ReadFile("../testdata/pgbackrest.json")
	if err != nil {
		t.Fatalf("failed to read pgbackrest.json: %v", err)
	}

	actual, err := parseBackrestInfo(data)
	if err != nil {
		t.Fatalf("failed to unmarshal json: %v", err)
	}

	expected := []types.PgbackrestInfo{
		{
			Name: "main",
			Backup: []types.Backup{
				{
					Label: "20260103-233937F",
					Type:  "full",
					Error: false,
					Database: types.PgbackrestDatabase{
						Id:      1,
						RepoKey: 1,
					},
					Info: types.BackupSizeInfo{
						Delta: 23246324,
						Size:  23246324,
						Repository: types.PgbackrestRepo{
							Delta: 3066593,
							Size:  3066593,
						},
					},
					Timestamp: types.BackupTimestamp{
						Start: 1767483577,
						Stop:  1767483580,
					},
				},
				{
					Label: "20260103-233937F_20260103-234057I",
					Type:  "incr",
					Error: false,
					Database: types.PgbackrestDatabase{
						Id:      1,
						RepoKey: 1,
					},
					Info: types.BackupSizeInfo{
						Delta: 1376514,
						Size:  23344628,
						Repository: types.PgbackrestRepo{
							Delta: 212178,
							Size:  3078147,
						},
					},
					Timestamp: types.BackupTimestamp{
						Start: 1767483657,
						Stop:  1767483664,
					},
				},
			},
			Db: []types.PgbackrestDb{
				{
					Id:       1,
					RepoKey:  1,
					SystemId: 7591284103390970243,
					Version:  "15",
				},
			},
		},
	}

	assert.Equal(t, expected, actual, "correct unmarshal of pgbackrest info")
}

func TestIsValidStanzaName(t *testing.T) {
	tests := []struct {
		name       string
		stanzaName string
		expected   bool
	}{
		{
			name:       "valid stanza name",
			stanzaName: "stanza-123",
			expected:   true,
		},
		{
			name:       "invalid stanza name",
			stanzaName: "0-stanza",
			expected:   false,
		},
		{
			name:       "too long stanza name",
			stanzaName: "abcdefghjkabcdefghjkabcdefghjkabcdefghjkabcdefghjkabcdefghjkabcdefghjk",
			expected:   false,
		},
		{
			name:       "empty stanza name is an error",
			stanzaName: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := isValidStanzaName(tt.stanzaName)
			assert.Equal(t, tt.expected, actual)

		})
	}
}

func TestGetServerInfo(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t)
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)
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

func TestListSchemas(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t, testutil.WithExtraDatabases("testdb", "app-db"), testutil.WithExtraSchemas(testutil.ExtraSchema{
		Name:     "testdb-schema",
		Database: "testdb",
	}, testutil.ExtraSchema{
		Name:     "app-db-schema",
		Database: "app-db",
	}))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)
	if err != nil {
		t.Fatalf("store initialization failed")
	}

	actual, err := store.ListSchemas(ctx)
	if err != nil {
		t.Fatalf("GetServerInfo failed: %v", err)
	}

	expected := []*types.Schema{
		{
			Oid:      2200,
			Name:     "public",
			Owner:    "pg_database_owner",
			Database: "postgres",
		},
		{
			Oid:      16384,
			Name:     "monitoring",
			Owner:    "postgres",
			Database: "postgres",
		},
		{
			Oid:      2200,
			Name:     "public",
			Owner:    "pg_database_owner",
			Database: "testdb",
		},
		{
			Oid:      16394,
			Name:     "testdb-schema",
			Owner:    "postgres",
			Database: "testdb",
		},
		{
			Oid:      2200,
			Name:     "public",
			Owner:    "pg_database_owner",
			Database: "app-db",
		},
		{
			Oid:      16395,
			Name:     "app-db-schema",
			Owner:    "postgres",
			Database: "app-db",
		},
	}

	assert.ElementsMatch(t, expected, actual)
}

func TestListExtensions(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t, testutil.WithExtraDatabases("testdb", "app-db"), testutil.WithExtraSchemas(testutil.ExtraSchema{
		Name:     "testdb-schema",
		Database: "testdb",
	}),
		testutil.WithInstalledExtensions(testutil.InstalledExtension{
			Name:     "pg_trgm",
			Database: "testdb",
		}),
	)
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)

	available, installed, err := store.ListExtensions(ctx)
	if err != nil {
		t.Fatalf("ListExtensions failed: %v", err)
	}

	expectedInstalled := []*types.InstalledExtension{
		{
			Name:     "pg_trgm",
			Version:  "1.6",
			Schema:   "public",
			Database: "testdb",
			Oid:      16395,
		},
		{
			Name:     "plpgsql",
			Version:  "1.0",
			Schema:   "pg_catalog",
			Database: "testdb",
			Oid:      13538,
		},
		{
			Name:     "plpgsql",
			Version:  "1.0",
			Schema:   "pg_catalog",
			Database: "app-db",
			Oid:      13538,
		},
		{
			Name:     "plpgsql",
			Version:  "1.0",
			Schema:   "pg_catalog",
			Database: "postgres",
			Oid:      13538,
		},
	}

	// The actual list is ~47 units long, depending
	assert.GreaterOrEqual(t, len(available), 0)
	assert.ElementsMatch(t, expectedInstalled, installed)
}

func TestListSequences(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t)
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)

	_, err = pool.Exec(ctx, "CREATE SEQUENCE test_seq")
	if err != nil {
		t.Fatalf("failed to create sequence: %v", err)
	}

	actual, err := store.ListSequences(ctx)
	if err != nil {
		t.Fatalf("failed to list sequences: %v", err)
	}

	expected := []*types.Sequence{
		{
			Oid:      16392,
			Name:     "test_seq",
			Owner:    "postgres",
			Schema:   "public",
			Database: "postgres",
			DataType: "int8",
		},
	}

	assert.ElementsMatch(t, actual, expected)
}

func TestListFunctions(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t, testutil.WithExtraDatabases("test-db"))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store, err := NewStore(pool, creds.ConnStr)

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

	_, err = pool.Exec(ctx, function)
	if err != nil {
		t.Fatalf("failed to create function: %v", err)
	}

	_, err = pool.Exec(ctx, function2)
	if err != nil {
		t.Fatalf("failed to create function: %v", err)
	}

	actual, err := store.ListFunctions(ctx)
	if err != nil {
		t.Fatalf("failed to list functions: %v", err)
	}

	expected := []*types.Function{
		{
			Oid:               16393,
			Name:              "sum",
			Schema:            "public",
			Owner:             "postgres",
			Database:          "postgres",
			Language:          "plpgsql",
			ReturnType:        "integer",
			IdentityArguments: "a integer, b integer",
		},
		{
			Oid:               16394,
			Name:              "sum",
			Schema:            "public",
			Owner:             "postgres",
			Database:          "postgres",
			Language:          "plpgsql",
			ReturnType:        "integer",
			IdentityArguments: "a integer",
		},
	}

	assert.ElementsMatch(t, actual, expected)
}
