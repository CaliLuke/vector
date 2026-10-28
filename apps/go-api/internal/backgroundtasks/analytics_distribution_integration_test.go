//go:build integration

package backgroundtasks

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestAnalyticsDistributionPreservesSegmentsAndMissingValues(t *testing.T) {
	_, _, pool := openBackgroundTaskDatabase(t)
	ctx := t.Context()
	_, err := pool.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO states (id,project_id,workspace_id,name,description,color,slug,sequence,"group","default",is_triage,created_at,updated_at)
 VALUES ('00000000-0000-4000-8000-000000000010',$1,$2,'Backlog','','','',1,'backlog',false,false,NOW(),NOW()),
 ('00000000-0000-4000-8000-000000000011',$1,$2,'Done','','','',2,'completed',false,false,NOW(),NOW())`, testsupport.LabelProject, testsupport.LabelWorkspace)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO issues (id,project_id,workspace_id,created_by_id,name,description_json,description_html,priority,state_id,sequence_id,sort_order,is_draft,created_at,updated_at)
 SELECT gen_random_uuid(),$1,$2,$3,'Distribution issue','{}','',seed.priority,seed.state_id::uuid,seed.sequence,0,false,NOW(),NOW()
 FROM (VALUES ('low','00000000-0000-4000-8000-000000000011',1),
 ('high','00000000-0000-4000-8000-000000000010',2),
 ('high','00000000-0000-4000-8000-000000000011',3),('high',NULL,4)) seed(priority,state_id,sequence)`, testsupport.LabelProject, testsupport.LabelWorkspace, testsupport.LabelActor)
	require.NoError(t, err)
	store := &analyticsExportStore{pool: pool}
	for _, test := range []struct {
		name, input string
		want        [][]string
	}{
		{"segmented count", `{"x_axis":"priority","y_axis":"issue_count","segment":"state_id"}`, [][]string{{"Priority", "Issue Count", "Backlog", "Done"}, {"low", "1", "0", "1"}, {"high", "3", "1", "1"}}},
		{"missing dimension", `{"x_axis":"state_id","y_axis":"issue_count"}`, [][]string{{"X-Axis", "Issue Count"}, {"Backlog", "1"}, {"Done", "2"}}},
		{"null estimates", `{"x_axis":"priority","y_axis":"estimate","segment":"state_id"}`, [][]string{{"Priority", "Estimate", "Backlog", "Done"}, {"low", "0", "0", "0"}, {"high", "0", "0", "0"}}},
		{"array filter", `{"x_axis":"priority","y_axis":"issue_count","priority":["high",42]}`, [][]string{{"Priority", "Issue Count"}, {"high", "3"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows, queryErr := store.rows(t.Context(), "milestones", json.RawMessage(test.input))
			require.NoError(t, queryErr)
			require.Equal(t, test.want, rows)
		})
	}
}
