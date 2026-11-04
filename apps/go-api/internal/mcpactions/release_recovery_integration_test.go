//go:build integration

package mcpactions_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/releases"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func releaseNativeMutation(t *testing.T, tc releaseMutationCase) (milestoneFixture, *releases.Repository, releases.Command) {
	t.Helper()
	f := templateFixture(t)
	repo := releases.NewRepository(f.pool)
	key := "name"
	if tc.tool == "release_tag" {
		key = "version"
	}
	cmd := releases.Command{Query: releases.Query{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", Tool: tc.tool, Action: "create"}, Fields: map[string]any{key: uuid.NewString()}}
	created, err := repo.Mutate(t.Context(), cmd)
	require.NoError(t, err)
	if tc.action != "create" {
		cmd.ID = uuid.MustParse(created.ID)
	}
	cmd.Action = tc.action
	cmd.Fields = nil
	switch tc.action {
	case "create", "update":
		cmd.Fields = map[string]any{key: uuid.NewString()}
	case "update_changelog":
		cmd.Fields = map[string]any{"description_stripped": "Accepted notes"}
	case "manage_workitems":
		cmd.Add = []uuid.UUID{uuid.MustParse(f.item(t))}
	case "attach", "detach":
		parent := cmd
		parent.ID = uuid.Nil
		parent.Tool = "release"
		parent.Action = "create"
		parent.Fields = map[string]any{"name": uuid.NewString()}
		release, err := repo.Mutate(t.Context(), parent)
		require.NoError(t, err)
		cmd.ID = uuid.Nil
		cmd.ReleaseID = uuid.MustParse(release.ID)
		cmd.Add = []uuid.UUID{uuid.MustParse(created.ID)}
		if tc.action == "detach" {
			seed := cmd
			seed.Action = "attach"
			_, err = repo.Mutate(t.Context(), seed)
			require.NoError(t, err)
		}
	}
	var rev int64
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM release_revisions WHERE workspace_id=$1`, cmd.WorkspaceID).Scan(&rev))
	revision := fmt.Sprintf("release:%s:%d", cmd.WorkspaceID, rev)
	cmd.ExpectedRevision = &revision
	request := uuid.New()
	cmd.RequestID = &request
	return f, repo, cmd
}

func TestReleaseFailedWritesPreserveAtomicRetry(t *testing.T) {
	for _, tc := range releaseMutationCases {
		t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
			f, repo, cmd := releaseNativeMutation(t, tc)
			assertCatalogueAtomicRetry(t, f, "release_events", releaseSnapshot, func(ctx context.Context) (projectmutations.Result, error) { return repo.Mutate(ctx, cmd) })
		})
	}
}

func TestReleaseConcurrentWritersPreserveAllRevisionBoundaries(t *testing.T) {
	for _, tc := range []releaseMutationCase{{"release", "update"}, {"release_tag", "update"}, {"release_label", "update"}, {"release", "update_changelog"}, {"release", "manage_workitems"}, {"release_label", "attach"}} {
		t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
			f, repo, first := releaseNativeMutation(t, tc)
			second := first
			request := uuid.New()
			second.RequestID = &request
			calls := [2]func(context.Context) (projectmutations.Result, error){
				func(ctx context.Context) (projectmutations.Result, error) { return repo.Mutate(ctx, first) },
				func(ctx context.Context) (projectmutations.Result, error) { return repo.Mutate(ctx, second) },
			}
			outcomes := competingScopeWrites(t, f, schemaMutationLock, first.WorkspaceID, calls)
			if outcomes[0].err != nil {
				outcomes[0], outcomes[1] = outcomes[1], outcomes[0]
			}
			require.NoError(t, outcomes[0].err)
			require.ErrorIs(t, outcomes[1].err, projectmutations.ErrRevision)
			var receipts int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE tool=$1`, tc.tool).Scan(&receipts))
			require.Equal(t, 1, receipts)
		})
	}
}
