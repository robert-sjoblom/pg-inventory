// Extractor - Connects to PostgreSQL instances and extracts inventory data from each database.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/jackc/pgx/v5/pgxpool"
	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/store"
)

func main() {
	fmt.Println("PostgreSQL Inventory Extractor")

	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatalf("DATABASE_URL not set")
	}

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// TODO: we need to check if the db is available, as in the original program
	// we don't want to spam connection attempts if the db is shutdown.
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	defer pool.Close()

	st := store.NewStore(pool)

	grpcServer := grpc.NewServer()
	extractorv1.RegisterExtractorServiceServer(grpcServer, extractor.NewServer(st))
	registerDev(grpcServer)

	log.Println("Extractor service listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
	}
}
