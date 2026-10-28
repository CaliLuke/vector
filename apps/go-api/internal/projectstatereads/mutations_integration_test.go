//go:build integration

package projectstatereads

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStateMutationReceiptsRevisionsAndDeletedTargetReplay(t *testing.T) {
	f := newStateMutationFixture(t)
	create := f.command("create", "")
	create.RequestID = stateValue(uuid.New())
	create.Changes = StateChanges{Name: stateValue("  Ready  "), Color: stateValue("#abc"), Sequence: stateValue(0.0)}
	created, err := f.repository.MutateState(t.Context(), create)
	require.NoError(t, err)
	require.False(t, *created.Replayed)
	require.NotNil(t, created.ReplayExpiresAt)
	var record map[string]any
	require.NoError(t, json.Unmarshal(created.Result, &record))
	require.Len(t, record, 18)
	require.Equal(t, "Ready", record["name"])
	require.Equal(t, float64(65535), record["sequence"])
	require.Equal(t, false, record["default"])
	require.Equal(t, "backlog", record["group"])
	require.NotContains(t, f.detail(t, created.StateID, false), "revision")
	require.Equal(t, created.Revision, f.detail(t, created.StateID, true)["revision"])

	create.Changes.Name, create.Changes.Default = stateValue("Ready"), stateValue(false)
	create.Changes.Sequence = stateValue(math.Copysign(0, -1))
	replayed, err := f.repository.MutateState(t.Context(), create)
	require.NoError(t, err)
	require.True(t, *replayed.Replayed)
	require.Equal(t, created.StateID, replayed.StateID)
	require.JSONEq(t, string(created.Result), string(replayed.Result))
	create.Changes.Name = stateValue("Different")
	_, err = f.repository.MutateState(t.Context(), create)
	require.ErrorIs(t, err, mutationreceipts.ErrConflict)

	update := f.command("update", created.StateID)
	update.RequestID, update.ExpectedRevision = stateValue(uuid.New()), &created.Revision
	update.Changes = StateChanges{Name: stateValue("Working"), Sequence: stateValue(0.0), Default: stateValue(false)}
	updated, err := f.repository.MutateState(t.Context(), update)
	require.NoError(t, err)
	require.NotEqual(t, created.Revision, updated.Revision)
	require.NoError(t, json.Unmarshal(updated.Result, &record))
	require.Equal(t, float64(0), record["sequence"])
	replayed, err = f.repository.MutateState(t.Context(), update)
	require.NoError(t, err)
	require.True(t, *replayed.Replayed)
	require.Equal(t, updated.Revision, replayed.Revision)
	update.RequestID = nil
	_, err = f.repository.MutateState(t.Context(), update)
	require.ErrorIs(t, err, ErrRevisionConflict)

	// A native HTTP-style write advances the same revision as the bound command.
	native := f.input
	native.StateID, native.Fields = created.StateID, map[string]any{"color": "#def"}
	_, err = f.repository.Update(t.Context(), native)
	require.NoError(t, err)
	current := f.detail(t, created.StateID, true)["revision"].(string)
	require.NotEqual(t, updated.Revision, current)
	update.ExpectedRevision = &updated.Revision
	_, err = f.repository.MutateState(t.Context(), update)
	require.ErrorIs(t, err, ErrRevisionConflict)

	remove := f.command("delete", created.StateID)
	remove.RequestID, remove.ExpectedRevision = stateValue(uuid.New()), &current
	deleted, err := f.repository.MutateState(t.Context(), remove)
	require.NoError(t, err)
	require.JSONEq(t, "null", string(deleted.Result))
	require.NotEqual(t, current, deleted.Revision)
	replayed, err = f.repository.MutateState(t.Context(), remove)
	require.NoError(t, err)
	require.True(t, *replayed.Replayed)
	require.Equal(t, deleted.Revision, replayed.Revision)
	_, err = f.db.Exec(t.Context(), `UPDATE project_members SET role=5`)
	require.NoError(t, err)
	_, err = f.repository.MutateState(t.Context(), remove)
	require.ErrorIs(t, err, ErrForbidden)
	var states, receipts, effects int
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM states),
 (SELECT count(*) FROM mutation_receipts),(SELECT count(*) FROM work_item_effects)`).Scan(&states, &receipts, &effects))
	require.Equal(t, 1, states)
	require.Equal(t, 3, receipts)
	require.Equal(t, 1, effects)
}

func TestStateMutationInvalidInputsDoNotCreateStateOrReceipt(t *testing.T) {
	f := newStateMutationFixture(t)
	for _, test := range []struct {
		name   string
		change func(*MutationCommand)
	}{
		{"actor", func(c *MutationCommand) { c.ActorID = uuid.Nil }},
		{"workspace", func(c *MutationCommand) { c.WorkspaceID = uuid.Nil }},
		{"slug", func(c *MutationCommand) { c.WorkspaceSlug = "" }},
		{"action", func(c *MutationCommand) { c.Action = "archive" }},
		{"create target", func(c *MutationCommand) { c.StateID = uuid.New() }},
		{"create revision", func(c *MutationCommand) { c.ExpectedRevision = stateValue("old") }},
		{"empty revision", func(c *MutationCommand) { c.ExpectedRevision = stateValue("") }},
		{"missing update target", func(c *MutationCommand) { c.Action = "update" }},
		{"nonfinite sequence", func(c *MutationCommand) { c.Changes.Sequence = stateValue(math.NaN()) }},
		{"infinite sequence", func(c *MutationCommand) { c.Changes.Sequence = stateValue(math.Inf(1)) }},
		{"workspace group required", func(c *MutationCommand) { c.ProjectID = uuid.Nil }},
		{"workspace default", func(c *MutationCommand) {
			c.ProjectID = uuid.Nil
			c.Changes.Group = stateValue("backlog")
			c.Changes.Default = stateValue(false)
		}},
		{"workspace sequence", func(c *MutationCommand) {
			c.ProjectID = uuid.Nil
			c.Changes.Group = stateValue("backlog")
			c.Changes.Sequence = stateValue(0.0)
		}},
		{"delete fields", func(c *MutationCommand) { c.Action = "delete"; c.StateID = uuid.New() }},
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd := f.command("create", "")
			cmd.RequestID = stateValue(uuid.New())
			cmd.Changes = StateChanges{Name: stateValue("Ready"), Color: stateValue("#abc")}
			test.change(&cmd)
			_, err := f.repository.MutateState(t.Context(), cmd)
			require.ErrorIs(t, err, ErrInvalidMutation)
		})
	}
	var count int
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM states)+(SELECT count(*) FROM mutation_receipts)`).Scan(&count))
	require.Zero(t, count)
}

func TestStateWorkspaceMutationsRemainUngoverned(t *testing.T) {
	f := newStateMutationFixture(t)
	for _, action := range []string{"create", "update", "delete"} {
		t.Run(action, func(t *testing.T) {
			cmd := f.command(action, "")
			cmd.ProjectID = uuid.Nil
			cmd.RequestID = stateValue(uuid.New())
			if action == "create" {
				cmd.Changes = StateChanges{Name: stateValue("Ready"), Color: stateValue("#abc"), Group: stateValue("backlog")}
			} else {
				cmd.StateID = uuid.New()
			}
			_, err := f.repository.MutateState(t.Context(), cmd)
			require.ErrorIs(t, err, ErrWorkspaceNotManaged)
		})
	}
	var count int
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM states)+(SELECT count(*) FROM mutation_receipts)`).Scan(&count))
	require.Zero(t, count)
}
