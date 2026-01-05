package types

import (
	"testing"

	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
	"github.com/stretchr/testify/assert"
)

func TestServerInfoToProto(t *testing.T) {
	info := ServerInfo{
		PgVersion:        "PostgreSQL 15.13 (Debian 15.13-1.pgdg110+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 10.2.1-6) 10.2.1 20210110, 64-bit",
		IsInRecovery:     false,
		IsReadOnly:       "off",
		SslEnabled:       "off",
		Port:             5432,
		MaxConnections:   100,
		ArchiveMode:      "off",
		DataDirectory:    "/var/lib/postgresql/data",
		SystemIdentifier: 1, // This is unknowable until the container spins up and starts PG
		TimelineID:       1,
		WalLevel:         "replica",
	}

	actual, err := ServerInfoToProto(&info, "cluster-001")
	if err != nil {
		t.Fatalf("could not convert ServerInfo to proto: %v", err)
	}

	expected := &extractorv1.GetServerInfoResponse{
		ClusterName:      "cluster-001",
		PgVersion:        "15.13",
		IsReadOnly:       false,
		SslEnabled:       false,
		Port:             5432,
		MaxConnections:   100,
		ArchiveMode:      "off",
		DataDirectory:    "/var/lib/postgresql/data",
		SystemIdentifier: 1,
		TimelineId:       1,
		WalLevel:         "replica",
		IsInRecovery:     false,
	}

	assert.Equal(t, expected, actual)
}

func TestParsePgVersionString(t *testing.T) {
	tests := []struct {
		name      string
		pgVersion string
		expected  string
		wantErr   bool
	}{
		{
			name:      "version() output returns correct version",
			pgVersion: "PostgreSQL 15.13 (Debian 15.13-1.pgdg110+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 10.2.1-6) 10.2.1 20210110, 64-bit",
			expected:  "15.13",
			wantErr:   false,
		},
		{
			name:      "no PG version returns error",
			pgVersion: "Not PostgreSQL 15.13...",
			expected:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := parsePgVersion(tt.pgVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
			}

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestParsePgOnOffToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
		wantErr  bool
	}{
		{
			name:     "off returns false",
			input:    "off",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "on returns true",
			input:    "on",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "uppercase OFF returns false",
			input:    "OFF",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "uppercase ON returns true",
			input:    "ON",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "empty string returns error",
			input:    "",
			expected: false,
			wantErr:  true,
		},
		{
			name:     "invalid value returns error",
			input:    "foo",
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := parsePgOnOffToBool(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr %v, got %v", tt.wantErr, err)
			}

			assert.Equal(t, tt.expected, actual)
		})
	}
}
