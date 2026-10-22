package systemapi

import (
	"context"
	"encoding/json"
	"fmt"
	api "github.com/CaliLuke/go-plane/gen/last_visited_workspace"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/lastvisitedworkspace"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

type lastVisitedWorkspaceRepository interface {
	Get(context.Context, uuid.UUID) (json.RawMessage, error)
}
type lastVisitedWorkspaceSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type lastVisitedWorkspaceService struct {
	repository lastVisitedWorkspaceRepository
	sessions   lastVisitedWorkspaceSessions
}
type lastVisitedWorkspacePrincipal struct{}

func NewLastVisitedWorkspace(r lastVisitedWorkspaceRepository, s lastVisitedWorkspaceSessions) (api.Service, error) {
	if r == nil || s == nil {
		return nil, fmt.Errorf("last visited workspace dependencies are required")
	}
	return &lastVisitedWorkspaceService{r, s}, nil
}
func (s *lastVisitedWorkspaceService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	p, e := s.sessions.Validate(ctx, key)
	if e != nil {
		return ctx, &api.LastVisitedWorkspaceUnauthenticated{Detail: unauthenticatedDetail}
	}
	id, e := uuid.Parse(p.UserID)
	if e != nil {
		return ctx, &api.LastVisitedWorkspaceUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, lastVisitedWorkspacePrincipal{}, id), nil
}
func (s *lastVisitedWorkspaceService) Get(ctx context.Context, _ *api.GetPayload) (*api.LastVisitedWorkspaceResult, error) {
	id, ok := ctx.Value(lastVisitedWorkspacePrincipal{}).(uuid.UUID)
	if !ok {
		return nil, &api.LastVisitedWorkspaceUnauthenticated{Detail: unauthenticatedDetail}
	}
	b, e := s.repository.Get(ctx, id)
	if e != nil {
		return nil, &api.LastVisitedWorkspaceInternal{Message: "Something went wrong please try again later"}
	}
	var v api.LastVisitedWorkspaceResult
	if e = json.Unmarshal(b, &v); e != nil {
		return nil, &api.LastVisitedWorkspaceInternal{Message: "Something went wrong please try again later"}
	}
	return &v, nil
}

var _ lastVisitedWorkspaceRepository = (*store.Repository)(nil)
