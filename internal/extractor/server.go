// Package extractor implements the gRPC server for extracting inventory data.
package extractor

import (
	"context"

	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/store"
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
		return nil, err
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
