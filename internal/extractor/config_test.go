package extractor

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbConnStrWithSslMode(t *testing.T) {
	db_credentials := DbCredentials{
		dbHost:        "localhost",
		dbPort:        5432,
		dbUser:        "testuser",
		dbName:        "testdb",
		dbSSLMode:     "verify-full",
		dbSSLCert:     "/path/to/cert.crt",
		dbSSLKey:      "/path/to/key.key",
		dbSSLRootCert: "/path/to/ca.crt",
	}

	actual := db_credentials.ConnStr()
	expected := "postgres://testuser@localhost:5432/testdb?sslcert=%2Fpath%2Fto%2Fcert.crt&sslkey=%2Fpath%2Fto%2Fkey.key&sslmode=verify-full&sslrootcert=%2Fpath%2Fto%2Fca.crt"

	assert.Equal(t, expected, actual, "correct conn string for ssl mode")
}

func TestDbConnStrWitoutSsl(t *testing.T) {
	db_credentials := DbCredentials{
		dbHost:        "localhost",
		dbPort:        5432,
		dbUser:        "testuser",
		dbName:        "testdb",
		dbSSLMode:     "disable",
		dbSSLCert:     "/path/to/cert.crt",
		dbSSLKey:      "/path/to/key.key",
		dbSSLRootCert: "/path/to/ca.crt",
	}

	actual := db_credentials.ConnStr()
	expected := "postgres://testuser@localhost:5432/testdb?sslmode=disable"

	assert.Equal(t, expected, actual, "correct conn string for ssl mode")
}

func TestIsValidSSLSettings(t *testing.T) {
	certFile := filepath.Join(t.TempDir(), "cert.crt")
	err := os.WriteFile(certFile, []byte("fake cert"), 0o600)
	if err != nil {
		log.Fatalf("could not write tempfile: %v", err)
	}

	tests := []struct {
		name        string
		sslmode     string
		sslcert     string
		sslkey      string
		sslrootcert string
		wantErr     bool
	}{
		{
			name:    "disable mode skips validation",
			sslmode: "disable",
			wantErr: false,
		},
		{
			name:        "emptry string when not disabled",
			sslmode:     "verify-full",
			sslcert:     "",
			sslkey:      "/path/to/key",
			sslrootcert: "/path/to/ca",
			wantErr:     true,
		},
		{
			name:        "cert present when not disabled",
			sslmode:     "verify-full",
			sslcert:     certFile,
			sslkey:      certFile,
			sslrootcert: certFile,
			wantErr:     false,
		},
		{
			name:        "missing cert when not disabled",
			sslmode:     "verify-full",
			sslcert:     "nonexistent",
			sslkey:      "/path/to/key",
			sslrootcert: "/path/to/ca",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isValidSSLSettings(tt.sslmode, tt.sslcert, tt.sslkey, tt.sslrootcert)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
			}
		})
	}
}
