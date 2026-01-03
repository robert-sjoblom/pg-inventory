package types

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackupInfoJsonToLocalType(t *testing.T) {
	data, err := os.ReadFile("../testdata/pgbackrest.json")
	if err != nil {
		t.Fatalf("failed to read pgbackrest.json: %v", err)
	}

	var actual []PgbackrestInfo
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatalf("failed to unmarshal json: %v", err)
	}

	expected := []PgbackrestInfo{
		{
			Name: "main",
			Backup: []Backup{
				{
					Label: "20260103-233937F",
					Type:  "full",
					Error: false,
					Database: PgbackrestDatabase{
						Id:      1,
						RepoKey: 1,
					},
					Info: BackupSizeInfo{
						Delta: 23246324,
						Size:  23246324,
						Repository: PgbackrestRepo{
							Delta: 3066593,
							Size:  3066593,
						},
					},
					Timestamp: BackupTimestamp{
						Start: 1767483577,
						Stop:  1767483580,
					},
				},
				{
					Label: "20260103-233937F_20260103-234057I",
					Type:  "incr",
					Error: false,
					Database: PgbackrestDatabase{
						Id:      1,
						RepoKey: 1,
					},
					Info: BackupSizeInfo{
						Delta: 1376514,
						Size:  23344628,
						Repository: PgbackrestRepo{
							Delta: 212178,
							Size:  3078147,
						},
					},
					Timestamp: BackupTimestamp{
						Start: 1767483657,
						Stop:  1767483664,
					},
				},
			},
			Db: []PgbackrestDb{
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
