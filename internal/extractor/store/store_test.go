package store

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestListDatabases(t *testing.T) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5433/postgres")

	if err != nil {
		t.Fatalf("Database connection failed: %v", err)
	}

	defer pool.Close()

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
		}
		break
	}

	if !found {
		t.Error("expected to 'postgres' database")
	}
}
