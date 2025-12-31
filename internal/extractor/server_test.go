package extractor

import (
	"context"
	"testing"

	extractorv1 "github.com/robert-sjoblom/pg-inventory/gen/extractor/v1"
	"github.com/robert-sjoblom/pg-inventory/internal/testutil"
)

func TestListDatabases(t *testing.T) {
	conn := testutil.DialBufconn(t, &Server{})
	client := extractorv1.NewExtractorServiceClient(conn)

	resp, err := client.ListDatabases(context.Background(), &extractorv1.ListDatabasesRequest{})
	if err != nil {
		t.Fatalf("ListDatabases failed: %v", err)
	}

	if len(resp.Databases) != 3 {
		t.Errorf("expected 3 databases, got %d", len(resp.Databases))
	}

	if resp.Databases[0].Name != "postgres" {
		t.Errorf("expected first db to be 'postgres', got %q", resp.Databases[0].Name)
	}
}
