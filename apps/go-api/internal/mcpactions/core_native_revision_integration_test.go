//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemrelations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestCoreNativeEditsInvalidateReviewedWrites(t *testing.T) {
	for _, tc := range []struct{ tool, table, key string }{
		{"workitem", "issues", "id"}, {"workitem_comment", "issue_comments", "id"}, {"workitem_link", "issue_links", "id"}, {"intake", "intake_issues", "issue_id"},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			f := coreFixture(t)
			cmd, mutate := coreNativeCommand(t, f, coreMutationCase{tc.tool, "update"})
			tag, err := f.pool.Exec(t.Context(), `UPDATE `+tc.table+` SET updated_at=updated_at+interval '1 second' WHERE `+tc.key+`=$1`, cmd.ResourceID)
			require.NoError(t, err)
			require.EqualValues(t, 1, tag.RowsAffected())
			before := coreSnapshot(t, f)
			_, err = mutate(t.Context(), cmd)
			require.ErrorIs(t, err, projectmutations.ErrRevision)
			require.Equal(t, before, coreSnapshot(t, f))
		})
	}
}
func TestCoreCompetingWritesPreserveSingleAcceptedRevision(t *testing.T) {
	for _, tool := range []string{"workitem", "workitem_comment", "workitem_link", "intake"} {
		t.Run(tool, func(t *testing.T) {
			f := coreFixture(t)
			first, mutate := coreNativeCommand(t, f, coreMutationCase{tool, "update"})
			second := first
			request := uuid.New()
			second.RequestID = &request
			outcomes := competingScopeWrites(t, f, schemaMutationLock, first.WorkspaceID, [2]func(context.Context) (projectmutations.Result, error){func(ctx context.Context) (projectmutations.Result, error) { return mutate(ctx, first) }, func(ctx context.Context) (projectmutations.Result, error) { return mutate(ctx, second) }})
			if outcomes[0].err != nil {
				outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
			}
			require.NoError(t, outcomes[0].err)
			require.ErrorIs(t, outcomes[1].err, projectmutations.ErrRevision)
		})
	}
}
func TestRelationNativeWritersAndCompetingRevisions(t *testing.T) {
	f := templateFixture(t)
	cmd := relationNativeCommand(t, f, "update_definition")
	repo := workitemrelations.NewRepository(f.pool)
	// A definition edit competes with another accepted review of the same workspace.
	second := cmd
	request := uuid.New()
	second.RequestID = &request
	outcomes := competingScopeWrites(t, f, schemaMutationLock, cmd.WorkspaceID, [2]func(context.Context) (projectmutations.Result, error){func(ctx context.Context) (projectmutations.Result, error) { return repo.Mutate(ctx, cmd) }, func(ctx context.Context) (projectmutations.Result, error) { return repo.Mutate(ctx, second) }})
	if outcomes[0].err != nil {
		outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
	}
	require.NoError(t, outcomes[0].err)
	require.ErrorIs(t, outcomes[1].err, projectmutations.ErrRevision)
	cmd.RequestID = nil
	first, target := f.item(t), f.item(t)
	record := uuid.New()
	args := pgx.NamedArgs{"record": record, "workspace": cmd.WorkspaceID, "definition": cmd.DefinitionID, "first": first, "target": target, "project": testsupport.LabelProject}
	for _, sql := range []string{
		`INSERT INTO work_item_relation_definitions(id,workspace_id,name,outward,inward) VALUES(@record,@workspace,'Native definition','before','after')`,
		`UPDATE work_item_relation_definitions SET name='Native edit' WHERE id=@record`,
		`DELETE FROM work_item_relation_definitions WHERE id=@record`,
		`INSERT INTO work_item_custom_relations(id,workspace_id,definition_id,issue_id,related_issue_id) VALUES(@record,@workspace,@definition,@first,@target)`,
		`UPDATE work_item_custom_relations SET deleted_at=now() WHERE id=@record`,
		`DELETE FROM work_item_custom_relations WHERE id=@record`,
		`INSERT INTO issue_relations(id,workspace_id,project_id,issue_id,related_issue_id,relation_type,created_at,updated_at) VALUES(@record,@workspace,@project,@first,@target,'blocked_by',now(),now())`,
		`UPDATE issue_relations SET deleted_at=now() WHERE id=@record`,
		`DELETE FROM issue_relations WHERE id=@record`,
	} {
		t.Run(sql, func(t *testing.T) {
			var revision int64
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM work_item_relation_revisions WHERE workspace_id=$1`, cmd.WorkspaceID).Scan(&revision))
			tag, err := f.pool.Exec(t.Context(), sql, args)
			require.NoError(t, err)
			require.EqualValues(t, 1, tag.RowsAffected())
			var after int64
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM work_item_relation_revisions WHERE workspace_id=$1`, cmd.WorkspaceID).Scan(&after))
			require.Greater(t, after, revision)
		})
	}
}
