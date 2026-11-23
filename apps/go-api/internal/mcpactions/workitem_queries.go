package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/workitems"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
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
	params, present, err := workitemQueryParameters(payload)
	if err != nil {
		return nil, "", err
	}
	action := params["action"]
	compact := action == "list" && params["format"] == "compact" && params["fields"] == "" && params["expand"] == ""
	if compact {
		params["fields"], params["expand"] = "id,name,sequence_id,project,state,priority,is_draft", "project,state"
		if !present["per_page"] {
			cursor := params["cursor"]
			size, sizeErr := compactPageSize(&cursor)
			if sizeErr != nil {
				return nil, action, sizeErr
			}
			params["per_page"], present["per_page"] = strconv.Itoa(size), true
		}
	}
	if scopeErr := validateWorkitemQueryScope(params); scopeErr != nil {
		return nil, action, scopeErr
	}
	workspace := params["workspace_slug"]
	var selector *string
	if workspace != "" {
		selector = &workspace
	}
	ctx, actor, err := s.scopedReadActor(ctx, selector)
	if err != nil {
		return nil, action, err
	}
	if action == "list" || action == "count" {
		project, projectErr := s.resolveWorkitemProject(ctx, actor, params["project_id"])
		if projectErr != nil {
			return nil, action, projectErr
		}
		params["project_id"] = project
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
		projection, queryErr := workitemQueryError(err, tool, action, params["pql"], compact)
		return projection, action, queryErr
	}
	value := apicontract.Timestamps(result.Body, actor.Timezone)
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
func workitemQueryParameters(payload any) (map[string]string, map[string]bool, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, workitemResultError()
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return nil, nil, api.MakeInvalidParams(errors.New("invalid work item query"))
	}
	params, present := map[string]string{}, map[string]bool{}
	for _, key := range []string{"group_by", "sub_group_by", "action", "format", "project_id", "workitem_id", "workitem_identifier", "query", "workspace_slug", "cycle_id", "module_id", "pql", "order_by", "cursor", "expand", "fields", "external_id", "external_source"} {
		var value string
		if field, ok := fields[key]; ok {
			if json.Unmarshal(field, &value) != nil {
				return nil, nil, api.MakeInvalidParams(errors.New("invalid work item query"))
			}
			if value != "" {
				params[key], present[key] = value, true
			}
		}
	}
	if raw, ok := fields["per_page"]; ok {
		var limit int
		if json.Unmarshal(raw, &limit) != nil || limit < 0 || limit > 1000 {
			return nil, nil, api.MakeInvalidParams(errors.New("invalid page size"))
		}
		if limit != 0 {
			params["per_page"], present["per_page"] = strconv.Itoa(limit), true
		}
	}
	var revision bool
	if raw, ok := fields["include_revision"]; ok && json.Unmarshal(raw, &revision) != nil {
		return nil, nil, api.MakeInvalidParams(errors.New("invalid revision selector"))
	}
	if revision {
		params["include_revision"] = "true"
	}
	return params, present, nil
}

func validateWorkitemQueryScope(params map[string]string) error {
	action := params["action"]
	projectRequired := action == "retrieve" || action == "list_archived" || action == "list_workitems"
	for _, key := range []string{"project_id", "workitem_id", "cycle_id", "module_id"} {
		value := params[key]
		if key == "project_id" && (action == "list" || action == "count") {
			continue // Resolved after authenticating the workspace.
		}
		if value != "" {
			if id, parseErr := uuid.Parse(value); parseErr != nil || id == uuid.Nil {
				return api.MakeInvalidParams(errors.New("invalid work item scope"))
			}
		}
	}
	if projectRequired && params["project_id"] == "" || action == "retrieve" && params["workitem_id"] == "" || action == "search" && params["query"] == "" {
		return api.MakeInvalidParams(errors.New("work item query target is required"))
	}
	return nil
}

func workitemQueryError(err error, tool, action, pql string, compact bool) (any, error) {
	var failure *apicontract.HTTPError
	if pql != "" && errors.As(err, &failure) && failure.Status == 400 {
		if body := failure.Body; body["pql"] != nil {
			if problem, ok := body["pql"].(string); compact && ok {
				return nil, loom.WithErrorRemedy(api.MakeInvalidParams(errors.New("invalid PQL")), &loom.ErrorRemedy{
					Code: "invalid_pql", SafeMessage: problem,
					RetryHint: "Correct the filter and retry. Use get_pql_reference with detail=brief for syntax.",
				})
			}
			projection := &api.WorkitemPQLFailure{}
			decodeWorkitemProjectionErr2 := decodeWorkitemProjection(map[string]any{"error": body["pql"], "failed_pql": pql, "pql_reference": pqlFull, "hint": fmt.Sprintf("The PQL above failed. Fix it using the reference and retry %s %s.", tool, action)}, projection)
			return projection, decodeWorkitemProjectionErr2
		}
	}
	return nil, resourceReadError(err, nil, nil)
}

func (s *Service) resolveWorkitemProject(ctx context.Context, actor mcpauth.Principal, reference string) (string, error) {
	if reference == "" {
		return "", nil
	}
	if id, err := uuid.Parse(reference); err == nil {
		if id == uuid.Nil {
			return "", api.MakeInvalidParams(errors.New("invalid work item scope"))
		}
		return id.String(), nil
	}
	body, err := s.readProjectReference(ctx, actor, reference, projects.APIProjectDetailOptions{Fields: "id"})
	if err != nil {
		return "", err
	}
	var record struct{ ID string }
	if json.Unmarshal(body, &record) != nil || record.ID == "" {
		return "", workitemResultError()
	}
	return record.ID, nil
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
