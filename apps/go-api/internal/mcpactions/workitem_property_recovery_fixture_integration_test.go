//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type propertyRecoveryCase struct {
	resourceID uuid.UUID
	milestoneFixture
	key         uuid.UUID
	tool, table string
	mutate      func(context.Context, uuid.UUID) (projectmutations.Result, error)
}

func newPropertyRecoveryCase(t *testing.T, action string) propertyRecoveryCase {
	t.Helper()
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `UPDATE workspace_members SET role=20; UPDATE project_members SET role=20`)
	require.NoError(t, err)
	repository := enableTypeCatalogue(t, f, "workspace-one")
	if strings.HasSuffix(action, "_value") {
		return propertyValueRecoveryCase(t, f, repository, action)
	}
	scope := workitemtypes.APIScope{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", Timezone: "UTC"}
	name, kind := "Recovery fixture", "OPTION"
	command := workitemtypes.PropertyAPIMutation{PropertyAPIQuery: workitemtypes.PropertyAPIQuery{APIScope: scope, Action: "create"}, Changes: workitemtypes.PropertyAPIChanges{DisplayName: &name, Kind: &kind, Options: []workitemtypes.PropertyAPIOption{{Name: "First"}, {Name: "Second"}}}}
	created, err := repository.MutatePropertyAPI(t.Context(), command)
	require.NoError(t, err)
	var record struct{ Options []struct{ ID uuid.UUID } }
	require.NoError(t, json.Unmarshal(created.Record, &record))
	require.Len(t, record.Options, 2)
	propertyID := uuid.MustParse(created.ID)
	command.Action, command.PropertyID = action, propertyID
	command.Changes = workitemtypes.PropertyAPIChanges{}
	table := "work_item_properties"
	switch action {
	case "create":
		newName := "New recovery property"
		command.PropertyID = uuid.Nil
		command.Changes = workitemtypes.PropertyAPIChanges{DisplayName: &newName, Kind: &kind, Options: []workitemtypes.PropertyAPIOption{{Name: "New option"}}}
	case "update":
		description := "Accepted property edit"
		command.Changes.Description = &description
	case "delete":
	case "manage_type_properties":
		typeName := "Recovery type"
		createdType, createErr := repository.MutateAPI(t.Context(), workitemtypes.APIMutation{APIScope: scope, Action: "create", Changes: workitemtypes.APIChanges{Name: &typeName}})
		require.NoError(t, createErr)
		command.PropertyID, command.TypeID, command.Attach = uuid.Nil, uuid.MustParse(createdType.ID), []uuid.UUID{propertyID}
		table = "work_item_type_properties"
	case "create_option":
		command.Option.Name = "New option"
	case "update_option", "delete_option":
		command.OptionID = record.Options[0].ID
		if action == "update_option" {
			command.Option.Name = "Changed option"
		}
	default:
		t.Fatal("unknown property recovery action", action)
	}
	raw, err := repository.ReadPropertyAPI(t.Context(), workitemtypes.PropertyAPIQuery{APIScope: scope, Action: "retrieve", PropertyID: propertyID, IncludeRevision: true})
	require.NoError(t, err)
	revision := propertyRecoveryRevision(t, raw)
	command.ExpectedRevision = &revision
	return propertyRecoveryCase{milestoneFixture: f, resourceID: propertyID, key: uuid.New(), tool: "workitem_property", table: table, mutate: func(ctx context.Context, key uuid.UUID) (projectmutations.Result, error) {
		attempt := command
		attempt.RequestID = &key
		return repository.MutatePropertyAPI(ctx, attempt)
	}}
}

func propertyValueRecoveryCase(t *testing.T, f milestoneFixture, properties *workitemtypes.Repository, action string) propertyRecoveryCase {
	t.Helper()
	handler := f.handler(t, nil)
	call := catalogueCall(func(tool string, input map[string]any) rpcResult {
		body, err := json.Marshal(input)
		require.NoError(t, err)
		return rpc(t, handler, "test-one", "tools/call", resourceCall(tool, string(body)))
	})
	input := preparePropertyMutationInput(t, f, call, action)
	command := projectmutations.Command{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", ProjectID: uuid.MustParse(testsupport.LabelProject), ResourceID: uuid.MustParse(input["workitem_id"].(string)), ParentID: uuid.MustParse(input["property_id"].(string)), Action: action, Timezone: "UTC", Fields: map[string]any{}}
	if action == "set_value" {
		command.Fields["value"] = input["value"]
	}
	raw, err := properties.ReadPropertyValueAPI(t.Context(), workitemtypes.PropertyValueAPIQuery{APIScope: workitemtypes.APIScope{ActorID: command.ActorID, WorkspaceID: command.WorkspaceID, WorkspaceSlug: command.WorkspaceSlug, ProjectID: command.ProjectID}, IssueID: command.ResourceID, PropertyID: command.ParentID, IncludeRevision: true})
	require.NoError(t, err)
	revision := propertyRecoveryRevision(t, raw)
	command.ExpectedRevision = &revision
	repository, err := workitems.NewRepository(f.pool, rosterPublisher{}, "https://vector.example")
	require.NoError(t, err)
	return propertyRecoveryCase{milestoneFixture: f, resourceID: command.ResourceID, key: uuid.New(), tool: "workitem_property_value", table: "issues", mutate: func(ctx context.Context, key uuid.UUID) (projectmutations.Result, error) {
		attempt := command
		attempt.RequestID = &key
		return repository.MutatePropertyValue(ctx, attempt)
	}}
}

func propertyRecoveryRevision(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var record struct{ Revision string }
	require.NoError(t, json.Unmarshal(raw, &record))
	require.NotEmpty(t, record.Revision)
	return record.Revision
}
