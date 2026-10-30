//go:build integration

package cyclereads

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestExtremeCycleCursorsExecuteWithoutOverflowingPostgresBounds(t *testing.T) {
	db, repo := cycleIssueQueryFixture(t)
	_, err := db.Exec(t.Context(), `ALTER TABLE workspace_members ADD COLUMN role integer;
 INSERT INTO project_members(project_id,workspace_id,member_id,role,is_active)
 VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000001',20,true)`)
	require.NoError(t, err)
	actor := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	project := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	cycle := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	for _, tc := range []struct {
		name, cursor, group string
		count               int
	}{
		{"infinite ungrouped first page", "1.e309:0:0", "", 2},
		{"huge grouped bound", "10000000000000000000.0:1:0", "priority", 0},
		{"infinite grouped bound", "1.e309:0:0", "priority", 0},
		{"last representable offset", "2:" + strconv.Itoa(math.MaxInt/2) + ":0", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, readErr := repo.Issues(t.Context(), actor, "demo", project, cycle, IssueListOptions{
				GroupBy: tc.group, PerPage: 2, PerPageSet: true, Cursor: tc.cursor, CursorSet: true,
			})
			require.NoError(t, readErr)
			var page struct {
				Count int  `json:"count"`
				Next  bool `json:"next_page_results"`
				Total int  `json:"total_results"`
			}
			require.NoError(t, json.Unmarshal(data, &page))
			require.Equal(t, tc.count, page.Count)
			require.False(t, page.Next)
			require.Equal(t, 2, page.Total)
		})
	}
}
