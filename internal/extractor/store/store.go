// Package store handles the database connection to host databases.
package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(p *pgxpool.Pool) *Store {
	return &Store{
		pool: p,
	}
}

func (s *Store) ListDatabases(ctx context.Context) ([]extractor.Database, error) {
	rows, err := s.pool.Query(ctx, "SELECT datname, oid FROM pg_database")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var databases []extractor.Database
	for rows.Next() {
		var db extractor.Database
		err := rows.Scan(&db.Name, &db.Oid)
		if err != nil {
			return nil, err
		}
		databases = append(databases, db)
	}

	return databases, nil
}
