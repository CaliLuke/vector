//go:build integration

package mcpactions_test

import (
	"context"
	"fmt"
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemrelations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var relationMutationActions = []string{"create_definition", "update_definition", "delete_definition", "create", "delete"}

func relationNativeCommand(t *testing.T, f milestoneFixture, action string) workitemrelations.Command {
	t.Helper()
	repo := workitemrelations.NewRepository(f.pool)
	name := "Definition " + uuid.NewString()
	cmd := workitemrelations.Command{Query: workitemrelations.Query{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", Action: "create_definition"}, Changes: workitemrelations.Changes{Name: &name}}
	definition, err := repo.Mutate(t.Context(), cmd)
	require.NoError(t, err)
	cmd.Action = action
	cmd.Changes = workitemrelations.Changes{}
	switch action {
	case "create_definition":
		name = "Created " + uuid.NewString()
		cmd.Changes.Name = &name
	case "update_definition":
		cmd.DefinitionID = uuid.MustParse(definition.ID)
		name = "Renamed " + uuid.NewString()
		cmd.Changes.Name = &name
	case "delete_definition":
		cmd.DefinitionID = uuid.MustParse(definition.ID)
	default:
		cmd.ProjectID = uuid.MustParse(testsupport.LabelProject)
		cmd.IssueID = uuid.MustParse(f.item(t))
		target := uuid.MustParse(f.item(t))
		cmd.Targets = []uuid.UUID{target}
		cmd.RelationType = "blocking"
		if action == "delete" {
			create := cmd
			create.Action = "create"
			_, err = repo.Mutate(t.Context(), create)
			require.NoError(t, err)
			cmd.Targets = nil
			cmd.RelatedID = target
			cmd.IsDependency = true
		}
	}
	var rev int64
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM work_item_relation_revisions WHERE workspace_id=$1`, cmd.WorkspaceID).Scan(&rev))
	token := fmt.Sprintf("workitem_relation:%s:%d", cmd.WorkspaceID, rev)
	cmd.ExpectedRevision = &token
	request := uuid.New()
	cmd.RequestID = &request
	return cmd
}
func relationWriteInput(t *testing.T, f milestoneFixture, action string) map[string]any {
	t.Helper()
	cmd := relationNativeCommand(t, f, action)
	input := map[string]any{"action": action, "workspace_slug": "workspace-one", "request_id": cmd.RequestID.String()}
	if cmd.Changes.Name != nil {
		input["name"] = *cmd.Changes.Name
	}
	if cmd.DefinitionID != uuid.Nil {
		input["definition_id"] = cmd.DefinitionID.String()
	}
	if cmd.ProjectID != uuid.Nil {
		input["project_id"] = cmd.ProjectID.String()
		input["workitem_id"] = cmd.IssueID.String()
	}
	if action == "create" {
		input["relation_type"] = cmd.RelationType
		input["workitem_ids"] = []string{cmd.Targets[0].String()}
	}
	if action == "delete" {
		input["related_workitem_id"] = cmd.RelatedID.String()
		input["is_dependency"] = true
	}
	return input
}
func TestRelationMutationsPreserveAtomicRetry(t *testing.T) {
	for _, action := range relationMutationActions {
		t.Run(action, func(t *testing.T) {
			f := templateFixture(t)
			cmd := relationNativeCommand(t, f, action)
			repo := workitemrelations.NewRepository(f.pool)
			assertCatalogueAtomicRetry(t, f, "mutation_receipts", coreSnapshot, func(ctx context.Context) (projectmutations.Result, error) { return repo.Mutate(ctx, cmd) })
		})
	}
}
func TestMCPRelationMutationsReauthorizeAfterWaiting(t *testing.T) {
	for _, action := range relationMutationActions {
		t.Run(action, func(t *testing.T) {
			runMutationAuthorizationWait(t, mutationWaitScenario{changes: mutationCredentialChanges(), tool: "workitem_relation", schemaLock: true, fixture: templateFixture, snapshot: coreSnapshot, prepare: func(t *testing.T, f milestoneFixture, _ catalogueCall) map[string]any {
				return relationWriteInput(t, f, action)
			}})
		})
	}
}
func TestMCPRelationReceiptsBindEveryActionInput(t *testing.T) {
	runCatalogue(t, templateFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		for _, action := range relationMutationActions {
			t.Run(action, func(t *testing.T) {
				input := relationWriteInput(t, f, action)
				require.Equal(t, false, milestoneObject(t, call("workitem_relation", input))["replayed"])
				before := coreSnapshot(t, f)
				require.Equal(t, true, milestoneObject(t, call("workitem_relation", input))["replayed"])
				conflict := maps.Clone(input)
				conflict["expected_revision"] = "changed-review"
				milestoneFailure(t, call("workitem_relation", conflict), "request_conflict")
				require.Equal(t, before, coreSnapshot(t, f))
			})
		}
	})
}
