// Package testutil contains test utilities like bufconn or test containers
package testutil

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type setupConfig struct {
	clusterName         string
	stanza              string
	extraDatabases      []string
	extraSchemas        []ExtraSchema
	installedExtensions []InstalledExtension
	extraRoles          []ExtraRole
	extraTables         []ExtraTable
}

type ExtraSchema struct {
	Name     string
	Database string
}

type InstalledExtension struct {
	Name     string
	Database string
}

// ExtraRoles own databases and schemas
type ExtraRole struct {
	Database string
	Schema   string
	Role     string
	Password string
}

type ExtraTable struct {
	Role     *ExtraRole
	Schema   string
	Database string
	DDL      string
}

type SetupOption func(*setupConfig)

// Sets up cluster configuration (monitoring.* tables)
func WithClusterConfig(clusterName, stanza string) SetupOption {
	return func(cfg *setupConfig) {
		cfg.clusterName = clusterName
		cfg.stanza = stanza
	}
}

// Sets up additional databases
func WithExtraDatabases(names ...string) SetupOption {
	return func(cfg *setupConfig) {
		cfg.extraDatabases = append(cfg.extraDatabases, names...)
	}
}

// Sets up extra schemas in given databases
func WithExtraSchemas(schemas ...ExtraSchema) SetupOption {
	return func(cfg *setupConfig) {
		cfg.extraSchemas = append(cfg.extraSchemas, schemas...)
	}
}

// Sets up installed extensions
func WithInstalledExtensions(extensions ...InstalledExtension) SetupOption {
	return func(cfg *setupConfig) {
		cfg.installedExtensions = append(cfg.installedExtensions, extensions...)
	}
}

// Sets up extra roles
func WithExtraRoles(roles ...ExtraRole) SetupOption {
	return func(cfg *setupConfig) {
		cfg.extraRoles = append(cfg.extraRoles, roles...)
	}
}

// Sets up extra tables in specified schema
func WithExtraTables(tables ...ExtraTable) SetupOption {
	return func(cfg *setupConfig) {
		cfg.extraTables = append(cfg.extraTables, tables...)
	}
}

type TestDbCredentials struct {
	container *postgres.PostgresContainer
	dbuser    string
	dbpass    string
	host      string
	port      nat.Port
}

func (d *TestDbCredentials) Terminate(ctx context.Context) {
	if err := testcontainers.TerminateContainer(d.container); err != nil {
		log.Default().Printf("failed to terminate container: %s", err)
	}
}

func (d *TestDbCredentials) ConnStr(db string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", d.dbuser, d.dbpass, d.host, d.port.Port(), db)
}

func StartSharedPostgres(ctx context.Context, opts ...SetupOption) (*TestDbCredentials, error) {
	db_user := "postgres"
	db_pass := "postgres"

	container, err := postgres.Run(ctx, "postgres:15-bullseye",
		postgres.WithUsername(db_user),
		postgres.WithPassword(db_pass),
		postgres.WithInitScripts(initScriptPath("00-monitoring.sql")),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pg testcontainer host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, fmt.Errorf("failed to get pg testcontainer port: %w", err)
	}

	cfg := &setupConfig{
		clusterName: "test-cluster",
		stanza:      "test-stanza",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	credentials := &TestDbCredentials{
		dbuser:    db_user,
		dbpass:    db_pass,
		host:      host,
		port:      port,
		container: container,
	}

	if err := applySetupConfiguration(ctx, credentials, cfg); err != nil {
		return nil, fmt.Errorf("failed to apply test configuration to database: %w", err)
	}

	return credentials, nil
}

// StartPostgres launches a postgres testcontainer instance
func StartPostgres(t *testing.T, opts ...SetupOption) *TestDbCredentials {
	t.Helper()
	ctx := context.Background()

	db_user := "postgres"
	db_pass := "postgres"

	container, err := postgres.Run(ctx, "postgres:15-bullseye",
		postgres.WithUsername(db_user),
		postgres.WithPassword(db_pass),
		postgres.WithInitScripts(initScriptPath("00-monitoring.sql")),
		postgres.BasicWaitStrategies(),
	)
	t.Cleanup(func() {
		if inner_err := testcontainers.TerminateContainer(container); inner_err != nil {
			t.Fatalf("failed to terminate container: %s", inner_err)
		}
	})

	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get pg testcontainer host")
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("failed to get pg testcontainer port")
	}

	cfg := &setupConfig{
		clusterName: "test-cluster",
		stanza:      "test-stanza",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	credentials := &TestDbCredentials{
		dbuser: db_user,
		dbpass: db_pass,
		host:   host,
		port:   port,
	}

	if err := applySetupConfiguration(ctx, credentials, cfg); err != nil {
		t.Fatalf("failed to apply test configuration to database: %v", err)
	}

	return credentials
}

func ConnectToDatabase(t *testing.T, credentials *TestDbCredentials, dbName string) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, credentials.ConnStr(dbName))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	t.Cleanup(func() { pool.Close() })

	return pool
}

func applySetupConfiguration(ctx context.Context, credentials *TestDbCredentials, cfg *setupConfig) error {
	pool, err := pgxpool.New(ctx, credentials.ConnStr("postgres"))
	if err != nil {
		return err
	}

	defer pool.Close()

	_, err = pool.Exec(ctx, "UPDATE monitoring.cluster_config SET value = $1 WHERE key = 'cluster_name'", cfg.clusterName)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, "UPDATE monitoring.cluster_config SET value = $1 WHERE key = 'stanza'", cfg.stanza)
	if err != nil {
		return err
	}

	for _, dbName := range cfg.extraDatabases {
		_, err = pool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{dbName}.Sanitize()))
		if err != nil {
			return fmt.Errorf("failed to create database %s: %w", dbName, err)
		}
	}

	for _, schema := range cfg.extraSchemas {
		schemaPool, schemaErr := pgxpool.New(ctx, credentials.ConnStr(schema.Database))
		if schemaErr != nil {
			return schemaErr
		}
		_, err = schemaPool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", pgx.Identifier{schema.Name}.Sanitize()))
		if err != nil {
			schemaPool.Close()
			return fmt.Errorf("failed to create schema in database %s: %w", schema.Database, err)
		}
		schemaPool.Close()
	}

	for _, extension := range cfg.installedExtensions {
		extPool, extErr := pgxpool.New(ctx, credentials.ConnStr(extension.Database))
		if extErr != nil {
			return extErr
		}

		_, err = extPool.Exec(ctx, fmt.Sprintf("CREATE EXTENSION %s", pgx.Identifier{extension.Name}.Sanitize()))
		if err != nil {
			extPool.Close()
			return err
		}
		extPool.Close()
	}

	for _, role := range cfg.extraRoles {
		_, err = pool.Exec(ctx, fmt.Sprintf("CREATE ROLE %s WITH LOGIN PASSWORD $$%s$$",
			pgx.Identifier{role.Role}.Sanitize(),
			role.Password))
		if err != nil {
			return fmt.Errorf("failed to create role %s: %w", role.Role, err)
		}

		_, err = pool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s OWNER %s",
			pgx.Identifier{role.Database}.Sanitize(),
			pgx.Identifier{role.Role}.Sanitize()))
		if err != nil {
			return fmt.Errorf("failed to create database %s: %w", role.Database, err)
		}

		rolePool, err := pgxpool.New(ctx, credentials.ConnStr(role.Database))
		if err != nil {
			return err
		}

		_, err = rolePool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s AUTHORIZATION %s",
			pgx.Identifier{role.Schema}.Sanitize(),
			pgx.Identifier{role.Role}.Sanitize()))
		if err != nil {
			rolePool.Close()
			return fmt.Errorf("failed to create schema %s in database %s: %w", role.Schema, role.Database, err)
		}
		rolePool.Close()
	}

	for _, table := range cfg.extraTables {
		var tablePool *pgxpool.Pool
		var err error

		if table.Role != nil {
			connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
				table.Role.Role, table.Role.Password,
				credentials.host, credentials.port.Port(), table.Database)
			tablePool, err = pgxpool.New(ctx, connStr)
		} else {
			return fmt.Errorf("WithExtraTables require a non-nil Role")
		}

		if err != nil {
			return err
		}

		if table.Schema != "" {
			_, err = tablePool.Exec(ctx, fmt.Sprintf("SET search_path TO %s", pgx.Identifier{table.Schema}.Sanitize()))
			if err != nil {
				tablePool.Close()
				return fmt.Errorf("failed to set search_path to %s: %w", table.Schema, err)
			}
		}

		_, err = tablePool.Exec(ctx, table.DDL)
		if err != nil {
			tablePool.Close()
			return fmt.Errorf("failed to execute DDL in database %s: %w", table.Database, err)
		}

		tablePool.Close()
	}

	return nil
}

func initScriptPath(scriptName string) string {
	wd, _ := os.Getwd()
	moduleRoot := findModuleRoot(wd)
	sqlPath := filepath.Join(moduleRoot, "internal", "testutil", "testdata", "setup", scriptName)
	return sqlPath
}

func findModuleRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// we got all the way to filesystem root
			return ""
		}
		dir = parent
	}
}
