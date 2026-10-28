package mcpactions

import (
	"errors"

	"github.com/CaliLuke/go-plane/internal/workitemrelations"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/google/uuid"
)

type relationInput struct {
	command            workitemrelations.Command
	workspace, request *string
}

func selectRelation(input *api.WorkitemRelationRequest) (relationInput, error) {
	var out relationInput
	var project, item, definition, related string
	var targets []string
	if input == nil {
		return out, api.MakeInvalidParams(errors.New("invalid relation action"))
	}
	if err := input.Validate(); err != nil {
		return out, relationInputError()
	}
	cmd := &out.command
	switch input.Kind() {
	case api.WorkitemRelationRequestKindRelationList:
		v, _ := input.AsRelationList()
		if v == nil {
			return out, relationInputError()
		}
		cmd.Action = v.Action
		out.workspace = v.WorkspaceSlug
		project, item = v.ProjectID, v.WorkitemID
		cmd.IncludeRevision = v.IncludeRevision != nil && *v.IncludeRevision
	case api.WorkitemRelationRequestKindRelationDefinitions:
		v, _ := input.AsRelationDefinitions()
		if v == nil {
			return out, relationInputError()
		}
		cmd.Action = v.Action
		out.workspace = v.WorkspaceSlug
		cmd.IsDefault, cmd.IsActive = v.IsDefault, v.IsActive
		cmd.IncludeRevision = v.IncludeRevision != nil && *v.IncludeRevision
	case api.WorkitemRelationRequestKindRelationCreate:
		v, _ := input.AsRelationCreate()
		if v == nil {
			return out, relationInputError()
		}
		cmd.Action = v.Action
		out.workspace, out.request = v.WorkspaceSlug, v.RequestID
		cmd.ExpectedRevision = v.ExpectedRevision
		project, item = v.ProjectID, v.WorkitemID
		targets = v.WorkitemIds
		if v.RelationType != nil {
			cmd.RelationType = *v.RelationType
		}
		if cmd.RelationType == "" {
			if v.RelationDefinitionID != nil {
				definition = *v.RelationDefinitionID
			}
			if v.RelationDefinitionLabel != nil {
				cmd.DefinitionLabel = *v.RelationDefinitionLabel
			}
		}
	case api.WorkitemRelationRequestKindRelationDelete:
		v, _ := input.AsRelationDelete()
		if v == nil {
			return out, relationInputError()
		}
		cmd.Action = v.Action
		out.workspace, out.request = v.WorkspaceSlug, v.RequestID
		cmd.ExpectedRevision = v.ExpectedRevision
		project, item, related = v.ProjectID, v.WorkitemID, v.RelatedWorkitemID
		cmd.IsDependency = v.IsDependency != nil && *v.IsDependency
	case api.WorkitemRelationRequestKindRelationDefinitionCreate:
		v, _ := input.AsRelationDefinitionCreate()
		if v == nil {
			return out, relationInputError()
		}
		cmd.Action = v.Action
		out.workspace, out.request = v.WorkspaceSlug, v.RequestID
		cmd.ExpectedRevision = v.ExpectedRevision
		cmd.Changes = relationChanges(&v.Name, v.Outward, v.Inward, v.Color, v.IsActive)
	case api.WorkitemRelationRequestKindRelationDefinitionUpdate:
		v, _ := input.AsRelationDefinitionUpdate()
		if v == nil {
			return out, relationInputError()
		}
		cmd.Action = v.Action
		out.workspace, out.request = v.WorkspaceSlug, v.RequestID
		cmd.ExpectedRevision = v.ExpectedRevision
		definition = v.DefinitionID
		cmd.Changes = relationChanges(v.Name, v.Outward, v.Inward, v.Color, v.IsActive)
	case api.WorkitemRelationRequestKindRelationDefinitionDelete:
		v, _ := input.AsRelationDefinitionDelete()
		if v == nil {
			return out, relationInputError()
		}
		cmd.Action = v.Action
		out.workspace, out.request = v.WorkspaceSlug, v.RequestID
		cmd.ExpectedRevision = v.ExpectedRevision
		definition = v.DefinitionID
	default:
		return out, relationInputError()
	}
	for _, pair := range []struct {
		value  string
		target *uuid.UUID
	}{{project, &cmd.ProjectID}, {item, &cmd.IssueID}, {definition, &cmd.DefinitionID}, {related, &cmd.RelatedID}} {
		if pair.value != "" {
			id, err := typeUUID(pair.value)
			if err != nil {
				return out, err
			}
			*pair.target = id
		}
	}
	for _, value := range targets {
		id, err := typeUUID(value)
		if err != nil {
			return out, err
		}
		cmd.Targets = append(cmd.Targets, id)
	}
	if out.request != nil {
		id, err := typeUUID(*out.request)
		if err != nil {
			return out, err
		}
		cmd.RequestID = &id
	}
	return out, nil
}
func relationChanges(name, outward, inward, color *string, active *bool) workitemrelations.Changes {
	nonempty := func(v *string) *string {
		if v != nil && *v != "" {
			return v
		}
		return nil
	}
	return workitemrelations.Changes{Name: nonempty(name), Outward: nonempty(outward), Inward: nonempty(inward), Color: nonempty(color), Active: active}
}
func relationInputError() error { return api.MakeInvalidParams(errors.New("invalid relation action")) }
