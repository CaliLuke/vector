package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"uuid"

	modules "github.com/CaliLuke/go-plane/internal/modulereads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func (s *Service) moduleRead(ctx context.Context, input *api.ModuleRequest) (*api.ModuleResult, error) {
	if input != nil {
		if items, ok := input.AsModuleListWorkitems(); ok {
			return planningWorkitems(ctx, s, "module", items, api.NewModuleResultWorkitemPQLFailure, api.NewModuleResultWorkitemPage)
		}
	}
	return readProjectResource(ctx, input, selectModuleAction, s.moduleReads, modules.ErrForbidden, modules.ErrNotFound, projectModuleResult)
}

func selectModuleAction(input *api.ModuleRequest) (modules.APIV1Request, *string, error) {
	request := modules.APIV1Request{Params: map[string]string{}, Present: map[string]bool{}}
	var workspace *string
	valid := false
	if input != nil {
		if list, ok := input.AsModuleList(); ok && list != nil && list.Action == "list" {
			valid = true
			request.Operation = "lite"
			request.Params["project_id"], workspace = list.ProjectID, list.WorkspaceSlug
			if list.Archived != nil && *list.Archived {
				request.Operation = "archived"
			}
			for key, value := range map[string]*string{"cursor": list.Cursor, "order_by": list.OrderBy} {
				if value != nil && *value != "" {
					request.Params[key], request.Present[key] = *value, true
				}
			}
			if list.PerPage != nil && *list.PerPage != 0 {
				request.Params["per_page"], request.Present["per_page"] = strconv.Itoa(*list.PerPage), true
				valid = valid && *list.PerPage > 0 && *list.PerPage <= 1000
			}
		} else if detail, ok := input.AsModuleRetrieve(); ok && detail != nil && detail.Action == "retrieve" {
			request.Operation = "detail"
			request.Params["project_id"], request.Params["pk"], workspace = detail.ProjectID, detail.ModuleID, detail.WorkspaceSlug
			if detail.IncludeRevision != nil && *detail.IncludeRevision {
				request.Params["include_revision"], request.Present["include_revision"] = "true", true
			}
			_, err := uuid.Parse(detail.ModuleID)
			valid = err == nil
		}
	}
	if _, err := uuid.Parse(request.Params["project_id"]); err != nil {
		valid = false
	}
	if !valid {
		return modules.APIV1Request{}, nil, api.MakeInvalidParams(errors.New("invalid module action"))
	}
	return request, workspace, nil
}

var moduleNullableFields = []string{"id", "created_at", "updated_at", "deleted_at", "description", "description_text", "description_html", "start_date", "target_date", "status", "view_props", "sort_order", "external_source", "external_id", "archived_at", "logo_props", "created_by", "updated_by", "project", "workspace", "lead", "members"}
var moduleMetricFields = []string{"total_issues", "cancelled_issues", "completed_issues", "started_issues", "unstarted_issues", "backlog_issues"}

func projectModuleResult(value any, operation string) (*api.ModuleResult, error) {
	return projectReadResult(value, operation == "detail", func(row map[string]any) (*api.ModuleRecord, error) {
		return projectModuleRecord(row, operation == "lite")
	}, api.NewModuleResultModuleRecord, func(page *api.ModulePage, records []*api.ModuleRecord) api.ModuleResult {
		page.Results = records
		return api.NewModuleResultModulePage(page)
	})
}

func projectModuleRecord(row map[string]any, lite bool) (*api.ModuleRecord, error) {
	name, ok := row["name"].(string)
	if !ok {
		return nil, errors.New("module name missing")
	}
	if status := row["status"]; status != nil {
		switch status {
		case "backlog", "planned", "in-progress", "paused", "completed", "cancelled":
		default:
			return nil, errors.New("invalid module status")
		}
	}
	wire := map[string]any{"name": name}
	for _, field := range moduleNullableFields {
		wire[field] = row[field]
	}
	if lite {
		for _, field := range []string{"project", "workspace"} {
			if _, ok := wire[field].(string); !ok {
				return nil, errors.New("module scope missing")
			}
		}
	} else {
		for _, field := range moduleMetricFields {
			value, err := sdkInteger(row[field])
			if err != nil {
				return nil, err
			}
			wire[field] = value
		}
	}
	if revision, ok := row["revision"]; ok {
		wire["revision"] = revision
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	record := &api.ModuleRecord{}
	if err = json.Unmarshal(body, record); err != nil {
		return nil, err
	}
	return record, nil
}
