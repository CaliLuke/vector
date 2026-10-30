package systemapi

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"strings"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_project_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/CaliLuke/loom/security"
)

const workspaceProjectInternalErrorDetail = "Something went wrong please try again later"

type workspaceProjectReadsRepository interface {
	UserWorkspaces(context.Context, uuid.UUID, store.WorkspaceOptions) (jsontext.Value, error)
	Workspaces(context.Context, uuid.UUID, store.WorkspaceOptions) (jsontext.Value, error)
	Workspace(context.Context, uuid.UUID, string, store.WorkspaceOptions) (jsontext.Value, error)
	ProjectStats(context.Context, uuid.UUID, string, []string, *string) (jsontext.Value, error)
	Projects(context.Context, uuid.UUID, string) (jsontext.Value, error)
	ProjectDetails(context.Context, uuid.UUID, string, store.ProjectDetailsOptions) (jsontext.Value, error)
	Project(context.Context, uuid.UUID, string, uuid.UUID) (jsontext.Value, error)
}

type workspaceProjectReadsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workspaceProjectRecentVisits interface {
	Record(context.Context, uuid.UUID, string, string, uuid.UUID, uuid.UUID) error
}

type workspaceProjectReadsService struct {
	repository   workspaceProjectReadsRepository
	sessions     workspaceProjectReadsSessions
	recentVisits workspaceProjectRecentVisits
}

type workspaceProjectReadsPrincipal struct{}

func NewWorkspaceProjectReads(repository workspaceProjectReadsRepository, sessions workspaceProjectReadsSessions, recentVisits workspaceProjectRecentVisits) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("workspace project reads repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	if recentVisits == nil {
		return nil, fmt.Errorf("workspace recent visits repository is required")
	}
	return &workspaceProjectReadsService{repository: repository, sessions: sessions, recentVisits: recentVisits}, nil
}

func (service *workspaceProjectReadsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, err := readServiceSessionUserUUID(ctx, key, service.sessions)
	if err != nil {
		return ctx, &api.WorkspaceProjectUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, workspaceProjectReadsPrincipal{}, userID), nil
}

func (service *workspaceProjectReadsService) AuthorizeMetadata(ctx context.Context, sessionID string) error {
	_, err := service.APIKeyAuth(ctx, sessionID, &security.APIKeyScheme{Name: "plane_session_cookie"})
	return err
}

func (service *workspaceProjectReadsService) UserWorkspaces(ctx context.Context, payload *api.UserWorkspacesPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	options, err := workspaceOptions(
		payload.Owner,
		readServiceOptionalQuery(ctx, "search", payload.Search),
	)
	if err != nil {
		return nil, nil, &api.WorkspaceProjectOwnerBadRequest{Owner: []string{"Enter a valid UUID."}}
	}
	data, err := service.repository.UserWorkspaces(ctx, userID, options)
	if err != nil {
		return nil, nil, service.repositoryError(err)
	}
	return workspaceProjectJSONResponse(data)
}

func (service *workspaceProjectReadsService) Workspaces(ctx context.Context, payload *api.WorkspacesPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	options, err := workspaceOptions(
		payload.Owner,
		readServiceOptionalQuery(ctx, "search", payload.Search),
	)
	if err != nil {
		return nil, nil, &api.WorkspaceProjectOwnerBadRequest{Owner: []string{"Enter a valid UUID."}}
	}
	data, err := service.repository.Workspaces(ctx, userID, options)
	if err != nil {
		return nil, nil, service.repositoryError(err)
	}
	return workspaceProjectJSONResponse(data)
}

func (service *workspaceProjectReadsService) Workspace(ctx context.Context, payload *api.WorkspacePayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	options, err := workspaceOptions(
		payload.Owner,
		readServiceOptionalQuery(ctx, "search", payload.Search),
	)
	if err != nil {
		return nil, nil, &api.WorkspaceProjectOwnerBadRequest{Owner: []string{"Enter a valid UUID."}}
	}
	data, err := service.repository.Workspace(ctx, userID, payload.Slug, options)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, &api.WorkspaceProjectDetailNotFound{Detail: "No Workspace matches the given query."}
	}
	if err != nil {
		return nil, nil, service.repositoryError(err)
	}
	return workspaceProjectJSONResponse(data)
}

func (service *workspaceProjectReadsService) ProjectStats(ctx context.Context, payload *api.ProjectStatsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.ProjectStats(
		ctx,
		userID,
		payload.Slug,
		commaValues(readServiceOptionalQuery(ctx, "fields", payload.Fields)),
		readServiceOptionalQuery(ctx, "project_ids", payload.ProjectIds),
	)
	if errors.Is(err, store.ErrBadRequest) {
		return nil, nil, &api.WorkspaceProjectValidationError{Message: "Please provide valid detail"}
	}
	if err != nil {
		return nil, nil, service.repositoryError(err)
	}
	return workspaceProjectJSONResponse(data)
}

func (service *workspaceProjectReadsService) Projects(ctx context.Context, payload *api.ProjectsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.Projects(ctx, userID, payload.Slug)
	if err != nil {
		return nil, nil, service.repositoryError(err)
	}
	return workspaceProjectJSONResponse(data)
}

func (service *workspaceProjectReadsService) ProjectDetails(ctx context.Context, payload *api.ProjectDetailsPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	data, err := service.repository.ProjectDetails(ctx, userID, payload.Slug, store.ProjectDetailsOptions{
		PerPage: readServiceOptionalQuery(ctx, "per_page", payload.PerPage),
		Cursor:  readServiceOptionalQuery(ctx, "cursor", payload.Cursor),
		OrderBy: readServiceOptionalQuery(ctx, "order_by", payload.OrderBy),
	})
	if errors.Is(err, store.ErrBadRequest) {
		detail := strings.TrimPrefix(err.Error(), store.ErrBadRequest.Error()+": ")
		if detail == err.Error() {
			detail = "Invalid cursor parameter."
		}
		return nil, nil, &api.WorkspaceProjectBadRequest{Detail: detail}
	}
	if err != nil {
		return nil, nil, service.repositoryError(err)
	}
	return workspaceProjectJSONResponse(data)
}

func (service *workspaceProjectReadsService) Project(ctx context.Context, payload *api.ProjectPayload) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, nil, err
	}
	projectID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, nil, &api.WorkspaceProjectNotFound{Message: "Project does not exist"}
	}
	data, err := service.repository.Project(ctx, userID, payload.Slug, projectID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, nil, &api.WorkspaceProjectNotFound{Message: "Project does not exist"}
	case errors.Is(err, store.ErrProjectForbidden):
		return nil, nil, &api.WorkspaceProjectForbidden{Message: "You do not have permission"}
	case errors.Is(err, store.ErrProjectConflict):
		return nil, nil, &api.WorkspaceProjectConflict{Message: "You are not a member of this project"}
	case err != nil:
		return nil, nil, service.repositoryError(err)
	}
	// Django intentionally treats recent-visit persistence as best effort.
	_ = service.recentVisits.Record(ctx, userID, payload.Slug, "project", projectID, projectID)
	return workspaceProjectJSONResponse(data)
}

func workspaceProjectJSONResponse(data jsontext.Value) (*api.RawJSONResponseMetadata, io.ReadCloser, error) {
	return &api.RawJSONResponseMetadata{ContentType: "application/json"}, rawJSONReadCloser(data), nil
}

func (service *workspaceProjectReadsService) UserWorkspacesOptions(ctx context.Context, _ *api.UserWorkspacesOptionsPayload) (*api.WorkspaceProjectOptions, error) {
	if _, err := service.userID(ctx); err != nil {
		return nil, err
	}
	return workspaceProjectOptions("User Work Spaces Endpoint"), nil
}

func (service *workspaceProjectReadsService) ProjectStatsOptions(ctx context.Context, _ *api.ProjectStatsOptionsPayload) (*api.WorkspaceProjectOptions, error) {
	if _, err := service.userID(ctx); err != nil {
		return nil, err
	}
	return workspaceProjectOptions("Project Stats Endpoint"), nil
}

func (service *workspaceProjectReadsService) ProjectDetailsOptions(ctx context.Context, _ *api.ProjectDetailsOptionsPayload) (*api.WorkspaceProjectOptions, error) {
	if _, err := service.userID(ctx); err != nil {
		return nil, err
	}
	return workspaceProjectOptions("Project"), nil
}

func workspaceProjectOptions(name string) *api.WorkspaceProjectOptions {
	return &api.WorkspaceProjectOptions{
		Name:        name,
		Description: "",
		Renders:     []string{"application/json"},
		Parses:      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	}
}

func (service *workspaceProjectReadsService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(workspaceProjectReadsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.WorkspaceProjectUnauthenticated{Detail: unauthenticatedDetail}
	}
	return userID, nil
}

func (service *workspaceProjectReadsService) repositoryError(err error) error {
	if errors.Is(err, store.ErrForbidden) {
		return &api.WorkspaceProjectForbidden{Message: "You don't have the required permissions."}
	}
	return service.internalError()
}

func (service *workspaceProjectReadsService) internalError() error {
	return &api.WorkspaceProjectInternalError{Message: workspaceProjectInternalErrorDetail}
}

func workspaceOptions(ownerValue, search *string) (store.WorkspaceOptions, error) {
	options := store.WorkspaceOptions{}
	if search != nil {
		options.SearchTerms = memberDirectorySearchPatterns(*search)
	}
	if ownerValue == nil || *ownerValue == "" {
		return options, nil
	}
	owner, err := uuid.Parse(*ownerValue)
	if err != nil {
		return store.WorkspaceOptions{}, err
	}
	options.Owner = &owner
	return options, nil
}

func commaValues(value *string) []string {
	if value == nil {
		return nil
	}
	return strings.Split(*value, ",")
}
