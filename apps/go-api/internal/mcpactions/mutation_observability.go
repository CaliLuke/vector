package mcpactions

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/platform"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/CaliLuke/loom/clue/log"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Option func(*Service)

// WithLogContext supplies application log outputs while retaining request trace
// correlation. Arguments, credential material and raw failures are never logged.
func WithLogContext(ctx context.Context) Option {
	return func(s *Service) { s.logContext = ctx }
}

func (s *Service) projectMutation(ctx context.Context, input *api.ProjectArchive) (result *api.ProjectResult, err error) {
	started := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp.project."+input.Action)
	actor, _ := mcpauth.PrincipalFromContext(ctx)
	defer func() {
		defer span.End()
		fields := mutationFields(actor, input, result, err)
		fields["latency_ms"] = time.Since(started).Milliseconds()
		s.recordMutation(ctx, span, fields, err)
	}()
	verified, current, authErr := actionActor(ctx, input.WorkspaceSlug, true)
	if authErr != nil {
		return nil, authErr
	}
	actor = current
	if actor.WorkspaceID == uuid.Nil {
		return nil, api.MakeInvalidParams(errors.New("workspace is required"))
	}
	return s.archiveProject(verified, actor, input)
}

func mutationFields(actor mcpauth.Principal, input *api.ProjectArchive, result *api.ProjectResult, err error) map[string]any {
	fields := mutationActorFields(actor, "project", input.Action, err)
	if err == nil {
		id, _ := uuid.Parse(input.ProjectID)
		fields["committed_project_id"] = id.String()
		if result != nil {
			if receipt, ok := result.AsProjectMutationReceipt(); ok && receipt != nil && receipt.Replayed != nil {
				fields["replayed"] = *receipt.Replayed
			}
		}
	}
	return fields
}

func mutationActorFields(actor mcpauth.Principal, tool, action string, err error) map[string]any {
	fields := map[string]any{"mcp_tool": tool, "mcp_action": action, "outcome": mutationOutcome(err), "replayed": false}
	if actor.UserID != uuid.Nil {
		fields["actor_id"], fields["credential_class"] = actor.UserID.String(), actor.CredentialClass()
	}
	if actor.WorkspaceID != uuid.Nil {
		fields["workspace_id"] = actor.WorkspaceID.String()
	}
	if actor.ClientID != "" {
		fields["client_id"] = actor.ClientID
	}
	return fields
}

func (s *Service) recordMutation(ctx context.Context, span trace.Span, fields map[string]any, err error) {
	if err != nil {
		span.SetStatus(codes.Error, mutationOutcome(err))
	}
	if requestID, ok := platform.RequestID(ctx); ok {
		fields[log.RequestIDKey] = requestID
	}
	attributes := make([]attribute.KeyValue, 0, len(fields))
	logFields := []log.Fielder{log.KV{K: log.MessageKey, V: "MCP mutation completed"}}
	for key, value := range fields {
		logFields = append(logFields, log.KV{K: key, V: value})
		switch v := value.(type) {
		case string:
			attributes = append(attributes, attribute.String(key, v))
		case bool:
			attributes = append(attributes, attribute.Bool(key, v))
		case int64:
			attributes = append(attributes, attribute.Int64(key, v))
		}
	}
	span.SetAttributes(attributes...)
	logger := ctx
	if s.logContext != nil {
		logger = log.WithContext(ctx, s.logContext)
	}
	if err != nil {
		log.Warn(logger, logFields...)
	} else {
		log.Info(logger, logFields...)
	}
}

func mutationOutcome(err error) string {
	if err == nil {
		return "committed"
	}
	var serviceError *loom.ServiceError
	if errors.As(err, &serviceError) && slices.Contains([]string{"unauthorized", "forbidden", "not_found", "invalid_params", "request_conflict", "conflict", "revision_conflict", "indeterminate"}, serviceError.Name) {
		return serviceError.Name
	}
	return "internal_error"
}
