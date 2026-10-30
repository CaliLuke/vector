package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/instance_admin_session"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/loom/security"
)

const instanceAdminUnauthenticatedDetail = "Authentication credentials were not provided."
const instanceAdminForbiddenDetail = "You do not have permission to perform this action."

type instanceAdminSessionRepository interface {
	IsAdmin(context.Context, uuid.UUID) (bool, error)
	User(context.Context, uuid.UUID) (json.RawMessage, error)
}
type instanceAdminMembershipRepository interface {
	HasAdminMembership(context.Context, uuid.UUID) (bool, error)
}
type instanceAdminSessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type InstanceAdminSessionService struct {
	repository instanceAdminSessionRepository
	sessions   instanceAdminSessionValidator
}

func NewInstanceAdminSession(repository instanceAdminSessionRepository, sessions instanceAdminSessionValidator) (api.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("instance admin session repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("session validator for Django is required")
	}
	return &InstanceAdminSessionService{repository: repository, sessions: sessions}, nil
}

func (s *InstanceAdminSessionService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	id, err := s.sessionActor(ctx, key)
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, instanceAdminPrincipalKey{}, id), nil
}

// sessionActor separates session read failures from invalid credentials.
// Session inspection can recover only the latter as an anonymous response.
func (s *InstanceAdminSessionService) sessionActor(ctx context.Context, key string) (uuid.UUID, error) {
	principal, err := s.sessions.Validate(ctx, key)
	if djangosession.IsReadFailure(err) {
		return uuid.UUID{}, instanceAdminInternal()
	}
	if err != nil {
		return uuid.UUID{}, &api.InstanceAdminSessionUnauthenticated{Detail: instanceAdminUnauthenticatedDetail}
	}
	id, err := uuid.Parse(principal.UserID)
	if err != nil {
		return uuid.UUID{}, &api.InstanceAdminSessionUnauthenticated{Detail: instanceAdminUnauthenticatedDetail}
	}
	return id, nil
}

func (s *InstanceAdminSessionService) Me(ctx context.Context, _ *api.MePayload) (*api.InstanceAdminUser, error) {
	id, ok := ctx.Value(instanceAdminPrincipalKey{}).(uuid.UUID)
	if !ok {
		return nil, &api.InstanceAdminSessionUnauthenticated{Detail: instanceAdminUnauthenticatedDetail}
	}
	admin, err := s.repository.IsAdmin(ctx, id)
	if err != nil {
		return nil, instanceAdminInternal()
	}
	if !admin {
		return nil, &api.InstanceAdminSessionForbidden{Detail: instanceAdminForbiddenDetail}
	}
	return s.user(ctx, id)
}

func (s *InstanceAdminSessionService) Session(ctx context.Context, payload *api.SessionPayload) (*api.SessionResult, error) {
	if payload == nil || payload.SessionID == nil {
		return &api.SessionResult{}, nil
	}
	id, err := s.sessionActor(ctx, *payload.SessionID)
	var invalid *api.InstanceAdminSessionUnauthenticated
	if errors.As(err, &invalid) {
		return &api.SessionResult{}, nil
	}
	if err != nil {
		return nil, err
	}
	membership, ok := s.repository.(instanceAdminMembershipRepository)
	if !ok {
		return nil, instanceAdminInternal()
	}
	admin, err := membership.HasAdminMembership(ctx, id)
	if err != nil {
		return nil, instanceAdminInternal()
	}
	if !admin {
		return &api.SessionResult{}, nil
	}
	user, err := s.user(ctx, id)
	if err != nil {
		return nil, instanceAdminInternal()
	}
	return &api.SessionResult{IsAuthenticated: true, User: user}, nil
}

func (s *InstanceAdminSessionService) user(ctx context.Context, id uuid.UUID) (*api.InstanceAdminUser, error) {
	raw, err := s.repository.User(ctx, id)
	if err != nil {
		return nil, instanceAdminInternal()
	}
	var user api.InstanceAdminUser
	if err := json.Unmarshal(raw, &user); err != nil {
		return nil, instanceAdminInternal()
	}
	return &user, nil
}

type instanceAdminPrincipalKey struct{}

func instanceAdminInternal() error {
	return &api.InstanceAdminSessionInternalError{Message: "Something went wrong please try again later"}
}
