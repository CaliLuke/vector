package mcpactions

import (
	"errors"

	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/google/uuid"
)

type workitemTypeInput struct {
	action, target                                string
	project, workspace, request, revision, cursor *string
	perPage                                       *int
	includeRevision                               bool
	changes                                       workitemtypes.APIChanges
	typeIDs                                       []uuid.UUID
}

func selectWorkitemType(input *api.WorkitemTypeRequest) (out workitemTypeInput, err error) {
	invalid := func() (workitemTypeInput, error) {
		return out, api.MakeInvalidParams(errors.New("invalid work item type action"))
	}
	if input == nil {
		return invalid()
	}
	var projects *string
	if v, ok := input.AsWorkitemTypeList(); ok && v != nil && v.Action == "list" {
		out = workitemTypeInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, cursor: v.Cursor, perPage: v.PerPage}
	} else if v, ok := input.AsWorkitemTypeRetrieve(); ok && v != nil && v.Action == "retrieve" {
		out = workitemTypeInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.WorkitemTypeID, includeRevision: v.IncludeRevision != nil && *v.IncludeRevision}
	} else if v, ok := input.AsWorkitemTypeCreate(); ok && v != nil && v.Action == "create" {
		out = workitemTypeInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, request: v.RequestID, revision: v.ExpectedRevision, changes: workitemtypes.APIChanges{Name: &v.Name, Description: v.Description, ExternalSource: v.ExternalSource, ExternalID: v.ExternalID}}
		if flag, ok := v.IsActive.Value(); ok {
			out.changes.Active = &flag
		}
		projects = v.ProjectIds
	} else if v, ok := input.AsWorkitemTypeUpdate(); ok && v != nil && v.Action == "update" {
		out = workitemTypeInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.WorkitemTypeID, request: v.RequestID, revision: v.ExpectedRevision, changes: workitemtypes.APIChanges{Name: v.Name, Description: v.Description, ExternalSource: v.ExternalSource, ExternalID: v.ExternalID}}
		if flag, ok := v.IsActive.Value(); ok {
			out.changes.Active = &flag
		}
		projects = v.ProjectIds
	} else if v, ok := input.AsWorkitemTypeDelete(); ok && v != nil && v.Action == "delete" {
		out = workitemTypeInput{action: v.Action, project: v.ProjectID, workspace: v.WorkspaceSlug, target: v.WorkitemTypeID, request: v.RequestID, revision: v.ExpectedRevision}
	} else if v, ok := input.AsWorkitemTypeResolve(); ok && v != nil && v.Action == "resolve" {
		out = workitemTypeInput{action: v.Action, project: &v.ProjectID, workspace: v.WorkspaceSlug, request: v.RequestID, revision: v.ExpectedRevision, changes: workitemtypes.APIChanges{Name: &v.Name}}
	} else if v, ok := input.AsWorkitemTypeImport(); ok && v != nil && v.Action == "import_to_project" {
		out = workitemTypeInput{action: v.Action, project: &v.ProjectID, workspace: v.WorkspaceSlug, request: v.RequestID, revision: v.ExpectedRevision}
		out.typeIDs, err = milestoneIDs(&v.WorkitemTypeIds)
		if err != nil {
			return invalid()
		}
	} else {
		return invalid()
	}
	out.changes.ProjectIDs, err = milestoneIDs(projects)
	if err != nil {
		return invalid()
	}
	for _, field := range []**string{&out.changes.Name, &out.changes.Description, &out.changes.ExternalSource, &out.changes.ExternalID} {
		if *field != nil && **field == "" {
			*field = nil
		}
	}
	return out, nil
}
