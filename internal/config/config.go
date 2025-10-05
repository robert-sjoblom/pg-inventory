package config

import "fmt"

// Config holds the database connection configuration.
type Config struct {
	User            string
	Password        string
	Host            string
	Port            string `default:"5432"`
	Database        string
	SSLMode         string `default:"disable"`
	ApplicationName string `default:"pg-inventory"`
}

// ConnString constructs the PostgreSQL connection string.
func (c *Config) ConnString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&application_name=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
		c.ApplicationName,
	)

}
