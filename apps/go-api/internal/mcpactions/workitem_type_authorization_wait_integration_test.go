//go:build integration

package mcpactions_test

import (
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
)

func TestMCPTypeWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "resolve", "import_to_project"} {
		t.Run(action, func(t *testing.T) {
			runMutationAuthorizationWait(t, mutationWaitScenario{
				schemaLock: true, tool: "workitem_type", action: action,
				fixture: newMilestoneFixture, snapshot: typeStorageSnapshot,
				prepare: func(t *testing.T, f milestoneFixture, _ catalogueCall) map[string]any {
					_, command := prepareTypeMutation(t, f, action)
					return typeMutationArguments(command)
				},
			})
		})
	}
}

func typeMutationArguments(command workitemtypes.APIMutation) map[string]any {
	input := map[string]any{"action": command.Action, "workspace_slug": command.WorkspaceSlug, "request_id": command.RequestID.String(), "expected_revision": *command.ExpectedRevision}
	if command.TypeID != uuid.Nil {
		input["workitem_type_id"] = command.TypeID.String()
	}
	if command.ProjectID != uuid.Nil {
		input["project_id"] = command.ProjectID.String()
	}
	if command.Changes.Name != nil {
		input["name"] = *command.Changes.Name
	}
	if command.Changes.Description != nil {
		input["description"] = *command.Changes.Description
	}
	if command.Changes.Active != nil {
		input["is_active"] = *command.Changes.Active
	}
	if len(command.TypeIDs) > 0 {
		ids := make([]string, len(command.TypeIDs))
		for i, id := range command.TypeIDs {
			ids[i] = id.String()
		}
		input["workitem_type_ids"] = strings.Join(ids, ",")
	}
	return input
}
