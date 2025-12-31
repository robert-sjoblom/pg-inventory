//go:build dev

package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func registerDev(s *grpc.Server) {
	reflection.Register(s)
}
