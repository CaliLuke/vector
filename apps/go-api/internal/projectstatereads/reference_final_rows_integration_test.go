//go:build integration

package projectstatereads

import (
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestStateReferenceGuardChecksFinalIssueAndDraftRows(t *testing.T) {
	for _, table := range []string{"issues", "draft_issues"} {
		t.Run(table, func(t *testing.T) {
			f := newStateMutationFixture(t)
			live, deleted := f.create(t, "Live"), f.create(t, "Deleted")
			input := f.input
			input.StateID = deleted
			require.NoError(t, f.repository.Delete(t.Context(), input))
			for _, change := range []struct {
				name, query string
				args        []any
			}{
				{"corrected selection", `UPDATE ` + table + ` SET state_id=$2 WHERE id=$1`, []any{live}},
				{"deleted row", `DELETE FROM ` + table + ` WHERE id=$1`, nil},
				{"soft-deleted row", `UPDATE ` + table + ` SET deleted_at=now() WHERE id=$1`, nil},
			} {
				t.Run(change.name, func(t *testing.T) {
					tx, err := f.db.Begin(t.Context())
					require.NoError(t, err)
					defer func() { testsupport.Rollback(t, tx, t.Context()) }()
					id := uuid.New()
					_, err = tx.Exec(t.Context(), stateReferenceInsert(table), id, testsupport.LabelWorkspace, testsupport.LabelProject, deleted)
					require.NoError(t, err)
					args := append([]any{id}, change.args...)
					_, err = tx.Exec(t.Context(), change.query, args...)
					require.NoError(t, err)
					require.NoError(t, tx.Commit(t.Context()))
				})
			}
			// A later restoration must validate its current state again.
			_, err := f.db.Exec(t.Context(), `UPDATE `+table+` SET deleted_at=NULL WHERE deleted_at IS NOT NULL`)
			var failure *pgconn.PgError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, "item_state_must_be_live", failure.ConstraintName)
		})
	}
}

func TestStateReferenceGuardRetainsDeferredParentCreationOrder(t *testing.T) {
	f := newStateMutationFixture(t)
	tx, err := f.db.Begin(t.Context())
	require.NoError(t, err)
	defer func() { testsupport.Rollback(t, tx, t.Context()) }()
	state := uuid.New()
	_, err = tx.Exec(t.Context(), stateReferenceInsert("issues"), uuid.New(), testsupport.LabelWorkspace, testsupport.LabelProject, state)
	require.NoError(t, err)
	_, err = tx.Exec(t.Context(), stateReferenceInsert("draft_issues"), uuid.New(), testsupport.LabelWorkspace, testsupport.LabelProject, state)
	require.NoError(t, err)
	_, err = tx.Exec(t.Context(), `INSERT INTO states(id,workspace_id,project_id,name,color,slug,description,sequence,"group","default",is_triage,created_at,updated_at)
 VALUES($1,$2,$3,'Later state','#abc','later','',1,'unstarted',false,false,now(),now())`, state, testsupport.LabelWorkspace, testsupport.LabelProject)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(t.Context()))
}
