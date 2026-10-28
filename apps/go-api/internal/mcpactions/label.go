package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"uuid"

	labels "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

func (s *Service) labelRead(ctx context.Context, input *api.LabelRequest) (*api.LabelResult, error) {
	return readProjectResource(ctx, input, selectLabelAction, s.labelReads, labels.ErrForbidden, labels.ErrNotFound, projectLabelResult)
}

func selectLabelAction(input *api.LabelRequest) (labels.APIV1Request, *string, error) {
	request := labels.APIV1Request{Params: map[string]string{}, Present: map[string]bool{}}
	var workspace *string
	valid := false
	if input != nil {
		if list, ok := input.AsLabelList(); ok && list != nil && list.Action == "list" {
			valid = true
			request.Operation = "list"
			request.Params["project_id"], workspace = list.ProjectID, list.WorkspaceSlug
			for key, value := range map[string]*string{"cursor": list.Cursor} {
				if value != nil && *value != "" {
					request.Params[key], request.Present[key] = *value, true
				}
			}
			if list.PerPage != nil && *list.PerPage != 0 {
				request.Params["per_page"], request.Present["per_page"] = strconv.Itoa(*list.PerPage), true
				valid = valid && *list.PerPage > 0 && *list.PerPage <= 1000
			}
		} else if detail, ok := input.AsLabelRetrieve(); ok && detail != nil && detail.Action == "retrieve" {
			request.Operation = "detail"
			request.Params["project_id"], request.Params["pk"], workspace = detail.ProjectID, detail.LabelID, detail.WorkspaceSlug
			if detail.IncludeRevision != nil && *detail.IncludeRevision {
				request.Params["include_revision"], request.Present["include_revision"] = "true", true
			}
			_, err := uuid.Parse(detail.LabelID)
			valid = err == nil
		}
	}
	if _, err := uuid.Parse(request.Params["project_id"]); err != nil {
		valid = false
	}
	if !valid {
		return labels.APIV1Request{}, nil, api.MakeInvalidParams(errors.New("invalid label action"))
	}
	return request, workspace, nil
}

var labelNullableFields = []string{"id", "created_at", "updated_at", "deleted_at", "description", "color", "sort_order", "external_source", "external_id", "created_by", "updated_by", "workspace", "project", "parent"}

func projectLabelResult(value any, operation string) (*api.LabelResult, error) {
	return projectReadResult(value, operation == "detail", func(row map[string]any) (*api.LabelRecord, error) {
		return projectLabelRecord(row)
	}, api.NewLabelResultLabelRecord, func(page *api.LabelPage, records []*api.LabelRecord) api.LabelResult {
		page.Results = records
		return api.NewLabelResultLabelPage(page)
	})
}

func projectLabelRecord(row map[string]any) (*api.LabelRecord, error) {
	name, ok := row["name"].(string)
	if !ok {
		return nil, errors.New("label name missing")
	}
	wire := map[string]any{"name": name}
	for _, field := range labelNullableFields {
		wire[field] = row[field]
	}
	if revision, ok := row["revision"]; ok {
		wire["revision"] = revision
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	record := &api.LabelRecord{}
	if err = json.Unmarshal(body, record); err != nil {
		return nil, err
	}
	return record, nil
}
