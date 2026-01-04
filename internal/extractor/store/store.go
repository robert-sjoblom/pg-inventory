// Package store handles the database connection to host databases.
package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/types"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(p *pgxpool.Pool) *Store {
	return &Store{
		pool: p,
	}
}

func (s *Store) ListDatabases(ctx context.Context) ([]types.Database, error) {
	rows, err := s.pool.Query(ctx, "SELECT datname, oid FROM pg_database")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var databases []types.Database
	for rows.Next() {
		var db types.Database
		err := rows.Scan(&db.Name, &db.Oid)
		if err != nil {
			return nil, err
		}
		databases = append(databases, db)
	}

	return databases, nil
}

func (s *Store) GetStanza(ctx context.Context) (string, error) {
	var stanza string

	err := s.pool.QueryRow(ctx, "SELECT value FROM monitoring.cluster_config WHERE key = 'stanza'").Scan(&stanza)
	if err != nil {
		return "", err
	}

	return stanza, nil
}
