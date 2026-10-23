package systemapi

import (
	"context"
	"encoding/json"
	"fmt"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_member_me"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacememberme"
	"github.com/CaliLuke/loom/security"
)

type workspaceMemberMeRepo interface {
	Get(context.Context, uuid.UUID, string) (json.RawMessage, error)
}
type workspaceMemberMeSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type workspaceMemberMeService struct {
	repo     workspaceMemberMeRepo
	sessions workspaceMemberMeSessions
}

func NewWorkspaceMemberMe(r workspaceMemberMeRepo, s workspaceMemberMeSessions) (api.Service, error) {
	if r == nil || s == nil {
		return nil, fmt.Errorf("dependencies required")
	}
	return &workspaceMemberMeService{r, s}, nil
}
func (s *workspaceMemberMeService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	p, e := s.sessions.Validate(ctx, key)
	if e != nil {
		return ctx, &api.WorkspaceMemberMeUnauthenticated{Detail: unauthenticatedDetail}
	}
	id, e := uuid.Parse(p.UserID)
	if e != nil {
		return ctx, &api.WorkspaceMemberMeUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, memberMePrincipal{}, id), nil
}
func (s *workspaceMemberMeService) Get(ctx context.Context, p *api.GetPayload) (*api.WorkspaceMemberMe, error) {
	id, ok := ctx.Value(memberMePrincipal{}).(uuid.UUID)
	if !ok {
		return nil, &api.WorkspaceMemberMeUnauthenticated{Detail: unauthenticatedDetail}
	}
	b, e := s.repo.Get(ctx, id, p.Slug)
	if e != nil {
		return nil, fmt.Errorf("internal")
	}
	var out api.WorkspaceMemberMe
	if e = json.Unmarshal(b, &out); e != nil {
		return nil, fmt.Errorf("internal")
	}
	return &out, nil
}

type memberMePrincipal struct{}

var _ workspaceMemberMeRepo = (*store.Repository)(nil)
