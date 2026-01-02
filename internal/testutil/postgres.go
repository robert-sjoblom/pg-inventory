// Package testutil contains test utilities like bufconn or test containers
package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type DbCredentials struct {
	dbuser string
	dbpass string
	host   string
	port   nat.Port
}

func (d *DbCredentials) ConnStr(db string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", d.dbuser, d.dbpass, d.host, d.port.Port(), db)
}

// StartPostgres launches a postgres testcontainer instance
func StartPostgres(t *testing.T) *DbCredentials {
	t.Helper()
	ctx := context.Background()

	db_user := "postgres"
	db_pass := "postgres"

	container, err := postgres.Run(ctx, "postgres:15-bullseye",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername(db_user),
		postgres.WithPassword(db_pass),
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

	return &DbCredentials{
		dbuser: db_user,
		dbpass: db_pass,
		host:   host,
		port:   port,
	}
}
