package cmd

import (
	"github.com/robert-sjoblom/pg-inventory/internal/config"
	"github.com/spf13/cobra"
)

// NewRootCommand creates and returns the root cobra command.
// The inventoryFunc is called when the command is executed.
func NewRootCommand(inventoryFunc func(*config.Config) error) *cobra.Command {
	var cfg config.Config

	rootCmd := &cobra.Command{
		Use:   "pg-inventory",
		Short: "PostgreSQL database inventory tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			return inventoryFunc(&cfg)
		},
	}

	// Database connection flags
	rootCmd.Flags().StringVarP(&cfg.User, "user", "u", "pgmonitor", "Database user")
	rootCmd.Flags().StringVarP(&cfg.Password, "password", "p", "password", "Database password")
	rootCmd.Flags().StringVar(&cfg.Host, "host", "localhost", "Database host")
	rootCmd.Flags().StringVarP(&cfg.Port, "port", "P", "15432", "Database port")
	rootCmd.Flags().StringVarP(&cfg.Database, "database", "d", "postgres", "Database name")
	rootCmd.Flags().StringVar(&cfg.SSLMode, "sslmode", "disable", "SSL mode (disable, require, verify-ca, verify-full)")

	return rootCmd
}
