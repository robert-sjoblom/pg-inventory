// Package extractor implements the gRPC server for extracting inventory data.
package extractor

import (
	"context"

	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/store"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	extractorv1.UnimplementedExtractorServiceServer
	store *store.Store
}

func NewServer(s *store.Store) *Server {
	return &Server{
		store: s,
	}
}

func (s *Server) ListDatabases(ctx context.Context, req *extractorv1.ListDatabasesRequest) (*extractorv1.ListDatabasesResponse, error) {
	dbs, err := s.store.ListDatabases(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list databases: %v", err)
	}

	databases := make([]*extractorv1.Database, 0, len(dbs))
	for _, db := range dbs {
		databases = append(databases, &extractorv1.Database{
			Oid:  db.Oid,
			Name: db.Name,
		})
	}

	return &extractorv1.ListDatabasesResponse{Databases: databases}, nil
}

func (s *Server) ListBackups(ctx context.Context, req *extractorv1.ListBackupsRequest) (*extractorv1.ListBackupsResponse, error) {
	pgbackrestInfo, err := s.store.ListBackups(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	if len(pgbackrestInfo) == 0 {
		return nil, status.Errorf(codes.NotFound, "no stanza found")
	}

	info := pgbackrestInfo[0]
	if len(info.Db) == 0 {
		return nil, status.Errorf(codes.Internal, "pgbackrest info is not complete, missing db field")
	}

	dbVersion := info.Db[0].Version
	if dbVersion == "" {
		return nil, status.Errorf(codes.Internal, "could not extract PG version from pgbackrest info")
	}

	stanza := info.Name
	backups := info.Backup

	backupInfo := make([]*extractorv1.BackupInfo, 0, len(backups))
	for _, backup := range backups {
		backupInfo = append(backupInfo, &extractorv1.BackupInfo{
			Label:           backup.Label,
			Type:            backup.Type,
			TimestampStart:  backup.Timestamp.Start,
			TimestampStop:   backup.Timestamp.Stop,
			BackupSize:      backup.Info.Size,
			RepoSize:        backup.Info.Repository.Size,
			DatabaseVersion: dbVersion,
			Error:           backup.Error,
			RepoKey:         uint32(backup.Database.RepoKey),
		})
	}

	return &extractorv1.ListBackupsResponse{
		Stanza:  stanza,
		Backups: backupInfo,
	}, nil
}

func (s *Server) GetServerInfo(ctx context.Context, req *extractorv1.GetServerInfoRequest) (*extractorv1.GetServerInfoResponse, error) {
	info, err := s.store.GetServerInfo(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get server info: %v", err)
	}

	resp, err := types.ServerInfoToProto(&info, s.store.ClusterName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert ServerInfo to RPC: %v", err)
	}

	return resp, nil
}
