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

func (s *Server) ListSchemas(ctx context.Context, req *extractorv1.ListSchemasRequest) (*extractorv1.ListSchemasResponse, error) {
	schemas, err := s.store.ListSchemas(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list schemas: %v", err)
	}

	resp := make([]*extractorv1.Schema, 0, len(schemas))
	for _, schema := range schemas {
		resp = append(resp, &extractorv1.Schema{
			Oid:      schema.Oid,
			Name:     schema.Name,
			Owner:    schema.Owner,
			Database: schema.Database,
		})
	}

	return &extractorv1.ListSchemasResponse{
		Schemas: resp,
	}, nil
}

func (s *Server) ListExtensions(ctx context.Context, req *extractorv1.ListExtensionsRequest) (*extractorv1.ListExtensionsResponse, error) {
	available, installed, err := s.store.ListExtensions(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list extensions: %v", err)
	}

	return types.ExtensionsToProto(available, installed), nil
}

func (s *Server) ListSequences(ctx context.Context, req *extractorv1.ListSequencesRequest) (*extractorv1.ListSequencesResponse, error) {
	seqs, err := s.store.ListSequences(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sequences: %v", err)
	}

	resp := make([]*extractorv1.Sequence, 0, len(seqs))
	for _, seq := range seqs {
		resp = append(resp, &extractorv1.Sequence{
			Oid:      seq.Oid,
			Name:     seq.Name,
			Schema:   seq.Schema,
			Owner:    seq.Owner,
			Database: seq.Database,
			DataType: seq.DataType,
		})
	}

	return &extractorv1.ListSequencesResponse{
		Sequences: resp,
	}, nil
}

func (s *Server) ListFunctions(ctx context.Context, req *extractorv1.ListFunctionsRequest) (*extractorv1.ListFunctionsResponse, error) {
	functions, err := s.store.ListFunctions(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list functions: %v", err)
	}

	resp := make([]*extractorv1.Function, 0, len(functions))
	for _, function := range functions {
		resp = append(resp, &extractorv1.Function{
			Oid:               function.Oid,
			Name:              function.Name,
			Schema:            function.Schema,
			Owner:             function.Owner,
			Database:          function.Database,
			Language:          function.Language,
			ReturnType:        function.ReturnType,
			IdentityArguments: function.IdentityArguments,
		})
	}

	return &extractorv1.ListFunctionsResponse{
		Functions: resp,
	}, nil
}

func (s *Server) ListTables(ctx context.Context, req *extractorv1.ListTablesRequest) (*extractorv1.ListTablesResponse, error) {
	tablesInfo, err := s.store.ListTables(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tables: %v", err)
	}

	databaseTables := make([]*extractorv1.DatabaseTables, 0, len(tablesInfo))
	for _, dbTables := range tablesInfo {
		tables := make([]*extractorv1.Table, 0, len(dbTables.Tables))
		for _, t := range dbTables.Tables {
			tables = append(tables, tableToProto(t))
		}
		databaseTables = append(databaseTables, &extractorv1.DatabaseTables{
			Database: dbTables.Database,
			Tables:   tables,
		})
	}

	return &extractorv1.ListTablesResponse{
		DatabaseTables: databaseTables,
	}, nil
}

func tableToProto(t *types.Table) *extractorv1.Table {
	columns := make([]*extractorv1.TableColumn, 0, len(t.TableColumns))
	for _, col := range t.TableColumns {
		columns = append(columns, &extractorv1.TableColumn{
			Name:        col.Name,
			Type:        col.Type,
			NotNull:     col.NotNull,
			IsInherited: col.IsInherited,
		})
	}

	indexes := make([]*extractorv1.TableIndex, 0, len(t.TableIndexes))
	for _, idx := range t.TableIndexes {
		indexes = append(indexes, &extractorv1.TableIndex{
			Name:        idx.Name,
			Type:        idx.Type,
			Columns:     idx.Columns,
			IsUnique:    idx.IsUnique,
			IsPrimary:   idx.IsPrimary,
			IsExclusion: idx.IsExclusion,
			IsPartial:   idx.IsPartial,
			IsValid:     idx.IsValid,
			IsInherited: idx.IsInherited,
			SizeBytes:   idx.SizeBytes,
			Definition:  idx.Definition,
		})
	}

	constraints := make([]*extractorv1.TableConstraint, 0, len(t.TableConstraints))
	for _, con := range t.TableConstraints {
		pc := &extractorv1.TableConstraint{
			Name:           con.Name,
			Type:           con.Type,
			LocalColumns:   con.LocalColumns,
			ForeignColumns: con.ForeignColumns,
			Definition:     con.Definition,
		}
		if con.ForeignTable != "" {
			pc.ForeignTable = &con.ForeignTable
		}
		constraints = append(constraints, pc)
	}

	parentTables := make([]*extractorv1.InheritanceRelation, 0, len(t.Inheritance.ParentTables))
	for _, parent := range t.Inheritance.ParentTables {
		parentTables = append(parentTables, &extractorv1.InheritanceRelation{
			Name: parent.Name,
			Oid:  parent.Oid,
		})
	}

	protoTable := &extractorv1.Table{
		Oid:         t.Oid,
		Name:        t.Name,
		Schema:      t.Schema,
		Owner:       t.Owner,
		Comment:     t.Comment,
		Columns:     columns,
		Indexes:     indexes,
		Constraints: constraints,
		Stats: &extractorv1.TableStats{
			RowEstimate:    t.Stats.RowEstimate,
			TotalSizeBytes: t.Stats.TotalSizeBytes,
			HeapSizeBytes:  t.Stats.HeapSizeBytes,
			ToastSizeBytes: t.Stats.ToastSizeBytes,
		},
		Inheritance: &extractorv1.TableInheritance{
			ParentTables: parentTables,
		},
	}

	return protoTable
}
