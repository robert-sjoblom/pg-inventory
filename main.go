package main

import (
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/robert-sjoblom/pg-inventory/internal/cmd"
	"github.com/robert-sjoblom/pg-inventory/internal/config"
	"github.com/robert-sjoblom/pg-inventory/internal/inventory"
)

func main() {
	rootCmd := cmd.NewRootCommand(runInventory)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runInventory(cfg *config.Config) error {
	fmt.Println("I have no idea what I'm doing. We'll get there.")

	dbNames, err := inventory.ListDatabases(cfg.ConnString())
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	fmt.Println("Databases:")
	for _, name := range dbNames {
		fmt.Printf("  - %s\n", name)
	}

	return nil
}
