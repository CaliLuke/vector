//go:build integration

package mcpactions_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type customerRequestLinkChange struct {
	name, code, disable, restore string
	arguments                    func(map[string]any) []any
}

func customerRequestLinkChanges() []customerRequestLinkChange {
	project := func(map[string]any) []any { return []any{testsupport.LabelProject} }
	member := func(map[string]any) []any { return []any{testsupport.LabelProject, testsupport.LabelActor} }
	item := func(input map[string]any) []any { return []any{input["workitem_ids"]} }
	return []customerRequestLinkChange{
		{"private project membership", "not_found", `UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, `UPDATE project_members SET is_active=true WHERE project_id=$1 AND member_id=$2`, member},
		{"project guest", "forbidden", `UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, `UPDATE project_members SET role=20 WHERE project_id=$1 AND member_id=$2`, member},
		{"archived project", "not_found", `UPDATE projects SET archived_at=now() WHERE id=$1`, `UPDATE projects SET archived_at=NULL WHERE id=$1`, project},
		{"deleted project", "not_found", `UPDATE projects SET deleted_at=now() WHERE id=$1`, `UPDATE projects SET deleted_at=NULL WHERE id=$1`, project},
		{"draft item", "not_found", `UPDATE issues SET is_draft=true WHERE id=$1`, `UPDATE issues SET is_draft=false WHERE id=$1`, item},
		{"deleted item", "not_found", `UPDATE issues SET deleted_at=now() WHERE id=$1`, `UPDATE issues SET deleted_at=NULL WHERE id=$1`, item},
	}
}

func (change customerRequestLinkChange) setActive(t *testing.T, ctx context.Context, f milestoneFixture, input map[string]any, active bool) {
	t.Helper()
	statement := change.disable
	if active {
		statement = change.restore
	}
	changed, err := f.pool.Exec(ctx, statement, change.arguments(input)...)
	require.NoError(t, err)
	require.EqualValues(t, 1, changed.RowsAffected())
}

func customerRequestLinkedInput(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
	t.Helper()
	input := customerMutationWriteInput(t, f, call, customerMutationScenario{tool: "customer_request", action: "create"})
	input["workitem_ids"] = f.item(t)
	return input
}

func TestMCPCustomerRequestLinkReceiptsRequireCurrentTargetAccess(t *testing.T) {
	runCustomerLinkTargetReplay(t, "customer_request", customerRequestLinkedInput, customerRequestLinkChanges())
}

func runCustomerLinkTargetReplay(t *testing.T, tool string, prepare func(*testing.T, milestoneFixture, catalogueCall) map[string]any, changes []customerRequestLinkChange) {
	t.Helper()
	runCatalogueFixtureMatrix(t, customerIdentityFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, change := range changes {
			t.Run(change.name, func(t *testing.T) {
				input := prepare(t, f, call)
				accepted := milestoneObject(t, call(tool, input))
				require.Equal(t, false, accepted["replayed"])
				t.Cleanup(func() {
					ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
					defer cancel()
					change.setActive(t, ctx, f, input, true)
				})
				change.setActive(t, t.Context(), f, input, false)
				before := customerStorageSnapshot(t, f)
				fresh := maps.Clone(input)
				retryField := "request_id"
				if tool == "customer_request" {
					retryField = "mutation_id"
				}
				fresh[retryField] = uuid.NewString()
				for _, attempt := range []map[string]any{fresh, input} {
					denied := call(tool, attempt)
					milestoneFailure(t, denied, change.code)
					require.Empty(t, denied.StructuredContent)
				}
				require.Equal(t, before, customerStorageSnapshot(t, f))
				change.setActive(t, t.Context(), f, input, true)
				restored := customerStorageSnapshot(t, f)
				replayed := milestoneObject(t, call(tool, input))
				require.Equal(t, true, replayed["replayed"])
				require.Equal(t, accepted["result"], replayed["result"])
				require.Equal(t, restored, customerStorageSnapshot(t, f), "restoring target access replays without another request, link or activity")
			})
		}
	})
}

func TestMCPCustomerRequestLinksReauthorizeTargetsAfterWaiting(t *testing.T) {
	runCustomerLinkTargetWait(t, "customer_request", customerRequestLinkedInput, customerRequestLinkChanges())
}

func runCustomerLinkTargetWait(t *testing.T, tool string, prepare func(*testing.T, milestoneFixture, catalogueCall) map[string]any, targets []customerRequestLinkChange) {
	t.Helper()
	var changes []mutationWaitChange
	for _, change := range targets {
		for _, mode := range []string{"personal token", "OAuth"} {
			changes = append(changes, mutationWaitChange{
				name: change.name + "/" + mode, mode: mode, code: change.code,
				apply: func(t *testing.T, ctx context.Context, writer pgx.Tx, input map[string]any) []byte {
					changed, err := writer.Exec(ctx, change.disable, change.arguments(input)...)
					require.NoError(t, err)
					require.EqualValues(t, 1, changed.RowsAffected())
					return nil
				},
			})
		}
	}
	runMutationAuthorizationWait(t, mutationWaitScenario{
		tool: tool, schemaLock: true, fixture: customerIdentityFixture,
		snapshot: customerStorageSnapshot, prepare: prepare, changes: changes,
	})
}
