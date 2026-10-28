package mcpactions

import (
	"context"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	"github.com/CaliLuke/go-plane/internal/modulereads"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type resourceWriter interface {
	MutateResource(context.Context, projectmutations.Command) (projectmutations.Result, error)
}
type resourceMutationInput struct {
	action, project, target, parent string
	workspace, request, revision    *string
	fields                          map[string]any
}

func (s *Service) mutateResource(ctx context.Context, tool string, input resourceMutationInput, writer resourceWriter) (value projectmutations.Result, err error) {
	started := time.Now()
	ctx, span := otel.Tracer("vector.mcp.actions").Start(ctx, "mcp."+tool+"."+input.action)
	actor, _ := mcpauth.PrincipalFromContext(ctx)
	defer func() {
		defer span.End()
		fields := mutationActorFields(actor, tool, input.action, err)
		fields["latency_ms"] = time.Since(started).Milliseconds()
		if err == nil {
			fields["committed_"+tool+"_id"] = value.ID
			fields["committed_project_id"] = input.project
			if value.Replayed != nil {
				fields["replayed"] = *value.Replayed
			}
		}
		s.recordMutation(ctx, span, fields, err)
	}()
	verified, current, authErr := actionActor(ctx, input.workspace, true)
	if authErr != nil {
		return value, authErr
	}
	actor = current
	cmd := projectmutations.Command{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug, Action: input.action, ExpectedRevision: input.revision, Fields: input.fields, Timezone: actor.Timezone}
	if tool != "project" || input.action != "create" {
		cmd.ProjectID, err = uuid.Parse(input.project)
		if err != nil || cmd.ProjectID == uuid.Nil {
			return value, api.MakeInvalidParams(errors.New("invalid project"))
		}
	}
	if input.action != "create" {
		cmd.ResourceID, err = uuid.Parse(input.target)
		if err != nil || cmd.ResourceID == uuid.Nil {
			return value, api.MakeInvalidParams(errors.New("invalid target"))
		}
	}
	if input.parent != "" {
		cmd.ParentID, err = uuid.Parse(input.parent)
		if err != nil || cmd.ParentID == uuid.Nil {
			return value, api.MakeInvalidParams(errors.New("invalid work item identifier"))
		}
	}
	if input.request != nil {
		id, parseErr := uuid.Parse(*input.request)
		if parseErr != nil || id == uuid.Nil {
			return value, api.MakeInvalidParams(errors.New("invalid request"))
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
	value, err = writer.MutateResource(verified, cmd)
	if err != nil {
		return value, archiveResourceError(err)
	}
	return value, nil
}
func archiveResourceError(err error) error {
	switch {
	case errors.Is(err, projectmutations.ErrForbidden), errors.Is(err, projects.ErrForbidden), errors.Is(err, cyclereads.ErrForbidden), errors.Is(err, modulereads.ErrForbidden):
		return api.MakeForbidden(errors.New("resource access denied"))
	case errors.Is(err, projectmutations.ErrNotFound), errors.Is(err, projects.ErrNotFound), errors.Is(err, cyclereads.ErrNotFound), errors.Is(err, modulereads.ErrNotFound):
		return api.MakeNotFound(errors.New("resource unavailable"))
	case errors.Is(err, projectmutations.ErrInvalid), errors.Is(err, mutationreceipts.ErrInvalidRequest):
		return api.MakeInvalidParams(errors.New("invalid mutation"))
	case errors.Is(err, projectmutations.ErrRevision), errors.Is(err, projects.ErrRevisionConflict):
		return api.MakeRevisionConflict(errors.New("resource revision changed"))
	case errors.Is(err, mutationreceipts.ErrConflict):
		return api.MakeRequestConflict(errors.New("request input changed"))
	case errors.Is(err, mutationreceipts.ErrIndeterminate):
		return api.MakeIndeterminate(errors.New("unacknowledged commit"))
	}
	var failure *djangoapi.HTTPError
	if errors.As(err, &failure) {
		switch failure.Status {
		case 400, 409:
			return api.MakeInvalidParams(errors.New("mutation rejected"))
		case 403:
			return api.MakeForbidden(errors.New("resource access denied"))
		case 404:
			return api.MakeNotFound(errors.New("resource unavailable"))
		}
	}
	var configuration *projectconfiguration.HTTPError
	if errors.As(err, &configuration) {
		switch configuration.Status {
		case 400, 409:
			return api.MakeInvalidParams(errors.New("mutation rejected"))
		case 403:
			return api.MakeForbidden(errors.New("resource access denied"))
		case 404:
			return api.MakeNotFound(errors.New("resource unavailable"))
		}
	}
	var validation *modulereads.ModuleValidationError
	if errors.As(err, &validation) {
		return api.MakeInvalidParams(errors.New("invalid module fields"))
	}
	var serviceError *loom.ServiceError
	if errors.As(err, &serviceError) {
		return err
	}
	return api.MakeInternalError(errors.New("resource mutation failed"))
}

func planningFields(add, remove *string) (map[string]any, error) {
	fields := map[string]any{}
	total := 0
	for key, input := range map[string]*string{"add_ids": add, "remove_ids": remove} {
		ids, err := milestoneIDs(input)
		if err != nil {
			return nil, api.MakeInvalidParams(errors.New("invalid work item identifiers"))
		}
		values := make([]string, len(ids))
		for i, id := range ids {
			values[i] = id.String()
		}
		fields[key] = values
		total += len(values)
	}
	if total == 0 {
		return nil, api.MakeInvalidParams(errors.New("add_ids or remove_ids is required"))
	}
	return fields, nil
}

type resourceMutationFunc func(context.Context, projectmutations.Command) (projectmutations.Result, error)

func (f resourceMutationFunc) MutateResource(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	return f(ctx, cmd)
}
