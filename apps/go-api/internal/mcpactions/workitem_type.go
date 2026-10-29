package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type typeApplication interface {
	ReadAPI(context.Context, workitemtypes.APIRead) (json.RawMessage, error)
	MutateAPI(context.Context, workitemtypes.APIMutation) (projectmutations.Result, error)
	APIVisibility(context.Context, workitemtypes.APIScope) (workitemtypes.APICapabilities, error)
}

func (s *Service) WorkitemType(ctx context.Context, input *api.WorkitemTypeRequest) (loom.Nullable[api.WorkitemTypeResult], error) {
	selected, err := selectWorkitemType(input)
	if err != nil {
		return loom.Nullable[api.WorkitemTypeResult]{}, err
	}
	if selected.action == "list" || selected.action == "retrieve" {
		return s.readWorkitemType(ctx, selected)
	}
	return s.mutateWorkitemType(ctx, selected)
}
func typeScope(actor mcpauth.Principal, project *string) (workitemtypes.APIScope, error) {
	scope := workitemtypes.APIScope{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug}
	if project != nil && *project != "" {
		id, err := typeUUID(*project)
		if err != nil {
			return scope, err
		}
		scope.ProjectID = id
	}
	return scope, nil
}
func typeUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, api.MakeInvalidParams(errors.New("invalid identifier"))
	}
	return id, nil
}
func (s *Service) readWorkitemType(ctx context.Context, in workitemTypeInput) (loom.Nullable[api.WorkitemTypeResult], error) {
	ctx, actor, err := actionActor(ctx, in.workspace, false)
	if err != nil {
		return loom.Nullable[api.WorkitemTypeResult]{}, err
	}
	scope, err := typeScope(actor, in.project)
	if err != nil {
		return loom.Nullable[api.WorkitemTypeResult]{}, err
	}
	query := workitemtypes.APIRead{APIScope: scope, IncludeRevision: in.includeRevision}
	if in.action == "retrieve" {
		query.TypeID, err = typeUUID(in.target)
		if err != nil {
			return loom.Nullable[api.WorkitemTypeResult]{}, err
		}
	}
	if in.action == "list" && scope.ProjectID != uuid.Nil {
		params := djangoapi.Query{Params: map[string]string{}, Present: map[string]bool{}}
		if in.perPage != nil && *in.perPage != 0 {
			params.Params["per_page"] = strconv.Itoa(*in.perPage)
			params.Present["per_page"] = true
		}
		if in.cursor != nil && *in.cursor != "" {
			params.Params["cursor"] = *in.cursor
			params.Present["cursor"] = true
		}
		query.Limit, err = djangoapi.PageSizeWithMinimum(params, 1)
		if err != nil {
			return loom.Nullable[api.WorkitemTypeResult]{}, api.MakeInvalidParams(errors.New("invalid page size"))
		}
		page, parseErr := djangoapi.CursorPage(params, query.Limit)
		if parseErr != nil || page < 0 || page > math.MaxInt/query.Limit {
			return loom.Nullable[api.WorkitemTypeResult]{}, api.MakeInvalidParams(errors.New("invalid cursor"))
		}
		query.Offset = page * query.Limit
	}
	raw, err := s.types.ReadAPI(ctx, query)
	if err != nil {
		return loom.Nullable[api.WorkitemTypeResult]{}, workitemTypeError(err)
	}
	if in.action == "list" {
		var values api.WorkitemTypes
		if json.Unmarshal(raw, &values) != nil || values == nil {
			return loom.Nullable[api.WorkitemTypeResult]{}, typeResultError()
		}
		return loom.NullableValue(api.NewWorkitemTypeResultWorkitemTypes(values)), nil
	}
	record, err := typeRecord(raw)
	if err != nil {
		return loom.Nullable[api.WorkitemTypeResult]{}, err
	}
	if in.includeRevision && record.Revision == nil {
		return loom.Nullable[api.WorkitemTypeResult]{}, typeResultError()
	}
	return loom.NullableValue(api.NewWorkitemTypeResultWorkitemTypeRecord(record)), nil
}
func (s *Service) mutateWorkitemType(ctx context.Context, in workitemTypeInput) (result loom.Nullable[api.WorkitemTypeResult], err error) {
	started := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp.workitem_type."+in.action)
	actor, _ := mcpauth.PrincipalFromContext(ctx)
	var value projectmutations.Result
	defer func() {
		defer span.End()
		fields := mutationActorFields(actor, "workitem_type", in.action, err)
		fields["latency_ms"] = time.Since(started).Milliseconds()
		if err == nil {
			fields["committed_resource_id"] = value.ID
			if value.Replayed != nil {
				fields["replayed"] = *value.Replayed
			}
		}
		s.recordMutation(ctx, span, fields, err)
	}()
	verified, current, authErr := actionActor(ctx, in.workspace, true)
	if authErr != nil {
		return result, authErr
	}
	actor = current
	scope, err := typeScope(actor, in.project)
	if err != nil {
		return result, err
	}
	cmd := workitemtypes.APIMutation{APIScope: scope, Action: in.action, Changes: in.changes, TypeIDs: in.typeIDs, ExpectedRevision: in.revision}
	if in.target != "" {
		cmd.TypeID, err = typeUUID(in.target)
		if err != nil {
			return result, err
		}
	}
	if in.request != nil {
		id, parseErr := typeUUID(*in.request)
		if parseErr != nil {
			return result, parseErr
		}
		cmd.RequestID = &id
	}
	cmd.Revalidate = mutationRevalidator(actor, in.workspace, false)
	value, err = s.types.MutateAPI(verified, cmd)
	if err != nil {
		return result, workitemTypeError(err)
	}
	var record *api.WorkitemTypeRecord
	if in.action != "delete" && in.action != "import_to_project" {
		record, err = typeRecord(value.Record)
		if err != nil {
			return result, err
		}
	}
	if in.request == nil && in.revision == nil {
		if record == nil {
			return loom.NullValue[api.WorkitemTypeResult](), nil
		}
		return loom.NullableValue(api.NewWorkitemTypeResultWorkitemTypeRecord(record)), nil
	}
	receipt := &api.WorkitemTypeMutationReceipt{ResourceID: value.ID, Revision: value.Revision, Result: loom.NullValue[api.WorkitemTypeRecord]()}
	if record != nil {
		receipt.Result = loom.NullableValue(*record)
	}
	if in.request != nil {
		if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
			return result, typeResultError()
		}
		id, expires := value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, value.Replayed
	}
	return loom.NullableValue(api.NewWorkitemTypeResultWorkitemTypeMutationReceipt(receipt)), nil
}
func typeRecord(raw json.RawMessage) (*api.WorkitemTypeRecord, error) {
	var record *api.WorkitemTypeRecord
	if json.Unmarshal(raw, &record) != nil || record == nil {
		return nil, typeResultError()
	}
	return record, nil
}
func typeResultError() error {
	return api.MakeInternalError(errors.New("invalid work item type result"))
}
func workitemTypeError(err error) error {
	if errors.Is(err, workitemtypes.ErrWorkspaceOwned) {
		return api.MakeWorkspaceOwned(errors.New("types are workspace owned"))
	}
	var native *workitemtypes.Error
	if errors.As(err, &native) {
		switch native.Status {
		case 400, 422:
			return api.MakeInvalidParams(errors.New(native.Message))
		case 403:
			return api.MakeForbidden(errors.New("work item type access denied"))
		case 404:
			return api.MakeNotFound(errors.New("work item type unavailable"))
		case 409:
			return api.MakeConflict(errors.New(native.Message))
		}
	}
	return archiveResourceError(err)
}
