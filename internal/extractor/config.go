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
	db_host := flag.String("db-host", "localhost", "database host to connect to (default localhost)")
	db_port := flag.Int("db-port", 5432, "the database port to connect to (default 5432)")
	db_user := flag.String("db-user", "pgmonitor", "the database user to connect with")
	db_name := flag.String("db-name", "postgres", "the database to connect to (default postgres)")
	db_sslmode := flag.String("db-sslmode", "verify-full", "SSL mode to use (default verify-full)")
	db_sslcert := flag.String("db-sslcert", "", "path to SSL certificate")
	db_sslkey := flag.String("db-sslkey", "", "path to SSL key")
	db_sslrootcert := flag.String("db-sslrootcert", "", "path to root CA cert")
	listenAddr := flag.String("listen", ":50051", "listen address (default :50051)")

	flag.Parse()

	return &Config{
		dbHost:        *db_host,
		dbPort:        *db_port,
		dbUser:        *db_user,
		dbName:        *db_name,
		dbSSLMode:     *db_sslmode,
		dbSSLCert:     *db_sslcert,
		dbSSLKey:      *db_sslkey,
		dbSSLRootCert: *db_sslrootcert,
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
