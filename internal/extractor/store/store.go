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
	pool *pgxpool.Pool
	// Connection string builder function for given dbName
	connStrFor func(dbName string) string
	// Connection semaphores for each database.
	// MUST be used by any function that connects to a different database than
	// the default one. These typically end in `ForDatabase`
	coordinator *ConnectionCoordinator
	Stanza      string
	ClusterName string
}

// NewStore initializes the store. Additionally, it also fetches stanza name and
// cluster name -- these are static (largely) values that only change occasionally,
// and so we should probably not taint the RPC routes with small calls to the DB
// for little gain. Perhaps a micro-optimization.
func NewStore(p *pgxpool.Pool, connStrFor func(s string) string) (*Store, error) {
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
		connStrFor:  connStrFor,
		coordinator: NewConnectionCoordinator(),
		ClusterName: clusterName,
		Stanza:      stanza,
	}, nil
}

func (s *Store) ListDatabases(ctx context.Context) ([]types.Database, error) {
	rows, err := s.pool.Query(ctx, "SELECT datname, oid FROM pg_database WHERE datallowconn AND NOT datistemplate")
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

// Connects to each database and returns the aggregated schema list
func (s *Store) ListSchemas(ctx context.Context) ([]*types.Schema, error) {
	databases, err := s.ListDatabases(ctx)
	if err != nil {
		return nil, err
	}

	type result struct {
		err     error
		schemas []*types.Schema
	}

	resultsCh := make(chan result, len(databases))

	/*
		Here is probably the first time I actually miss Rust.
		The following compiles in go:

		for _, db := range databases {
			go func() {
				name := db.Name
			}())
		}

		However, the goroutine captures the _variable_, not the value. So db
		will be reassigned before the goroutine reads it (race-ish, I guess).
		When we pass as a parameter instead (the }(db.Name)) part), we create a
		copy.

		In Rust you can't even compile it; capturing a &mut in a closure is
		prevented.

		Nice footgun, go.
	*/
	for _, db := range databases {
		go func(dbName string) {
			schemas, err := s.listSchemasForDatabase(ctx, dbName)
			resultsCh <- result{
				schemas: schemas,
				err:     err,
			}
		}(db.Name)
	}

	var allSchemas []*types.Schema
	var errs []error
	for range databases {
		res := <-resultsCh
		if res.err != nil {
			errs = append(errs, res.err)
		} else {
			allSchemas = append(allSchemas, res.schemas...)
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to query %d/%d databases: %v", len(errs), len(databases), errs)
	}

	return allSchemas, nil
}

func (s *Store) ListExtensions(ctx context.Context) ([]*types.AvailableExtension, []*types.InstalledExtension, error) {
	availableExtensions, err := s.listAvailableExtensions(ctx)
	if err != nil {
		return nil, nil, err
	}

	databases, err := s.ListDatabases(ctx)
	if err != nil {
		return nil, nil, err
	}

	type result struct {
		err       error
		installed []*types.InstalledExtension
	}

	resultsCh := make(chan result, len(databases))
	for _, db := range databases {
		go func(dbName string) {
			installed, err := s.listInstalledExtensionsForDatabase(ctx, dbName)
			resultsCh <- result{
				installed: installed,
				err:       err,
			}
		}(db.Name)
	}

	var installedExtensions []*types.InstalledExtension
	var errs []error
	for range databases {
		res := <-resultsCh
		if res.err != nil {
			errs = append(errs, res.err)
		} else {
			installedExtensions = append(installedExtensions, res.installed...)
		}
	}

	if len(errs) > 0 {
		return nil, nil, fmt.Errorf("failed to query %d/%d databases: %v", len(errs), len(databases), errs)
	}

	return availableExtensions, installedExtensions, nil
}

const availableExtensions = `
SELECT
    name,
    default_version
FROM pg_available_extensions
`

func (s *Store) listAvailableExtensions(ctx context.Context) ([]*types.AvailableExtension, error) {
	rows, err := s.pool.Query(ctx, availableExtensions)
	if err != nil {
		return nil, fmt.Errorf("query available extensions: %w", err)
	}
	defer rows.Close()

	var extensions []*types.AvailableExtension
	for rows.Next() {
		var ext types.AvailableExtension
		if err := rows.Scan(&ext.Name, &ext.DefaultVersion); err != nil {
			return nil, fmt.Errorf("scan available extension: %w", err)
		}
		extensions = append(extensions, &ext)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate available extensions: %w", err)
	}

	return extensions, nil
}

const queryListInstalledExtensions = `
SELECT
    e.oid,
    e.extname AS name,
    e.extversion AS version,
    n.nspname AS schema
FROM pg_extension e
JOIN pg_namespace n ON e.extnamespace = n.oid
ORDER BY e.extname;
`

func (s *Store) listInstalledExtensionsForDatabase(ctx context.Context, dbName string) ([]*types.InstalledExtension, error) {
	err := s.coordinator.Acquire(ctx, dbName)
	if err != nil {
		return nil, fmt.Errorf("acquire connection to %q: %w", dbName, err)
	}
	defer s.coordinator.Release(dbName)

	pool, err := pgxpool.New(ctx, s.connStrFor(dbName))
	if err != nil {
		return nil, fmt.Errorf("connect to %q for installed extensions: %w", dbName, err)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, queryListInstalledExtensions)
	if err != nil {
		return nil, fmt.Errorf("query installed extensions in database %q: %w", dbName, err)
	}

	defer rows.Close()

	var extensions []*types.InstalledExtension
	for rows.Next() {
		var extension types.InstalledExtension
		err := rows.Scan(&extension.Oid, &extension.Name, &extension.Version, &extension.Schema)
		if err != nil {
			return nil, fmt.Errorf("query installed extensions in database %q: %w", dbName, err)
		}
		extension.Database = dbName
		extensions = append(extensions, &extension)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate installed extensions in database %q: %w", dbName, err)
	}
	return extensions, nil
}

const queryListSchemas = `
SELECT
	n.oid,
	n.nspname AS name,
	pg_catalog.pg_get_userbyid(n.nspowner) AS owner
FROM pg_catalog.pg_namespace n
WHERE n.nspname NOT LIKE 'pg_%'
	AND n.nspname != 'information_schema'
`

func (s *Store) listSchemasForDatabase(ctx context.Context, dbName string) ([]*types.Schema, error) {
	err := s.coordinator.Acquire(ctx, dbName)
	if err != nil {
		return nil, fmt.Errorf("acquire connection to %q: %w", dbName, err)
	}
	defer s.coordinator.Release(dbName)

	pool, err := pgxpool.New(ctx, s.connStrFor(dbName))
	if err != nil {
		return nil, fmt.Errorf("connect to %q for schemas: %w", dbName, err)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, queryListSchemas)
	if err != nil {
		return nil, fmt.Errorf("query schemas in database %q: %w", dbName, err)
	}

	defer rows.Close()

	var schemas []*types.Schema
	for rows.Next() {
		var schema types.Schema
		err := rows.Scan(&schema.Oid, &schema.Name, &schema.Owner)
		if err != nil {
			return nil, fmt.Errorf("query schemas in database %q: %w", dbName, err)
		}
		schema.Database = dbName
		schemas = append(schemas, &schema)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schemas in database %q: %w", dbName, err)
	}
	return schemas, nil
}

var stanzaNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,62}$`)

func isValidStanzaName(s string) bool {
	return stanzaNamePattern.MatchString(s)
}
