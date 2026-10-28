package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mcpauth"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
)

type projectReader interface {
	APIProjects(context.Context, uuid.UUID, string, projects.APIProjectListOptions) (json.RawMessage, error)
	APIProject(context.Context, uuid.UUID, string, uuid.UUID, projects.APIProjectDetailOptions) (json.RawMessage, error)
}

func (s *Service) Project(ctx context.Context, input *api.ProjectRequest) (loom.Nullable[api.ProjectResult], error) {
	if input != nil && (input.Kind() == api.ProjectRequestKindProjectGetFeatures || input.Kind() == api.ProjectRequestKindProjectUpdateFeatures) {
		raw, err := s.featureOperation(ctx, input)
		return decodeReleaseResult[api.ProjectResult](raw, err)
	}
	if input != nil {
		if summary, ok := input.AsProjectWorklogSummary(); ok {
			return s.projectWorklogSummary(ctx, summary)
		}
	}
	if input != nil && (input.Kind() == api.ProjectRequestKindProjectCreate || input.Kind() == api.ProjectRequestKindProjectUpdate || input.Kind() == api.ProjectRequestKindProjectDelete) {
		return s.projectRecordMutation(ctx, input)
	}
	result, err := s.project(ctx, input)
	if err != nil {
		return loom.Nullable[api.ProjectResult]{}, err
	}
	if result == nil {
		return loom.NullValue[api.ProjectResult](), nil
	}
	return loom.NullableValue(*result), nil
}

func (s *Service) project(ctx context.Context, input *api.ProjectRequest) (*api.ProjectResult, error) {
	if input == nil {
		return nil, api.MakeInvalidParams(errors.New("invalid project action"))
	}
	list, isList := input.AsProjectList()
	detail, isDetail := input.AsProjectRetrieve()
	archive, isArchive := input.AsProjectArchive()
	var workspace *string
	mutation := false
	switch {
	case isList && list != nil && list.Action == "list":
		workspace = list.WorkspaceSlug
	case isDetail && detail != nil && detail.Action == "retrieve":
		workspace = detail.WorkspaceSlug
	case isArchive && archive != nil && (archive.Action == "archive" || archive.Action == "unarchive"):
		workspace = archive.WorkspaceSlug
		mutation = true
	default:
		return nil, api.MakeInvalidParams(errors.New("invalid project action"))
	}
	if mutation {
		return s.projectMutation(ctx, archive)
	}
	ctx, actor, err := actionActor(ctx, workspace, false)
	if err != nil {
		return nil, err
	}
	if actor.WorkspaceSlug == "" {
		return nil, api.MakeInvalidParams(errors.New("workspace is required"))
	}
	if isList {
		return s.projectList(ctx, actor, list)
	}
	return s.projectDetail(ctx, actor, detail)
}

func (s *Service) projectList(ctx context.Context, actor mcpauth.Principal, input *api.ProjectList) (*api.ProjectResult, error) {
	size := 100
	if input.PerPage != nil && *input.PerPage != 0 {
		size = *input.PerPage
	}
	if size < 1 || size > 1000 {
		return nil, api.MakeInvalidParams(errors.New("invalid project page size"))
	}
	perPage := strconv.Itoa(size)
	workspaceID := uuid.UUID(actor.WorkspaceID)
	options := projects.APIProjectListOptions{PerPage: &perPage, Cursor: input.Cursor, OrderBy: input.OrderBy, IncludeArchived: true, ExpectedWorkspaceID: &workspaceID}
	if options.Cursor != nil && *options.Cursor == "" {
		options.Cursor = nil
	}
	body, err := s.projectReads.APIProjects(ctx, uuid.UUID(actor.UserID), actor.WorkspaceSlug, options)
	if err != nil {
		return nil, projectReadError(err)
	}
	var page *api.ProjectPage
	if err := json.Unmarshal(body, &page); err != nil || page == nil {
		return nil, projectReadFailure()
	}
	result := api.NewProjectResultProjectPage(page)
	return &result, nil
}

func (s *Service) projectDetail(ctx context.Context, actor mcpauth.Principal, input *api.ProjectRetrieve) (*api.ProjectResult, error) {
	id, err := uuid.Parse(input.ProjectID)
	if err != nil {
		return nil, api.MakeInvalidParams(errors.New("invalid project identifier"))
	}
	workspaceID := uuid.UUID(actor.WorkspaceID)
	body, err := s.projectReads.APIProject(ctx, uuid.UUID(actor.UserID), actor.WorkspaceSlug, id, projects.APIProjectDetailOptions{ExpectedWorkspaceID: &workspaceID, IncludeRevision: input.IncludeRevision != nil && *input.IncludeRevision})
	if err != nil {
		return nil, projectReadError(err)
	}
	// The HTTP detail omits per-member sort_order; the pinned SDK defaults it to null.
	detail := &api.ProjectDetail{SortOrder: loom.NullValue[float64]()}
	if err := json.Unmarshal(body, &detail); err != nil || detail == nil {
		return nil, projectReadFailure()
	}
	if input.IncludeRevision != nil && *input.IncludeRevision && detail.Revision == nil {
		return nil, projectReadFailure()
	}
	result := api.NewProjectResultProjectDetail(detail)
	return &result, nil
}

func projectReadError(err error) error {
	switch {
	case errors.Is(err, projects.ErrForbidden):
		return api.MakeForbidden(errors.New("project access denied"))
	case errors.Is(err, projects.ErrNotFound):
		return api.MakeNotFound(errors.New("project unavailable"))
	case errors.Is(err, projects.ErrBadRequest):
		return api.MakeInvalidParams(errors.New("invalid project query"))
	}
	var failure *djangoapi.HTTPError
	if errors.As(err, &failure) && failure.Status == http.StatusBadRequest {
		return api.MakeInvalidParams(errors.New("invalid project query"))
	}
	return projectReadFailure()
}

func projectReadFailure() error {
	return api.MakeInternalError(errors.New("project read failed"))
}
