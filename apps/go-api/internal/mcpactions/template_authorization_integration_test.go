//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func templateWriteInput(t *testing.T, call catalogueCall, scope templateScope, action string) map[string]any {
	t.Helper()
	input := templateScopedInput(scope, "create")
	input["name"], input["template_data"] = uuid.NewString(), `{}`
	if action != "create" {
		id := milestoneObject(t, call("template", input))["id"]
		input = templateScopedInput(scope, action)
		input["template_id"] = id
		if action == "update" {
			input["name"] = "Revised template"
		}
	}
	input["request_id"] = uuid.NewString()
	return input
}

func TestMCPTemplateWritesAndReplayRecheckScopeRoleAndGovernance(t *testing.T) {
	for _, scope := range templateScopes() {
		for _, action := range []string{"create", "update", "delete"} {
			t.Run(scope.kind+"/"+scope.project+"/"+action, func(t *testing.T) {
				runCatalogueCredentials(t, newMilestoneFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
					input := templateWriteInput(t, call, scope, action)
					accepted := milestoneObject(t, call("template", input))
					fresh := maps.Clone(input)
					fresh["request_id"] = uuid.NewString()
					table, column, id := "workspace_members", "workspace_id", testsupport.LabelWorkspace
					if scope.project != "" {
						table, column, id = "project_members", "project_id", scope.project
					}
					for _, role := range []int{5, 15} {
						_, err := f.pool.Exec(t.Context(), `UPDATE `+table+` SET role=$3 WHERE `+column+`=$1 AND member_id=$2`, id, testsupport.LabelActor, role)
						require.NoError(t, err)
						before := templateStorageSnapshot(t, f)
						milestoneFailure(t, call("template", input), "forbidden")
						milestoneFailure(t, call("template", fresh), "forbidden")
						catalogueArray(t, call("template", templateScopedInput(scope, "list")))
						require.Equal(t, before, templateStorageSnapshot(t, f))
					}
					_, err := f.pool.Exec(t.Context(), `UPDATE `+table+` SET role=20 WHERE `+column+`=$1 AND member_id=$2`, id, testsupport.LabelActor)
					require.NoError(t, err)
					if scope.project != "" {
						_, err = f.pool.Exec(t.Context(), `UPDATE workspace_template_settings SET owned_by_workspace=true WHERE workspace_id=$1`, testsupport.LabelWorkspace)
						require.NoError(t, err)
						before := templateStorageSnapshot(t, f)
						milestoneFailure(t, call("template", input), "workspace_owned")
						milestoneFailure(t, call("template", fresh), "workspace_owned")
						catalogueArray(t, call("template", templateScopedInput(scope, "list")))
						require.Equal(t, before, templateStorageSnapshot(t, f))
						_, err = f.pool.Exec(t.Context(), `UPDATE workspace_template_settings SET owned_by_workspace=false WHERE workspace_id=$1`, testsupport.LabelWorkspace)
						require.NoError(t, err)
					}
					before := templateStorageSnapshot(t, f)
					accepted["replayed"] = true
					require.Equal(t, accepted, milestoneObject(t, call("template", input)))
					require.Equal(t, before, templateStorageSnapshot(t, f))
				})
			})
		}
	}
}
