//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func planningDetailRevision(t *testing.T, call catalogueCall, kind, id string) string {
	t.Helper()
	detail := milestoneObject(t, call(kind, map[string]any{"action": "retrieve", kind + "_id": id, "include_revision": true}))
	revision, ok := detail["revision"].(string)
	require.True(t, ok, "revision tokens must be strings")
	require.NotEmpty(t, revision)
	return revision
}

func TestMCPPlanningNativeRelationWritersAdvanceRevisions(t *testing.T) {
	runPlanningMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall, kind string) {
		t.Run("work items", func(t *testing.T) {
			assertPlanningRelationRevisions(t, f, call, kind, kind+"_issues", "issue_id", f.item(t))
		})
		if kind == "module" {
			t.Run("members", func(t *testing.T) {
				assertPlanningRelationRevisions(t, f, call, kind, "module_members", "member_id", testsupport.LabelActor)
			})
		}
	})
}

func assertPlanningRelationRevisions(t *testing.T, f milestoneFixture, call catalogueCall, kind, table, referenceColumn, reference string) {
	t.Helper()
	firstInput, first := createPlanningReceipt(t, call, kind)
	secondInput, second := createPlanningReceipt(t, call, kind)
	firstID, secondID, relation := first[kind+"_id"].(string), second[kind+"_id"].(string), uuid.NewString()
	tableSQL, parentSQL, referenceSQL := pgx.Identifier{table}.Sanitize(), pgx.Identifier{kind + "_id"}.Sanitize(), pgx.Identifier{referenceColumn}.Sanitize()
	_, err := f.pool.Exec(t.Context(), "INSERT INTO "+tableSQL+" (id,"+parentSQL+","+referenceSQL+",workspace_id,project_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,now(),now())", relation, firstID, reference, testsupport.LabelWorkspace, testsupport.LabelProject)
	require.NoError(t, err)
	inserted := planningDetailRevision(t, call, kind, firstID)
	require.NotEqual(t, first["revision"], inserted)
	require.Equal(t, second["revision"], planningDetailRevision(t, call, kind, secondID))

	_, err = f.pool.Exec(t.Context(), "UPDATE "+tableSQL+" SET "+parentSQL+"=$2 WHERE id=$1", relation, secondID)
	require.NoError(t, err)
	movedFirst, movedSecond := planningDetailRevision(t, call, kind, firstID), planningDetailRevision(t, call, kind, secondID)
	require.NotEqual(t, inserted, movedFirst, "moving a relation must invalidate its former parent")
	require.NotEqual(t, second["revision"], movedSecond, "moving a relation must invalidate its new parent")
	_, err = f.pool.Exec(t.Context(), "DELETE FROM "+tableSQL+" WHERE id=$1", relation)
	require.NoError(t, err)
	require.NotEqual(t, movedSecond, planningDetailRevision(t, call, kind, secondID))
	require.Equal(t, movedFirst, planningDetailRevision(t, call, kind, firstID))

	before := planningMutationSnapshot(t, f)
	for _, created := range []map[string]any{first, second} {
		milestoneFailure(t, call(kind, map[string]any{"action": "update", kind + "_id": created[kind+"_id"], "name": "stale edit", "expected_revision": created["revision"], "request_id": uuid.NewString()}), "revision_conflict")
	}
	for i, input := range []map[string]any{firstInput, secondInput} {
		original := []map[string]any{first, second}[i]
		replay := milestoneObject(t, call(kind, input))
		require.Equal(t, true, replay["replayed"])
		require.Equal(t, original["result"], replay["result"])
	}
	require.Equal(t, before, planningMutationSnapshot(t, f))
}
