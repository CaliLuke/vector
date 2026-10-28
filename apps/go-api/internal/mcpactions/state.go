package mcpactions

import (
	"context"
	"errors"
	"strconv"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	states "github.com/CaliLuke/go-plane/internal/projectstatereads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
)

type stateApplication interface {
	projectResourceReader
	VisibleActions(context.Context, uuid.UUID, uuid.UUID, string) (states.StateCapabilities, error)
	MutateState(context.Context, states.MutationCommand) (states.MutationResult, error)
}
type stateInput struct {
	action, target                                string
	project, workspace, cursor, request, revision *string
	perPage                                       *int
	includeRevision                               bool
	changes                                       states.StateChanges
}

func (s *Service) State(ctx context.Context, input *api.StateRequest) (loom.Nullable[api.StateResult], error) {
	command, err := stateCommand(input)
	if err != nil {
		return loom.Nullable[api.StateResult]{}, err
	}
	var result *api.StateResult
	if command.action == "list" || command.action == "retrieve" {
		result, err = readProjectResource(ctx, &command, selectStateRead, s.states, states.ErrForbidden, states.ErrNotFound, projectStateResult)
	} else {
		result, err = s.stateMutation(ctx, command)
	}
	if err != nil {
		return loom.Nullable[api.StateResult]{}, err
	}
	if result == nil {
		return loom.NullValue[api.StateResult](), nil
	}
	return loom.NullableValue(*result), nil
}
func stateCommand(input *api.StateRequest) (stateInput, error) {
	var command stateInput
	if input == nil {
		return command, stateInvalid()
	}
	switch input.Kind() {
	case api.StateRequestKindStateList:
		v, ok := input.AsStateList()
		if !ok || v == nil || v.Action != "list" {
			return command, stateInvalid()
		}
		command = stateInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, cursor: v.Cursor, perPage: v.PerPage}
	case api.StateRequestKindStateRetrieve:
		v, ok := input.AsStateRetrieve()
		if !ok || v == nil || v.Action != "retrieve" {
			return command, stateInvalid()
		}
		command = stateInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.StateID, includeRevision: v.IncludeRevision != nil && *v.IncludeRevision}
	case api.StateRequestKindStateCreate:
		v, ok := input.AsStateCreate()
		if !ok || v == nil || v.Action != "create" {
			return command, stateInvalid()
		}
		command = stateInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, request: v.RequestID,
			changes: stateChanges(&v.Name, &v.Color, v.Description, v.Group, v.ExternalSource, v.ExternalID, v.Sequence, v.Default)}
	case api.StateRequestKindStateUpdate:
		v, ok := input.AsStateUpdate()
		if !ok || v == nil || v.Action != "update" {
			return command, stateInvalid()
		}
		command = stateInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.StateID, request: v.RequestID, revision: v.ExpectedRevision,
			changes: stateChanges(v.Name, v.Color, v.Description, v.Group, v.ExternalSource, v.ExternalID, v.Sequence, v.Default)}
	case api.StateRequestKindStateDelete:
		v, ok := input.AsStateDelete()
		if !ok || v == nil || v.Action != "delete" {
			return command, stateInvalid()
		}
		command = stateInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.StateID, request: v.RequestID, revision: v.ExpectedRevision}
	default:
		return command, stateInvalid()
	}
	if command.project != nil && *command.project != "" {
		if _, err := stateUUID(*command.project); err != nil {
			return command, err
		}
	}
	if command.action != "list" && command.action != "create" {
		if _, err := stateUUID(command.target); err != nil {
			return command, err
		}
	}
	return command, nil
}
func stateChanges(name, color, description, group, source, external *string, sequence loom.Nullable[float64], defaultState loom.Nullable[bool]) states.StateChanges {
	omit := func(value *string) *string {
		if value != nil && *value != "" {
			return value
		}
		return nil
	}
	return states.StateChanges{Name: omit(name), Color: omit(color), Description: omit(description), Group: omit(group), ExternalSource: omit(source), ExternalID: omit(external), Sequence: nullableStateInput(sequence), Default: nullableStateInput(defaultState)}
}
func stateUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, stateInvalid()
	}
	return id, nil
}
func stateInvalid() error { return api.MakeInvalidParams(errors.New("invalid state input")) }
func selectStateRead(input *stateInput) (djangoapi.ProjectReadRequest, *string, error) {
	request := djangoapi.ProjectReadRequest{Operation: "list", Params: map[string]string{}, Present: map[string]bool{}}
	if input.project != nil {
		request.Params["project_id"] = *input.project
	}
	if input.action == "retrieve" {
		request.Operation = "detail"
		request.Params["state_id"] = input.target
		if input.includeRevision {
			request.Params["include_revision"] = "true"
			request.Present["include_revision"] = true
		}
	} else {
		size := 1000
		if input.perPage != nil && *input.perPage != 0 {
			size = *input.perPage
		}
		if size < 1 || size > 1000 {
			return request, nil, stateInvalid()
		}
		request.Params["per_page"], request.Present["per_page"] = strconv.Itoa(size), true
		if input.cursor != nil && *input.cursor != "" {
			request.Params["cursor"], request.Present["cursor"] = *input.cursor, true
		}
	}
	return request, input.workspace, nil
}

func nullableStateInput[T any](input loom.Nullable[T]) *T {
	value, ok := input.Value()
	if !ok {
		return nil
	}
	return &value
}
