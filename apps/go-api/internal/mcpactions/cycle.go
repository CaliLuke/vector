package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"uuid"

	cycles "github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

type cycleReader interface {
	ReadAPIV1(context.Context, string, cycles.APIV1Request) (any, error)
}

func (s *Service) Cycle(ctx context.Context, input *api.CycleRequest) (*api.CycleResult, error) {
	request, workspace, err := selectCycleAction(input)
	if err != nil {
		return nil, err
	}
	ctx, actor, err := readActor(ctx, workspace)
	if err != nil {
		return nil, err
	}
	if actor.WorkspaceSlug == "" {
		return nil, api.MakeInvalidParams(errors.New("workspace is required"))
	}
	workspaceID := uuid.UUID(actor.WorkspaceID)
	request.ExpectedWorkspaceID = &workspaceID
	request.Params["slug"] = actor.WorkspaceSlug
	value, err := s.cycleReads.ReadAPIV1(ctx, actor.UserID.String(), request)
	if err != nil {
		return nil, cycleReadError(err)
	}
	result, err := projectCycleResult(djangoapi.Timestamps(value, actor.Timezone), request.Operation)
	if err != nil {
		return nil, api.MakeInternalError(errors.New("cycle projection unavailable"))
	}
	return result, nil
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

func cycleReadError(err error) error {
	switch {
	case errors.Is(err, cycles.ErrForbidden):
		return api.MakeForbidden(errors.New("cycle access denied"))
	case errors.Is(err, cycles.ErrNotFound):
		return api.MakeNotFound(errors.New("cycle unavailable"))
	}
	var failure *djangoapi.HTTPError
	if errors.As(err, &failure) && failure.Status == 400 {
		return api.MakeInvalidParams(errors.New("invalid cycle query"))
	}
	return api.MakeInternalError(errors.New("cycle read unavailable"))
}

var cycleNullableFields = []string{"id", "created_at", "updated_at", "deleted_at", "description", "start_date", "end_date", "external_source", "external_id", "archived_at", "timezone", "created_by", "updated_by", "project", "workspace", "owned_by", "view_props", "progress_snapshot", "logo_props", "sort_order", "version"}
var cycleMetricFields = []string{"total_issues", "cancelled_issues", "completed_issues", "started_issues", "unstarted_issues", "backlog_issues", "total_estimates", "completed_estimates", "started_estimates"}

func projectCycleResult(value any, operation string) (*api.CycleResult, error) {
	document, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("invalid cycle document")
	}
	if operation == "detail" {
		record, err := projectCycleRecord(document, false)
		if err != nil {
			return nil, err
		}
		result := api.NewCycleResultCycleRecord(record)
		return &result, nil
	}
	rows, ok := document["results"].([]any)
	if !ok {
		return nil, errors.New("invalid cycle page")
	}
	// Decode pagination separately from entries so SDK integer coercion is exact.
	envelope := make(map[string]any, len(document))
	for key, value := range document {
		if key != "results" {
			envelope[key] = value
		}
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	page := &api.CyclePage{Results: make([]*api.CycleRecord, 0, len(rows))}
	if err = json.Unmarshal(body, page); err != nil {
		return nil, err
	}
	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		if !ok {
			return nil, errors.New("invalid cycle row")
		}
		record, err := projectCycleRecord(row, operation == "lite")
		if err != nil {
			return nil, err
		}
		page.Results = append(page.Results, record)
	}
	result := api.NewCycleResultCyclePage(page)
	return &result, nil
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
			value, err := cycleMetric(row[field])
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

// The HTTP serializer emits integral estimates as e.g. 3.0; the pinned SDK
// coerces them to integers. Preserve precision rather than converting via float.
func cycleMetric(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	number, ok := new(big.Rat).SetString(string(raw))
	if !ok || !number.IsInt() || !number.Num().IsInt64() {
		return nil, errors.New("invalid cycle metric")
	}
	return number.Num().Int64(), nil
}
