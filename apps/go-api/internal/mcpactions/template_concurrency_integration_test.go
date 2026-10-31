//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/templates"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func templateMutationCase(t *testing.T, action string) (milestoneFixture, *templates.Repository, templates.Command) {
	t.Helper()
	f := templateFixture(t)
	repository := templates.NewRepository(f.pool)
	name := "Original template"
	command := templates.Command{Query: templates.Query{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", ProjectID: uuid.MustParse(testsupport.LabelProject), Kind: "workitem"}, Action: "create", Name: &name, Data: map[string]any{"nested": map[string]any{"retained": true}}}
	created, err := repository.Mutate(t.Context(), command)
	require.NoError(t, err)
	command.Action = action
	command.ExpectedRevision = &created.Revision
	if action != "create" {
		command.ID = uuid.MustParse(created.ID)
	}
	if action == "delete" {
		command.Name, command.Data = nil, nil
	} else {
		name = "Accepted template"
	}
	key := uuid.New()
	command.RequestID = &key
	return f, repository, command
}

func TestTemplateCompetingWritesPreserveReceiptAndRevisionOwnership(t *testing.T) {
	for _, action := range []string{"create", "update", "delete"} {
		for _, conflict := range []string{"replay", "input", "revision"} {
			t.Run(action+"/"+conflict, func(t *testing.T) {
				f, repository, first := templateMutationCase(t, action)
				second := first
				expected := error(nil)
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
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM template_events),(SELECT count(*) FROM mutation_receipts WHERE tool='template')`).Scan(&events, &receipts))
				require.Equal(t, 2, events, "one seed and one accepted mutation")
				require.Equal(t, 1, receipts)
				before := templateStorageSnapshot(t, f)
				replay, err := repository.Mutate(t.Context(), first)
				if err != nil {
					require.ErrorIs(t, err, expected)
					replay, err = repository.Mutate(t.Context(), second)
				}
				require.NoError(t, err)
				require.NotNil(t, replay.Replayed)
				require.True(t, *replay.Replayed)
				require.Equal(t, outcomes[0].value.Record, replay.Record)
				require.Equal(t, before, templateStorageSnapshot(t, f))
			})
		}
	}
}
