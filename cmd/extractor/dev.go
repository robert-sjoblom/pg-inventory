//go:build dev

package main

import (
	"log/slog"
	"os"

	"github.com/robert-sjoblom/pg-inventory/internal/extractor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func registerDev(s *grpc.Server) {
	reflection.Register(s)
}

func initializeLogger(c *extractor.Config) (*slog.Logger, error) {
	level, err := c.LogLevel()
	if err != nil {
		return nil, err
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})), nil
}
