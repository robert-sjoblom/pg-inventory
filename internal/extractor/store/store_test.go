//go:build integration

package store

import (
	"context"
	"testing"

	"github.com/robert-sjoblom/pg-inventory/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestListDatabases(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t, testutil.WithExtraDatabases("testdb", "app-db"))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store := NewStore(pool)

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

func TestStoreGetStanza(t *testing.T) {
	ctx := context.Background()
	creds := testutil.StartPostgres(t, testutil.WithClusterConfig("another-name", "stanza-name"))
	pool := testutil.ConnectToDatabase(t, creds, "postgres")

	store := NewStore(pool)

	expected, err := store.GetStanza(ctx)
	if err != nil {
		t.Fatalf("GetStanza failed: %v", err)
	}

	assert.Equal(t, "stanza-name", expected)
}
