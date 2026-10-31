//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestEstimateNativeWritersRollbackRejectedCommit(t *testing.T) {
	for _, action := range []string{"create estimate", "create point", "edit point"} {
		t.Run(action, func(t *testing.T) {
			f, repository, command := estimatePointEditFixture(t)
			table := "estimate_points"
			if action == "create estimate" {
				table = "estimates"
			}
			write := func() (json.RawMessage, error) {
				switch action {
				case "create estimate":
					return repository.CreateEstimate(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, strings.NewReader(`{"estimate":{"name":"Accepted native creation"}}`), "application/json")
				case "create point":
					return repository.CreateEstimatePoint(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, command.ResourceID.String(), strings.NewReader(`{"key":3,"value":"native"}`), "application/json")
				default:
					return repository.PatchEstimatePoint(t.Context(), testsupport.LabelActor, "workspace-one", testsupport.LabelProject, command.ResourceID.String(), command.ParentID.String(), strings.NewReader(`{"value":"native"}`), "application/json")
				}
			}
			_, err := f.pool.Exec(t.Context(), `CREATE FUNCTION reject_estimate_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected estimate commit rejection'; END $$;
 CREATE CONSTRAINT TRIGGER reject_estimate_commit AFTER INSERT OR UPDATE ON `+pgx.Identifier{table}.Sanitize()+` DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_estimate_commit()`)
			require.NoError(t, err)
			before := estimateStorageSnapshot(t, f)
			result, err := write()
			require.Error(t, err)
			require.Nil(t, result)
			require.Equal(t, before, estimateStorageSnapshot(t, f), "a rejected commit must not leave catalogue rows or revisions")
			_, err = f.pool.Exec(t.Context(), `DROP TRIGGER reject_estimate_commit ON `+pgx.Identifier{table}.Sanitize()+`; DROP FUNCTION reject_estimate_commit()`)
			require.NoError(t, err)
			result, err = write()
			require.NoError(t, err)
			require.NotEmpty(t, result)
			require.NotEqual(t, before, estimateStorageSnapshot(t, f))
		})
	}
}
