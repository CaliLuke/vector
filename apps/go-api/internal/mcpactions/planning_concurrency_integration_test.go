//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/modulereads"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func planningRepositoryWriter(t *testing.T, f milestoneFixture, kind string) func(context.Context, projectmutations.Command) (projectmutations.Result, error) {
	t.Helper()
	if kind == "cycle" {
		r, err := cyclereads.NewLifecycleRepository(f.pool, rosterPublisher{}, "https://vector.example")
		require.NoError(t, err)
		return r.MutateResource
	}
	r, err := modulereads.NewLifecycleRepository(f.pool, rosterPublisher{}, "https://vector.example")
	require.NoError(t, err)
	return r.MutateResource
}

func TestPlanningCompetingReceiptsAndRevisions(t *testing.T) {
	for _, kind := range []string{"cycle", "module"} {
		for _, scenario := range []string{"identical request", "changed request", "competing revision"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				f := planningAuthorizationWaitFixture(t)
				write := planningRepositoryWriter(t, f, kind)
				key := uuid.New()
				base := projectmutations.Command{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "workspace-one", Action: "create", RequestID: &key, Fields: map[string]any{"name": uuid.NewString(), "description": "initial"}}
				if kind == "cycle" {
					base.Fields["owned_by"] = testsupport.LabelActor
				}
				commands := [2]projectmutations.Command{base, base}
				var expectedError error
				var initialRevision int
				switch scenario {
				case "changed request":
					commands[1].Fields = map[string]any{"name": uuid.NewString(), "description": "different"}
					if kind == "cycle" {
						commands[1].Fields["owned_by"] = testsupport.LabelActor
					}
					expectedError = mutationreceipts.ErrConflict
				case "competing revision":
					base.RequestID = nil
					created, err := write(t.Context(), base)
					require.NoError(t, err)
					require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT revision FROM "+pgx.Identifier{kind + "_edit_revisions"}.Sanitize()+" WHERE "+pgx.Identifier{kind + "_id"}.Sanitize()+"=$1", created.ID).Scan(&initialRevision))
					firstKey, secondKey := uuid.New(), uuid.New()
					base.Action, base.ResourceID, base.ExpectedRevision = "update", uuid.MustParse(created.ID), &created.Revision
					commands = [2]projectmutations.Command{base, base}
					commands[0].RequestID, commands[0].Fields = &firstKey, map[string]any{"name": uuid.NewString()}
					commands[1].RequestID, commands[1].Fields = &secondKey, map[string]any{"description": "competing edit"}
					expectedError = projectmutations.ErrRevision
				}
				calls := [2]func(context.Context) (projectmutations.Result, error){}
				for i, command := range commands {
					calls[i] = func(ctx context.Context) (projectmutations.Result, error) { return write(ctx, command) }
				}
				outcomes := competingScopeWrites(t, f, projectMutationLock, base.ProjectID, calls)
				if outcomes[0].err != nil {
					outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
				}
				require.NoError(t, outcomes[0].err)
				accepted := outcomes[0].value
				require.NotNil(t, accepted.Replayed)
				if expectedError == nil {
					require.NoError(t, outcomes[1].err)
					require.NotNil(t, outcomes[1].value.Replayed)
					require.NotEqual(t, *accepted.Replayed, *outcomes[1].value.Replayed)
					require.Equal(t, accepted.ID, outcomes[1].value.ID)
					require.Equal(t, accepted.Revision, outcomes[1].value.Revision)
					require.JSONEq(t, string(accepted.Record), string(outcomes[1].value.Record))
				} else {
					require.ErrorIs(t, outcomes[1].err, expectedError)
					require.False(t, *accepted.Replayed)
				}
				var saved, committed struct{ Name, Description string }
				require.NoError(t, json.Unmarshal(accepted.Record, &committed))
				require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT name,description FROM "+pgx.Identifier{kind + "s"}.Sanitize()+" WHERE id=$1", accepted.ID).Scan(&saved.Name, &saved.Description))
				require.Equal(t, committed, saved)
				var records, receipts, revision int
				require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+pgx.Identifier{kind + "s"}.Sanitize()+" WHERE project_id=$1", base.ProjectID).Scan(&records))
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE tool=$1 AND request_id IN ($2,$3)`, kind, commands[0].RequestID, commands[1].RequestID).Scan(&receipts))
				require.NoError(t, f.pool.QueryRow(t.Context(), "SELECT revision FROM "+pgx.Identifier{kind + "_edit_revisions"}.Sanitize()+" WHERE "+pgx.Identifier{kind + "_id"}.Sanitize()+"=$1", accepted.ID).Scan(&revision))
				require.Equal(t, 1, records)
				require.Equal(t, 1, receipts)
				require.Equal(t, initialRevision+1, revision)
			})
		}
	}
}
