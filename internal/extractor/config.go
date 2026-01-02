package extractor

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
)

// Holds all the config for the extractor application; can in turn create derived
// structs, such as DbCredentials
type Config struct {
	ListenAddr string
	logLevel   string
	DbCredentials
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
	logLevel := flag.String("log-level", "info", "log level, choose between [debug, info, warn, error] (default info)")

	flag.Parse()

	return &Config{
		DbCredentials: DbCredentials{
			dbHost:        *dbHost,
			dbPort:        *dbPort,
			dbUser:        *dbUser,
			dbName:        *dbName,
			dbSSLMode:     *dbSslmode,
			dbSSLCert:     *dbSslcert,
			dbSSLKey:      *dbSslkey,
			dbSSLRootCert: *dbSslrootcert,
		},
		logLevel:   *logLevel,
		ListenAddr: *listenAddr,
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
	if err := isValidSSLSettings(c.dbSSLMode, c.dbSSLCert, c.dbSSLKey, c.dbSSLRootCert); err != nil {
		return nil, err
	}

	return &c.DbCredentials, nil
}

func (c *Config) LogLevel() (slog.Level, error) {
	switch strings.ToLower(c.logLevel) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("log-level must be one of [debug, info, warn, error], got: %v", c.logLevel)
	}
}

// Generate a certificate-based connection string for the default (Config) database
func (d *DbCredentials) ConnStr() string {
	return d.ConnStrForDb(d.dbName)
}

// Generate a certificate-based connection string for the given database name
func (d *DbCredentials) ConnStrForDb(dbName string) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.User(d.dbUser),
		Host:   fmt.Sprintf("%s:%d", d.dbHost, d.dbPort),
		Path:   dbName,
	}

	q := u.Query()
	if d.dbSSLMode == "disable" {
		q.Set("sslmode", "disable")
	}

	if d.dbSSLMode == "verify-ca" || d.dbSSLMode == "verify-full" {
		q.Set("sslmode", d.dbSSLMode)
		q.Set("sslcert", d.dbSSLCert)
		q.Set("sslkey", d.dbSSLKey)
		q.Set("sslrootcert", d.dbSSLRootCert)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func isValidSSLSettings(sslmode, sslcert, sslkey, sslrootcert string) error {
	needsCerts := sslmode == "verify-ca" || sslmode == "verify-full"

	if !needsCerts {
		return nil
	}

	if sslcert == "" || sslkey == "" || sslrootcert == "" {
		return errors.New("SSL certificates required for verify-ca and verify-full modes")
	}

	for _, path := range []string{sslcert, sslkey, sslrootcert} {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("SSL certificate file not found: %s", path)
			}
			return fmt.Errorf("cannot access: %w", err)
		}
	}

	return nil
}
