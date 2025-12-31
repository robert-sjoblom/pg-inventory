// Extractor - Connects to PostgreSQL instances and extracts inventory data from each database.
package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor"
)

func main() {
	fmt.Println("PostgreSQL Inventory Extractor")

	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	extractorv1.RegisterExtractorServiceServer(grpcServer, newServer())
	registerDev(grpcServer)

	log.Println("Extractor service listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func newServer() *extractor.Server {
	return &extractor.Server{}
}
