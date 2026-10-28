//go:build integration

package projectstatereads

import (
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestStateDeleteRetainsLiveDraft(t *testing.T) {
	f := newStateMutationFixture(t)
	state := f.create(t, "Draft state")
	_, err := f.db.Exec(t.Context(), `INSERT INTO draft_issues(id,workspace_id,project_id,state_id,name,description_json,description_html,priority,sort_order,created_at,updated_at)
 VALUES($1,$2,$3,$4,'Draft','{}','','none',0,now(),now())`, uuid.New(), testsupport.LabelWorkspace, testsupport.LabelProject, state)
	require.NoError(t, err)
	input := f.input
	input.StateID = state
	var failure *WriteFailure
	require.ErrorAs(t, f.repository.Delete(t.Context(), input), &failure)
	require.Equal(t, 400, failure.Status)
	var live bool
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT deleted_at IS NULL FROM states WHERE id=$1`, state).Scan(&live))
	require.True(t, live)
}

func TestStateDeleteRecordsDurableEffect(t *testing.T) {
	f := newStateMutationFixture(t)
	state := f.create(t, "Empty state")
	input := f.input
	input.StateID = state
	require.NoError(t, f.repository.Delete(t.Context(), input))
	var effects int
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects
 WHERE kind='soft_delete' AND payload=jsonb_build_object('app','db','model','state','id',$1::text)`, state).Scan(&effects))
	require.Equal(t, 1, effects)
}
