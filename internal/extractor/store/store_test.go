package store

import (
	"os"
	"testing"

	"github.com/robert-sjoblom/pg-inventory/internal/extractor/types"
	"github.com/stretchr/testify/assert"
)

func TestBackupInfoJsonToLocalType(t *testing.T) {
	data, err := os.ReadFile("../testdata/pgbackrest.json")
	if err != nil {
		t.Fatalf("failed to read pgbackrest.json: %v", err)
	}

	actual, err := parseBackrestInfo(data)
	if err != nil {
		t.Fatalf("failed to unmarshal json: %v", err)
	}

	expected := []types.PgbackrestInfo{
		{
			Name: "main",
			Backup: []types.Backup{
				{
					Label: "20260103-233937F",
					Type:  "full",
					Error: false,
					Database: types.PgbackrestDatabase{
						Id:      1,
						RepoKey: 1,
					},
					Info: types.BackupSizeInfo{
						Delta: 23246324,
						Size:  23246324,
						Repository: types.PgbackrestRepo{
							Delta: 3066593,
							Size:  3066593,
						},
					},
					Timestamp: types.BackupTimestamp{
						Start: 1767483577,
						Stop:  1767483580,
					},
				},
				{
					Label: "20260103-233937F_20260103-234057I",
					Type:  "incr",
					Error: false,
					Database: types.PgbackrestDatabase{
						Id:      1,
						RepoKey: 1,
					},
					Info: types.BackupSizeInfo{
						Delta: 1376514,
						Size:  23344628,
						Repository: types.PgbackrestRepo{
							Delta: 212178,
							Size:  3078147,
						},
					},
					Timestamp: types.BackupTimestamp{
						Start: 1767483657,
						Stop:  1767483664,
					},
				},
			},
			Db: []types.PgbackrestDb{
				{
					Id:       1,
					RepoKey:  1,
					SystemId: 7591284103390970243,
					Version:  "15",
				},
			},
		},
	}

	assert.Equal(t, expected, actual, "correct unmarshal of pgbackrest info")
}

func TestIsValidStanzaName(t *testing.T) {
	tests := []struct {
		name       string
		stanzaName string
		expected   bool
	}{
		{
			name:       "valid stanza name",
			stanzaName: "stanza-123",
			expected:   true,
		},
		{
			name:       "invalid stanza name",
			stanzaName: "0-stanza",
			expected:   false,
		},
		{
			name:       "too long stanza name",
			stanzaName: "abcdefghjkabcdefghjkabcdefghjkabcdefghjkabcdefghjkabcdefghjkabcdefghjk",
			expected:   false,
		},
		{
			name:       "empty stanza name is an error",
			stanzaName: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := isValidStanzaName(tt.stanzaName)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
