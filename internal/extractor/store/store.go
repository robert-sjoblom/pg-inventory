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

const queryServerInfo = `
SELECT
    pg_is_in_recovery() AS is_in_recovery,
    current_setting('transaction_read_only') AS is_read_only,
    current_setting('ssl') AS ssl,
    current_setting('port')::int AS port,
    current_setting('max_connections')::int AS max_connections,
    current_setting('archive_mode') AS archive_mode,
    current_setting('wal_level') AS wal_level,
    current_setting('data_directory') AS data_directory,
    (SELECT system_identifier FROM pg_control_system()) AS system_identifier,
    (SELECT timeline_id FROM pg_control_checkpoint()) AS timeline_id,
    version() AS pg_version;
`

// Returns the PG server info
func (s *Store) GetServerInfo(ctx context.Context) (types.ServerInfo, error) {
	var info types.ServerInfo
	err := s.pool.QueryRow(ctx, queryServerInfo).Scan(
		&info.IsInRecovery,
		&info.IsReadOnly,
		&info.SslEnabled,
		&info.Port,
		&info.MaxConnections,
		&info.ArchiveMode,
		&info.WalLevel,
		&info.DataDirectory,
		&info.SystemIdentifier,
		&info.TimelineID,
		&info.PgVersion,
	)
	if err != nil {
		return info, err
	}

	return info, nil
}

var stanzaNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,62}$`)

func isValidStanzaName(s string) bool {
	return stanzaNamePattern.MatchString(s)
}
