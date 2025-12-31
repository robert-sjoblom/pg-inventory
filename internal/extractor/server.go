// Package extractor implements the gRPC server for extracting inventory data.
package extractor

import (
	"context"

	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
)

type Server struct {
	extractorv1.UnimplementedExtractorServiceServer
}

func (s *Server) ListDatabases(ctx context.Context, req *extractorv1.ListDatabasesRequest) (*extractorv1.ListDatabasesResponse, error) {
	databases := []*extractorv1.Database{
		{
			Name: "postgres",
			Oid:  1,
		},
		{
			Name: "database_one",
			Oid:  2,
		},
		{
			Name: "database_two",
			Oid:  1,
		},
	}
	return &extractorv1.ListDatabasesResponse{Databases: databases}, nil
}
