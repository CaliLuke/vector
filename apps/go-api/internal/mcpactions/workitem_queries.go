package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
)

func (s *Service) readWorkitem(ctx context.Context, input *api.WorkitemRequest) (loom.Nullable[api.WorkitemResult], error) {
	var payload any
	if v, ok := input.AsWorkitemCount(); ok {
		payload = v
	} else if v, ok := input.AsWorkitemList(); ok {
		payload = v
	} else if v, ok := input.AsWorkitemListArchived(); ok {
		payload = v
	} else if v, ok := input.AsWorkitemRetrieve(); ok {
		payload = v
	} else if v, ok := input.AsWorkitemIdentifier(); ok {
		payload = v
	} else if v, ok := input.AsWorkitemSearch(); ok {
		payload = v
	}
	value, action, err := s.queryWorkitems(ctx, "workitem", payload)
	if err != nil {
		return loom.Nullable[api.WorkitemResult]{}, err
	}
	if failure, ok := value.(*api.WorkitemPQLFailure); ok {
		return loom.NullableValue(api.NewWorkitemResultWorkitemPQLFailure(failure)), nil
	}
	switch action {
	case "count":
		result := &api.WorkitemCountResult{}
		decodeWorkitemProjectionErr := decodeWorkitemProjection(value, result)
		return loom.NullableValue(api.NewWorkitemResultWorkitemCountResult(result)), decodeWorkitemProjectionErr
	case "list", "list_archived":
		page, workitemPageProjectionErr := workitemPageProjection(value)
		return loom.NullableValue(api.NewWorkitemResultWorkitemPage(page)), workitemPageProjectionErr
	case "search":
		result := &api.WorkitemSearchResult{}
		if err = decodeWorkitemProjection(value, result); err != nil {
			return loom.Nullable[api.WorkitemResult]{}, err
		}
		return loom.NullableValue(api.NewWorkitemResultWorkitemSearchResult(result)), nil
	default:
		record, err := workitemReadProjection(value, false)
		return loom.NullableValue(api.NewWorkitemResultWorkitemReadRecord(record)), err
	}
}

// Generated action variants supply the closed input contract. This shared
// selector only translates their common query fields into the native request.
func (s *Service) queryWorkitems(ctx context.Context, tool string, payload any) (any, string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, "", workitemResultError()
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return nil, "", api.MakeInvalidParams(errors.New("invalid work item query"))
	}
	params, present := map[string]string{}, map[string]bool{}
	for _, key := range []string{"group_by", "sub_group_by", "action", "project_id", "workitem_id", "workitem_identifier", "query", "workspace_slug", "cycle_id", "module_id", "pql", "order_by", "cursor", "expand", "fields", "external_id", "external_source"} {
		var value string
		if field, ok := fields[key]; ok {
			if json.Unmarshal(field, &value) != nil {
				return nil, "", api.MakeInvalidParams(errors.New("invalid work item query"))
			}
			if value != "" {
				params[key], present[key] = value, true
			}
		}
	}
	if raw, ok := fields["per_page"]; ok {
		var limit int
		if json.Unmarshal(raw, &limit) != nil || limit < 0 || limit > 1000 {
			return nil, "", api.MakeInvalidParams(errors.New("invalid page size"))
		}
		if limit != 0 {
			params["per_page"], present["per_page"] = strconv.Itoa(limit), true
		}
	}
	var revision bool
	if raw, ok := fields["include_revision"]; ok && json.Unmarshal(raw, &revision) != nil {
		return nil, "", api.MakeInvalidParams(errors.New("invalid revision selector"))
	}
	if revision {
		params["include_revision"] = "true"
	}
	action := params["action"]
	projectRequired := action == "retrieve" || action == "list_archived" || action == "list_workitems"
	for _, key := range []string{"project_id", "workitem_id", "cycle_id", "module_id"} {
		value := params[key]
		if value != "" {
			if id, parseErr := uuid.Parse(value); parseErr != nil || id == uuid.Nil {
				return nil, action, api.MakeInvalidParams(errors.New("invalid work item scope"))
			}
		}
	}
	if projectRequired && params["project_id"] == "" || action == "retrieve" && params["workitem_id"] == "" || action == "search" && params["query"] == "" {
		return nil, action, api.MakeInvalidParams(errors.New("work item query target is required"))
	}
	workspace := params["workspace_slug"]
	var selector *string
	if workspace != "" {
		selector = &workspace
	}
	ctx, actor, err := readActor(ctx, selector)
	if err != nil {
		return nil, action, err
	}
	if actor.WorkspaceSlug == "" {
		return nil, action, api.MakeInvalidParams(errors.New("workspace is required"))
	}
	operation := action
	if tool == "cycle" || tool == "module" {
		if action != "list_workitems" || params[tool+"_id"] == "" {
			return nil, action, api.MakeInvalidParams(errors.New("invalid parent item query"))
		}
		operation = tool + "_items"
		params["parent_id"] = params[tool+"_id"]
	}
	params["identifier"], params["search"] = params["workitem_identifier"], params["query"]
	params["fields"] = normalizeWorkitemFields(params["fields"])
	result, err := s.collaborationReads.ReadWorkitem(ctx, workitems.Request{ExpectedWorkspaceID: &actor.WorkspaceID, API: true, Method: "GET", Operation: operation, ActorID: actor.UserID.String(), Slug: actor.WorkspaceSlug, ProjectID: params["project_id"], IssueID: params["workitem_id"], Timezone: actor.Timezone, Params: params, Present: present})
	if err != nil {
		var failure *djangoapi.HTTPError
		if params["pql"] != "" && errors.As(err, &failure) && failure.Status == 400 {
			if body := failure.Body; body["pql"] != nil {
				projection := &api.WorkitemPQLFailure{}
				decodeWorkitemProjectionErr2 := decodeWorkitemProjection(map[string]any{"error": body["pql"], "failed_pql": params["pql"], "pql_reference": pqlFull, "hint": fmt.Sprintf("The PQL above failed. Fix it using the reference and retry %s %s.", tool, action)}, projection)
				return projection, action, decodeWorkitemProjectionErr2
			}
		}
		return nil, action, resourceReadError(err, nil, nil)
	}
	value := djangoapi.Timestamps(result.Body, actor.Timezone)
	if action == "list" || action == "list_archived" || action == "list_workitems" {
		page, ok := value.(map[string]any)
		if !ok {
			return nil, action, workitemResultError()
		}
		rows, ok := page["results"].([]any)
		if !ok {
			return nil, action, workitemResultError()
		}
		for i, row := range rows {
			projected, err := workitemReadProjection(row, params["fields"] != "")
			if err != nil {
				return nil, action, err
			}
			rows[i] = projected
		}
	}
	return value, action, nil
}
func normalizeWorkitemFields(raw string) string {
	out := []string{}
	for _, field := range strings.Split(raw, ",") {
		if field = strings.TrimSpace(field); field != "" {
			out = append(out, field)
		}
	}
	return strings.Join(out, ",")
}
func workitemReadProjection(value any, sparse bool) (*api.WorkitemReadRecord, error) {
	row, ok := value.(map[string]any)
	if !ok {
		return nil, workitemResultError()
	}
	if !sparse {
		for _, key := range []string{"id", "type_id", "created_at", "updated_at", "deleted_at", "name", "description_html", "description_stripped", "description_binary", "priority", "start_date", "target_date", "completed_at", "archived_at", "external_source", "external_id", "created_by", "updated_by", "project", "workspace", "parent", "state", "estimate_point", "type", "point", "sequence_id", "sort_order", "is_draft"} {
			if _, ok := row[key]; !ok {
				row[key] = nil
			}
		}
		for _, key := range []string{"assignees", "labels"} {
			if row[key] == nil {
				row[key] = []any{}
			}
		}
	}
	record := &api.WorkitemReadRecord{}
	return record, decodeWorkitemProjection(row, record)
}
func workitemPageProjection(value any) (*api.WorkitemPage, error) {
	page := &api.WorkitemPage{}
	return page, decodeWorkitemProjection(value, page)
}
func decodeWorkitemProjection(value, target any) error {
	raw, err := json.Marshal(value)
	if err != nil || json.Unmarshal(raw, target) != nil {
		return workitemResultError()
	}
	return nil
}
