//go:build integration

package cyclereads

import (
	"encoding/json"
	"fmt"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestCycleGroupingReturnsScopedBucketsAndPages(t *testing.T) {
	connection, repository := cycleIssueQueryFixture(t)
	ctx := t.Context()
	actor := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	project := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	cycle := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	const authorA = "00000000-0000-4000-8000-000000000011"
	const authorB = "00000000-0000-4000-8000-000000000012"
	_, err := connection.Exec(ctx, `
 ALTER TABLE workspace_members ADD COLUMN role integer;
 INSERT INTO project_members(project_id,workspace_id,member_id,role,is_active)
 VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000001',20,true);
 UPDATE issues SET sequence_id=5,created_by_id='00000000-0000-4000-8000-000000000011',start_date='2026-10-01',target_date='2026-10-10'
 WHERE id='00000000-0000-4000-8000-000000000005';
 UPDATE issues SET sequence_id=6,created_by_id='00000000-0000-4000-8000-000000000012',start_date='2026-10-02'
 WHERE id='00000000-0000-4000-8000-000000000006';
 INSERT INTO issues(id,name,workspace_id,project_id,priority,sequence_id,created_by_id,start_date,target_date,created_at,updated_at)
 SELECT '00000000-0000-4000-8000-000000000007','Second urgent',workspace_id,project_id,priority,7,created_by_id,start_date,target_date,created_at,updated_at
 FROM issues WHERE id='00000000-0000-4000-8000-000000000005';
 INSERT INTO issues(id,name,workspace_id,project_id,priority,sequence_id,created_at,updated_at)
 VALUES('00000000-0000-4000-8000-000000000008','Unassigned','00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000002','urgent',8,now(),now());
 INSERT INTO workspaces(id,slug) VALUES('00000000-0000-4000-8000-000000000020','other');
 INSERT INTO projects(id,workspace_id) VALUES('00000000-0000-4000-8000-000000000021','00000000-0000-4000-8000-000000000020');
 INSERT INTO issues(id,name,workspace_id,project_id,priority,sequence_id,created_by_id,start_date,target_date,created_at,updated_at)
 VALUES('00000000-0000-4000-8000-000000000009','Hidden foreign','00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000021','urgent',9,'00000000-0000-4000-8000-000000000099','2030-01-01','2030-02-01',now(),now());
 INSERT INTO issues(id,name,workspace_id,project_id,priority,sequence_id,created_by_id,start_date,target_date,created_at,updated_at,deleted_at)
 SELECT '00000000-0000-4000-8000-000000000010','Deleted',workspace_id,project_id,priority,10,'00000000-0000-4000-8000-000000000098','2031-01-01','2031-02-01',now(),now(),now()
 FROM issues WHERE id='00000000-0000-4000-8000-000000000005';
 INSERT INTO projects(id,workspace_id) VALUES('00000000-0000-4000-8000-000000000022','00000000-0000-4000-8000-000000000004');
 INSERT INTO issues(id,name,workspace_id,project_id,priority,sequence_id,created_by_id,start_date,target_date,created_at,updated_at)
 VALUES('00000000-0000-4000-8000-000000000013','Hidden project','00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000022','urgent',13,'00000000-0000-4000-8000-000000000097','2032-01-01','2032-02-01',now(),now());
 INSERT INTO cycle_issues(issue_id,cycle_id,created_at)
 SELECT id,'00000000-0000-4000-8000-000000000003',now() FROM issues WHERE sequence_id>=7;
 `)
	require.NoError(t, err)
	for _, field := range []struct {
		name   string
		groups map[string][]int
	}{
		{"created_by", map[string][]int{authorA: {5, 7}, authorB: {6}, "None": {8}}},
		{"start_date", map[string][]int{"2026-10-01": {5, 7}, "2026-10-02": {6}, "None": {8}}},
		{"target_date", map[string][]int{"2026-10-10": {5, 7}, "None": {6, 8}}},
		{"state_id", map[string][]int{"None": {5, 6, 7, 8}}},
	} {
		for _, mode := range []string{"primary", "primary-with-priority", "secondary-to-priority"} {
			t.Run(field.name+"/"+mode, func(t *testing.T) {
				group, subgroup := field.name, ""
				if mode == "primary-with-priority" {
					subgroup = "priority"
				}
				if mode == "secondary-to-priority" {
					group, subgroup = "priority", field.name
				}
				expected := dynamicGroupingBuckets(field.groups, mode)
				pages := 0
				for _, ids := range expected {
					pages = max(pages, len(ids))
				}
				for page := 0; page < pages; page++ {
					t.Run(fmt.Sprintf("page-%d", page), func(t *testing.T) {
						data, queryErr := repository.Issues(ctx, actor, "demo", project, cycle, IssueListOptions{
							GroupBy: group, SubGroupBy: subgroup, OrderBy: "sequence_id", PerPage: 1, PerPageSet: true, Cursor: fmt.Sprintf("1:%d:0", page), CursorSet: true,
						})
						require.NoError(t, queryErr)
						checkDynamicGroupingPage(t, data, subgroup != "", expected, page, pages)
					})
				}
			})
		}
	}
}

func dynamicGroupingBuckets(groups map[string][]int, mode string) map[string][]int {
	buckets := map[string][]int{}
	for key, ids := range groups {
		if mode == "primary" {
			buckets[key] = ids
			continue
		}
		for _, id := range ids {
			priority := "urgent"
			if id == 6 {
				priority = "low"
			}
			bucket := key + "/" + priority
			if mode == "secondary-to-priority" {
				bucket = priority + "/" + key
			}
			buckets[bucket] = append(buckets[bucket], id)
		}
	}
	return buckets
}

type dynamicGroupingBucket struct {
	Results json.RawMessage `json:"results"`
	Total   int             `json:"total_results"`
}

func checkDynamicGroupingPage(t *testing.T, data []byte, nested bool, expected map[string][]int, page, pages int) {
	t.Helper()
	var envelope struct {
		Results map[string]dynamicGroupingBucket `json:"results"`
		Total   int                              `json:"total_count"`
		Count   int                              `json:"count"`
		Pages   int                              `json:"total_pages"`
		Next    bool                             `json:"next_page_results"`
	}
	require.NoError(t, json.Unmarshal(data, &envelope))
	require.Equal(t, 4, envelope.Total)
	require.Equal(t, pages, envelope.Pages, "nested pagination must use the largest bucket")
	require.Equal(t, page+1 < pages, envelope.Next)
	actual := map[string]dynamicGroupingBucket{}
	for key, bucket := range envelope.Results {
		if !nested {
			actual[key] = bucket
			continue
		}
		var subgroups map[string]dynamicGroupingBucket
		require.NoError(t, json.Unmarshal(bucket.Results, &subgroups))
		for subkey, subgroup := range subgroups {
			actual[key+"/"+subkey] = subgroup
		}
	}
	require.Len(t, actual, len(expected), "unexpected group key may expose an out-of-scope row")
	selected := 0
	for key, ids := range expected {
		bucket, exists := actual[key]
		require.True(t, exists, "missing bucket %s", key)
		require.Equal(t, len(ids), bucket.Total, "bucket %s total", key)
		var rows []struct {
			Sequence int `json:"sequence_id"`
		}
		require.NoError(t, json.Unmarshal(bucket.Results, &rows))
		if page >= len(ids) {
			require.Empty(t, rows)
			continue
		}
		require.Len(t, rows, 1)
		require.Equal(t, ids[page], rows[0].Sequence, "bucket %s page %d", key, page)
		selected++
	}
	require.Equal(t, selected, envelope.Count)
}
