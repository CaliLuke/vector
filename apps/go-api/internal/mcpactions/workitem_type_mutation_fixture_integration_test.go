//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Each command changes storage, including resolve and import. The target is an
// unused, active workspace definition which has not been imported into a project.
func typeMutationFixture(t *testing.T, action string) (milestoneFixture, *workitemtypes.Repository, workitemtypes.APIMutation) {
	t.Helper()
	f := newMilestoneFixture(t)
	repository, command := prepareTypeMutation(t, f, action)
	return f, repository, command
}

func prepareTypeMutation(t *testing.T, f milestoneFixture, action string) (*workitemtypes.Repository, workitemtypes.APIMutation) {
	t.Helper()
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20`)
	require.NoError(t, err)
	repository := enableTypeCatalogue(t, f, "workspace-one")
	name, active := "Type mutation fixture", true
	command := workitemtypes.APIMutation{
		APIScope: workitemtypes.APIScope{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", Timezone: "UTC"},
		Action:   "create", Changes: workitemtypes.APIChanges{Name: &name, Active: &active},
	}
	created, err := repository.MutateAPI(t.Context(), command)
	require.NoError(t, err)
	id, key := uuid.MustParse(created.ID), uuid.New()
	command.Action, command.RequestID, command.ExpectedRevision = action, &key, &created.Revision
	command.Changes = workitemtypes.APIChanges{}
	switch action {
	case "create":
		newName := "New definition"
		command.Changes = workitemtypes.APIChanges{Name: &newName, Active: &active}
	case "update":
		description := "Changed definition"
		command.TypeID, command.Changes.Description = id, &description
	case "delete":
		command.TypeID = id
	case "resolve":
		command.ProjectID, command.Changes.Name = uuid.MustParse(testsupport.LabelProject), &name
	case "import_to_project":
		command.ProjectID, command.TypeIDs = uuid.MustParse(testsupport.LabelProject), []uuid.UUID{id}
	default:
		t.Fatal("unknown type mutation fixture action", action)
	}
	return repository, command
}
