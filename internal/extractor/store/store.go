// Package store handles the database connection to host databases.
package store

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/types"
)

type Store struct {
	pool        *pgxpool.Pool
	Stanza      string
	ClusterName string
}

// NewStore initializes the store. Additionally, it also fetches stanza name and
// cluster name -- these are static (largely) values that only change occasionally,
// and so we should probably not taint the RPC routes with small calls to the DB
// for little gain. Perhaps a micro-optimization.
func NewStore(p *pgxpool.Pool) (*Store, error) {
	ctx := context.Background()

	var stanza, clusterName string
	err := p.QueryRow(ctx, "SELECT value FROM monitoring.cluster_config WHERE key = 'stanza'").Scan(&stanza)
	if err != nil {
		return nil, fmt.Errorf("failed to load stanza: %w", err)
	}

	if !isValidStanzaName(stanza) {
		return nil, fmt.Errorf("invalid stanza name: %q", stanza)
	}

	err = p.QueryRow(ctx, "SELECT value FROM monitoring.cluster_config WHERE key = 'cluster_name'").Scan(&clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster_name: %w", err)
	}

	return &Store{
		pool:        p,
		ClusterName: clusterName,
		Stanza:      stanza,
	}, nil
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

var stanzaNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,62}$`)

func isValidStanzaName(s string) bool {
	return stanzaNamePattern.MatchString(s)
}
