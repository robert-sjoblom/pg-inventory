//go:build !dev

package main

import (
	"log/slog"
	"os"

	"github.com/robert-sjoblom/pg-inventory/internal/extractor"
)

func registerDev(s any) {
	// no-op
}

func initializeLogger(c *extractor.Config) (*slog.Logger, error) {
	level, err := c.LogLevel()
	if err != nil {
		return nil, err
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})), nil
}
