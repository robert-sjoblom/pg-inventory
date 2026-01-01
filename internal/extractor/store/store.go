// Package store handles the database connection to host databases.
package store

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(p *pgxpool.Pool) *Store {
	return &Store{
		pool: p,
	}
}
