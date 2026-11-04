//go:build integration

package mcpactions_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/initiatives"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func initiativeMutationCase(t *testing.T, action string) (milestoneFixture, *initiatives.Repository, initiatives.Command) {
	t.Helper()
	f := initiativeFixture(t)
	repository := initiatives.NewRepository(f.pool)
	cmd := initiatives.Command{Query: initiatives.Query{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", Action: "create"}, Fields: map[string]any{"name": "Original initiative"}}
	created, err := repository.Mutate(t.Context(), cmd)
	require.NoError(t, err)
	if action != "create" {
		cmd.ID = uuid.MustParse(created.ID)
	}
	cmd.Action = action
	cmd.Fields = nil
	switch action {
	case "create", "update":
		cmd.Fields = map[string]any{"name": "Accepted initiative"}
	case "add_projects", "remove_projects":
		cmd.Add = []uuid.UUID{uuid.MustParse(testsupport.LabelProject)}
		if action == "remove_projects" {
			seed := cmd
			seed.Action = "add_projects"
			_, err = repository.Mutate(t.Context(), seed)
			require.NoError(t, err)
		}
	case "manage_workitems":
		cmd.Add = []uuid.UUID{uuid.MustParse(f.item(t))}
	}
	var revision int64
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM initiative_revisions WHERE workspace_id=$1`, cmd.WorkspaceID).Scan(&revision))
	expected := fmt.Sprintf("initiative:%s:%d", cmd.WorkspaceID, revision)
	cmd.ExpectedRevision = &expected
	key := uuid.New()
	cmd.RequestID = &key
	return f, repository, cmd
}

func TestInitiativeFailedWritesPreserveAtomicRetry(t *testing.T) {
	for _, action := range initiativeMutationActions {
		t.Run(action, func(t *testing.T) {
			f, repository, cmd := initiativeMutationCase(t, action)
			assertCatalogueAtomicRetry(t, f, "initiative_events", initiativeSnapshot, func(ctx context.Context) (projectmutations.Result, error) {
				return repository.Mutate(ctx, cmd)
			})
		})
	}
}

func TestInitiativeConcurrentWritersRetainReceiptAndRevisionOwnership(t *testing.T) {
	// Row, project-link and item-link writers each have their own revision triggers.
	for _, action := range []string{"update", "add_projects", "manage_workitems"} {
		for _, conflict := range []string{"replay", "input", "revision"} {
			t.Run(action+"/"+conflict, func(t *testing.T) {
				f, repository, first := initiativeMutationCase(t, action)
				second := first
				var expected error
				switch conflict {
				case "input":
					second.ExpectedRevision = nil
					expected = mutationreceipts.ErrConflict
				case "revision":
					key := uuid.New()
					second.RequestID = &key
					expected = projectmutations.ErrRevision
				}
				calls := [2]func(context.Context) (projectmutations.Result, error){
					func(ctx context.Context) (projectmutations.Result, error) { return repository.Mutate(ctx, first) },
					func(ctx context.Context) (projectmutations.Result, error) { return repository.Mutate(ctx, second) },
				}
				outcomes := competingScopeWrites(t, f, schemaMutationLock, first.WorkspaceID, calls)
				if outcomes[0].err != nil {
					outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
				}
				require.NoError(t, outcomes[0].err)
				require.ErrorIs(t, outcomes[1].err, expected)
				if expected == nil {
					assertConcurrentReceiptReplay(t, outcomes[0].value, outcomes[1].value)
				}
				var events, receipts int
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM initiative_events),(SELECT count(*) FROM mutation_receipts WHERE tool='initiative')`).Scan(&events, &receipts))
				require.Equal(t, 2, events)
				require.Equal(t, 1, receipts)
			})
		}
	}
}
