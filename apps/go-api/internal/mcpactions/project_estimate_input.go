package mcpactions

import (
	"errors"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func selectProjectEstimate(input *api.ProjectEstimateRequest) (selected resourceMutationInput, read, revision bool, err error) {
	selected.fields = map[string]any{}
	var text map[string]*string
	switch {
	case input.Kind() == api.ProjectEstimateRequestKindProjectEstimateRetrieve:
		v, ok := input.AsProjectEstimateRetrieve()
		if !ok || v == nil || v.Action != "retrieve" {
			break
		}
		selected.action, selected.project, selected.workspace = v.Action, v.ProjectID, v.WorkspaceSlug
		return selected, true, v.IncludeRevision != nil && *v.IncludeRevision, nil
	case input.Kind() == api.ProjectEstimateRequestKindProjectEstimateListPoints:
		v, ok := input.AsProjectEstimateListPoints()
		if !ok || v == nil || v.Action != "list_points" {
			break
		}
		selected.action, selected.project, selected.target, selected.workspace = v.Action, v.ProjectID, v.EstimateID, v.WorkspaceSlug
		return selected, true, false, nil
	case input.Kind() == api.ProjectEstimateRequestKindProjectEstimateCreate:
		v, ok := input.AsProjectEstimateCreate()
		if !ok || v == nil || v.Action != "create" {
			break
		}
		selected.action, selected.project, selected.workspace, selected.request, selected.revision = v.Action, v.ProjectID, v.WorkspaceSlug, v.RequestID, v.ExpectedRevision
		text = map[string]*string{"name": &v.Name, "type": v.Type, "description": v.Description, "external_source": v.ExternalSource, "external_id": v.ExternalID}
		if flag, ok := v.LastUsed.Value(); ok {
			selected.fields["last_used"] = flag
		}
	case input.Kind() == api.ProjectEstimateRequestKindProjectEstimateUpdate:
		v, ok := input.AsProjectEstimateUpdate()
		if !ok || v == nil || v.Action != "update" {
			break
		}
		selected.action, selected.project, selected.target, selected.workspace, selected.request, selected.revision = v.Action, v.ProjectID, v.ProjectID, v.WorkspaceSlug, v.RequestID, v.ExpectedRevision
		text = map[string]*string{"name": v.Name, "description": v.Description, "external_source": v.ExternalSource, "external_id": v.ExternalID}
	case input.Kind() == api.ProjectEstimateRequestKindProjectEstimateDelete:
		v, ok := input.AsProjectEstimateDelete()
		if !ok || v == nil || v.Action != "delete" {
			break
		}
		selected.action, selected.project, selected.target, selected.workspace, selected.request, selected.revision = v.Action, v.ProjectID, v.ProjectID, v.WorkspaceSlug, v.RequestID, v.ExpectedRevision
	case input.Kind() == api.ProjectEstimateRequestKindProjectEstimateLink:
		v, ok := input.AsProjectEstimateLink()
		if !ok || v == nil || v.Action != "link" {
			break
		}
		selected.action, selected.project, selected.target, selected.workspace, selected.request, selected.revision = v.Action, v.ProjectID, v.EstimateID, v.WorkspaceSlug, v.RequestID, v.ExpectedRevision
	case input.Kind() == api.ProjectEstimateRequestKindProjectEstimateCreatePoints:
		v, ok := input.AsProjectEstimateCreatePoints()
		if !ok || v == nil || v.Action != "create_points" {
			break
		}
		selected.action, selected.project, selected.target, selected.workspace, selected.request, selected.revision = v.Action, v.ProjectID, v.EstimateID, v.WorkspaceSlug, v.RequestID, v.ExpectedRevision
		selected.fields["points"] = v.Points
	case input.Kind() == api.ProjectEstimateRequestKindProjectEstimateUpdatePoint:
		v, ok := input.AsProjectEstimateUpdatePoint()
		if !ok || v == nil || v.Action != "update_point" {
			break
		}
		selected.action, selected.project, selected.target, selected.parent, selected.workspace, selected.request, selected.revision = v.Action, v.ProjectID, v.EstimateID, v.EstimatePointID, v.WorkspaceSlug, v.RequestID, v.ExpectedRevision
		text = map[string]*string{"value": v.Value, "description": v.Description, "external_source": v.ExternalSource, "external_id": v.ExternalID}
		if key, ok := v.Key.Value(); ok {
			selected.fields["key"] = key
		}
	case input.Kind() == api.ProjectEstimateRequestKindProjectEstimateDeletePoint:
		v, ok := input.AsProjectEstimateDeletePoint()
		if !ok || v == nil || v.Action != "delete_point" {
			break
		}
		selected.action, selected.project, selected.target, selected.parent, selected.workspace, selected.request, selected.revision = v.Action, v.ProjectID, v.EstimateID, v.EstimatePointID, v.WorkspaceSlug, v.RequestID, v.ExpectedRevision
	}
	if selected.action == "" {
		return selected, false, false, api.MakeInvalidParams(errors.New("invalid estimate action"))
	}
	for key, value := range text {
		if value != nil && *value != "" {
			selected.fields[key] = *value
		}
	}
	return selected, false, false, nil
}
