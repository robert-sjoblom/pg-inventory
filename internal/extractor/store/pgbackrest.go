package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/robert-sjoblom/pg-inventory/internal/extractor/types"
)

func (s *Store) ListBackups(ctx context.Context) ([]types.PgbackrestInfo, error) {
	res := exec.CommandContext(ctx, "pgbackrest", "info", "--stanza="+s.Stanza, "--output=json")

	data, err := res.Output()
	if err != nil {
		return nil, fmt.Errorf("pgbackrest info failed: %w", err)
	}

	return parseBackrestInfo(data)
}

func parseBackrestInfo(data []byte) ([]types.PgbackrestInfo, error) {
	var info []types.PgbackrestInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return info, nil
}
