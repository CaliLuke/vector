package mcpactions

import (
	"context"
	"errors"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
)

type projectResourceReader interface {
	ReadAPIV1(context.Context, string, apicontract.ProjectReadRequest) (any, error)
}

func readProjectResource[Input, Result any](ctx context.Context, s *Service, input *Input, selectAction func(*Input) (apicontract.ProjectReadRequest, *string, error), reader projectResourceReader, forbidden, notFound error, project func(any, string) (*Result, error)) (*Result, error) {
	request, workspace, err := selectAction(input)
	if err != nil {
		return nil, err
	}
	ctx, actor, err := s.scopedReadActor(ctx, workspace)
	if err != nil {
		return nil, err
	}
	workspaceID := uuid.UUID(actor.WorkspaceID)
	request.ExpectedWorkspaceID = &workspaceID
	request.Params["slug"] = actor.WorkspaceSlug
	value, err := reader.ReadAPIV1(ctx, actor.UserID.String(), request)
	if err != nil {
		return nil, resourceReadError(err, forbidden, notFound)
	}
	result, err := project(apicontract.Timestamps(value, actor.Timezone), request.Operation)
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
	var failure *apicontract.HTTPError
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

// Compact lists default to a small page and can continue using only the cursor.
func compactPageSize(cursor *string) (int, error) {
	if cursor == nil || *cursor == "" {
		return 20, nil
	}
	size, _, _, err := apicontract.CursorWindow(cursor)
	if err != nil {
		return 0, api.MakeInvalidParams(errors.New("invalid pagination cursor"))
	}
	return size, nil
}
