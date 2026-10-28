package mcpactions

import (
	"context"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

func (s *Service) observeMutation(ctx context.Context, actor mcpauth.Principal, tool, action string, mutate func(context.Context) (projectmutations.Result, error)) (value projectmutations.Result, err error) {
	start := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp."+tool+"."+action)
	defer func() {
		defer span.End()
		fields := mutationActorFields(actor, tool, action, err)
		fields["latency_ms"] = time.Since(start).Milliseconds()
		if err == nil {
			fields["committed_resource_id"] = value.ID
			if value.Replayed != nil {
				fields["replayed"] = *value.Replayed
			}
		}
		s.recordMutation(ctx, span, fields, err)
	}()
	return mutate(ctx)
}

func mutationRevalidator(actor mcpauth.Principal, selector *string, requireWorkspace bool) func(context.Context) error {
	return func(ctx context.Context) error {
		_, current, err := actionActor(ctx, selector, true)
		if err != nil {
			return err
		}
		if current.UserID != actor.UserID || current.WorkspaceID != actor.WorkspaceID || (requireWorkspace && current.WorkspaceID == uuid.Nil) {
			return api.MakeForbidden(errors.New("authorized scope changed"))
		}
		return nil
	}
}
