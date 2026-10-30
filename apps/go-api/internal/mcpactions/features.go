package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/features"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type featureApplication interface {
	WorkspaceVisibility(context.Context, features.Query) (features.Capabilities, error)
	Read(context.Context, features.Query) (json.RawMessage, error)
	Mutate(context.Context, features.Command) (projectmutations.Result, error)
}

func (s *Service) featureOperation(ctx context.Context, input interface{ Validate() error }) (json.RawMessage, error) {
	if input.Validate() != nil {
		return nil, api.MakeInvalidParams(errors.New("invalid feature input"))
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, workitemResultError()
	}
	var in struct {
		Action          string  `json:"action"`
		Project         string  `json:"project_id"`
		Workspace       *string `json:"workspace_slug"`
		Request         *string `json:"request_id"`
		Revision        *string `json:"expected_revision"`
		IncludeRevision bool    `json:"include_revision"`
	}
	if json.Unmarshal(raw, &in) != nil {
		return nil, workitemResultError()
	}
	cmd := features.Command{Query: features.Query{IncludeRevision: in.IncludeRevision}, ExpectedRevision: in.Revision, Values: map[string]bool{}}
	if in.Project != "" {
		cmd.ProjectID, err = typeUUID(in.Project)
		if err != nil {
			return nil, err
		}
	}
	if in.Request != nil {
		id, typeUUIDErr := typeUUID(*in.Request)
		if typeUUIDErr != nil {
			return nil, typeUUIDErr
		}
		cmd.RequestID = &id
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return nil, workitemResultError()
	}
	for key, value := range fields {
		switch key {
		case "action", "project_id", "workspace_slug", "request_id", "expected_revision", "include_revision":
			continue
		}
		var flag *bool
		if json.Unmarshal(value, &flag) != nil {
			return nil, api.MakeInvalidParams(errors.New("feature flags must be booleans"))
		}
		if flag == nil {
			continue
		}
		if key == "workitem_types" {
			key = "work_item_types"
		}
		cmd.Values[key] = *flag
	}
	write := in.Action == "update_features"
	ctx, actor, err := actionActor(ctx, in.Workspace, write)
	if err != nil {
		return nil, err
	}
	cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug = actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug
	if !write {
		raw, err = s.features.Read(ctx, cmd.Query)
	} else {
		var value projectmutations.Result
		value, err = s.mutateFeatures(ctx, actor, in.Workspace, cmd)
		if err == nil {
			raw, err = resourceReceiptJSON(value, in.Request != nil || in.Revision != nil, in.Request != nil)
		}
	}
	if err != nil {
		return nil, archiveResourceError(err)
	}
	return raw, nil
}
func (s *Service) mutateFeatures(ctx context.Context, actor mcpauth.Principal, selector *string, cmd features.Command) (value projectmutations.Result, err error) {
	tool := "workspace"
	if cmd.ProjectID != uuid.Nil {
		tool = "project"
	}
	start := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp."+tool+".update_features")
	defer func() {
		defer span.End()
		fields := mutationActorFields(actor, tool, "update_features", err)
		fields["latency_ms"] = time.Since(start).Milliseconds()
		if err == nil {
			fields["committed_resource_id"] = value.ID
			if value.Replayed != nil {
				fields["replayed"] = *value.Replayed
			}
		}
		s.recordMutation(ctx, span, fields, err)
	}()
	cmd.Revalidate = mutationRevalidator(actor, selector, false)
	return s.features.Mutate(ctx, cmd)
}
