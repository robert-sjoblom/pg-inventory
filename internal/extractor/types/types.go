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

type Sequence struct {
	Name     string `json:"name"`
	Schema   string `json:"schema"`
	Owner    string `json:"owner"`
	Database string `json:"database"`
	DataType string `json:"data_type"`
	Oid      uint32 `json:"oid"`
}

type Function struct {
	Name              string `json:"name"`
	Schema            string `json:"schema"`
	Owner             string `json:"owner"`
	Database          string `json:"database"`
	Language          string `json:"language"`
	ReturnType        string `json:"return_type"`
	IdentityArguments string `json:"identity_arguments"`
	Oid               uint32 `json:"oid"`
}

type TablesInfo struct {
	Database string
	Tables   []*Table
}

type Table struct {
	Name             string             `json:"name"`
	Schema           string             `json:"schema"`
	Owner            string             `json:"owner"`
	Comment          *string            `json:"comment"`
	TableColumns     []*TableColumn     `json:"table_columns"`
	TableIndexes     []*TableIndex      `json:"table_indexes"`
	TableConstraints []*TableConstraint `json:"table_constraints"`
	Inheritance      TableInheritance   `json:"inheritance"`
	Stats            TableStats         `json:"stats"`
	Oid              uint32             `json:"oid"`
}

type TableStats struct {
	RowEstimate    int64  `json:"row_estimate"`
	TotalSizeBytes uint64 `json:"total_size_bytes"`
	HeapSizeBytes  uint64 `json:"heap_size_bytes"`
	ToastSizeBytes uint64 `json:"toast_size_bytes"`
}

type TableInheritance struct {
	ParentTables []*InheritanceRelation `json:"parent_tables"` // Tables this table inherits from
}

type InheritanceRelation struct {
	Name string `json:"name"` // Fully qualified name (e.g., "schema".table)
	Oid  uint32 `json:"oid"`
}

type TableColumn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	NotNull     bool   `json:"not_null"`
	IsInherited bool   `json:"is_inherited"`
}

type TableIndex struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // Access method: btree, hash, gin, gist, brin, etc.
	Definition  string   `json:"definition"`
	Columns     []string `json:"columns"` // Column names in index order
	SizeBytes   uint64   `json:"size_bytes"`
	IsUnique    bool     `json:"is_unique"`
	IsPrimary   bool     `json:"is_primary"`
	IsExclusion bool     `json:"is_exclusion"`
	IsPartial   bool     `json:"is_partial"` // Has a "WHERE" clause in definition
	IsValid     bool     `json:"is_valid"`
	IsInherited bool     `json:"is_inherited"` // Propagated from parent table (e.g., partition inherits from partitioned table)
}

type TableConstraint struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"` // PK, UNIQ, FK, CHK, EXCL
	ForeignTable   string   `json:"foreign_table"`
	Definition     string   `json:"definition"`
	LocalColumns   []string `json:"local_columns"`
	ForeignColumns []string `json:"foreign_columns"`
}

func ExtensionsToProto(avail []*AvailableExtension, inst []*InstalledExtension) *extractorv1.ListExtensionsResponse {
	installed := make([]*extractorv1.InstalledExtension, 0, len(inst))
	for _, ins := range inst {
		installed = append(installed, &extractorv1.InstalledExtension{
			Oid:      ins.Oid,
			Name:     ins.Name,
			Version:  ins.Version,
			Schema:   ins.Schema,
			Database: ins.Database,
		})
	}

	available := make([]*extractorv1.AvailableExtension, 0, len(avail))
	for _, av := range avail {
		available = append(available, &extractorv1.AvailableExtension{
			Name:           av.Name,
			DefaultVersion: av.DefaultVersion,
		})
	}

	return &extractorv1.ListExtensionsResponse{
		Installed: installed,
		Available: available,
	}
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

func (d Database) ToProto() *extractorv1.Database {
	return &extractorv1.Database{
		Oid:  d.Oid,
		Name: d.Name,
	}
}

func (s *Schema) ToProto() *extractorv1.Schema {
	return &extractorv1.Schema{
		Oid:      s.Oid,
		Name:     s.Name,
		Owner:    s.Owner,
		Database: s.Database,
	}
}

func (s *Sequence) ToProto() *extractorv1.Sequence {
	return &extractorv1.Sequence{
		Oid:      s.Oid,
		Name:     s.Name,
		Schema:   s.Schema,
		Owner:    s.Owner,
		Database: s.Database,
		DataType: s.DataType,
	}
}

func (f *Function) ToProto() *extractorv1.Function {
	return &extractorv1.Function{
		Oid:               f.Oid,
		Name:              f.Name,
		Schema:            f.Schema,
		Owner:             f.Owner,
		Database:          f.Database,
		Language:          f.Language,
		ReturnType:        f.ReturnType,
		IdentityArguments: f.IdentityArguments,
	}
}

func (b *Backup) ToProto(dbVersion string) *extractorv1.BackupInfo {
	return &extractorv1.BackupInfo{
		Label:           b.Label,
		Type:            b.Type,
		TimestampStart:  b.Timestamp.Start,
		TimestampStop:   b.Timestamp.Stop,
		BackupSize:      b.Info.Size,
		RepoSize:        b.Info.Repository.Size,
		DatabaseVersion: dbVersion,
		Error:           b.Error,
		RepoKey:         uint32(b.Database.RepoKey),
	}
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
