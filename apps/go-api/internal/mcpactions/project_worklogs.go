package mcpactions

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/worklogs"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

func (s *Service) projectWorklogSummary(ctx context.Context, input *api.ProjectWorklogSummary) (loom.Nullable[api.ProjectResult], error) {
	if input == nil || input.Action != "worklog_summary" {
		return loom.Nullable[api.ProjectResult]{}, api.MakeInvalidParams(errors.New("invalid project worklog summary"))
	}
	project, err := typeUUID(input.ProjectID)
	if err != nil {
		return loom.Nullable[api.ProjectResult]{}, err
	}
	ctx, actor, err := readActor(ctx, input.WorkspaceSlug)
	if err != nil {
		return loom.Nullable[api.ProjectResult]{}, err
	}
	raw, err := s.workLogs.Summary(ctx, worklogs.Query{ActorID: actor.UserID, WorkspaceID: actor.WorkspaceID, WorkspaceSlug: actor.WorkspaceSlug, ProjectID: project})
	if err != nil {
		return loom.Nullable[api.ProjectResult]{}, resourceReadError(err, projectmutations.ErrForbidden, projectmutations.ErrNotFound)
	}
	values := api.ProjectWorklogTotals{}
	if json.Unmarshal(raw, &values) != nil || values == nil {
		return loom.Nullable[api.ProjectResult]{}, workitemResultError()
	}
	return loom.NullableValue(api.NewProjectResultProjectWorklogTotals(values)), nil
}
