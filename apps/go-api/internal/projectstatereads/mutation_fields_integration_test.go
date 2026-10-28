//go:build integration

package projectstatereads

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateNativeUpdatesCheckFinalExternalIdentityPair(t *testing.T) {
	f := newStateMutationFixture(t)
	first, second := f.create(t, "First"), f.create(t, "Second")
	input := f.input
	input.StateID, input.Fields = first, map[string]any{"external_id": "42", "external_source": "left"}
	_, err := f.repository.Update(t.Context(), input)
	require.NoError(t, err)
	input.StateID, input.Fields = second, map[string]any{"external_id": "42", "external_source": "right"}
	_, err = f.repository.Update(t.Context(), input)
	require.NoError(t, err)
	input.Fields = map[string]any{"external_source": "left"}
	_, err = f.repository.Update(t.Context(), input)
	var failure *WriteFailure
	require.ErrorAs(t, err, &failure)
	require.Equal(t, 409, failure.Status)
	require.Equal(t, "right", f.detail(t, second, false)["external_source"])
	input.Fields = map[string]any{"external_source": "left", "external_id": nil}
	_, err = f.repository.Update(t.Context(), input)
	require.NoError(t, err)
	require.Nil(t, f.detail(t, second, false)["external_id"])
	input.Fields = map[string]any{"external_id": ""}
	_, err = f.repository.Update(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, "", f.detail(t, second, false)["external_id"])
}

func TestStateNativeValidationReservesSystemName(t *testing.T) {
	f := newStateMutationFixture(t)
	for _, name := range []string{"Triage", "triage", "  TRIAGE  "} {
		input := f.input
		input.Fields = map[string]any{"name": name, "color": "#abc"}
		_, err := f.repository.Create(t.Context(), input)
		var failure *WriteFailure
		require.ErrorAs(t, err, &failure)
		require.Equal(t, 400, failure.Status)
	}
}

func TestStateDefaultChangesAdvanceAllAffectedRevisions(t *testing.T) {
	f := newStateMutationFixture(t)
	first, second := f.create(t, "First"), f.create(t, "Second")
	firstBefore, secondBefore := f.detail(t, first, true)["revision"], f.detail(t, second, true)["revision"]
	cmd := f.command("update", first)
	cmd.Changes.Default = stateValue(true)
	_, err := f.repository.MutateState(t.Context(), cmd)
	require.NoError(t, err)
	firstAfter, secondAfter := f.detail(t, first, true), f.detail(t, second, true)
	require.NotEqual(t, firstBefore, firstAfter["revision"])
	require.NotEqual(t, secondBefore, secondAfter["revision"])
	require.Equal(t, true, firstAfter["default"])
	require.Equal(t, false, secondAfter["default"])
	native := f.input
	native.StateID = second
	require.NoError(t, f.repository.MarkDefault(t.Context(), native))
	require.NotEqual(t, firstAfter["revision"], f.detail(t, first, true)["revision"])
	require.NotEqual(t, secondAfter["revision"], f.detail(t, second, true)["revision"])
	require.Equal(t, false, f.detail(t, first, false)["default"])
	require.Equal(t, true, f.detail(t, second, false)["default"])
	cmd = f.command("delete", second)
	_, err = f.repository.MutateState(t.Context(), cmd)
	var failure *WriteFailure
	require.ErrorAs(t, err, &failure)
	require.Equal(t, 400, failure.Status)
}

func TestStateDefaultAndPlacementIgnoreForeignOwnedRows(t *testing.T) {
	f := newStateMutationFixture(t)
	foreign := f.create(t, "Malformed foreign row")
	_, err := f.db.Exec(t.Context(), `INSERT INTO workspaces SELECT (jsonb_populate_record(NULL::workspaces,to_jsonb(w)||jsonb_build_object('id','00000000-0000-4000-8000-000000000099','slug','foreign'))).* FROM workspaces w;
 UPDATE states SET workspace_id='00000000-0000-4000-8000-000000000099',sequence=9000000,"default"=true`)
	require.NoError(t, err)
	cmd := f.command("create", "")
	cmd.Changes = StateChanges{Name: stateValue("Owned"), Color: stateValue("#123456"), Default: stateValue(true)}
	result, err := f.repository.MutateState(t.Context(), cmd)
	require.NoError(t, err)
	var sequence float64
	var retained bool
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT sequence FROM states WHERE id=$1`, result.StateID).Scan(&sequence))
	require.Equal(t, float64(65535), sequence)
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT "default" FROM states WHERE id=$1`, foreign).Scan(&retained))
	require.True(t, retained)
	cmd = f.command("update", result.StateID)
	cmd.Changes.Default = stateValue(true)
	_, err = f.repository.MutateState(t.Context(), cmd)
	require.NoError(t, err)
	input := f.input
	input.StateID = result.StateID
	require.NoError(t, f.repository.MarkDefault(t.Context(), input))
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT "default" FROM states WHERE id=$1`, foreign).Scan(&retained))
	require.True(t, retained)
}
