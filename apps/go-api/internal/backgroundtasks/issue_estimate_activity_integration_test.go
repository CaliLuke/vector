//go:build integration

package backgroundtasks

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEstimateActivityRetainsLabelsAndScopesReferences(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	t.Setenv("REDIS_URL", "")
	store := f.activityStore(t)
	item := f.activityItem(t).String()
	actor := testsupport.LabelActor
	points := map[string]string{}
	for _, kind := range []string{"categories", "points", "time"} {
		estimate, point := uuid.NewString(), uuid.NewString()
		f.exec(t, `INSERT INTO estimates(id,workspace_id,project_id,name,description,type,last_used,created_at,updated_at) VALUES($1,$2,$3,$5,'',$4,true,now(),now())`, estimate, testsupport.LabelWorkspace, testsupport.LabelProject, kind, kind+" estimate")
		f.exec(t, `INSERT INTO estimate_points(id,workspace_id,project_id,estimate_id,key,value,description,created_at,updated_at) VALUES($1,$2,$3,$4,0,$5,'',now(),now())`, point, testsupport.LabelWorkspace, testsupport.LabelProject, estimate, kind+" label")
		points[kind] = point
	}
	foreignProject, foreignEstimate, foreignPoint := uuid.NewString(), uuid.NewString(), uuid.NewString()
	f.exec(t, `INSERT INTO projects SELECT (jsonb_populate_record(NULL::projects,to_jsonb(p)||jsonb_build_object('id',$1::uuid,'identifier','OTHER','name','Other project'))).* FROM projects p WHERE id=$2`, foreignProject, testsupport.LabelProject)
	f.exec(t, `INSERT INTO estimates(id,workspace_id,project_id,name,description,type,last_used,created_at,updated_at) VALUES($1,$2,$3,'Private','','points',true,now(),now())`, foreignEstimate, testsupport.LabelWorkspace, foreignProject)
	f.exec(t, `INSERT INTO estimate_points(id,workspace_id,project_id,estimate_id,key,value,description,created_at,updated_at) VALUES($1,$2,$3,$4,0,'private label','',now(),now())`, foreignPoint, testsupport.LabelWorkspace, foreignProject, foreignEstimate)
	for _, tc := range []struct {
		name, old, next, kind, verb string
		oldVisible, newVisible      bool
	}{
		{"assignment", "", points["categories"], "categories", "updated", false, true},
		{"change", points["categories"], points["time"], "time", "updated", true, true},
		{"removal", points["points"], "", "points", "removed", true, false},
		{"foreign old hidden", foreignPoint, points["points"], "points", "updated", false, true},
		{"foreign new hidden", points["time"], foreignPoint, "time", "removed", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requested, current := map[string]any{"estimate_point": nil}, map[string]any{"estimate_point": nil}
			if tc.next != "" {
				requested["estimate_point"] = tc.next
			}
			if tc.old != "" {
				current["estimate_point"] = tc.old
			}
			raw, err := json.Marshal(requested)
			require.NoError(t, err)
			old, err := json.Marshal(current)
			require.NoError(t, err)
			event := uuid.NewString()
			input := IssueActivityInput{Type: "issue.activity.updated", ProjectID: testsupport.LabelProject, IssueID: &item, ActorID: &actor, RequestedData: raw, CurrentInstance: old}
			ids, err := store.RecordIssueActivity(f.ctx, event, input)
			require.NoError(t, err)
			require.Len(t, ids, 1)
			var field, verb string
			var oldID, newID, oldValue, newValue *string
			require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT field,verb,old_identifier::text,new_identifier::text,old_value,new_value FROM issue_activities WHERE id=$1`, ids[0]).Scan(&field, &verb, &oldID, &newID, &oldValue, &newValue))
			require.Equal(t, "estimate_"+tc.kind, field)
			require.Equal(t, tc.verb, verb)
			if tc.oldVisible {
				require.NotNil(t, oldID)
				require.Equal(t, tc.old, *oldID)
				require.NotNil(t, oldValue)
				for kind, id := range points {
					if id == tc.old {
						require.Equal(t, kind+" label", *oldValue)
					}
				}
			} else {
				require.Nil(t, oldID)
				require.Nil(t, oldValue)
			}
			if tc.newVisible {
				require.NotNil(t, newID)
				require.Equal(t, tc.next, *newID)
				require.Equal(t, tc.kind+" label", *newValue)
			} else {
				require.Nil(t, newID)
				require.Nil(t, newValue)
			}
			// Retained tombstones must still supply the original activity labels, and
			// replay must keep the same activity ID instead of adding another record.
			f.exec(t, `UPDATE estimate_points SET deleted_at=now() WHERE id=$1`, points[tc.kind])
			replay, err := store.RecordIssueActivity(f.ctx, event, input)
			require.NoError(t, err)
			require.Equal(t, ids, replay)
		})
	}
	t.Run("unchanged and foreign references do not create history", func(t *testing.T) {
		for _, pair := range [][2]string{{`{}`, `{}`}, {`{"estimate_point":null}`, `{}`}, {`{"estimate_point":"` + points["points"] + `"}`, `{"estimate_point":"` + points["points"] + `"}`}, {`{"estimate_point":"` + foreignPoint + `"}`, `{}`}} {
			ids, err := store.RecordIssueActivity(f.ctx, uuid.NewString(), IssueActivityInput{Type: "issue.activity.updated", ProjectID: testsupport.LabelProject, IssueID: &item, ActorID: &actor, RequestedData: json.RawMessage(pair[0]), CurrentInstance: json.RawMessage(pair[1])})
			require.NoError(t, err)
			require.Empty(t, ids)
		}
		_, err := store.RecordIssueActivity(f.ctx, uuid.NewString(), IssueActivityInput{Type: "issue.activity.updated", ProjectID: testsupport.LabelProject, IssueID: &item, ActorID: &actor, RequestedData: json.RawMessage(`{"estimate_point":"invalid"}`), CurrentInstance: json.RawMessage(`{}`)})
		require.ErrorIs(t, err, ErrPermanentIssueActivity)
	})
	t.Run("new history preserves existing event row identifiers", func(t *testing.T) {
		event := uuid.NewString()
		input := IssueActivityInput{Type: "issue.activity.updated", ProjectID: testsupport.LabelProject, IssueID: &item, ActorID: &actor, RequestedData: json.RawMessage(`{"archived_at":"2024-01-01T00:00:00Z"}`), CurrentInstance: json.RawMessage(`{"archived_at":null}`)}
		// Seed the archive row an older worker produced when it ignored the
		// estimate field. Replaying the mixed event must retain that row's ID.
		prior, err := store.RecordIssueActivity(f.ctx, event, input)
		require.NoError(t, err)
		require.Len(t, prior, 1)
		raw, err := json.Marshal(map[string]any{"archived_at": "2024-01-01T00:00:00Z", "estimate_point": points["points"]})
		require.NoError(t, err)
		input.RequestedData = raw
		replay, err := store.RecordIssueActivity(f.ctx, event, input)
		require.NoError(t, err)
		require.Len(t, replay, 2)
		require.Equal(t, prior[0], replay[0])
		var fields []string
		require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT array_agg(field ORDER BY field) FROM issue_activities WHERE id=ANY($1::uuid[])`, replay).Scan(&fields))
		require.Equal(t, []string{"archived_at", "estimate_points"}, fields)
		again, err := store.RecordIssueActivity(f.ctx, event, input)
		require.NoError(t, err)
		require.Equal(t, replay, again)
	})

}
