package inventory

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ListDatabases connects to the PostgreSQL server using the provided connection string
// and returns a list of non-template database names.
func ListDatabases(connString string) ([]string, error) {
	// TODO: this shouldn't handle connections directly, of course.
	db, err := sqlx.Connect("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect: %w", err)
	}

	defer func() {
		err = db.Close()
		if err != nil {
			fmt.Printf("failed to close database connection: %v\n", err)
		}
	}()

	var dbNames []string
	err = db.Select(&dbNames, "SELECT datname FROM pg_database WHERE datistemplate = false;")
	if err != nil {
		return nil, fmt.Errorf("unable to query: %w", err)
	}

	return dbNames, nil
}
