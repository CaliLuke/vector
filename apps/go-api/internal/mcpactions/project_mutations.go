package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) projectRecordMutation(ctx context.Context, input *api.ProjectRequest) (loom.Nullable[api.ProjectResult], error) {
	mutation, err := selectProjectMutation(input)
	if err != nil {
		return loom.Nullable[api.ProjectResult]{}, err
	}
	value, err := s.mutateResource(ctx, "project", mutation, s.projectWrites)
	if err != nil {
		return loom.Nullable[api.ProjectResult]{}, err
	}
	var record *api.ProjectDetail
	if len(value.Record) > 0 && string(value.Record) != "null" {
		record = &api.ProjectDetail{SortOrder: loom.NullValue[float64]()}
		if err := json.Unmarshal(value.Record, record); err != nil {
			return loom.Nullable[api.ProjectResult]{}, api.MakeInternalError(errors.New("invalid project mutation result"))
		}
	}
	if mutation.request == nil && mutation.revision == nil {
		if record == nil {
			return loom.NullValue[api.ProjectResult](), nil
		}
		return loom.NullableValue(api.NewProjectResultProjectDetail(record)), nil
	}
	receipt := &api.ProjectMutationReceipt{ProjectID: value.ID, Revision: value.Revision, Result: loom.NullValue[api.ProjectDetail]()}
	if record != nil {
		receipt.Result = loom.NullableValue(*record)
	}
	if mutation.request != nil {
		if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
			return loom.Nullable[api.ProjectResult]{}, api.MakeInternalError(errors.New("missing project receipt"))
		}
		id, expires := value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, value.Replayed
	}
	return loom.NullableValue(api.NewProjectResultProjectMutationReceipt(receipt)), nil
}

func selectProjectMutation(input *api.ProjectRequest) (resourceMutationInput, error) {
	selected := resourceMutationInput{fields: map[string]any{}}
	var text map[string]*string
	var flags map[string]loom.Nullable[bool]
	var archiveIn, closeIn *int
	if value, ok := input.AsProjectCreate(); ok && value != nil && value.Action == "create" {
		selected.action, selected.workspace, selected.request = value.Action, value.WorkspaceSlug, value.RequestID
		text = map[string]*string{"name": &value.Name, "identifier": &value.Identifier, "description": value.Description, "description_html": value.DescriptionHTML, "project_lead": value.ProjectLead, "default_assignee": value.DefaultAssignee, "emoji": value.Emoji, "cover_image": value.CoverImage, "timezone": value.Timezone, "external_source": value.ExternalSource, "external_id": value.ExternalID}
		flags = map[string]loom.Nullable[bool]{"module_view": value.Modules, "cycle_view": value.Cycles, "issue_views_view": value.Views, "page_view": value.Pages, "intake_view": value.Intakes, "is_issue_type_enabled": value.WorkitemTypes}
		archiveIn, closeIn = value.ArchiveIn, value.CloseIn
	} else if value, ok := input.AsProjectUpdate(); ok && value != nil && value.Action == "update" {
		selected.action, selected.project, selected.target, selected.workspace, selected.request, selected.revision = value.Action, value.ProjectID, value.ProjectID, value.WorkspaceSlug, value.RequestID, value.ExpectedRevision
		text = map[string]*string{"name": value.Name, "identifier": value.Identifier, "description": value.Description, "description_html": value.DescriptionHTML, "project_lead": value.ProjectLead, "default_assignee": value.DefaultAssignee, "emoji": value.Emoji, "cover_image": value.CoverImage, "timezone": value.Timezone, "external_source": value.ExternalSource, "external_id": value.ExternalID, "default_state": value.DefaultState, "estimate": value.Estimate}
		flags = map[string]loom.Nullable[bool]{"module_view": value.Modules, "cycle_view": value.Cycles, "issue_views_view": value.Views, "page_view": value.Pages, "intake_view": value.Intakes, "is_issue_type_enabled": value.WorkitemTypes, "is_time_tracking_enabled": value.IsTimeTrackingEnabled}
		if network, ok := value.Network.Value(); ok {
			selected.fields["network"] = network
		}
		archiveIn, closeIn = value.ArchiveIn, value.CloseIn
	} else if value, ok := input.AsProjectDelete(); ok && value != nil && value.Action == "delete" {
		selected.action, selected.project, selected.target, selected.workspace, selected.request, selected.revision = value.Action, value.ProjectID, value.ProjectID, value.WorkspaceSlug, value.RequestID, value.ExpectedRevision
	} else {
		return selected, api.MakeInvalidParams(errors.New("invalid project mutation"))
	}
	for key, value := range text {
		if value != nil && *value != "" {
			selected.fields[key] = *value
		}
	}
	for key, value := range flags {
		if flag, ok := value.Value(); ok {
			selected.fields[key] = flag
		}
	}
	for key, value := range map[string]*int{"archive_in": archiveIn, "close_in": closeIn} {
		if value != nil && *value != 0 {
			selected.fields[key] = *value
		}
	}
	if selected.fields["description_html"] == nil {
		if plain, ok := selected.fields["description"].(string); ok {
			escape := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#x27;", "\n", "<br/>")
			selected.fields["description_html"] = "<p>" + escape.Replace(plain) + "</p>"
		}
	}
	return selected, nil
}
