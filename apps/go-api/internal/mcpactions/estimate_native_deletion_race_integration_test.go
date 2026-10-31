//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestEstimateNativeWritersRecheckCommittedDeletion(t *testing.T) {
	for _, scenario := range []string{"create after parent deletion", "edit after parent deletion", "edit after point deletion", "create after project deletion", "edit after project deletion", "create after workspace deletion", "edit after workspace deletion", "create estimate after project deletion", "edit estimate after project deletion", "create estimate after workspace deletion", "edit estimate after workspace deletion"} {
		t.Run(scenario, func(t *testing.T) {
			f, repository, command := estimatePointEditFixture(t)
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			deletion, err := f.pool.Begin(ctx)
			require.NoError(t, err)
			defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, deletion.Rollback, nil)) }()
			writer, err := projectconfiguration.NewRepository(deletion)
			require.NoError(t, err)
			switch {
			case scenario == "edit after point deletion":
				_, err = writer.DeleteEstimatePoint(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, command.ResourceID.String(), command.ParentID.String(), strings.NewReader(`{}`), "application/json")
			case strings.Contains(scenario, "project deletion"), strings.Contains(scenario, "workspace deletion"):
				_, err = deletion.Exec(ctx, `SELECT work_item_schema_lock($1)`, testsupport.LabelWorkspace)
				require.NoError(t, err)
				if strings.Contains(scenario, "project deletion") {
					_, err = deletion.Exec(ctx, `UPDATE projects SET deleted_at=clock_timestamp() WHERE id=$1`, testsupport.LabelProject)
				} else {
					_, err = deletion.Exec(ctx, `UPDATE workspaces SET deleted_at=clock_timestamp() WHERE id=$1`, testsupport.LabelWorkspace)
				}
			default:
				err = writer.DeleteEstimate(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, command.ResourceID.String())
			}
			require.NoError(t, err)
			const snapshotSQL = `SELECT jsonb_build_object(
    'estimates',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM estimates e),
    'points',(SELECT jsonb_agg(to_jsonb(ep) ORDER BY id) FROM estimate_points ep),
    'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY project_id) FROM project_estimate_revisions r),
    'effects',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e))`
			var before, after []byte
			require.NoError(t, deletion.QueryRow(ctx, snapshotSQL).Scan(&before))
			done := make(chan concurrentMutationOutcome[json.RawMessage], 1)
			go func() {
				value, writeErr := writeNativeEstimateRace(ctx, repository, command, scenario)
				done <- concurrentMutationOutcome[json.RawMessage]{value, writeErr}
			}()
			waitEstimateWriters(t, ctx, f, 1)
			require.NoError(t, deletion.Commit(ctx))
			var result concurrentMutationOutcome[json.RawMessage]
			select {
			case result = <-done:
			case <-ctx.Done():
				t.Fatal("native point writer did not finish after deletion")
			}
			var unavailable *projectconfiguration.HTTPError
			require.ErrorAs(t, result.err, &unavailable)
			require.Equal(t, 404, unavailable.Status)
			require.Nil(t, result.value)
			require.NoError(t, f.pool.QueryRow(ctx, snapshotSQL).Scan(&after))
			require.Equal(t, before, after, "a stale native writer must not change points, revisions or cleanup effects")
		})
	}
}

func writeNativeEstimateRace(ctx context.Context, repository *projectconfiguration.Repository, command projectmutations.Command, scenario string) (json.RawMessage, error) {
	switch {
	case strings.HasPrefix(scenario, "create estimate"):
		return repository.CreateEstimate(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, strings.NewReader(`{"estimate":{"name":"late"}}`), "application/json")
	case strings.HasPrefix(scenario, "edit estimate"):
		body := fmt.Sprintf(`{"estimate":{"name":"late"},"estimate_points":[{"id":%q,"value":"late"}]}`, command.ParentID.String())
		return repository.PatchEstimate(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, command.ResourceID.String(), strings.NewReader(body), "application/json")
	case strings.HasPrefix(scenario, "create"):
		return repository.CreateEstimatePoint(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, command.ResourceID.String(), strings.NewReader(`{"key":3,"value":"late"}`), "application/json")
	default:
		return repository.PatchEstimatePoint(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, command.ResourceID.String(), command.ParentID.String(), strings.NewReader(`{"value":"late"}`), "application/json")
	}
}
