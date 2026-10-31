//go:build integration

package mcpactions_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCustomerIdentityReceiptsRequireCurrentRoleAndFeature(t *testing.T) {
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, action := range []string{"create", "upsert", "update", "delete"} {
			for _, change := range []struct {
				name, statement, code string
				arguments             []any
				disabled, enabled     any
			}{
				{"guest", `UPDATE workspace_members SET role=$3 WHERE workspace_id=$1 AND member_id=$2`, "forbidden", []any{testsupport.LabelWorkspace, testsupport.LabelActor}, 5, 20},
				{"member", `UPDATE workspace_members SET role=$3 WHERE workspace_id=$1 AND member_id=$2`, "forbidden", []any{testsupport.LabelWorkspace, testsupport.LabelActor}, 15, 20},
				{"disabled feature", `UPDATE workspace_features SET customers=$2 WHERE workspace_id=$1`, "feature_disabled", []any{testsupport.LabelWorkspace}, false, true},
			} {
				t.Run(action+"/"+change.name, func(t *testing.T) {
					input := customerIdentityWriteInput(t, call, action)
					accepted := milestoneObject(t, call("customer", input))
					before := customerStorageSnapshot(t, f)
					set := func(value any) {
						ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
						defer cancel()
						arguments := append(slices.Clone(change.arguments), value)
						changed, err := f.pool.Exec(ctx, change.statement, arguments...)
						require.NoError(t, err)
						require.EqualValues(t, 1, changed.RowsAffected())
					}
					t.Cleanup(func() { set(change.enabled) })
					set(change.disabled)
					fresh := maps.Clone(input)
					fresh["request_id"] = uuid.NewString()
					for _, attempt := range []map[string]any{fresh, input} {
						denied := call("customer", attempt)
						milestoneFailure(t, denied, change.code)
						require.Empty(t, denied.StructuredContent)
					}
					require.Equal(t, before, customerStorageSnapshot(t, f))
					set(change.enabled)
					replay := milestoneObject(t, call("customer", input))
					require.Equal(t, true, replay["replayed"])
					require.Equal(t, accepted["result"], replay["result"])
					require.Equal(t, before, customerStorageSnapshot(t, f))
				})
			}
		}
	})
}
