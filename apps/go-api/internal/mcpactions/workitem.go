package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) Workitem(ctx context.Context, input *api.WorkitemRequest) (loom.Nullable[api.WorkitemResult], error) {
	if input != nil {
		switch input.Kind() {
		case api.WorkitemRequestKindWorkitemCount, api.WorkitemRequestKindWorkitemList, api.WorkitemRequestKindWorkitemListArchived, api.WorkitemRequestKindWorkitemRetrieve, api.WorkitemRequestKindWorkitemIdentifier, api.WorkitemRequestKindWorkitemSearch:
			return s.readWorkitem(ctx, input)
		}
	}
	selected, err := selectWorkitemMutation(input)
	if err != nil {
		return loom.Nullable[api.WorkitemResult]{}, err
	}
	value, err := s.mutateResource(ctx, "workitem", selected, resourceMutationFunc(s.collaborationReads.MutateWorkitem))
	if err != nil {
		return loom.Nullable[api.WorkitemResult]{}, err
	}
	var record *api.WorkitemRecord
	var archive *api.WorkitemArchiveStatus
	if selected.action == "archive" {
		if json.Unmarshal(value.Record, &archive) != nil || archive == nil {
			return loom.Nullable[api.WorkitemResult]{}, workitemResultError()
		}
	} else if selected.action != "delete" {
		record, err = projectWorkitemRecord(value.Record)
		if err != nil {
			return loom.Nullable[api.WorkitemResult]{}, err
		}
	}
	if selected.request == nil && selected.revision == nil {
		if archive != nil {
			return loom.NullableValue(api.NewWorkitemResultWorkitemArchiveStatus(archive)), nil
		}
		if record == nil {
			return loom.NullValue[api.WorkitemResult](), nil
		}
		projected := &api.WorkitemReadRecord{}
		if err := decodeWorkitemProjection(record, projected); err != nil {
			return loom.Nullable[api.WorkitemResult]{}, err
		}
		return loom.NullableValue(api.NewWorkitemResultWorkitemReadRecord(projected)), nil
	}
	receipt := &api.WorkitemMutationReceipt{WorkitemID: value.ID, Revision: value.Revision, Result: loom.NullValue[api.WorkitemRecord]()}
	if record != nil {
		receipt.Result = loom.NullableValue(*record)
	}
	if archive != nil {
		receipt.Archived = &archive.Archived
	}
	if selected.request != nil {
		if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
			return loom.Nullable[api.WorkitemResult]{}, workitemResultError()
		}
		id, expires := value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, value.Replayed
	}
	return loom.NullableValue(api.NewWorkitemResultWorkitemMutationReceipt(receipt)), nil
}
func projectWorkitemRecord(raw json.RawMessage) (*api.WorkitemRecord, error) {
	var row map[string]any
	if json.Unmarshal(raw, &row) != nil || row == nil {
		return nil, workitemResultError()
	}
	// SDK WorkItem defaults all omitted scalar fields to null. Native extras are
	// explicitly projected, including its unexpanded assignment and property data.
	for _, key := range []string{"id", "type_id", "created_at", "updated_at", "deleted_at", "name", "description_html", "description_stripped", "description_binary", "priority", "start_date", "target_date", "completed_at", "archived_at", "external_source", "external_id", "created_by", "updated_by", "project", "workspace", "parent", "state", "estimate_point", "type", "point", "sequence_id", "sort_order", "is_draft"} {
		if _, ok := row[key]; !ok {
			row[key] = nil
		}
	}
	for _, key := range []string{"assignees", "labels"} {
		if row[key] == nil {
			row[key] = []string{}
		}
	}
	raw, err := json.Marshal(djangoapi.Timestamps(row, "UTC"))
	if err != nil {
		return nil, workitemResultError()
	}
	var record api.WorkitemRecord
	if json.Unmarshal(raw, &record) != nil {
		return nil, workitemResultError()
	}
	return &record, nil
}
func workitemResultError() error {
	return api.MakeInternalError(errors.New("invalid work item result"))
}
