package extractor

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

// Holds all the config for the extractor application; can in turn create derived
// structs, such as DbCredentials
type Config struct {
	dbHost        string
	dbUser        string
	dbName        string
	dbSSLMode     string
	dbSSLCert     string
	dbSSLKey      string
	dbSSLRootCert string
	ListenAddr    string
	dbPort        int
}

func NewConfig() *Config {
	dbHost := flag.String("db-host", "localhost", "database host to connect to (default localhost)")
	dbPort := flag.Int("db-port", 5432, "the database port to connect to (default 5432)")
	dbUser := flag.String("db-user", "pgmonitor", "the database user to connect with")
	dbName := flag.String("db-name", "postgres", "the database to connect to (default postgres)")
	dbSslmode := flag.String("db-sslmode", "verify-full", "SSL mode to use (default verify-full)")
	dbSslcert := flag.String("db-sslcert", "", "path to SSL certificate")
	dbSslkey := flag.String("db-sslkey", "", "path to SSL key")
	dbSslrootcert := flag.String("db-sslrootcert", "", "path to root CA cert")
	listenAddr := flag.String("listen", ":50051", "listen address (default :50051)")

	flag.Parse()

	return &Config{
		dbHost:        *dbHost,
		dbPort:        *dbPort,
		dbUser:        *dbUser,
		dbName:        *dbName,
		dbSSLMode:     *dbSslmode,
		dbSSLCert:     *dbSslcert,
		dbSSLKey:      *dbSslkey,
		dbSSLRootCert: *dbSslrootcert,
		ListenAddr:    *listenAddr,
	}
}

type DbCredentials struct {
	dbHost        string
	dbUser        string
	dbName        string
	dbSSLMode     string
	dbSSLCert     string
	dbSSLKey      string
	dbSSLRootCert string
	dbPort        int
}

func (c *Config) NewCredentials() (*DbCredentials, error) {
	if err := isValidSSLSettings(&c.dbSSLMode, &c.dbSSLCert, &c.dbSSLKey, &c.dbSSLRootCert); err != nil {
		return nil, err
	}

	return &DbCredentials{
		dbHost:        c.dbHost,
		dbPort:        c.dbPort,
		dbUser:        c.dbUser,
		dbName:        c.dbName,
		dbSSLMode:     c.dbSSLMode,
		dbSSLCert:     c.dbSSLCert,
		dbSSLKey:      c.dbSSLKey,
		dbSSLRootCert: c.dbSSLRootCert,
	}, nil
}

// Generate a connection string for the default (Config) database
func (d *DbCredentials) ConnStr() string {
	return d.ConnStrForDb(d.dbName)
}

func (d *DbCredentials) ConnStrForDb(dbName string) string {
	base := fmt.Sprintf("postgres://%s@%s:%d/%s", d.dbUser, d.dbHost, d.dbPort, dbName)

	if d.dbSSLMode == "disable" {
		return base + "?sslmode=disable"
	}

	return fmt.Sprintf("%s?sslmode=%s&sslcert=%s&sslkey=%s&sslrootcert=%s",
		base, d.dbSSLMode, d.dbSSLCert, d.dbSSLKey, d.dbSSLRootCert,
	)
}

func isValidSSLSettings(sslmode, sslcert, sslkey, sslrootcert *string) error {
	if *sslmode == "disable" {
		return nil
	}

	if *sslcert == "" || *sslkey == "" || *sslrootcert == "" {
		return errors.New("SSL certificates are required when SSL mode is not 'disable'")
	}

	for _, path := range []string{*sslcert, *sslkey, *sslrootcert} {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("SSL certificate file not found: %s", path)
			}
			return fmt.Errorf("cannot access: %w", err)
		}
	}

	return nil
}
