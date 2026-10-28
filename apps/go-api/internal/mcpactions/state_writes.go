package mcpactions

import (
	"context"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	states "github.com/CaliLuke/go-plane/internal/projectstatereads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"go.opentelemetry.io/otel"
)

func (s *Service) stateMutation(ctx context.Context, input stateInput) (result *api.StateResult, err error) {
	started := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp.state."+input.action)
	actor, _ := mcpauth.PrincipalFromContext(ctx)
	var value states.MutationResult
	defer func() {
		defer span.End()
		fields := mutationActorFields(actor, "state", input.action, err)
		fields["latency_ms"] = time.Since(started).Milliseconds()
		if err == nil {
			fields["committed_state_id"] = value.StateID
			if input.project != nil {
				fields["committed_project_id"] = *input.project
			}
			if value.Replayed != nil {
				fields["replayed"] = *value.Replayed
			}
		}
		s.recordMutation(ctx, span, fields, err)
	}()
	verified, current, authErr := actionActor(ctx, input.workspace, true)
	if authErr != nil {
		return nil, authErr
	}
	actor = current
	cmd := states.MutationCommand{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug,
		Action: input.action, Changes: input.changes, ExpectedRevision: input.revision}
	if input.project != nil && *input.project != "" {
		cmd.ProjectID, err = stateUUID(*input.project)
		if err != nil {
			return nil, err
		}
	}
	if input.target != "" {
		cmd.StateID, err = stateUUID(input.target)
		if err != nil {
			return nil, err
		}
	}
	if input.request != nil {
		id, parseErr := stateUUID(*input.request)
		if parseErr != nil {
			return nil, parseErr
		}
		cmd.RequestID = &id
	}
	cmd.Revalidate = func(ctx context.Context) error {
		_, fresh, authErr := actionActor(ctx, input.workspace, true)
		if authErr != nil {
			return authErr
		}
		if fresh.UserID != actor.UserID || fresh.WorkspaceID != actor.WorkspaceID {
			return api.MakeForbidden(errors.New("authorized scope changed"))
		}
		return nil
	}
	value, err = s.states.MutateState(verified, cmd)
	if err != nil {
		return nil, stateError(err)
	}
	return stateMutationResult(input, value, actor.Timezone)
}
func stateError(err error) error {
	switch {
	case errors.Is(err, states.ErrForbidden):
		return api.MakeForbidden(errors.New("state access denied"))
	case errors.Is(err, states.ErrNotFound):
		return api.MakeNotFound(errors.New("state unavailable"))
	case errors.Is(err, states.ErrInvalidMutation), errors.Is(err, mutationreceipts.ErrInvalidRequest):
		return stateInvalid()
	case errors.Is(err, states.ErrWorkspaceNotManaged):
		return api.MakeWorkspaceNotManaged(errors.New("workspace state writes unavailable"))
	case errors.Is(err, states.ErrRevisionConflict):
		return api.MakeRevisionConflict(errors.New("state revision changed"))

	}
	if receiptErr := receiptOperationError(err); receiptErr != nil {
		return receiptErr
	}
	var failure *states.WriteFailure
	if errors.As(err, &failure) {
		switch failure.Status {
		case 400:
			return stateInvalid()
		case 403:
			return api.MakeForbidden(errors.New("state access denied"))
		case 404:
			return api.MakeNotFound(errors.New("state unavailable"))
		case 409:
			return api.MakeConflict(errors.New("state conflict"))
		}
	}
	var serviceError *loom.ServiceError
	if errors.As(err, &serviceError) {
		return err
	}
	return api.MakeInternalError(errors.New("state operation failed"))
}
