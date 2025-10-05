package main

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("I have no idea what I'm doing. We'll get there.")

	db, err := sqlx.Connect("postgres", "user=pgmonitor dbname=postgres sslmode=disable password=password port=15432")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	dbNames := []string{}
	err = db.Select(&dbNames, "SELECT datname FROM pg_database WHERE datistemplate = false;")

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to query database names: %v\n", err)
		os.Exit(1)
	}

	for _, name := range dbNames {
		fmt.Printf("Database: %s\n", name)
	}

	if err := db.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing database connection: %v\n", err)
		os.Exit(1)
	}
}
