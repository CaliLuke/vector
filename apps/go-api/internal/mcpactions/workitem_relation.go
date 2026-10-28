package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemrelations"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"go.opentelemetry.io/otel"
)

type relationApplication interface {
	Read(context.Context, workitemrelations.Query) (json.RawMessage, error)
	Mutate(context.Context, workitemrelations.Command) (projectmutations.Result, error)
}

func (s *Service) WorkitemRelation(ctx context.Context, input *api.WorkitemRelationRequest) (result loom.Nullable[api.WorkitemRelationResult], err error) {
	in, err := selectRelation(input)
	if err != nil {
		return result, err
	}
	write := in.command.Action != "list" && in.command.Action != "list_definitions"
	ctx, actor, err := actionActor(ctx, in.workspace, write)
	if err != nil {
		return result, err
	}
	in.command.ActorID, in.command.WorkspaceID, in.command.WorkspaceSlug = actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug
	var raw json.RawMessage
	if !write {
		raw, err = s.relations.Read(ctx, in.command.Query)
	} else {
		var value projectmutations.Result
		value, err = s.mutateRelation(ctx, actor, in)
		if err == nil {
			if in.request != nil || in.command.ExpectedRevision != nil {
				receipt := &api.RelationMutationReceipt{ResourceID: value.ID, Revision: value.Revision, Result: loom.NullValue[loom.JSONValue]()}
				if len(value.Record) > 0 && string(value.Record) != "null" {
					var projected any
					if json.Unmarshal(value.Record, &projected) != nil {
						return result, workitemResultError()
					}
					if err = decodeWorkitemProjection(projectNestedTimestamps(projected, actor.Timezone), &receipt.Result); err != nil {
						return result, err
					}
				}
				if in.request != nil {
					if value.RequestID == nil || value.Replayed == nil || value.ReplayExpiresAt == nil {
						return result, workitemResultError()
					}
					id, expires := value.RequestID.String(), value.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
					receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, value.Replayed
				}
				return loom.NullableValue(api.NewWorkitemRelationResultRelationMutationReceipt(receipt)), nil
			}
			raw = value.Record
		}
	}
	if err != nil {
		return result, archiveResourceError(err)
	}
	if string(raw) == "null" {
		return loom.NullValue[api.WorkitemRelationResult](), nil
	}
	return projectRelationResult(raw, in.command.Action, actor.Timezone)
}
func (s *Service) mutateRelation(ctx context.Context, actor mcpauth.Principal, in relationInput) (value projectmutations.Result, err error) {
	started := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp.workitem_relation."+in.command.Action)
	defer func() {
		defer span.End()
		fields := mutationActorFields(actor, "workitem_relation", in.command.Action, err)
		fields["latency_ms"] = time.Since(started).Milliseconds()
		if err == nil {
			fields["committed_resource_id"] = value.ID
			if value.Replayed != nil {
				fields["replayed"] = *value.Replayed
			}
		}
		s.recordMutation(ctx, span, fields, err)
	}()
	in.command.Revalidate = func(ctx context.Context) error {
		_, current, err := actionActor(ctx, in.workspace, true)
		if err != nil {
			return err
		}
		if current.UserID != actor.UserID || current.WorkspaceID != actor.WorkspaceID {
			return api.MakeForbidden(errors.New("authorized scope changed"))
		}
		return nil
	}
	return s.relations.Mutate(ctx, in.command)
}
func projectNestedTimestamps(value any, timezone string) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			typed[key] = projectNestedTimestamps(child, timezone)
		}
		return djangoapi.Timestamps(typed, timezone)
	case []any:
		for index, child := range typed {
			typed[index] = projectNestedTimestamps(child, timezone)
		}
	}
	return value
}
func projectRelationResult(raw json.RawMessage, action, timezone string) (loom.Nullable[api.WorkitemRelationResult], error) {
	var data any
	if json.Unmarshal(raw, &data) != nil {
		return loom.Nullable[api.WorkitemRelationResult]{}, workitemResultError()
	}
	encoded, err := json.Marshal(projectNestedTimestamps(data, timezone))
	if err != nil {
		return loom.Nullable[api.WorkitemRelationResult]{}, workitemResultError()
	}
	var result api.WorkitemRelationResult
	switch action {
	case "list":
		var value api.RelationCollection
		err = json.Unmarshal(encoded, &value)
		result = api.NewWorkitemRelationResultRelationCollection(&value)
	case "list_definitions":
		var value api.RelationDefinitionCollection
		err = json.Unmarshal(encoded, &value)
		result = api.NewWorkitemRelationResultRelationDefinitionCollection(&value)
	case "create":
		value := api.RelationItems{}
		err = json.Unmarshal(encoded, &value)
		result = api.NewWorkitemRelationResultRelationItems(value)
	default:
		var value api.RelationDefinitionRecord
		err = json.Unmarshal(encoded, &value)
		result = api.NewWorkitemRelationResultRelationDefinitionRecord(&value)
	}
	if err != nil {
		return loom.Nullable[api.WorkitemRelationResult]{}, workitemResultError()
	}
	return loom.NullableValue(result), nil
}
