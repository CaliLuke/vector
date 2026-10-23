package systemapi

import (
	"context"
	"encoding/json"
	"fmt"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_recent_visits"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacerecentvisits"
	"github.com/CaliLuke/loom/security"
)

type workspaceRecentVisitsRepository interface {
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	List(context.Context, uuid.UUID, string, *string) (json.RawMessage, error)
}

type workspaceRecentVisitsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workspaceRecentVisitsService struct {
	repository workspaceRecentVisitsRepository
	sessions   workspaceRecentVisitsSessions
}

type workspaceRecentVisitsPrincipal struct{}

func NewWorkspaceRecentVisits(repository workspaceRecentVisitsRepository, sessions workspaceRecentVisitsSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("workspace recent visits dependencies are required")
	}
	return &workspaceRecentVisitsService{repository: repository, sessions: sessions}, nil
}

func (s *workspaceRecentVisitsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := s.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.WorkspaceRecentVisitUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.WorkspaceRecentVisitUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, workspaceRecentVisitsPrincipal{}, userID), nil
}

func (s *workspaceRecentVisitsService) authorize(ctx context.Context, slug string) (uuid.UUID, error) {
	userID, ok := ctx.Value(workspaceRecentVisitsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.WorkspaceRecentVisitUnauthenticated{Detail: unauthenticatedDetail}
	}
	authorized, err := s.repository.Authorized(ctx, userID, slug)
	if err != nil {
		return uuid.Nil(), recentVisitsInternalError()
	}
	if !authorized {
		return uuid.Nil(), &api.WorkspaceRecentVisitForbidden{Message: "You don't have the required permissions."}
	}
	return userID, nil
}

func (s *workspaceRecentVisitsService) List(ctx context.Context, payload *api.ListPayload) ([]*api.WorkspaceRecentVisit, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	var entityName *string
	// Django only applies this filter for a truthy query parameter; an empty
	// entity_name is equivalent to omitting it.
	if payload.EntityName != nil && *payload.EntityName != "" {
		entityName = payload.EntityName
	}
	body, err := s.repository.List(ctx, userID, payload.Slug, entityName)
	if err != nil {
		return nil, recentVisitsInternalError()
	}
	var result []*api.WorkspaceRecentVisit
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, recentVisitsInternalError()
	}
	return result, nil
}

func recentVisitsInternalError() error {
	return &api.WorkspaceRecentVisitInternal{Message: "Something went wrong please try again later"}
}

var _ workspaceRecentVisitsRepository = (*store.Repository)(nil)
