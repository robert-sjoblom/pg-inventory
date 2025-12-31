// Package testutil contains test utilities like bufconn or test containers
package testutil

import (
	"context"
	"net"
	"testing"

	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// DialBufconn creates an in-memory gRPC connection for testing.
// It returns a gRPC ClientConn and a cleanup function.
func DialBufconn(t *testing.T, srv extractorv1.ExtractorServiceServer) *grpc.ClientConn {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()

	extractorv1.RegisterExtractorServiceServer(s, srv)

	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial bufconn: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
		s.Stop()
	})

	return conn
}
