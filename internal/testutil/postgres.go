// Package testutil contains test utilities like bufconn or test containers
package testutil

import (
	"context"
	"fmt"
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
}

type ExtraSchema struct {
	Name     string
	Database string
}

type InstalledExtension struct {
	Name     string
	Database string
}

type SetupOption func(*setupConfig)

func WithClusterConfig(clusterName, stanza string) SetupOption {
	return func(cfg *setupConfig) {
		cfg.clusterName = clusterName
		cfg.stanza = stanza
	}
}

func WithExtraDatabases(names ...string) SetupOption {
	return func(cfg *setupConfig) {
		cfg.extraDatabases = append(cfg.extraDatabases, names...)
	}
}

func WithExtraSchemas(schemas ...ExtraSchema) SetupOption {
	return func(cfg *setupConfig) {
		cfg.extraSchemas = append(cfg.extraSchemas, schemas...)
	}
}

func WithInstalledExtensions(extensions ...InstalledExtension) SetupOption {
	return func(cfg *setupConfig) {
		cfg.installedExtensions = append(cfg.installedExtensions, extensions...)
	}
}

type TestDbCredentials struct {
	dbuser string
	dbpass string
	host   string
	port   nat.Port
}

func (d *TestDbCredentials) ConnStr(db string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", d.dbuser, d.dbpass, d.host, d.port.Port(), db)
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
		pool, err = pgxpool.New(ctx, credentials.ConnStr(schema.Database))
		if err != nil {
			return err
		}
		_, err = pool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", pgx.Identifier{schema.Name}.Sanitize()))
		if err != nil {
			return fmt.Errorf("failed to create schema in database %s: %w", schema.Database, err)
		}
		pool.Close()
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
