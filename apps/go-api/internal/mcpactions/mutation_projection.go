package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func mutationRecord[T any](raw json.RawMessage, project func(map[string]any) (*T, error)) (*T, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var row map[string]any
	if json.Unmarshal(raw, &row) != nil {
		return nil, api.MakeInternalError(errors.New("invalid mutation result"))
	}
	record, err := project(row)
	if err != nil {
		return nil, api.MakeInternalError(errors.New("invalid mutation result"))
	}
	return record, nil
}

func mutationReceiptMetadata(value projectmutations.Result) (*string, *string, *bool, error) {
	if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
		return nil, nil, nil, api.MakeInternalError(errors.New("missing receipt"))
	}
	id, expires := value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
	return &id, &expires, value.Replayed, nil
}

func decodeRecord[T any](wire map[string]any) (*T, error) {
	record := new(T)
	raw, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, record); err != nil {
		return nil, err
	}
	return record, nil
}

type resourceReceipt[T any] struct {
	ID, Revision               string
	Result                     loom.Nullable[T]
	RequestID, ReplayExpiresAt *string
	Replayed                   *bool
}

func mutationResponse[T, R any](input resourceMutationInput, value projectmutations.Result, record *T, project func(*T) R, receiptResult func(resourceReceipt[T]) R) (loom.Nullable[R], error) {
	if input.request == nil && input.revision == nil {
		if record == nil {
			return loom.NullValue[R](), nil
		}
		return loom.NullableValue(project(record)), nil
	}
	receipt := resourceReceipt[T]{ID: value.ID, Revision: value.Revision, Result: loom.NullValue[T]()}
	if record != nil {
		receipt.Result = loom.NullableValue(*record)
	}
	if input.request != nil {
		var err error
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed, err = mutationReceiptMetadata(value)
		if err != nil {
			return loom.Nullable[R]{}, err
		}
	}
	return loom.NullableValue(receiptResult(receipt)), nil
}

func nullableRead[T any](result *T, err error) (loom.Nullable[T], error) {
	if err != nil {
		return loom.Nullable[T]{}, err
	}
	return loom.NullableValue(*result), nil
}

func planningWorkitems[R any](ctx context.Context, s *Service, tool string, input any, failureResult func(*api.WorkitemPQLFailure) R, pageResult func(*api.WorkitemPage) R) (*R, error) {
	value, _, err := s.queryWorkitems(ctx, tool, input)
	if err != nil {
		return nil, err
	}
	if failure, ok := value.(*api.WorkitemPQLFailure); ok {
		result := failureResult(failure)
		return &result, nil
	}
	page, err := workitemPageProjection(value)
	if err != nil {
		return nil, err
	}
	result := pageResult(page)
	return &result, nil
}

type resourceProjection[T, R any] struct {
	record  func(map[string]any) (*T, error)
	result  func(*T) R
	receipt func(resourceReceipt[T]) R
}

func mutateAndProjectResource[T, R any](ctx context.Context, service *Service, tool string, input resourceMutationInput, selectionErr error, writer resourceWriter, projection resourceProjection[T, R]) (loom.Nullable[R], error) {
	if selectionErr != nil {
		return loom.Nullable[R]{}, selectionErr
	}
	value, err := service.mutateResource(ctx, tool, input, writer)
	if err != nil {
		return loom.Nullable[R]{}, err
	}
	record, err := mutationRecord(value.Record, projection.record)
	if err != nil {
		return loom.Nullable[R]{}, err
	}
	return mutationResponse(input, value, record, projection.result, projection.receipt)
}

func dispatchResource[I, R any](ctx context.Context, input *I, invalid string, isRead func(*I) bool, read func(context.Context, *I) (*R, error), mutate func(context.Context, *I) (loom.Nullable[R], error)) (loom.Nullable[R], error) {
	if input == nil {
		return loom.Nullable[R]{}, api.MakeInvalidParams(errors.New(invalid))
	}
	if isRead(input) {
		return nullableRead(read(ctx, input))
	}
	return mutate(ctx, input)
}
