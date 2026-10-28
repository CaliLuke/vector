package mcpactions

import (
	"encoding/json"
	"errors"

	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	states "github.com/CaliLuke/go-plane/internal/projectstatereads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func projectStateResult(value any, operation string) (*api.StateResult, error) {
	return projectReadResult(value, operation == "detail", projectStateRecord, api.NewStateResultStateRecord,
		func(page *api.StatePage, records []*api.StateRecord) api.StateResult {
			page.Results = records
			return api.NewStateResultStatePage(page)
		})
}
func projectStateRecord(row map[string]any) (*api.StateRecord, error) {
	if _, ok := row["name"].(string); !ok {
		return nil, errors.New("state name missing")
	}
	if _, ok := row["color"].(string); !ok {
		return nil, errors.New("state color missing")
	}
	wire := map[string]any{}
	for _, key := range []string{"name", "color", "id", "created_at", "updated_at", "deleted_at", "description", "sequence", "group", "is_triage", "default", "external_source", "external_id", "created_by", "updated_by", "project", "workspace"} {
		wire[key] = row[key]
	}
	for _, key := range []string{"slug", "revision"} {
		if value, ok := row[key]; ok {
			wire[key] = value
		}
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	record := &api.StateRecord{}
	if err = json.Unmarshal(body, record); err != nil {
		return nil, err
	}
	return record, nil
}
func stateMutationResult(input stateInput, value states.MutationResult, timezone string) (*api.StateResult, error) {
	var record *api.StateRecord
	if input.action != "delete" {
		var row map[string]any
		if json.Unmarshal(value.Result, &row) != nil || row == nil {
			return nil, stateError(nil)
		}
		var err error
		record, err = projectStateRecord(djangoapi.Timestamps(row, timezone).(map[string]any))
		if err != nil {
			return nil, stateError(nil)
		}
	}
	if input.request == nil && input.revision == nil {
		if record == nil {
			return nil, nil
		}
		result := api.NewStateResultStateRecord(record)
		return &result, nil
	}
	if _, err := stateUUID(value.StateID); err != nil || value.Revision == "" {
		return nil, stateError(nil)
	}
	receipt := &api.StateMutationReceipt{StateID: value.StateID, Revision: value.Revision, Result: loom.NullValue[api.StateRecord]()}
	if record != nil {
		receipt.Result = loom.NullableValue(*record)
	}
	if input.request != nil {
		if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
			return nil, stateError(nil)
		}
		id, expires := value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, value.Replayed
	}
	result := api.NewStateResultStateMutationReceipt(receipt)
	return &result, nil
}
