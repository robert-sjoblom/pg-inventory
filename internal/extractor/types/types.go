// Package types holds domain types for the extractor
package types

import (
	"fmt"
	"regexp"
	"strings"

	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
)

type Database struct {
	Name string
	Oid  uint32
}

type PgbackrestInfo struct {
	Name   string         `json:"name"`
	Backup []Backup       `json:"backup"`
	Db     []PgbackrestDb `json:"db"`
}

type Backup struct {
	Label     string             `json:"label"`
	Type      string             `json:"type"`
	Timestamp BackupTimestamp    `json:"timestamp"`
	Info      BackupSizeInfo     `json:"info"`
	Database  PgbackrestDatabase `json:"database"`
	Error     bool               `json:"error"`
}

type PgbackrestDatabase struct {
	Id      uint8 `json:"id"`
	RepoKey uint8 `json:"repo-key"`
}

type BackupSizeInfo struct {
	Delta      uint64         `json:"delta"`
	Repository PgbackrestRepo `json:"repository"`
	Size       uint64         `json:"size"`
}

type BackupTimestamp struct {
	Start int64 `json:"start"`
	Stop  int64 `json:"stop"`
}

type PgbackrestRepo struct {
	Delta uint64 `json:"delta"`
	Size  uint64 `json:"size"`
}

type PgbackrestDb struct {
	Version  string `json:"version"`
	SystemId uint64 `json:"system-id"`
	RepoKey  uint8  `json:"repo-key"`
	Id       uint8  `json:"id"`
}

// ServerInfo from PG server. Most of the fields are strings, because that's what
// settings return as. Native types where applicable.
type ServerInfo struct {
	PgVersion        string `json:"pg_version"`
	IsReadOnly       string `json:"is_read_only"`
	SslEnabled       string `json:"ssl_enabled"`
	ArchiveMode      string `json:"archive_mode"`
	DataDirectory    string `json:"data_directory"`
	WalLevel         string `json:"wal_level"`
	IsInRecovery     bool   `json:"is_in_recovery"`
	TimelineID       int8   `json:"timeline_id"`
	Port             int32  `json:"port"`
	MaxConnections   int32  `json:"max_connections"`
	SystemIdentifier int64  `json:"system_identifier"`
}

type Schema struct {
	Database string `json:"database"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	Oid      uint32 `json:"oid"`
}

type InstalledExtension struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Schema   string `json:"schema"`
	Database string `json:"database"`
	Oid      uint32 `json:"oid"`
}

type AvailableExtension struct {
	Name           string `json:"name"`
	DefaultVersion string `json:"default_version"`
}

func ServerInfoToProto(info *ServerInfo, clusterName string) (*extractorv1.GetServerInfoResponse, error) {
	pgVersion, err := parsePgVersion(info.PgVersion)
	if err != nil {
		return nil, err
	}

	isReadOnly, err := parsePgOnOffToBool(info.IsReadOnly)
	if err != nil {
		return nil, err
	}

	sslEnabled, err := parsePgOnOffToBool(info.SslEnabled)
	if err != nil {
		return nil, err
	}

	return &extractorv1.GetServerInfoResponse{
		PgVersion:        pgVersion,
		IsReadOnly:       isReadOnly,
		ClusterName:      clusterName,
		IsInRecovery:     info.IsInRecovery,
		SslEnabled:       sslEnabled,
		Port:             info.Port,
		MaxConnections:   info.MaxConnections,
		ArchiveMode:      info.ArchiveMode,
		DataDirectory:    info.DataDirectory,
		SystemIdentifier: info.SystemIdentifier,
		TimelineId:       int32(info.TimelineID),
		WalLevel:         info.WalLevel,
	}, nil
}

var pgVersionPattern = regexp.MustCompile(`^PostgreSQL (\d+\.\d+)`)

func parsePgVersion(pgVersion string) (string, error) {
	matches := pgVersionPattern.FindStringSubmatch(pgVersion)
	if len(matches) >= 2 {
		return matches[1], nil
	}

	return "", fmt.Errorf("invalid PostgreSQL version string: %q", pgVersion)
}

func parsePgOnOffToBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "off":
		return false, nil
	case "on":
		return true, nil
	default:
		return false, fmt.Errorf("could not parse as bool: %q", s)
	}
}
