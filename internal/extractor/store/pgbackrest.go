package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/robert-sjoblom/pg-inventory/internal/extractor/types"
)

func (s *Store) ListBackups(ctx context.Context) ([]types.PgbackrestInfo, error) {
	stanza, err := s.getStanza(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgbackrest JSON: %w", err)
	}

	if !isValidStanzaName(stanza) {
		return nil, fmt.Errorf("invalid stanza name: %q", stanza)
	}

	res := exec.CommandContext(ctx, "pgbackrest", "info", "--stanza="+stanza, "--output=json")

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

var stanzaNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,62}$`)

func isValidStanzaName(s string) bool {
	return stanzaNamePattern.MatchString(s)
}
