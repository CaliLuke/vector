package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"uuid"

	cycles "github.com/CaliLuke/go-plane/internal/cyclereads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func (s *Service) Cycle(ctx context.Context, input *api.CycleRequest) (*api.CycleResult, error) {
	return readProjectResource(ctx, input, selectCycleAction, s.cycleReads, cycles.ErrForbidden, cycles.ErrNotFound, projectCycleResult)
}

func selectCycleAction(input *api.CycleRequest) (cycles.APIV1Request, *string, error) {
	request := cycles.APIV1Request{Params: map[string]string{}, Present: map[string]bool{}}
	var workspace *string
	valid := false
	if input != nil {
		if list, ok := input.AsCycleList(); ok && list != nil && list.Action == "list" {
			valid = true
			request.Operation = "lite"
			request.Params["project_id"], workspace = list.ProjectID, list.WorkspaceSlug
			if list.Archived != nil && *list.Archived {
				request.Operation = "archived"
			}
			for key, value := range map[string]*string{"cursor": list.Cursor, "order_by": list.OrderBy, "status": list.Status} {
				if value != nil && *value != "" {
					request.Params[key], request.Present[key] = *value, true
				}
			}
			if request.Operation == "lite" {
				switch request.Params["status"] {
				case "", "current", "upcoming", "completed", "draft", "incomplete":
				default:
					valid = false
				}
			}
			if list.PerPage != nil && *list.PerPage != 0 {
				request.Params["per_page"], request.Present["per_page"] = strconv.Itoa(*list.PerPage), true
				valid = valid && *list.PerPage > 0 && *list.PerPage <= 1000
			}
		} else if detail, ok := input.AsCycleRetrieve(); ok && detail != nil && detail.Action == "retrieve" {
			request.Operation = "detail"
			request.Params["project_id"], request.Params["pk"], workspace = detail.ProjectID, detail.CycleID, detail.WorkspaceSlug
			_, err := uuid.Parse(detail.CycleID)
			valid = err == nil
		}
	}
	if _, err := uuid.Parse(request.Params["project_id"]); err != nil {
		valid = false
	}
	if !valid {
		return cycles.APIV1Request{}, nil, api.MakeInvalidParams(errors.New("invalid cycle action"))
	}
	return request, workspace, nil
}

var cycleNullableFields = []string{"id", "created_at", "updated_at", "deleted_at", "description", "start_date", "end_date", "external_source", "external_id", "archived_at", "timezone", "created_by", "updated_by", "project", "workspace", "owned_by", "view_props", "progress_snapshot", "logo_props", "sort_order", "version"}
var cycleMetricFields = []string{"total_issues", "cancelled_issues", "completed_issues", "started_issues", "unstarted_issues", "backlog_issues", "total_estimates", "completed_estimates", "started_estimates"}

func projectCycleResult(value any, operation string) (*api.CycleResult, error) {
	return projectReadResult(value, operation == "detail", func(row map[string]any) (*api.CycleRecord, error) {
		return projectCycleRecord(row, operation == "lite")
	}, api.NewCycleResultCycleRecord, func(page *api.CyclePage, records []*api.CycleRecord) api.CycleResult {
		page.Results = records
		return api.NewCycleResultCyclePage(page)
	})
}

func projectCycleRecord(row map[string]any, lite bool) (*api.CycleRecord, error) {
	name, ok := row["name"].(string)
	if !ok {
		return nil, errors.New("cycle name missing")
	}
	wire := map[string]any{"name": name}
	for _, field := range cycleNullableFields {
		wire[field] = row[field]
	}
	if lite {
		for _, field := range []string{"project", "workspace", "owned_by"} {
			if _, ok := wire[field].(string); !ok {
				return nil, errors.New("cycle scope missing")
			}
		}
	} else {
		for _, field := range cycleMetricFields {
			value, err := sdkInteger(row[field])
			if err != nil {
				return nil, err
			}
			wire[field] = value
		}
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	record := &api.CycleRecord{}
	if err = json.Unmarshal(body, record); err != nil {
		return nil, err
	}
	return record, nil
}
