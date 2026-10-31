package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"go.opentelemetry.io/otel"
)

type propertyApplication interface {
	ReadPropertyValueAPI(context.Context, workitemtypes.PropertyValueAPIQuery) (json.RawMessage, error)
	ReadPropertyAPI(context.Context, workitemtypes.PropertyAPIQuery) (json.RawMessage, error)
	MutatePropertyAPI(context.Context, workitemtypes.PropertyAPIMutation) (projectmutations.Result, error)
}

func (s *Service) WorkitemProperty(ctx context.Context, input *api.WorkitemPropertyRequest) (result loom.Nullable[api.WorkitemPropertyResult], err error) {
	in, err := selectProperty(input)
	if err != nil {
		return result, err
	}
	if strings.HasSuffix(in.Action, "_value") {
		return s.propertyValue(ctx, in)
	}
	read := in.Action == "list" || in.Action == "retrieve" || in.Action == "list_options" || in.Action == "retrieve_option"
	ctx, actor, err := actionActor(ctx, in.Workspace, !read)
	if err != nil {
		return result, err
	}
	scope, err := typeScope(actor, in.Project)
	if err != nil {
		return result, err
	}
	q, err := in.query(scope)
	if err != nil {
		return result, err
	}
	var raw json.RawMessage
	if read {
		raw, err = s.properties.ReadPropertyAPI(ctx, q)
	} else {
		cmd, selectionErr := in.command(q)
		if selectionErr != nil {
			return result, selectionErr
		}
		var value projectmutations.Result
		value, err = s.mutateProperty(ctx, actor, in, cmd)
		if err == nil {
			if in.Request != nil || in.Revision != nil {
				return propertyMutationReceipt(value, in.Request != nil)
			}
			raw = value.Record
		}
	}
	if err != nil {
		return result, workitemTypeError(err)
	}
	if string(raw) == "null" {
		return loom.NullValue[api.WorkitemPropertyResult](), nil
	}
	return projectPropertyResult(raw, in.Action, actor.Timezone)
}
func (s *Service) mutateProperty(ctx context.Context, actor mcpauth.Principal, in propertyInput, cmd workitemtypes.PropertyAPIMutation) (value projectmutations.Result, err error) {
	started := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp.workitem_property."+in.Action)
	defer func() {
		defer span.End()
		fields := mutationActorFields(actor, "workitem_property", in.Action, err)
		fields["latency_ms"] = time.Since(started).Milliseconds()
		if err == nil {
			fields["committed_resource_id"] = value.ID
			if value.Replayed != nil {
				fields["replayed"] = *value.Replayed
			}
		}
		s.recordMutation(ctx, span, fields, err)
	}()
	cmd.Revalidate = func(ctx context.Context) error {
		_, current, err := actionActor(ctx, in.Workspace, true)
		if err != nil {
			return err
		}
		if current.UserID != actor.UserID || current.WorkspaceID != actor.WorkspaceID {
			return api.MakeForbidden(errors.New("authorized scope changed"))
		}
		return nil
	}
	return s.properties.MutatePropertyAPI(ctx, cmd)
}
func projectPropertyResult(raw json.RawMessage, action, timezone string) (loom.Nullable[api.WorkitemPropertyResult], error) {
	var data any
	if json.Unmarshal(raw, &data) != nil {
		return loom.Nullable[api.WorkitemPropertyResult]{}, workitemResultError()
	}
	data = projectNestedTimestamps(data, timezone)
	var result api.WorkitemPropertyResult
	var err error
	switch action {
	case "list", "list_options", "manage_type_properties":
		var value api.WorkitemPropertyArray
		err = decodeWorkitemProjection(data, &value)
		result = api.NewWorkitemPropertyResultWorkitemPropertyArray(value)
	default:
		if strings.HasSuffix(action, "_option") {
			var value api.WorkitemPropertyOptionRecord
			err = decodeWorkitemProjection(data, &value)
			result = api.NewWorkitemPropertyResultWorkitemPropertyOptionRecord(&value)
		} else {
			var value api.WorkitemPropertyRecord
			err = decodeWorkitemProjection(data, &value)
			result = api.NewWorkitemPropertyResultWorkitemPropertyRecord(&value)
		}
	}
	if err != nil {
		return loom.Nullable[api.WorkitemPropertyResult]{}, err
	}
	return loom.NullableValue(result), nil
}

func propertyMutationReceipt(value projectmutations.Result, requested bool) (result loom.Nullable[api.WorkitemPropertyResult], err error) {
	receipt := &api.WorkitemPropertyMutationReceipt{ResourceID: value.ID, Revision: value.Revision, Result: loom.NullValue[loom.JSONValue]()}
	if len(value.Record) > 0 && string(value.Record) != "null" {
		if json.Unmarshal(value.Record, &receipt.Result) != nil {
			return result, workitemResultError()
		}
	}
	if requested {
		if value.RequestID == nil || value.ReplayExpiresAt == nil || value.Replayed == nil {
			return result, workitemResultError()
		}
		id, expires := value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, value.Replayed
	}
	return loom.NullableValue(api.NewWorkitemPropertyResultWorkitemPropertyMutationReceipt(receipt)), nil
}
