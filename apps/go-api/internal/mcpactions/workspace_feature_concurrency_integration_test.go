//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/features"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFeatureConcurrentRevisionAllowsOneCommit(t *testing.T) {
	for _, scope := range []struct {
		tool, first, second, readSQL, rowID string
		project                             uuid.UUID
	}{
		{"workspace", "customers", "teams", `SELECT customers,teams FROM workspace_features WHERE workspace_id=$1`, testsupport.LabelWorkspace, uuid.Nil},
		{"project", "cycles", "modules", `SELECT cycle_view,module_view FROM projects WHERE id=$1`, testsupport.LabelProject, uuid.MustParse(testsupport.LabelProject)},
	} {
		t.Run(scope.tool, func(t *testing.T) {
			f := newMilestoneFixture(t)
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			_, err := f.pool.Exec(ctx, `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20; UPDATE workspace_features SET customers=false,teams=false; UPDATE projects SET cycle_view=false,module_view=false`)
			require.NoError(t, err)
			repository := features.NewRepository(f.pool)
			query := features.Query{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", ProjectID: scope.project, IncludeRevision: true}
			raw, err := repository.Read(ctx, query)
			require.NoError(t, err)
			var before struct {
				Revision string `json:"revision"`
			}
			require.NoError(t, json.Unmarshal(raw, &before))
			require.NotEmpty(t, before.Revision)

			blocker, err := f.pool.Begin(ctx)
			require.NoError(t, err)
			defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, blocker.Rollback, nil)) }()
			_, err = blocker.Exec(ctx, `SELECT revision FROM workspace_features WHERE workspace_id=$1 FOR UPDATE`, testsupport.LabelWorkspace)
			require.NoError(t, err)
			type outcome struct {
				value projectmutations.Result
				err   error
			}
			results := make(chan outcome, 2)
			for _, flag := range []string{scope.first, scope.second} {
				key := uuid.New()
				command := features.Command{Query: query, Values: map[string]bool{flag: true}, RequestID: &key, ExpectedRevision: &before.Revision}
				go func() {
					value, mutationErr := repository.Mutate(ctx, command)
					results <- outcome{value, mutationErr}
				}()
			}
			require.Eventually(t, func() bool {
				var waiting int
				queryErr := f.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'`).Scan(&waiting)
				return queryErr == nil && waiting == 2
			}, 10*time.Second, 10*time.Millisecond, "both feature writes must wait before the competing commits")
			require.NoError(t, blocker.Commit(ctx))
			var committed []projectmutations.Result
			var rejected []error
			for range 2 {
				select {
				case result := <-results:
					if result.err != nil {
						rejected = append(rejected, result.err)
					} else {
						committed = append(committed, result.value)
					}
				case <-ctx.Done():
					t.Fatal("feature writes did not finish after releasing the row lock")
				}
			}
			require.Len(t, committed, 1)
			require.Len(t, rejected, 1)
			require.ErrorIs(t, rejected[0], projectmutations.ErrRevision)
			var firstEnabled, secondEnabled bool
			require.NoError(t, f.pool.QueryRow(ctx, scope.readSQL, scope.rowID).Scan(&firstEnabled, &secondEnabled))
			require.NotEqual(t, firstEnabled, secondEnabled, "only the winning update may change its flag")
			require.Equal(t, 1, workspaceFeatureAuditCount(t, f))
			var receipts int
			require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM mutation_receipts WHERE workspace_id=$1 AND tool=$2 AND action='update_features'`, testsupport.LabelWorkspace, scope.tool).Scan(&receipts))
			require.Equal(t, 1, receipts)
			query.IncludeRevision = false
			current, err := repository.Read(ctx, query)
			require.NoError(t, err)
			require.JSONEq(t, string(committed[0].Record), string(current))

		})
	}
}
