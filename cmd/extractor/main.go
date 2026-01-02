// Extractor - Connects to PostgreSQL instances and extracts inventory data from each database.
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/jackc/pgx/v5/pgxpool"
	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor"
	"github.com/robert-sjoblom/pg-inventory/internal/extractor/store"
)

func main() {
	// This calls flag.Parse() behind the scenes
	config := extractor.NewConfig()

	logger, err := initializeLogger(config)
	if err != nil {
		log.Fatalf("%v", err)
	}

	logger.Info("PostgreSQL Inventory Extractor")

	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", config.ListenAddr)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		os.Exit(1)
	}

	dbCredentials, err := config.NewCredentials()
	if err != nil {
		logger.Error("failed to parse db credentials", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), dbCredentials.ConnStr())
	if err != nil {
		logger.Error("failed to create pool to database", "err", err)
		os.Exit(1)
	}

	// TODO: we need to check if the db is available, as in the original program
	// we don't want to spam connection attempts if the db is shutdown.
	if err := pool.Ping(context.Background()); err != nil {
		logger.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}

	st := store.NewStore(pool)

	grpcServer := grpc.NewServer()
	extractorv1.RegisterExtractorServiceServer(grpcServer, extractor.NewServer(st))
	registerDev(grpcServer)

	errCh := make(chan error, 1)

	logger.Info("Extractor service listening", "address", config.ListenAddr)
	go func() {
		errCh <- grpcServer.Serve(lis)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	select {
	case <-sigCh:
		logger.Info("Shutdown signal received, stopping gracefully")
		grpcServer.GracefulStop()
	case err := <-errCh:
		if err != nil {
			logger.Error("server stopped with error", "err", err)
		}
	}

	pool.Close()
	logger.Info("Shutdown complete")
}
