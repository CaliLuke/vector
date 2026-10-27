package mcpactions

import (
	"context"
	"errors"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

type projectResourceReader interface {
	ReadAPIV1(context.Context, string, djangoapi.ProjectReadRequest) (any, error)
}

func readProjectResource[Input, Result any](ctx context.Context, input *Input, selectAction func(*Input) (djangoapi.ProjectReadRequest, *string, error), reader projectResourceReader, forbidden, notFound error, project func(any, string) (*Result, error)) (*Result, error) {
	request, workspace, err := selectAction(input)
	if err != nil {
		return nil, err
	}
	ctx, actor, err := readActor(ctx, workspace)
	if err != nil {
		return nil, err
	}
	if actor.WorkspaceSlug == "" {
		return nil, api.MakeInvalidParams(errors.New("workspace is required"))
	}
	workspaceID := uuid.UUID(actor.WorkspaceID)
	request.ExpectedWorkspaceID = &workspaceID
	request.Params["slug"] = actor.WorkspaceSlug
	value, err := reader.ReadAPIV1(ctx, actor.UserID.String(), request)
	if err != nil {
		return nil, resourceReadError(err, forbidden, notFound)
	}
	result, err := project(djangoapi.Timestamps(value, actor.Timezone), request.Operation)
	if err != nil {
		return nil, api.MakeInternalError(errors.New("resource projection unavailable"))
	}
	return result, nil
}
func resourceReadError(err, forbidden, notFound error) error {
	switch {
	case errors.Is(err, forbidden):
		return api.MakeForbidden(errors.New("resource access denied"))
	case errors.Is(err, notFound):
		return api.MakeNotFound(errors.New("resource unavailable"))
	}
	var failure *djangoapi.HTTPError
	if errors.As(err, &failure) {
		switch failure.Status {
		case 400:
			return api.MakeInvalidParams(errors.New("invalid resource query"))
		case 403:
			return api.MakeForbidden(errors.New("resource access denied"))
		case 404:
			return api.MakeNotFound(errors.New("resource unavailable"))
		}
	}
	return api.MakeInternalError(errors.New("resource read unavailable"))
}
