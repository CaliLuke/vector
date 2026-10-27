package mcpactions

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/milestones"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
)

type milestoneApplication interface {
	VisibleActions(context.Context, uuid.UUID, uuid.UUID, string) (milestones.Capabilities, error)
	List(context.Context, milestones.Scope, djangoapi.Query) (milestones.Page[milestones.Record], error)
	Retrieve(context.Context, milestones.Scope, uuid.UUID, bool) (milestones.Record, error)
	ListWorkItems(context.Context, milestones.Scope, uuid.UUID, djangoapi.Query) (milestones.Page[milestones.Link], error)
	Create(context.Context, milestones.WriteOptions, milestones.Fields) (milestones.MutationResult, error)
	Update(context.Context, milestones.WriteOptions, uuid.UUID, milestones.Fields) (milestones.MutationResult, error)
	Delete(context.Context, milestones.WriteOptions, uuid.UUID) (milestones.MutationResult, error)
	ManageWorkItems(context.Context, milestones.WriteOptions, uuid.UUID, []uuid.UUID, []uuid.UUID) (milestones.MutationResult, error)
}

type milestoneInput struct {
	action, project, target              string
	workspace, cursor, request, revision *string
	perPage                              *int
	includeRevision                      bool
	fields                               milestones.Fields
	add, remove                          *string
}

func (s *Service) Milestone(ctx context.Context, input *api.MilestoneRequest) (loom.Nullable[api.MilestoneResult], error) {
	command, err := milestoneCommand(input)
	if err != nil {
		return loom.Nullable[api.MilestoneResult]{}, err
	}
	var result *api.MilestoneResult
	if command.action == "list" || command.action == "retrieve" || command.action == "list_workitems" {
		result, err = s.milestoneRead(ctx, command)
	} else {
		result, err = s.milestoneMutation(ctx, command)
	}
	if err != nil {
		return loom.Nullable[api.MilestoneResult]{}, err
	}
	if result == nil {
		return loom.NullValue[api.MilestoneResult](), nil
	}
	return loom.NullableValue(*result), nil
}

func milestoneCommand(input *api.MilestoneRequest) (milestoneInput, error) {
	var command milestoneInput
	if input == nil {
		return command, milestoneInvalid()
	}
	switch {
	case input.Kind() == api.MilestoneRequestKindMilestoneList:
		v, ok := input.AsMilestoneList()
		if !ok || v == nil || v.Action != "list" {
			return command, milestoneInvalid()
		}
		command = milestoneInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, cursor: v.Cursor, perPage: v.PerPage}
	case input.Kind() == api.MilestoneRequestKindMilestoneRetrieve:
		v, ok := input.AsMilestoneRetrieve()
		if !ok || v == nil || v.Action != "retrieve" {
			return command, milestoneInvalid()
		}
		command = milestoneInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.MilestoneID, includeRevision: v.IncludeRevision != nil && *v.IncludeRevision}
	case input.Kind() == api.MilestoneRequestKindMilestoneListWorkitems:
		v, ok := input.AsMilestoneListWorkitems()
		if !ok || v == nil || v.Action != "list_workitems" {
			return command, milestoneInvalid()
		}
		command = milestoneInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.MilestoneID, cursor: v.Cursor, perPage: v.PerPage}
	case input.Kind() == api.MilestoneRequestKindMilestoneCreate:
		v, ok := input.AsMilestoneCreate()
		if !ok || v == nil || v.Action != "create" {
			return command, milestoneInvalid()
		}
		command = milestoneInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, request: v.RequestID, revision: v.ExpectedRevision, fields: milestoneFields(&v.Title, v.TargetDate, v.ExternalSource, v.ExternalID)}
	case input.Kind() == api.MilestoneRequestKindMilestoneUpdate:
		v, ok := input.AsMilestoneUpdate()
		if !ok || v == nil || v.Action != "update" {
			return command, milestoneInvalid()
		}
		command = milestoneInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.MilestoneID, request: v.RequestID, revision: v.ExpectedRevision, fields: milestoneFields(v.Title, v.TargetDate, v.ExternalSource, v.ExternalID)}
	case input.Kind() == api.MilestoneRequestKindMilestoneDelete:
		v, ok := input.AsMilestoneDelete()
		if !ok || v == nil || v.Action != "delete" {
			return command, milestoneInvalid()
		}
		command = milestoneInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.MilestoneID, request: v.RequestID, revision: v.ExpectedRevision}
	case input.Kind() == api.MilestoneRequestKindMilestoneManageWorkitems:
		v, ok := input.AsMilestoneManageWorkitems()
		if !ok || v == nil || v.Action != "manage_workitems" {
			return command, milestoneInvalid()
		}
		command = milestoneInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.MilestoneID, request: v.RequestID, revision: v.ExpectedRevision, add: v.AddIds, remove: v.RemoveIds}
	default:
		return command, milestoneInvalid()
	}
	return command, nil
}

func milestoneScope(actor mcpauth.Principal, input milestoneInput) (milestones.Scope, uuid.UUID, error) {
	scope := milestones.Scope{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug, Timezone: actor.Timezone}
	project, err := milestoneUUID(input.project)
	if err != nil || scope.WorkspaceID == uuid.Nil || scope.WorkspaceSlug == "" {
		return scope, uuid.Nil, milestoneInvalid()
	}
	scope.ProjectID = project
	if input.action == "list" || input.action == "create" {
		return scope, uuid.Nil, nil
	}
	id, err := milestoneUUID(input.target)
	return scope, id, err
}
func milestoneUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, milestoneInvalid()
	}
	return id, nil
}
func milestoneInvalid() error { return api.MakeInvalidParams(errors.New("invalid milestone input")) }
func milestoneFields(title, date, source, external *string) milestones.Fields {
	field := func(value *string) milestones.StringField {
		if value == nil || *value == "" {
			return milestones.StringField{}
		}
		return milestones.StringField{Present: true, Value: value}
	}
	return milestones.Fields{Title: field(title), TargetDate: field(date), ExternalSource: field(source), ExternalID: field(external)}
}
