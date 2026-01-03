// Package types holds domain types for the extractor
package types

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
